// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	app core.App
	// temporalClient returns the client of the caller's namespace, used to
	// look up the caller's parent session.
	temporalClient func(namespace string) (client.Client, error)
}

// NewSaveAuthoredWalletActionActivity binds approved-action persistence to the application.
func NewSaveAuthoredWalletActionActivity(app core.App) *SaveAuthoredWalletActionActivity {
	return &SaveAuthoredWalletActionActivity{
		BaseActivity:   workflowengine.BaseActivity{Name: SaveAuthoredWalletActionActivityName},
		app:            app,
		temporalClient: temporalclient.GetTemporalClientWithNamespace,
	}
}

// Name returns the registered Temporal activity name.
func (a *SaveAuthoredWalletActionActivity) Name() string { return a.BaseActivity.Name }

// saveCaller is the workflow execution that scheduled the save.
type saveCaller struct {
	Namespace    string
	WorkflowType string
	WorkflowID   string
	RunID        string
}

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
	info := activity.GetInfo(ctx)
	receipt, err := a.save(ctx, saveCaller{
		Namespace:    info.Namespace,
		WorkflowType: info.WorkflowType.Name,
		WorkflowID:   info.WorkflowExecution.ID,
		RunID:        info.WorkflowExecution.RunID,
	}, payload)
	if err != nil {
		return result, err
	}
	result.Output = receipt
	return result, nil
}

// save accepts only the ActionAuthor child of a running session in the
// caller's namespace, and saves on behalf of the session's actor, build and
// platform rather than those of the payload.
func (a *SaveAuthoredWalletActionActivity) save(
	ctx context.Context,
	caller saveCaller,
	payload actionauthor.SaveRequest,
) (actionauthor.Receipt, error) {
	session, err := a.callerSession(ctx, caller)
	if err != nil {
		return actionauthor.Receipt{}, err
	}
	if !matchesSession(payload.ActorID, session.ActorID) ||
		!matchesSession(payload.ActorCollection, session.ActorCollection) ||
		!matchesSession(payload.WalletVersionIdentifier, session.WalletVersionIdentifier) ||
		!matchesSession(payload.Platform, session.Platform) {
		return actionauthor.Receipt{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: save request does not match the session", actionauthor.ErrForbidden,
		))
	}
	payload.BuildRequest = session
	receipt, err := actionauthor.Save(
		a.app,
		caller.Namespace,
		caller.WorkflowID,
		caller.RunID,
		payload,
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

// callerSession returns the build request of the running session whose
// ActionAuthor child is the caller, taken from the session's start input.
func (a *SaveAuthoredWalletActionActivity) callerSession(
	ctx context.Context,
	caller saveCaller,
) (actionauthor.BuildRequest, error) {
	var none actionauthor.BuildRequest
	forbidden := func(reason string) error {
		return actionAuthorError(
			&a.BaseActivity,
			fmt.Errorf("%w: %s", actionauthor.ErrForbidden, reason),
		)
	}
	sessionID, isAuthor := strings.CutSuffix(caller.WorkflowID, ActionAuthorWorkflowIDSuffix)
	if caller.Namespace == "" || caller.WorkflowType != ActionAuthorWorkflowType || !isAuthor ||
		len(sessionID) <= len(ActionAuthorSessionWorkflowIDPrefix) ||
		!strings.HasPrefix(sessionID, ActionAuthorSessionWorkflowIDPrefix) {
		return none, forbidden("caller is not an action author session agent")
	}
	temporalClient, err := a.temporalClient(caller.Namespace)
	if err != nil {
		return none, a.lookupError(err)
	}
	self, err := temporalClient.DescribeWorkflowExecution(ctx, caller.WorkflowID, caller.RunID)
	if err != nil {
		return none, a.describeError(err, forbidden)
	}
	parent := self.GetWorkflowExecutionInfo().GetParentExecution()
	if self.GetWorkflowExecutionInfo().GetType().GetName() != ActionAuthorWorkflowType ||
		parent.GetWorkflowId() != sessionID || parent.GetRunId() == "" {
		return none, forbidden("caller is not the child of its action author session")
	}
	described, err := temporalClient.DescribeWorkflowExecution(ctx, sessionID, parent.GetRunId())
	if err != nil {
		return none, a.describeError(err, forbidden)
	}
	sessionInfo := described.GetWorkflowExecutionInfo()
	if sessionInfo.GetType().GetName() != ActionAuthorSessionWorkflowName ||
		sessionInfo.GetStatus() != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return none, forbidden("action author session is not running")
	}
	history := temporalClient.GetWorkflowHistory(
		ctx,
		sessionID,
		parent.GetRunId(),
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	if !history.HasNext() {
		return none, forbidden("action author session has no start input")
	}
	event, err := history.Next()
	if err != nil {
		return none, a.lookupError(err)
	}
	started := event.GetWorkflowExecutionStartedEventAttributes()
	var input workflowengine.WorkflowInput
	if started == nil ||
		temporalcrypto.DataConverter().FromPayloads(started.GetInput(), &input) != nil {
		return none, forbidden("action author session has no start input")
	}
	// The session's SessionInput carries the build request fields.
	session, err := workflowengine.DecodePayload[actionauthor.BuildRequest](input.Payload)
	if err != nil || session.ActorID == "" || session.ActorCollection == "" ||
		session.WalletVersionIdentifier == "" || session.Platform == "" {
		return none, forbidden("action author session has no actor or build")
	}
	return session, nil
}

// describeError rejects an unknown execution and retries transient failures.
func (a *SaveAuthoredWalletActionActivity) describeError(
	err error,
	forbidden func(string) error,
) error {
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return forbidden("action author session not found")
	}
	return a.lookupError(err)
}

func (a *SaveAuthoredWalletActionActivity) lookupError(err error) error {
	return a.NewCodedError(
		errorcodes.UnexpectedWorkflowError,
		true,
		fmt.Errorf("look up action author session: %w", err),
	)
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
