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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

type ReportMobileDeviceSemaphoreDoneInput struct {
	OwnerNamespace string `json:"owner_namespace"`
	LeaderDeviceID string `json:"leader_device_id"`
	TicketID       string `json:"ticket_id"`
	WorkflowID     string `json:"workflow_id"`
	RunID          string `json:"run_id"`
	WorkflowResult string `json:"workflow_result,omitempty"`
}

type ReportMobileDeviceSemaphoreDoneActivity struct {
	workflowengine.BaseActivity
}

func NewReportMobileDeviceSemaphoreDoneActivity() *ReportMobileDeviceSemaphoreDoneActivity {
	return &ReportMobileDeviceSemaphoreDoneActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Report mobile device semaphore done",
		},
	}
}

func (a *ReportMobileDeviceSemaphoreDoneActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ReportMobileDeviceSemaphoreDoneActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ReportMobileDeviceSemaphoreDoneInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	ticketID := strings.TrimSpace(payload.TicketID)
	leaderDeviceID := strings.TrimSpace(payload.LeaderDeviceID)
	ownerNamespace := strings.TrimSpace(payload.OwnerNamespace)
	if ticketID == "" || leaderDeviceID == "" || ownerNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "ticket_id, leader_device_id, and owner_namespace are required",
		})
	}

	if isMobileDeviceSemaphoreDisabled() {
		return result, nil
	}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return result, err
	}

	updateReq := mobiledevicesemaphore.MobileDeviceSemaphoreRunDoneRequest{
		TicketID:       ticketID,
		OwnerNamespace: ownerNamespace,
		WorkflowID:     strings.TrimSpace(payload.WorkflowID),
		RunID:          strings.TrimSpace(payload.RunID),
		WorkflowResult: strings.TrimSpace(payload.WorkflowResult),
	}

	handle, err := temporalClient.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   mobiledevicesemaphore.WorkflowID(leaderDeviceID),
		UpdateName:   mobiledevicesemaphore.RunDoneUpdate,
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

	var status mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView
	if err := handle.Get(ctx, &status); err != nil {
		if isNotFoundError(err) {
			return result, nil
		}
		return result, err
	}

	return result, nil
}

func isMobileDeviceSemaphoreDisabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("MOBILE_DEVICE_SEMAPHORE_DISABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func isNotFoundError(err error) bool {
	var notFound *serviceerror.NotFound
	return errors.As(err, &notFound)
}
