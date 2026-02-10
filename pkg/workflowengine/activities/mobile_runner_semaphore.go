// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

type ReleaseMobileRunnerPermitActivity struct {
	workflowengine.BaseActivity
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

func isNotFoundError(err error) bool {
	var notFound *serviceerror.NotFound
	return errors.As(err, &notFound)
}
