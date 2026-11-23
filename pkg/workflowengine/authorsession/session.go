// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package authorsession runs one action author session on a runner device:
// it prepares the device with the pipeline's own code, runs the runner's
// Python agent as a child workflow, and cleans and releases the device on
// every exit.
package authorsession

import (
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// WorkflowName is the registered name of the session workflow.
	WorkflowName = activities.ActionAuthorSessionWorkflowName
	// AuthorWorkflowType is the Python agent's workflow on the runner.
	AuthorWorkflowType = activities.ActionAuthorWorkflowType
	// AuthorTaskQueueSuffix follows the runner ID in the Python agent's task queue.
	AuthorTaskQueueSuffix = "-ActionAuthor"
	// ReleaseDeviceActivityName is the Python agent's activity that frees the
	// session's device: it deletes the iOS simulator clones, or stops the
	// Android device's Maestro process.
	ReleaseDeviceActivityName = "action_author.release_device"
	// StepID labels the prepared device in errors and run data.
	StepID = "action-author"
	// ReleaseDeviceSignal asks the session to clean and release its device
	// while the agent keeps running (e.g. while the user reviews). A second
	// signal is a no-op.
	ReleaseDeviceSignal = "action_author_release_device"

	authorWorkflowIDSuffix     = activities.ActionAuthorWorkflowIDSuffix
	authorExecutionTimeout     = 2 * time.Hour
	fcafAuthorExecutionTimeout = 8 * time.Hour
	deviceTypeIOSSimulator     = "ios_simulator"
	resultSuccess              = "success"
	resultFailed               = "failed"
	resultCanceled             = "canceled"
	releaseStartToClose        = 2 * time.Minute
	releaseScheduleToStart     = 30 * time.Second
	releaseMaximumAttempts     = 3
	cleanupStartToClose        = 10 * time.Minute
	cleanupScheduleToClose     = 20 * time.Minute
	cleanupHeartbeatTimeout    = 30 * time.Second
	cleanupMaximumAttempts     = 3
	sessionOutputAuthorKey     = "author_workflow_id"
	sessionOutputDeviceKey     = "device"
	// authorReadySignal is sent by the Python agent once its workflow runs.
	authorReadySignal = "action_author_ready"
	// agentReadyTimeout bounds the wait for authorReadySignal.
	agentReadyTimeout = 3 * time.Minute
)

// SessionInput selects the device and build of a session.
type SessionInput struct {
	DeviceID string `json:"device_id"`
	// Platform is "android" or "ios", derived from the device type.
	Platform                string `json:"platform"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	ActorID                 string `json:"actor_id"`
	// Namespace is the runner namespace the session runs in and
	// Organization the canonified_name of the caller's organization. Both
	// are bound by TicketGrant and Grant; the workflow does not read them.
	Namespace    string `json:"namespace"`
	Organization string `json:"organization"`
	// TicketGrant is Credimi's signature of the enqueued session
	// (actionauthor.SignTicket), set by POST /api/action-author/sessions
	// after its checks. The activity that starts the session accepts only a
	// ticket signed for exactly the session it starts, and replaces it with
	// Grant.
	TicketGrant string `json:"ticket_grant,omitempty"`
	// Grant is Credimi's signature of the session (actionauthor.SignSession),
	// set by the activity that starts the session (never in the semaphore
	// ticket) and checked by every Credimi activity the session and its
	// agent call. The workflow never reads it.
	Grant string `json:"grant,omitempty"`
	// Mode is actionauthor.SessionModeFCAF for an FCAF authoring session,
	// empty for chat; FCAFSuite is the suite an FCAF session authors.
	Mode      string `json:"mode,omitempty"`
	FCAFSuite string `json:"fcaf_suite,omitempty"`
}

// AuthoringBuild is the resolved build the agent authors against. Store
// builds have no version or installer digest: the agent installs PackageID
// from StoreURL at session start.
type AuthoringBuild struct {
	WalletID          string `json:"wallet_id"`
	VersionID         string `json:"version_id"`
	VersionIdentifier string `json:"version_identifier"`
	OwnerNamespace    string `json:"owner_namespace"`
	Platform          string `json:"platform"`
	InstallerSHA256   string `json:"installer_sha256"`
	Source            string `json:"source"`
	PackageID         string `json:"package_id,omitempty"`
	StoreURL          string `json:"store_url,omitempty"`
}

// AuthoringDevice is the prepared runner device the agent drives.
type AuthoringDevice struct {
	DeviceID      string `json:"device_id"`
	RunnerID      string `json:"runner_id"`
	Serial        string `json:"serial"`
	Type          string `json:"type"`
	PackageID     string `json:"package_id"`
	InstallerPath string `json:"installer_path"`
}

// AuthoringContext is the trusted session context passed to the Python
// agent; its JSON shape is the runner agent's contract. The agent sets each
// action's plan in the chat.
type AuthoringContext struct {
	WalletVersionIdentifier string          `json:"wallet_version_identifier"`
	ActorID                 string          `json:"actor_id"`
	Platform                string          `json:"platform"`
	Build                   AuthoringBuild  `json:"build"`
	Device                  AuthoringDevice `json:"device"`
	Mode                    string          `json:"mode,omitempty"`
	FCAFSuite               string          `json:"fcaf_suite,omitempty"`
}

// Workflow is the "Action author session" workflow.
type Workflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// New returns the session workflow.
func New() *Workflow {
	w := &Workflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

// Name returns the registered workflow name.
func (w *Workflow) Name() string {
	return WorkflowName
}

// GetOptions returns the pipeline's default activity options.
func (w *Workflow) GetOptions() workflow.ActivityOptions {
	return pipeline.PrepareWorkflowOptions(pipelineinternal.RuntimeConfig{}).ActivityOptions
}

// Workflow runs the session.
func (w *Workflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// ExecuteWorkflow prepares the device, runs the agent until it ends and
// always cleans the device and releases the semaphore ticket. Any outcome of
// the agent ends the session normally.
func (w *Workflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, finalErr error) {
	logger := workflow.GetLogger(ctx)
	execution := workflow.GetInfo(ctx).WorkflowExecution
	ao := w.GetOptions()
	if input.ActivityOptions != nil {
		ao = *input.ActivityOptions
	}

	settedDevices := map[string]any{}
	runData := map[string]any{
		"run_identifier": execution.ID,
		"setted_devices": settedDevices,
	}
	var prepared mobiledevice.PreparedMobileDevice
	released := false
	defer func() {
		if released {
			return
		}
		releaseSessionDevice(releaseSessionDeviceInput{
			ctx:       ctx,
			logger:    logger,
			ao:        ao,
			config:    input.Config,
			runData:   runData,
			prepared:  prepared,
			execution: execution,
			result:    sessionResult(ctx, finalErr),
		})
	}()
	// The live view reads the prepared device with the pipeline's query.
	if err := workflow.SetQueryHandler(
		ctx,
		pipeline.PipelineMobileDevicesQuery,
		func() (map[string]any, error) {
			return settedDevices, nil
		},
	); err != nil {
		return result, err
	}
	drainRunGrant(ctx)

	session, err := workflowengine.DecodePayload[SessionInput](input.Payload)
	if err != nil {
		return result, invalidSessionError(fmt.Sprintf("invalid session input: %v", err))
	}
	session.DeviceID = canonify.NormalizePath(session.DeviceID)
	if session.DeviceID == "" || session.WalletVersionIdentifier == "" || session.Platform == "" {
		return result, invalidSessionError(
			"device_id, platform and wallet_version_identifier are required",
		)
	}
	switch session.Mode {
	case "":
	case actionauthor.SessionModeFCAF:
		if session.FCAFSuite != actionauthor.FCAFRelyingPartySuite ||
			session.Platform != actionauthor.PlatformAndroid {
			return result, invalidSessionError(
				"FCAF sessions run the relying-party suite on Android",
			)
		}
	default:
		return result, invalidSessionError("mode must be fcaf or empty")
	}

	build, err := resolveBuild(ctx, session)
	if err != nil {
		return result, err
	}

	deviceRequest := mobiledevice.MobileDeviceRequest{
		DeviceID:          session.DeviceID,
		VersionIdentifier: build.VersionIdentifier,
		StepID:            StepID,
		RunIdentifier:     execution.ID,
	}
	// A store wallet is installed by the agent; cleanup removes it by the diff
	// against the initial_installed_apps recorded at prepare.
	if build.Source == actionauthor.SourceStore {
		deviceRequest.VersionIdentifier = pipeline.MobileExternalSourceVersionID
		deviceRequest.PackageID = build.PackageID
	}
	device, err := pipeline.PrepareMobileDevice(ctx, ao, settedDevices, deviceRequest)
	if err != nil {
		return result, err
	}
	prepared = device
	// onRelease frees the device while the agent keeps running; the deferred
	// release then has nothing left to do. Clearing settedDevices ends the
	// session's live view (the device query reports no device).
	onRelease := func() {
		if released {
			return
		}
		released = true
		// The release reports success because the device work ended
		// normally; a later failure of the agent is not a device outcome.
		releaseSessionDevice(releaseSessionDeviceInput{
			ctx:       ctx,
			logger:    logger,
			ao:        ao,
			config:    input.Config,
			runData:   runData,
			prepared:  prepared,
			execution: execution,
			result:    resultSuccess,
		})
		clear(settedDevices)
	}

	authorWorkflowID := execution.ID + authorWorkflowIDSuffix
	authorTaskQueue := prepared.RunnerID + AuthorTaskQueueSuffix
	childTimeout := authorExecutionTimeout
	if session.Mode == actionauthor.SessionModeFCAF {
		childTimeout = fcafAuthorExecutionTimeout
	}
	childCtx, cancelChild := workflow.WithCancel(ctx)
	childCtx = workflow.WithChildOptions(childCtx, workflow.ChildWorkflowOptions{
		WorkflowID:               authorWorkflowID,
		TaskQueue:                authorTaskQueue,
		WorkflowExecutionTimeout: childTimeout,
		ParentClosePolicy:        enumspb.PARENT_CLOSE_POLICY_TERMINATE,
	})
	child := workflow.ExecuteChildWorkflow(
		childCtx,
		AuthorWorkflowType,
		// The Python harness agent takes its config, then the trusted session context.
		map[string]any{},
		authoringContext(session, build, prepared),
	)
	ready, childErr := awaitAuthor(ctx, child, onRelease)
	if !ready {
		// Closing the session terminates the child; the deferred release
		// frees the device now instead of after the child's timeout.
		cancelChild()
		return result, agentNotStartedError(authorTaskQueue)
	}
	if childErr != nil && temporal.IsCanceledError(ctx.Err()) {
		return result, childErr
	}
	if childErr != nil {
		logger.Warn("action author agent ended with an error", "error", childErr)
	}

	result.Message = "action author session ended"
	result.Output = map[string]any{
		sessionOutputAuthorKey: authorWorkflowID,
		sessionOutputDeviceKey: prepared,
	}
	return result, nil
}

// awaitAuthor waits for the agent's ready signal, then for the agent to end,
// calling onRelease when ReleaseDeviceSignal arrives meanwhile. It reports
// ready false when neither the signal nor the agent's end arrives within
// agentReadyTimeout: a runner without a polling agent worker must not hold
// the device until the child's execution timeout.
func awaitAuthor(
	ctx workflow.Context,
	child workflow.ChildWorkflowFuture,
	onRelease func(),
) (ready bool, childErr error) {
	timerCtx, cancelTimer := workflow.WithCancel(ctx)

	ended := false
	timedOut := false
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(
		workflow.GetSignalChannel(ctx, authorReadySignal),
		func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
		},
	)
	selector.AddFuture(workflow.NewTimer(timerCtx, agentReadyTimeout), func(f workflow.Future) {
		// A canceled session cancels the timer too; the child then reports it.
		timedOut = f.Get(timerCtx, nil) == nil
	})
	selector.AddFuture(child, func(f workflow.Future) {
		ended = true
		childErr = f.Get(ctx, nil)
	})
	selector.Select(ctx)

	if timedOut {
		return false, nil
	}
	// The agent is ready or ended: the timer must not stay pending (and fire)
	// while the session waits up to the child's execution timeout.
	cancelTimer()
	releaseSignal := workflow.GetSignalChannel(ctx, ReleaseDeviceSignal)
	for !ended {
		selector := workflow.NewSelector(ctx)
		selector.AddFuture(child, func(f workflow.Future) {
			ended = true
			childErr = f.Get(ctx, nil)
		})
		selector.AddReceive(releaseSignal, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			onRelease()
		})
		selector.Select(ctx)
	}
	return true, childErr
}

func agentNotStartedError(taskQueue string) error {
	errCode := errorcodes.Codes[errorcodes.ActionAuthorAgentNotStarted]
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf(
			"action author agent did not start within %s on task queue %s",
			agentReadyTimeout,
			taskQueue,
		),
	})
}

func resolveBuild(ctx workflow.Context, session SessionInput) (actionauthor.Build, error) {
	var resolved workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		ctx,
		activities.ResolveActionAuthorBuildActivityName,
		workflowengine.ActivityInput{Payload: buildRequest(session)},
	).Get(ctx, &resolved); err != nil {
		return actionauthor.Build{}, err
	}
	build, err := workflowengine.DecodePayload[actionauthor.Build](resolved.Output)
	if err != nil || build.VersionIdentifier == "" {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		return actionauthor.Build{}, workflowengine.NewAppError(workflowengine.WorkflowError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "invalid action author build",
			Details: map[string]any{"payload": resolved.Output},
		})
	}
	return build, nil
}

func buildRequest(session SessionInput) actionauthor.BuildRequest {
	return actionauthor.BuildRequest{
		Actor:                   actionauthor.Actor{ActorID: session.ActorID},
		WalletVersionIdentifier: session.WalletVersionIdentifier,
		Platform:                session.Platform,
	}
}

func authoringContext(
	session SessionInput,
	build actionauthor.Build,
	prepared mobiledevice.PreparedMobileDevice,
) AuthoringContext {
	return AuthoringContext{
		WalletVersionIdentifier: session.WalletVersionIdentifier,
		ActorID:                 session.ActorID,
		Platform:                session.Platform,
		Mode:                    session.Mode,
		FCAFSuite:               session.FCAFSuite,
		Build: AuthoringBuild{
			WalletID:          build.WalletID,
			VersionID:         build.VersionID,
			VersionIdentifier: build.VersionIdentifier,
			OwnerNamespace:    build.OwnerNamespace,
			Platform:          build.Platform,
			InstallerSHA256:   build.InstallerSHA256,
			Source:            build.Source,
			PackageID:         build.PackageID,
			StoreURL:          build.StoreURL,
		},
		Device: AuthoringDevice{
			DeviceID:      session.DeviceID,
			RunnerID:      prepared.RunnerID,
			Serial:        prepared.Serial,
			Type:          prepared.Type,
			PackageID:     prepared.PackageID,
			InstallerPath: prepared.InstallerPath,
		},
	}
}

type releaseSessionDeviceInput struct {
	ctx       workflow.Context
	logger    log.Logger
	ao        workflow.ActivityOptions
	config    map[string]any
	runData   map[string]any
	prepared  mobiledevice.PreparedMobileDevice
	execution workflow.Execution
	result    string
}

// releaseSessionDevice has the agent free the device (on iOS it deletes the
// session's clones, on Android it stops the device's Maestro process), runs
// the pipeline cleanup and releases the semaphore ticket, in that order, even
// when the session was canceled. Failures are logged and never fail the session.
func releaseSessionDevice(input releaseSessionDeviceInput) {
	dctx, _ := workflow.NewDisconnectedContext(input.ctx)

	// The agent's own teardown does not run when it is terminated, so the
	// session asks it to free the device. A device that was never prepared
	// holds nothing. A missing agent worker times out instead of blocking
	// cleanup and RunDone.
	if input.prepared.RunnerID != "" {
		release := map[string]any{
			"platform": actionauthor.PlatformAndroid,
			"template": input.prepared.Serial,
		}
		if input.prepared.Type == deviceTypeIOSSimulator {
			release["platform"] = actionauthor.PlatformIOS
		}
		releaseCtx := workflow.WithActivityOptions(
			dctx,
			releaseDeviceOptions(input.prepared.RunnerID),
		)
		err := workflow.ExecuteActivity(releaseCtx, ReleaseDeviceActivityName, release).
			Get(releaseCtx, nil)
		if err != nil {
			input.logger.Error("failed to release action author device", "error", err)
		}
	}

	cleanupAO := workflow.ActivityOptions{
		StartToCloseTimeout:    cleanupStartToClose,
		ScheduleToCloseTimeout: cleanupScheduleToClose,
		HeartbeatTimeout:       cleanupHeartbeatTimeout,
		RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: cleanupMaximumAttempts},
	}
	// Sessions have no pipeline results, so cleanup warnings are dropped.
	output := map[string]any{}
	if err := pipeline.MobileAutomationCleanupHook(
		workflow.WithActivityOptions(dctx, input.ao),
		nil,
		&cleanupAO,
		input.config,
		input.runData,
		&output,
	); err != nil {
		input.logger.Error("action author session device cleanup failed", "error", err)
	}

	pipeline.ReportMobileDeviceSemaphoreDone(
		workflow.WithActivityOptions(dctx, input.ao),
		input.logger,
		input.config,
		input.execution.ID,
		input.execution.RunID,
		input.result,
	)
}

// releaseDeviceOptions schedules the clone release on the runner's agent
// queue; it gives up when no agent worker picks it up in time.
func releaseDeviceOptions(runnerID string) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		TaskQueue:              runnerID + AuthorTaskQueueSuffix,
		ScheduleToStartTimeout: releaseScheduleToStart,
		StartToCloseTimeout:    releaseStartToClose,
		RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: releaseMaximumAttempts},
	}
}

// sessionResult reports the session outcome with the values pipelines use.
func sessionResult(ctx workflow.Context, err error) string {
	if temporal.IsCanceledError(ctx.Err()) {
		return resultCanceled
	}
	if err != nil {
		return resultFailed
	}
	return resultSuccess
}

func invalidSessionError(message string) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: message,
	})
}

// drainRunGrant receives and ignores the start activity's run grant signals,
// which Credimi reads from the history only, so the SDK does not warn about
// unhandled signals. Receiving adds no commands, so it is replay safe.
func drainRunGrant(ctx workflow.Context) {
	signals := workflow.GetSignalChannel(ctx, activities.ActionAuthorRunGrantSignal)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			signals.Receive(ctx, nil)
		}
	})
}
