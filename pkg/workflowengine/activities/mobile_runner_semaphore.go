// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	tclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

type AcquireMobileRunnerPermitInput struct {
	RunnerID string `json:"runner_id"`
}

type AcquireMobileRunnerPermitActivity struct {
	workflowengine.BaseActivity
}

type ReleaseMobileRunnerPermitActivity struct {
	workflowengine.BaseActivity
}

// Default wait timeout is 24 hours
const defaultMobileRunnerSemaphoreWaitTimeout = 24 * time.Hour

func NewAcquireMobileRunnerPermitActivity() *AcquireMobileRunnerPermitActivity {
	return &AcquireMobileRunnerPermitActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Acquire mobile runner permit",
		},
	}
}

func (a *AcquireMobileRunnerPermitActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *AcquireMobileRunnerPermitActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[AcquireMobileRunnerPermitInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	runnerID := strings.TrimSpace(payload.RunnerID)
	if runnerID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, errCode.Description)
	}

	if isMobileRunnerSemaphoreDisabled() {
		result.Output = mobilerunnersemaphore.MobileRunnerSemaphorePermit{RunnerID: runnerID}
		return result, nil
	}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return result, a.mapAcquireError(err, runnerID, 0, nil)
	}

	workflowID := mobilerunnersemaphore.WorkflowID(runnerID)
	startInput := workflowengine.WorkflowInput{
		Payload: mobilerunnersemaphore.MobileRunnerSemaphoreWorkflowInput{
			RunnerID: runnerID,
			Capacity: 1,
		},
	}
	_, err = temporalClient.ExecuteWorkflow(
		ctx,
		tclient.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: mobilerunnersemaphore.TaskQueue,
		},
		mobilerunnersemaphore.WorkflowName,
		startInput,
	)
	if err != nil && !temporal.IsWorkflowExecutionAlreadyStartedError(err) {
		return result, a.mapAcquireError(err, runnerID, 0, nil)
	}

	info := activity.GetInfo(ctx)
	leaseID := mobilerunnersemaphore.PermitLeaseID(
		info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID,
		runnerID,
	)

	waitTimeout := mobileRunnerSemaphoreWaitTimeout()
	updateReq := mobilerunnersemaphore.MobileRunnerSemaphoreAcquireRequest{
		RequestID:       leaseID,
		LeaseID:         leaseID,
		OwnerNamespace:  info.WorkflowNamespace,
		OwnerWorkflowID: info.WorkflowExecution.ID,
		OwnerRunID:      info.WorkflowExecution.RunID,
		WaitTimeout:     waitTimeout,
	}

	handle, err := temporalClient.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   mobilerunnersemaphore.AcquireUpdate,
		UpdateID:     mobilerunnersemaphore.AcquireUpdateID(leaseID),
		Args:         []interface{}{updateReq},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		queueLen := resolveQueueLen(ctx, temporalClient, workflowID, err)
		return result, a.mapAcquireError(err, runnerID, waitTimeout, queueLen)
	}

	var permit mobilerunnersemaphore.MobileRunnerSemaphorePermit
	if err := handle.Get(ctx, &permit); err != nil {
		queueLen := resolveQueueLen(ctx, temporalClient, workflowID, err)
		return result, a.mapAcquireError(err, runnerID, waitTimeout, queueLen)
	}

	result.Output = permit
	return result, nil
}

func (a *AcquireMobileRunnerPermitActivity) mapAcquireError(
	err error,
	runnerID string,
	waitTimeout time.Duration,
	queueLen *int,
) error {
	if err == nil {
		return nil
	}

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == mobilerunnersemaphore.ErrTimeout {
		errCode := errorcodes.Codes[errorcodes.MobileRunnerBusy]
		details := map[string]any{
			"runner_id": runnerID,
			"waited_ms": waitTimeout.Milliseconds(),
		}
		if queueLen != nil {
			details["queue_len"] = *queueLen
		}
		return a.NewActivityError(errCode.Code, errCode.Description, details)
	}

	errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
	return a.NewActivityError(errCode.Code, fmt.Sprintf("%s: %v", errCode.Description, err))
}

func NewReleaseMobileRunnerPermitActivity() *ReleaseMobileRunnerPermitActivity {
	return &ReleaseMobileRunnerPermitActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Release mobile runner permit",
		},
	}
}

func (a *ReleaseMobileRunnerPermitActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ReleaseMobileRunnerPermitActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[mobilerunnersemaphore.MobileRunnerSemaphorePermit](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	leaseID := strings.TrimSpace(payload.LeaseID)
	if leaseID == "" || isMobileRunnerSemaphoreDisabled() {
		return result, nil
	}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return result, err
	}

	workflowID := mobilerunnersemaphore.WorkflowID(payload.RunnerID)

	handle, err := temporalClient.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID: workflowID,
		UpdateName: mobilerunnersemaphore.ReleaseUpdate,
		UpdateID:   mobilerunnersemaphore.ReleaseUpdateID(leaseID),
		Args: []interface{}{
			mobilerunnersemaphore.MobileRunnerSemaphoreReleaseRequest{LeaseID: leaseID},
		},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		if isNotFoundError(err) {
			return result, nil
		}
		return result, err
	}

	var updateResult mobilerunnersemaphore.MobileRunnerSemaphoreReleaseResult
	if err := handle.Get(ctx, &updateResult); err != nil {
		if isNotFoundError(err) {
			return result, nil
		}
		return result, err
	}

	return result, nil
}

func isMobileRunnerSemaphoreDisabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("MOBILE_RUNNER_SEMAPHORE_DISABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func mobileRunnerSemaphoreWaitTimeout() time.Duration {
	value := strings.TrimSpace(os.Getenv("MOBILE_RUNNER_SEMAPHORE_WAIT_TIMEOUT"))
	if value == "" {
		return defaultMobileRunnerSemaphoreWaitTimeout
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultMobileRunnerSemaphoreWaitTimeout
	}
	return duration
}

func resolveQueueLen(
	ctx context.Context,
	temporalClient tclient.Client,
	workflowID string,
	err error,
) *int {
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) || appErr.Type() != mobilerunnersemaphore.ErrTimeout {
		return nil
	}

	encoded, queryErr := temporalClient.QueryWorkflow(
		ctx,
		workflowID,
		"",
		mobilerunnersemaphore.StateQuery,
	)
	if queryErr != nil {
		return nil
	}

	var state mobilerunnersemaphore.MobileRunnerSemaphoreStateView
	if decodeErr := encoded.Get(&state); decodeErr != nil {
		return nil
	}

	queueLen := state.QueueLen
	return &queueLen
}

func isNotFoundError(err error) bool {
	var notFound *serviceerror.NotFound
	return errors.As(err, &notFound)
}
