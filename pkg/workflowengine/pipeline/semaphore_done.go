// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

const (
	mobileDeviceSemaphoreOwnerNamespaceKey = "mobile_device_semaphore_owner_namespace"
	mobileDeviceSemaphoreLeaderDeviceIDKey = "mobile_device_semaphore_leader_device_id"
)

func reportMobileDeviceSemaphoreDone(
	ctx workflow.Context,
	logger log.Logger,
	config map[string]any,
	workflowID string,
	runID string,
	workflowResult string,
) {
	if config == nil {
		return
	}
	ticketID, _ := config[mobileDeviceSemaphoreTicketIDConfigKey].(string)
	ownerNamespace, _ := config[mobileDeviceSemaphoreOwnerNamespaceKey].(string)
	leaderDeviceID, _ := config[mobileDeviceSemaphoreLeaderDeviceIDKey].(string)
	if ticketID == "" || ownerNamespace == "" || leaderDeviceID == "" {
		return
	}

	reportActivity := activities.NewReportMobileDeviceSemaphoreDoneActivity()
	payload := activities.ReportMobileDeviceSemaphoreDoneInput{
		OwnerNamespace: ownerNamespace,
		LeaderDeviceID: leaderDeviceID,
		TicketID:       ticketID,
		WorkflowID:     workflowID,
		RunID:          runID,
		WorkflowResult: workflowResult,
	}

	if err := workflow.ExecuteActivity(
		ctx,
		reportActivity.Name(),
		workflowengine.ActivityInput{Payload: payload},
	).Get(ctx, nil); err != nil {
		logger.Error(
			"failed to report mobile device semaphore done",
			"ticket_id",
			ticketID,
			"leader_device_id",
			leaderDeviceID,
			"error",
			err,
		)
	}
}
