// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
)

// ResolveActionAuthorBuildActivityName is the organization worker's build-resolution activity.
const ResolveActionAuthorBuildActivityName = "Resolve action author build"

// SaveAuthoredWalletActionActivityName is the organization worker's approved-action writer.
const SaveAuthoredWalletActionActivityName = "Save authored wallet action"

// ActionAuthorSessionWorkflowName is the organization worker's action author
// session workflow (pkg/workflowengine/authorsession), which a granted
// action_author semaphore ticket starts.
const ActionAuthorSessionWorkflowName = "Action author session"

// Identity of the Python agent's ActionAuthor child workflow that the
// session workflow (pkg/workflowengine/authorsession) starts with the ID
// "<session workflow ID>-author"; the session's ID is
// "ActionAuthorSession-<ticket>".
const (
	ActionAuthorWorkflowType            = "ActionAuthor"
	ActionAuthorWorkflowIDSuffix        = "-author"
	ActionAuthorSessionWorkflowIDPrefix = "ActionAuthorSession-"
)

// ResolveActionAuthorBuildActivity authorizes and resolves the workflow's selected build.
type ResolveActionAuthorBuildActivity struct {
	workflowengine.BaseActivity
	app core.App
}

// NewResolveActionAuthorBuildActivity binds build resolution to the application's database.
func NewResolveActionAuthorBuildActivity(app core.App) *ResolveActionAuthorBuildActivity {
	return &ResolveActionAuthorBuildActivity{
		BaseActivity: workflowengine.BaseActivity{Name: ResolveActionAuthorBuildActivityName},
		app:          app,
	}
}

// Name returns the registered Temporal activity name.
func (a *ResolveActionAuthorBuildActivity) Name() string { return a.BaseActivity.Name }

// Execute resolves the selected build in the activity's actual organization namespace.
func (a *ResolveActionAuthorBuildActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[actionauthor.BuildRequest](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	info := activity.GetInfo(ctx)
	if info.Namespace == "" {
		return result, actionAuthorError(&a.BaseActivity, actionauthor.ErrForbidden)
	}
	build, err := actionauthor.Resolve(a.app, info.Namespace, payload)
	if err != nil {
		return result, actionAuthorError(&a.BaseActivity, err)
	}
	result.Output = build
	return result, nil
}

// SaveAuthoredWalletActionActivity persists a candidate approved through the workflow.
type SaveAuthoredWalletActionActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewSaveAuthoredWalletActionActivity binds approved-action persistence to the application.
func NewSaveAuthoredWalletActionActivity(app core.App) *SaveAuthoredWalletActionActivity {
	return &SaveAuthoredWalletActionActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: SaveAuthoredWalletActionActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *SaveAuthoredWalletActionActivity) Name() string { return a.BaseActivity.Name }

// Execute saves the candidate with the activity's actual namespace and workflow identity.
func (a *SaveAuthoredWalletActionActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[actionauthor.SaveRequest](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	receipt, err := a.save(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = receipt
	return result, nil
}

// save accepts only the ActionAuthor child of a running session in the
// caller's namespace and a Credimi-signed approval of the candidate and plan
// by the session's actor, and saves on behalf of the session's actor, build
// and platform rather than those of the payload, with the sent plan checked
// again for the session's build.
func (a *SaveAuthoredWalletActionActivity) save(
	ctx context.Context,
	caller authorCaller,
	payload actionauthor.SaveRequest,
) (actionauthor.Receipt, error) {
	ref, err := a.callerSession(ctx, &a.BaseActivity, caller)
	if err != nil {
		return actionauthor.Receipt{}, err
	}
	session := ref.Session
	if !matchesSession(payload.ActorID, session.ActorID) ||
		!matchesSession(payload.WalletVersionIdentifier, session.WalletVersionIdentifier) ||
		!matchesSession(payload.Platform, session.Platform) {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: save request does not match the session", actionauthor.ErrForbidden,
		))
	}
	selection, err := actionauthor.Select(a.app, caller.Namespace, session)
	if err != nil {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, err)
	}
	plan, err := SessionPlan(a.app, selection, payload.Spec)
	if err != nil {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, err)
	}
	payload.BuildRequest = session
	payload.Spec = plan
	approval, err := actionauthor.VerifyApproval(
		a.app,
		payload.Approval,
		caller.WorkflowID,
		payload.CandidateSHA256,
		time.Now(),
	)
	if err != nil {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, err)
	}
	if approval.ApproverID != session.ActorID {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: the approval is not from the session's user", actionauthor.ErrForbidden,
		))
	}
	if approval.PlanSHA256 != actionauthor.PlanSHA256(payload.Spec) {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: approval is for another plan", actionauthor.ErrForbidden,
		))
	}
	receipt, err := actionauthor.Save(
		a.app,
		caller.Namespace,
		caller.WorkflowID,
		caller.RunID,
		payload,
		approval,
	)
	if err != nil {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, err)
	}
	return receipt, nil
}

// matchesSession accepts an omitted payload value or the session's own.
func matchesSession(requested, session string) bool {
	return requested == "" || requested == session
}

// authorCaller is the workflow execution that scheduled an action author activity.
type authorCaller struct {
	Namespace    string
	WorkflowType string
	WorkflowID   string
	RunID        string
}

// activityCaller returns the workflow execution that scheduled the activity.
func activityCaller(ctx context.Context) authorCaller {
	info := activity.GetInfo(ctx)
	return authorCaller{
		Namespace:    info.Namespace,
		WorkflowType: info.WorkflowType.Name,
		WorkflowID:   info.WorkflowExecution.ID,
		RunID:        info.WorkflowExecution.RunID,
	}
}

// authorSessionLookup finds the running session of an ActionAuthor caller.
type authorSessionLookup struct {
	// temporalClient returns the client of the caller's namespace, used to
	// look up the caller's parent session.
	temporalClient func(namespace string) (client.Client, error)
}

func newAuthorSessionLookup() authorSessionLookup {
	return authorSessionLookup{temporalClient: temporalclient.GetTemporalClientWithNamespace}
}

// callerSession accepts only the ActionAuthor child of a running session in
// the caller's namespace and returns the session and its start input.
func (l authorSessionLookup) callerSession(
	ctx context.Context,
	base *workflowengine.BaseActivity,
	caller authorCaller,
) (ActionAuthorSessionRef, error) {
	if _, isAuthor := actionAuthorSessionID(caller.WorkflowID); caller.Namespace == "" ||
		caller.WorkflowType != ActionAuthorWorkflowType || !isAuthor {
		return ActionAuthorSessionRef{}, actionAuthorError(base, fmt.Errorf(
			"%w: caller is not an action author session agent", actionauthor.ErrForbidden,
		))
	}
	temporalClient, err := l.temporalClient(caller.Namespace)
	if err != nil {
		return ActionAuthorSessionRef{}, lookupError(
			base,
			fmt.Errorf("look up action author session: %w", err),
		)
	}
	ref, err := LookupActionAuthorSession(ctx, temporalClient, caller.WorkflowID, caller.RunID)
	if errors.Is(err, actionauthor.ErrForbidden) {
		return ActionAuthorSessionRef{}, actionAuthorError(base, err)
	}
	if err != nil {
		return ActionAuthorSessionRef{}, lookupError(base, err)
	}
	return ref, nil
}

// actionAuthorSessionID returns the session workflow ID of an ActionAuthor
// workflow ID, "<session workflow ID>-author".
func actionAuthorSessionID(authorWorkflowID string) (string, bool) {
	sessionID, isAuthor := strings.CutSuffix(authorWorkflowID, ActionAuthorWorkflowIDSuffix)
	return sessionID, isAuthor && len(sessionID) > len(ActionAuthorSessionWorkflowIDPrefix) &&
		strings.HasPrefix(sessionID, ActionAuthorSessionWorkflowIDPrefix)
}

// ActionAuthorSessionRef identifies a running session, its build request and
// the device it runs on.
type ActionAuthorSessionRef struct {
	SessionWorkflowID string
	SessionRunID      string
	Session           actionauthor.BuildRequest
	DeviceID          string
}

// LookupActionAuthorSession returns the running session whose ActionAuthor
// child is the given workflow (its latest run when authorRunID is empty),
// taken from the session's start input. Errors wrapping
// actionauthor.ErrForbidden reject the workflow; others are transient.
func LookupActionAuthorSession(
	ctx context.Context,
	c client.Client,
	authorWorkflowID string,
	authorRunID string,
) (ActionAuthorSessionRef, error) {
	var ref ActionAuthorSessionRef
	forbidden := func(reason string) error {
		return fmt.Errorf("%w: %s", actionauthor.ErrForbidden, reason)
	}
	sessionID, isAuthor := actionAuthorSessionID(authorWorkflowID)
	if !isAuthor {
		return ref, forbidden("caller is not an action author session agent")
	}
	self, err := c.DescribeWorkflowExecution(ctx, authorWorkflowID, authorRunID)
	if err != nil {
		return ref, describeLookupError(err)
	}
	parent := self.GetWorkflowExecutionInfo().GetParentExecution()
	if self.GetWorkflowExecutionInfo().GetType().GetName() != ActionAuthorWorkflowType ||
		parent.GetWorkflowId() != sessionID || parent.GetRunId() == "" {
		return ref, forbidden("caller is not the child of its action author session")
	}
	described, err := c.DescribeWorkflowExecution(ctx, sessionID, parent.GetRunId())
	if err != nil {
		return ref, describeLookupError(err)
	}
	sessionInfo := described.GetWorkflowExecutionInfo()
	if sessionInfo.GetType().GetName() != ActionAuthorSessionWorkflowName ||
		sessionInfo.GetStatus() != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return ref, forbidden("action author session is not running")
	}
	history := c.GetWorkflowHistory(
		ctx,
		sessionID,
		parent.GetRunId(),
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	if !history.HasNext() {
		return ref, forbidden("action author session has no start input")
	}
	event, err := history.Next()
	if err != nil {
		return ref, fmt.Errorf("look up action author session: %w", err)
	}
	started := event.GetWorkflowExecutionStartedEventAttributes()
	var input workflowengine.WorkflowInput
	if started == nil ||
		temporalcrypto.DataConverter().FromPayloads(started.GetInput(), &input) != nil {
		return ref, forbidden("action author session has no start input")
	}
	// The session's SessionInput carries the build request and the device;
	// its other fields are ignored.
	session, err := workflowengine.DecodePayload[actionauthor.BuildRequest](input.Payload)
	if err != nil || session.ActorID == "" ||
		session.WalletVersionIdentifier == "" || session.Platform == "" {
		return ref, forbidden("action author session has no actor or build")
	}
	device, err := workflowengine.DecodePayload[struct {
		DeviceID string `json:"device_id"`
	}](input.Payload)
	if err != nil {
		return ref, forbidden("action author session has no device")
	}
	return ActionAuthorSessionRef{
		SessionWorkflowID: sessionID,
		SessionRunID:      parent.GetRunId(),
		Session:           session,
		DeviceID:          device.DeviceID,
	}, nil
}

// describeLookupError rejects an unknown execution and retries transient failures.
func describeLookupError(err error) error {
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return fmt.Errorf("%w: action author session not found", actionauthor.ErrForbidden)
	}
	return fmt.Errorf("look up action author session: %w", err)
}

// lookupError retries a session lookup that failed for a transient reason.
func lookupError(base *workflowengine.BaseActivity, err error) error {
	return base.NewCodedError(errorcodes.UnexpectedWorkflowError, true, err)
}

// actionAuthorError makes authorization, validation and approval failures
// non-retryable and leaves database and storage failures retryable.
func actionAuthorError(a *workflowengine.BaseActivity, err error) error {
	switch {
	case errors.Is(err, actionauthor.ErrForbidden):
		return a.NewCodedError(errorcodes.RecordNotAccessible, false, err)
	case errors.Is(err, actionauthor.ErrInvalid):
		return a.NewCodedError(errorcodes.MissingOrInvalidPayload, false, err)
	case errors.Is(err, actionauthor.ErrConflict):
		return a.NewCodedError(errorcodes.DatabaseOperationFailed, false, err)
	default:
		return a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
}
