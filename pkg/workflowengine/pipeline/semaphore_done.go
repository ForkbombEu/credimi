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
	mobileRunnerSemaphoreOwnerNamespaceKey = "mobile_runner_semaphore_owner_namespace"
	mobileRunnerSemaphoreLeaderRunnerIDKey = "mobile_runner_semaphore_leader_runner_id"
)

func reportMobileRunnerSemaphoreDone(
	ctx workflow.Context,
	logger log.Logger,
	config map[string]any,
	workflowID string,
	runID string,
) {
	if config == nil {
		return
	}
	ticketID, _ := config[mobileRunnerSemaphoreTicketIDConfigKey].(string)
	ownerNamespace, _ := config[mobileRunnerSemaphoreOwnerNamespaceKey].(string)
	leaderRunnerID, _ := config[mobileRunnerSemaphoreLeaderRunnerIDKey].(string)
	if ticketID == "" || ownerNamespace == "" || leaderRunnerID == "" {
		return
	}

	reportActivity := activities.NewReportMobileRunnerSemaphoreDoneActivity()
	payload := activities.ReportMobileRunnerSemaphoreDoneInput{
		OwnerNamespace: ownerNamespace,
		LeaderRunnerID: leaderRunnerID,
		TicketID:       ticketID,
		WorkflowID:     workflowID,
		RunID:          runID,
	}

	if err := workflow.ExecuteActivity(
		ctx,
		reportActivity.Name(),
		workflowengine.ActivityInput{Payload: payload},
	).Get(ctx, nil); err != nil {
		logger.Error(
			"failed to report mobile runner semaphore done",
			"ticket_id",
			ticketID,
			"leader_runner_id",
			leaderRunnerID,
			"error",
			err,
		)
	}
}
