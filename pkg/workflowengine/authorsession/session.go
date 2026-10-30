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
	// ReleaseDeviceActivityName is the Python agent's activity that deletes
	// the session's iOS simulator clones.
	ReleaseDeviceActivityName = "action_author.release_device"
	// StepID labels the prepared device in errors and run data.
	StepID = "action-author"

	authorWorkflowIDSuffix  = activities.ActionAuthorWorkflowIDSuffix
	authorExecutionTimeout  = 2 * time.Hour
	deviceTypeIOSSimulator  = "ios_simulator"
	resultSuccess           = "success"
	resultFailed            = "failed"
	resultCanceled          = "canceled"
	releaseStartToClose     = 2 * time.Minute
	releaseScheduleToStart  = 30 * time.Second
	releaseMaximumAttempts  = 3
	cleanupStartToClose     = 10 * time.Minute
	cleanupScheduleToClose  = 20 * time.Minute
	cleanupHeartbeatTimeout = 30 * time.Second
	cleanupMaximumAttempts  = 3
	sessionOutputAuthorKey  = "author_workflow_id"
	sessionOutputDeviceKey  = "device"
)

// SessionInput selects the device, build and slot of a session.
type SessionInput struct {
	DeviceID string `json:"device_id"`
	// Platform is "android" or "ios", derived from the device type.
	Platform                string `json:"platform"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	ActorID                 string `json:"actor_id"`
	ActorCollection         string `json:"actor_collection"`
	Slot                    string `json:"slot"`
	AppID                   string `json:"app_id,omitempty"`
}

// AuthoringBuild is the resolved build the agent authors against.
type AuthoringBuild struct {
	WalletID          string `json:"wallet_id"`
	VersionID         string `json:"version_id"`
	VersionIdentifier string `json:"version_identifier"`
	OwnerNamespace    string `json:"owner_namespace"`
	Platform          string `json:"platform"`
	InstallerSHA256   string `json:"installer_sha256"`
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
// agent; its JSON shape is the runner agent's contract.
type AuthoringContext struct {
	WalletVersionIdentifier string          `json:"wallet_version_identifier"`
	ActorID                 string          `json:"actor_id"`
	ActorCollection         string          `json:"actor_collection"`
	Slot                    string          `json:"slot"`
	AppID                   string          `json:"app_id"`
	Platform                string          `json:"platform"`
	Build                   AuthoringBuild  `json:"build"`
	Device                  AuthoringDevice `json:"device"`
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
	var prepared pipeline.PreparedMobileDevice
	defer func() {
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

	build, err := resolveBuild(ctx, session)
	if err != nil {
		return result, err
	}

	device, err := pipeline.PrepareMobileDevice(
		ctx,
		ao,
		settedDevices,
		pipeline.MobileDeviceRequest{
			DeviceID:          session.DeviceID,
			VersionIdentifier: build.VersionIdentifier,
			StepID:            StepID,
			RunIdentifier:     execution.ID,
		},
	)
	if err != nil {
		return result, err
	}
	prepared = device

	if session.AppID != "" && session.AppID != prepared.PackageID {
		return result, invalidSessionError(
			fmt.Sprintf("selected build is %s, not %s", prepared.PackageID, session.AppID),
		)
	}

	authorWorkflowID := execution.ID + authorWorkflowIDSuffix
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               authorWorkflowID,
		TaskQueue:                prepared.RunnerID + AuthorTaskQueueSuffix,
		WorkflowExecutionTimeout: authorExecutionTimeout,
		ParentClosePolicy:        enumspb.PARENT_CLOSE_POLICY_TERMINATE,
	})
	childErr := workflow.ExecuteChildWorkflow(
		childCtx,
		AuthorWorkflowType,
		// The Python harness agent takes its config, then the trusted session context.
		map[string]any{},
		authoringContext(session, build, prepared),
	).Get(childCtx, nil)
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

func resolveBuild(ctx workflow.Context, session SessionInput) (actionauthor.Build, error) {
	var resolved workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		ctx,
		activities.ResolveActionAuthorBuildActivityName,
		workflowengine.ActivityInput{Payload: actionauthor.BuildRequest{
			Actor: actionauthor.Actor{
				ActorID:         session.ActorID,
				ActorCollection: session.ActorCollection,
			},
			WalletVersionIdentifier: session.WalletVersionIdentifier,
			Platform:                session.Platform,
		}},
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

func authoringContext(
	session SessionInput,
	build actionauthor.Build,
	prepared pipeline.PreparedMobileDevice,
) AuthoringContext {
	return AuthoringContext{
		WalletVersionIdentifier: session.WalletVersionIdentifier,
		ActorID:                 session.ActorID,
		ActorCollection:         session.ActorCollection,
		Slot:                    session.Slot,
		AppID:                   session.AppID,
		Platform:                session.Platform,
		Build: AuthoringBuild{
			WalletID:          build.WalletID,
			VersionID:         build.VersionID,
			VersionIdentifier: build.VersionIdentifier,
			OwnerNamespace:    build.OwnerNamespace,
			Platform:          build.Platform,
			InstallerSHA256:   build.InstallerSHA256,
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
	prepared  pipeline.PreparedMobileDevice
	execution workflow.Execution
	result    string
}

// releaseSessionDevice deletes the agent's iOS clones, runs the pipeline
// cleanup and releases the semaphore ticket, in that order, even when the
// session was canceled. Failures are logged and never fail the session.
func releaseSessionDevice(input releaseSessionDeviceInput) {
	dctx, _ := workflow.NewDisconnectedContext(input.ctx)

	// The agent's own teardown does not run when it is terminated, so the
	// session deletes its clones. A device that was never prepared has none.
	// A missing agent worker times out instead of blocking cleanup and RunDone.
	if input.prepared.Type == deviceTypeIOSSimulator && input.prepared.RunnerID != "" {
		releaseCtx := workflow.WithActivityOptions(
			dctx,
			releaseDeviceOptions(input.prepared.RunnerID),
		)
		if err := workflow.ExecuteActivity(
			releaseCtx,
			ReleaseDeviceActivityName,
			map[string]any{"platform": actionauthor.PlatformIOS, "template": input.prepared.Serial},
		).Get(releaseCtx, nil); err != nil {
			input.logger.Error("failed to release action author simulator clones", "error", err)
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
