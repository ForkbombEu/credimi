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

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesession"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const (
	AwaitDeviceSessionActivityName   = "Wait for a mobile device session"
	ReleaseDeviceSessionActivityName = "Release a mobile device session"
	CheckPipelineRunActivityName     = "Check a pipeline run"

	// runnerWorkflowExecutionTimeout bounds every workflow Credimi starts in a
	// runner namespace.
	runnerWorkflowExecutionTimeout = 24 * time.Hour
	runnerWorkflowHeartbeatEvery   = 10 * time.Second
	// runnerWorkflowCancelWait bounds the wait for a canceled workflow to close.
	runnerWorkflowCancelWait = 10 * time.Minute
	orphanStepsPageSize      = 100
)

// ReleaseDeviceSessionPayload ends a device session.
type ReleaseDeviceSessionPayload struct {
	Ref         mobiledevicesession.RunnerWorkflowRef `json:"ref"`
	SkipCleanup bool                                  `json:"skip_cleanup"`
}

// CheckPipelineRunPayload asks whether a pipeline run still runs.
type CheckPipelineRunPayload struct {
	Pipeline        mobiledevicesession.PipelineRunRef `json:"pipeline"`
	RunnerNamespace string                             `json:"runner_namespace"`
}

// CheckPipelineRunResult is the output of CheckPipelineRunActivityName.
type CheckPipelineRunResult struct {
	Running bool `json:"running"`
}

// runnerWorkflowClients returns the Temporal client of a namespace.
type runnerWorkflowClients func(namespace string) (client.Client, error)

func defaultRunnerWorkflowClients(namespace string) (client.Client, error) {
	return temporalclient.GetTemporalClientWithNamespace(namespace)
}

func requireRunnerNamespace(a *workflowengine.BaseActivity, namespace string) error {
	if !strings.HasPrefix(namespace, canonify.RunnerNamespacePrefix) ||
		len(namespace) == len(canonify.RunnerNamespacePrefix) {
		return a.NewCodedError(
			errorcodes.MissingOrInvalidPayload,
			false,
			fmt.Errorf("%q is not a runner namespace", namespace),
		)
	}
	return nil
}

func runnerNamespaceClient(
	a *workflowengine.BaseActivity,
	clients runnerWorkflowClients,
	namespace string,
) (client.Client, error) {
	if clients == nil {
		clients = defaultRunnerWorkflowClients
	}
	c, err := clients(namespace)
	if err != nil {
		return nil, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	}
	return c, nil
}

// startOrReuseRunnerWorkflow starts a workflow in a runner namespace with the
// pipeline run memo; with REJECT_DUPLICATE/USE_EXISTING a retried start
// resolves to the same run, also after it closed.
func startOrReuseRunnerWorkflow(
	ctx context.Context,
	a *workflowengine.BaseActivity,
	c client.Client,
	workflowID, workflowName, summary string,
	memo map[string]any,
	pipelineRun mobiledevicesession.PipelineRunRef,
	input workflowengine.WorkflowInput,
) (client.WorkflowRun, error) {
	fullMemo := make(map[string]any, len(memo)+1)
	for key, value := range memo {
		fullMemo[key] = value
	}
	fullMemo[mobiledevicesession.PipelineRunMemoKey] = pipelineRun
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                pipelineTaskQueue,
		Memo:                     fullMemo,
		StaticSummary:            summary,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		WorkflowExecutionTimeout: runnerWorkflowExecutionTimeout,
	}, workflowName, input)
	switch {
	case err == nil:
		return run, nil
	case temporal.IsWorkflowExecutionAlreadyStartedError(err):
		// A retry after the run closed: reuse that run.
		return c.GetWorkflow(ctx, workflowID, ""), nil
	default:
		return nil, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	}
}

// heartbeatUntil heartbeats details every runnerWorkflowHeartbeatEvery until
// done is closed or ctx is done; it reports whether done was closed.
func heartbeatUntil(ctx context.Context, done <-chan struct{}, details ...any) bool {
	ticker := time.NewTicker(runnerWorkflowHeartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return true
		case <-ctx.Done():
			return false
		case <-ticker.C:
			workflowengine.RecordHeartbeat(ctx, details...)
		}
	}
}

// RunRunnerWorkflowActivity runs a workflow in a runner namespace and returns
// its result. Canceling the activity cancels the workflow.
type RunRunnerWorkflowActivity struct {
	workflowengine.BaseActivity
	clients runnerWorkflowClients
}

func NewRunRunnerWorkflowActivity() *RunRunnerWorkflowActivity {
	return &RunRunnerWorkflowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: mobiledevicesession.RunRunnerWorkflowActivityName,
		},
	}
}

func (a *RunRunnerWorkflowActivity) Name() string { return a.BaseActivity.Name }

func (a *RunRunnerWorkflowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[mobiledevicesession.RunRunnerWorkflowPayload](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if err := requireRunnerNamespace(&a.BaseActivity, payload.Namespace); err != nil {
		return result, err
	}
	if payload.WorkflowID == "" || payload.WorkflowName == "" {
		return result, a.NewCodedError(
			errorcodes.MissingOrInvalidPayload,
			false,
			errors.New("workflow_id and workflow_name are required"),
		)
	}
	c, err := runnerNamespaceClient(&a.BaseActivity, a.clients, payload.Namespace)
	if err != nil {
		return result, err
	}
	run, err := startOrReuseRunnerWorkflow(
		ctx,
		&a.BaseActivity,
		c,
		payload.WorkflowID,
		payload.WorkflowName,
		payload.Summary,
		payload.Memo,
		payload.Pipeline,
		payload.Input,
	)
	if err != nil {
		return result, err
	}
	runID := run.GetRunID()
	workflowengine.RecordHeartbeat(ctx, runID)

	var workflowResult workflowengine.WorkflowResult
	var runErr error
	getCtx, stopGet := context.WithCancel(context.Background())
	defer stopGet()
	done := make(chan struct{})
	go func() {
		defer close(done)
		runErr = run.Get(getCtx, &workflowResult)
	}()
	if !heartbeatUntil(ctx, done, runID) {
		// A failed cancel leaves the workflow to its own timeout; a sent one is
		// given time to clean up, so a retried step finds the device free.
		if err := c.CancelWorkflow(context.Background(), payload.WorkflowID, runID); err == nil {
			select {
			case <-done:
			case <-time.After(runnerWorkflowCancelWait):
			}
		}
		return result, temporal.NewCanceledError()
	}
	if runErr != nil {
		failure := workflowengine.ParseWorkflowError(runErr)
		if failure.Code != "" {
			return result, workflowengine.NewNonRetryableAppError(failure)
		}
		return result, a.NewCodedError(errorcodes.PipelineExecutionError, false, fmt.Errorf(
			"workflow %s in %s ended: %w",
			payload.WorkflowID,
			payload.Namespace,
			runErr,
		))
	}
	result.Output = mobiledevicesession.RunRunnerWorkflowResult{
		Namespace:  payload.Namespace,
		WorkflowID: payload.WorkflowID,
		RunID:      runID,
		Result:     workflowResult,
	}
	return result, nil
}

// StartDeviceSessionActivity starts the device session of a pipeline run.
type StartDeviceSessionActivity struct {
	workflowengine.BaseActivity
	clients runnerWorkflowClients
}

func NewStartDeviceSessionActivity() *StartDeviceSessionActivity {
	return &StartDeviceSessionActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: mobiledevicesession.StartDeviceSessionActivityName,
		},
	}
}

func (a *StartDeviceSessionActivity) Name() string { return a.BaseActivity.Name }

func (a *StartDeviceSessionActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[mobiledevicesession.StartDeviceSessionPayload](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if err := requireRunnerNamespace(&a.BaseActivity, payload.Namespace); err != nil {
		return result, err
	}
	if payload.WorkflowID == "" || payload.DeviceID == "" {
		return result, a.NewCodedError(
			errorcodes.MissingOrInvalidPayload,
			false,
			errors.New("workflow_id and device_id are required"),
		)
	}
	c, err := runnerNamespaceClient(&a.BaseActivity, a.clients, payload.Namespace)
	if err != nil {
		return result, err
	}
	run, err := startOrReuseRunnerWorkflow(
		ctx,
		&a.BaseActivity,
		c,
		payload.WorkflowID,
		mobiledevicesession.WorkflowName,
		"device "+payload.DeviceID,
		nil,
		payload.Pipeline,
		payload.Input,
	)
	if err != nil {
		return result, err
	}
	result.Output = mobiledevicesession.RunnerWorkflowRef{
		Namespace:  payload.Namespace,
		WorkflowID: payload.WorkflowID,
		RunID:      run.GetRunID(),
	}
	return result, nil
}

// updateDeviceSession sends an update to a device session and waits for its
// result, heartbeating meanwhile.
func updateDeviceSession(
	ctx context.Context,
	c client.Client,
	ref mobiledevicesession.RunnerWorkflowRef,
	updateName, updateID string,
	args []any,
	valuePtr any,
) error {
	var updateErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		handle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
			WorkflowID:   ref.WorkflowID,
			RunID:        ref.RunID,
			UpdateName:   updateName,
			UpdateID:     updateID,
			Args:         args,
			WaitForStage: client.WorkflowUpdateStageCompleted,
		})
		if err != nil {
			updateErr = err
			return
		}
		updateErr = handle.Get(ctx, valuePtr)
	}()
	if !heartbeatUntil(ctx, done, ref.RunID) {
		<-done
		return ctx.Err()
	}
	return updateErr
}

// AwaitDeviceSessionActivity waits until a device session prepared its device
// and returns what the session answered.
type AwaitDeviceSessionActivity struct {
	workflowengine.BaseActivity
	clients runnerWorkflowClients
}

func NewAwaitDeviceSessionActivity() *AwaitDeviceSessionActivity {
	return &AwaitDeviceSessionActivity{
		BaseActivity: workflowengine.BaseActivity{Name: AwaitDeviceSessionActivityName},
	}
}

func (a *AwaitDeviceSessionActivity) Name() string { return a.BaseActivity.Name }

func (a *AwaitDeviceSessionActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	ref, err := workflowengine.DecodePayload[mobiledevicesession.RunnerWorkflowRef](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if err := requireRunnerNamespace(&a.BaseActivity, ref.Namespace); err != nil {
		return result, err
	}
	c, err := runnerNamespaceClient(&a.BaseActivity, a.clients, ref.Namespace)
	if err != nil {
		return result, err
	}
	var ready map[string]any
	err = updateDeviceSession(ctx, c, ref, mobiledevicesession.ReadyUpdate, "ready", nil, &ready)
	if err != nil {
		if failure := workflowengine.ParseWorkflowError(err); failure.Code != "" {
			return result, workflowengine.NewNonRetryableAppError(failure)
		}
		return result, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	}
	result.Output = ready
	return result, nil
}

// ReleaseDeviceSessionActivity ends a device session and returns its cleanup.
type ReleaseDeviceSessionActivity struct {
	workflowengine.BaseActivity
	clients runnerWorkflowClients
}

func NewReleaseDeviceSessionActivity() *ReleaseDeviceSessionActivity {
	return &ReleaseDeviceSessionActivity{
		BaseActivity: workflowengine.BaseActivity{Name: ReleaseDeviceSessionActivityName},
	}
}

func (a *ReleaseDeviceSessionActivity) Name() string { return a.BaseActivity.Name }

func (a *ReleaseDeviceSessionActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ReleaseDeviceSessionPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if err := requireRunnerNamespace(&a.BaseActivity, payload.Ref.Namespace); err != nil {
		return result, err
	}
	c, err := runnerNamespaceClient(&a.BaseActivity, a.clients, payload.Ref.Namespace)
	if err != nil {
		return result, err
	}
	var cleanup mobiledevicesession.Cleanup
	err = updateDeviceSession(
		ctx,
		c,
		payload.Ref,
		mobiledevicesession.ReleaseUpdate,
		"release",
		[]any{mobiledevicesession.Release{SkipCleanup: payload.SkipCleanup}},
		&cleanup,
	)
	var notFound *serviceerror.NotFound
	switch {
	case err == nil:
	case errors.As(err, &notFound):
		cleanup = mobiledevicesession.Cleanup{Warnings: []string{
			fmt.Sprintf("device session %s had already ended", payload.Ref.WorkflowID),
		}}
	default:
		return result, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	}
	result.Output = cleanup
	return result, nil
}

// CheckPipelineRunActivity reports whether a pipeline run still runs. When it
// does not, it cancels the step workflows the run left running in the runner
// namespace, such as those of a terminated pipeline.
type CheckPipelineRunActivity struct {
	workflowengine.BaseActivity
	clients runnerWorkflowClients
}

func NewCheckPipelineRunActivity() *CheckPipelineRunActivity {
	return &CheckPipelineRunActivity{
		BaseActivity: workflowengine.BaseActivity{Name: CheckPipelineRunActivityName},
	}
}

func (a *CheckPipelineRunActivity) Name() string { return a.BaseActivity.Name }

func (a *CheckPipelineRunActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[CheckPipelineRunPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if err := requireRunnerNamespace(&a.BaseActivity, payload.RunnerNamespace); err != nil {
		return result, err
	}
	if payload.Pipeline.Namespace == "" || payload.Pipeline.WorkflowID == "" {
		return result, a.NewCodedError(
			errorcodes.MissingOrInvalidPayload,
			false,
			errors.New("pipeline namespace and workflow_id are required"),
		)
	}
	clients := a.clients
	if clients == nil {
		clients = defaultRunnerWorkflowClients
	}
	pipelineClient, err := clients(payload.Pipeline.Namespace)
	if err != nil {
		return result, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	}
	desc, err := temporalclient.Describe(
		ctx,
		pipelineClient,
		payload.Pipeline.WorkflowID,
		payload.Pipeline.RunID,
	)
	var notFound *serviceerror.NotFound
	switch {
	case errors.As(err, &notFound):
	case err != nil:
		return result, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	case desc.GetWorkflowExecutionInfo().GetStatus() ==
		enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		result.Output = CheckPipelineRunResult{Running: true}
		return result, nil
	}

	runnerClient, err := clients(payload.RunnerNamespace)
	if err != nil {
		return result, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	}
	if err := cancelOrphanSteps(ctx, runnerClient, payload); err != nil {
		return result, a.NewCodedError(errorcodes.PipelineExecutionError, true, err)
	}
	result.Output = CheckPipelineRunResult{Running: false}
	return result, nil
}

func cancelOrphanSteps(
	ctx context.Context,
	c client.Client,
	payload CheckPipelineRunPayload,
) error {
	query := fmt.Sprintf(
		"WorkflowType = %q AND ExecutionStatus = 'Running' AND WorkflowId STARTS_WITH %q",
		mobiledevicesession.MobileAutomationWorkflowName,
		payload.Pipeline.WorkflowID+"-",
	)
	var token []byte
	for {
		resp, err := c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     payload.RunnerNamespace,
			PageSize:      orphanStepsPageSize,
			NextPageToken: token,
			Query:         query,
		})
		if err != nil {
			return fmt.Errorf("list orphaned steps of %s: %w", payload.Pipeline.WorkflowID, err)
		}
		for _, execution := range resp.GetExecutions() {
			id := execution.GetExecution().GetWorkflowId()
			err := c.CancelWorkflow(ctx, id, execution.GetExecution().GetRunId())
			var notFound *serviceerror.NotFound
			if err != nil && !errors.As(err, &notFound) {
				return fmt.Errorf("cancel orphaned step %s: %w", id, err)
			}
		}
		token = resp.GetNextPageToken()
		if len(token) == 0 {
			return nil
		}
	}
}
