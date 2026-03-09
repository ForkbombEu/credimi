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

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

type ReportMobileRunnerSemaphoreDoneInput struct {
	OwnerNamespace string `json:"owner_namespace"`
	LeaderRunnerID string `json:"leader_runner_id"`
	TicketID       string `json:"ticket_id"`
	WorkflowID     string `json:"workflow_id"`
	RunID          string `json:"run_id"`
}

type ReportMobileRunnerSemaphoreDoneActivity struct {
	workflowengine.BaseActivity
}

func NewReportMobileRunnerSemaphoreDoneActivity() *ReportMobileRunnerSemaphoreDoneActivity {
	return &ReportMobileRunnerSemaphoreDoneActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Report mobile runner semaphore done",
		},
	}
}

func (a *ReportMobileRunnerSemaphoreDoneActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ReportMobileRunnerSemaphoreDoneActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ReportMobileRunnerSemaphoreDoneInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	ticketID := strings.TrimSpace(payload.TicketID)
	leaderRunnerID := strings.TrimSpace(payload.LeaderRunnerID)
	ownerNamespace := strings.TrimSpace(payload.OwnerNamespace)
	if ticketID == "" || leaderRunnerID == "" || ownerNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, errCode.Description)
	}

	if isMobileRunnerSemaphoreDisabled() {
		return result, nil
	}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return result, err
	}

	updateReq := mobilerunnersemaphore.MobileRunnerSemaphoreRunDoneRequest{
		TicketID:       ticketID,
		OwnerNamespace: ownerNamespace,
		WorkflowID:     strings.TrimSpace(payload.WorkflowID),
		RunID:          strings.TrimSpace(payload.RunID),
	}

	handle, err := temporalClient.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   mobilerunnersemaphore.WorkflowID(leaderRunnerID),
		UpdateName:   mobilerunnersemaphore.RunDoneUpdate,
		UpdateID:     fmt.Sprintf("run-done/%s", ticketID),
		Args:         []interface{}{updateReq},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		if isNotFoundError(err) {
			return result, nil
		}
		return result, err
	}

	var status mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView
	if err := handle.Get(ctx, &status); err != nil {
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
