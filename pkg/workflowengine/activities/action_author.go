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
	authorSessionLookup
	app core.App
}

// NewResolveActionAuthorBuildActivity binds build resolution to the application's database.
func NewResolveActionAuthorBuildActivity(app core.App) *ResolveActionAuthorBuildActivity {
	return &ResolveActionAuthorBuildActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: ResolveActionAuthorBuildActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(app),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *ResolveActionAuthorBuildActivity) Name() string { return a.BaseActivity.Name }

// Execute resolves the build of the calling session workflow.
func (a *ResolveActionAuthorBuildActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[actionauthor.BuildRequest](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	build, err := a.resolve(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = build
	return result, nil
}

// resolve accepts only a running session workflow that Credimi signed, whose
// build request is the payload, and resolves the session's build in its own
// organization namespace.
func (a *ResolveActionAuthorBuildActivity) resolve(
	ctx context.Context,
	caller authorCaller,
	payload actionauthor.BuildRequest,
) (actionauthor.Build, error) {
	if caller.Namespace == "" ||
		caller.WorkflowType != ActionAuthorSessionWorkflowName ||
		!strings.HasPrefix(caller.WorkflowID, ActionAuthorSessionWorkflowIDPrefix) {
		return actionauthor.Build{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: caller is not an action author session", actionauthor.ErrForbidden,
		))
	}
	temporalClient, err := a.temporalClient(caller.Namespace)
	if err != nil {
		return actionauthor.Build{}, lookupError(
			&a.BaseActivity,
			fmt.Errorf("look up action author session: %w", err),
		)
	}
	ref, err := lookupSessionStart(
		ctx,
		a.app,
		temporalClient,
		caller.Namespace,
		caller.WorkflowID,
		caller.RunID,
	)
	if errors.Is(err, actionauthor.ErrForbidden) {
		return actionauthor.Build{}, actionAuthorError(&a.BaseActivity, err)
	}
	if err != nil {
		return actionauthor.Build{}, lookupError(&a.BaseActivity, err)
	}
	if payload.ActorID != ref.Session.ActorID ||
		payload.WalletVersionIdentifier != ref.Session.WalletVersionIdentifier ||
		payload.Platform != ref.Session.Platform {
		return actionauthor.Build{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: build request does not match the session", actionauthor.ErrForbidden,
		))
	}
	build, err := actionauthor.Resolve(a.app, caller.Namespace, ref.Session)
	if err != nil {
		return actionauthor.Build{}, actionAuthorError(&a.BaseActivity, err)
	}
	return build, nil
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
		authorSessionLookup: newAuthorSessionLookup(app),
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
	// app derives the key that checks the session's grant.
	app core.App
	// temporalClient returns the client of the caller's namespace, used to
	// look up the caller's parent session.
	temporalClient func(namespace string) (client.Client, error)
}

func newAuthorSessionLookup(app core.App) authorSessionLookup {
	return authorSessionLookup{
		app:            app,
		temporalClient: temporalclient.GetTemporalClientWithNamespace,
	}
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
	ref, err := LookupActionAuthorSession(
		ctx,
		l.app,
		temporalClient,
		caller.Namespace,
		caller.WorkflowID,
		caller.RunID,
	)
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
// taken from the session's start input, after checking Credimi's grant for
// the session in namespace. Errors wrapping actionauthor.ErrForbidden reject
// the workflow; others are transient.
func LookupActionAuthorSession(
	ctx context.Context,
	app core.App,
	c client.Client,
	namespace string,
	authorWorkflowID string,
	authorRunID string,
) (ActionAuthorSessionRef, error) {
	sessionID, isAuthor := actionAuthorSessionID(authorWorkflowID)
	if !isAuthor {
		return ActionAuthorSessionRef{}, sessionForbidden(
			"caller is not an action author session agent",
		)
	}
	self, err := c.DescribeWorkflowExecution(ctx, authorWorkflowID, authorRunID)
	if err != nil {
		return ActionAuthorSessionRef{}, describeLookupError(err)
	}
	parent := self.GetWorkflowExecutionInfo().GetParentExecution()
	if self.GetWorkflowExecutionInfo().GetType().GetName() != ActionAuthorWorkflowType ||
		parent.GetWorkflowId() != sessionID || parent.GetRunId() == "" {
		return ActionAuthorSessionRef{}, sessionForbidden(
			"caller is not the child of its action author session",
		)
	}
	return lookupSessionStart(ctx, app, c, namespace, sessionID, parent.GetRunId())
}

// sessionForbidden rejects a session lookup.
func sessionForbidden(reason string) error {
	return fmt.Errorf("%w: %s", actionauthor.ErrForbidden, reason)
}

// lookupSessionStart returns the running session's start input after
// checking Credimi's grant for it.
func lookupSessionStart(
	ctx context.Context,
	app core.App,
	c client.Client,
	namespace, sessionID, sessionRunID string,
) (ActionAuthorSessionRef, error) {
	var ref ActionAuthorSessionRef
	described, err := c.DescribeWorkflowExecution(ctx, sessionID, sessionRunID)
	if err != nil {
		return ref, describeLookupError(err)
	}
	sessionInfo := described.GetWorkflowExecutionInfo()
	if sessionInfo.GetType().GetName() != ActionAuthorSessionWorkflowName ||
		sessionInfo.GetStatus() != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return ref, sessionForbidden("action author session is not running")
	}
	history := c.GetWorkflowHistory(
		ctx,
		sessionID,
		sessionRunID,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	if !history.HasNext() {
		return ref, sessionForbidden("action author session has no start input")
	}
	event, err := history.Next()
	if err != nil {
		return ref, fmt.Errorf("look up action author session: %w", err)
	}
	started := event.GetWorkflowExecutionStartedEventAttributes()
	var input workflowengine.WorkflowInput
	if started == nil ||
		temporalcrypto.DataConverter().FromPayloads(started.GetInput(), &input) != nil {
		return ref, sessionForbidden("action author session has no start input")
	}
	// The session's SessionInput carries the build request, the device, the
	// app and Credimi's grant; its other fields are ignored.
	session, err := workflowengine.DecodePayload[actionauthor.BuildRequest](input.Payload)
	if err != nil || session.ActorID == "" ||
		session.WalletVersionIdentifier == "" || session.Platform == "" {
		return ref, sessionForbidden("action author session has no actor or build")
	}
	start, err := workflowengine.DecodePayload[struct {
		DeviceID string `json:"device_id"`
		AppID    string `json:"app_id"`
		Grant    string `json:"grant"`
	}](input.Payload)
	if err != nil {
		return ref, sessionForbidden("action author session has no device")
	}
	if err := actionauthor.VerifySession(app, start.Grant, actionauthor.SessionGrant{
		SessionWorkflowID:       sessionID,
		Namespace:               namespace,
		DeviceID:                start.DeviceID,
		Platform:                session.Platform,
		WalletVersionIdentifier: session.WalletVersionIdentifier,
		ActorID:                 session.ActorID,
		AppID:                   start.AppID,
	}); err != nil {
		return ref, err
	}
	return ActionAuthorSessionRef{
		SessionWorkflowID: sessionID,
		SessionRunID:      sessionRunID,
		Session:           session,
		DeviceID:          start.DeviceID,
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
