// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/runqueue"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
)

const (
	semaphoreWorkflowPageSize  int32         = 200
	semaphoreWorkflowPageCap                 = 10
	semaphoreQueuedRunsTimeout time.Duration = 2 * time.Second
)

var listMobileRunnerSemaphoreWorkflows = listMobileRunnerSemaphoreWorkflowsTemporal
var queryMobileRunnerSemaphoreQueuedRuns = queryMobileRunnerSemaphoreQueuedRunsTemporal

type QueuedPipelineRunAggregate struct {
	TicketID           string
	PipelineIdentifier string
	EnqueuedAt         time.Time
	LeaderRunnerID     string
	RequiredRunnerIDs  []string
	RunnerIDs          []string
	Status             workflows.MobileRunnerSemaphoreRunStatus
	Position           int
	LineLen            int
}

func listQueuedPipelineRuns(
	ctx context.Context,
	orgNamespace string,
) (map[string]QueuedPipelineRunAggregate, error) {
	if orgNamespace == "" {
		return nil, nil
	}

	runnerIDs, err := listMobileRunnerSemaphoreWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	aggregates := make(map[string]QueuedPipelineRunAggregate)
	statuses := make(map[string][]runqueue.RunnerStatus)

	for _, runnerID := range runnerIDs {
		runnerCtx, cancel := context.WithTimeout(ctx, semaphoreQueuedRunsTimeout)
		views, err := queryMobileRunnerSemaphoreQueuedRuns(runnerCtx, runnerID, orgNamespace)
		cancel()
		if err != nil {
			continue
		}

		for _, view := range views {
			if view.Status != workflowengine.MobileRunnerSemaphoreRunQueued {
				continue
			}

			statuses[view.TicketID] = append(
				statuses[view.TicketID],
				runqueue.RunnerStatus{
					RunnerID: runnerID,
					Status:   view.Status,
					Position: view.Position,
					LineLen:  view.LineLen,
				},
			)

			agg, ok := aggregates[view.TicketID]
			if !ok {
				aggregates[view.TicketID] = QueuedPipelineRunAggregate{
					TicketID:           view.TicketID,
					PipelineIdentifier: view.PipelineIdentifier,
					EnqueuedAt:         view.EnqueuedAt,
					LeaderRunnerID:     view.LeaderRunnerID,
					RequiredRunnerIDs:  copyStringSlice(view.RequiredRunnerIDs),
					RunnerIDs:          copyStringSlice(view.RequiredRunnerIDs),
				}
				continue
			}

			if agg.PipelineIdentifier == "" {
				agg.PipelineIdentifier = view.PipelineIdentifier
			}
			if agg.EnqueuedAt.IsZero() {
				agg.EnqueuedAt = view.EnqueuedAt
			}
			if agg.LeaderRunnerID == "" {
				agg.LeaderRunnerID = view.LeaderRunnerID
			}
			if len(agg.RequiredRunnerIDs) == 0 && len(view.RequiredRunnerIDs) > 0 {
				agg.RequiredRunnerIDs = copyStringSlice(view.RequiredRunnerIDs)
				agg.RunnerIDs = copyStringSlice(view.RequiredRunnerIDs)
			}

			aggregates[view.TicketID] = agg
		}
	}

	for ticketID, runnerStatuses := range statuses {
		aggregateStatus := runqueue.AggregateRunnerStatuses(runnerStatuses)
		agg := aggregates[ticketID]
		agg.Status = aggregateStatus.Status
		agg.Position = aggregateStatus.Position
		agg.LineLen = aggregateStatus.LineLen
		aggregates[ticketID] = agg
	}

	return aggregates, nil
}

func listMobileRunnerSemaphoreWorkflowsTemporal(ctx context.Context) ([]string, error) {
	client, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		"WorkflowType = \"%s\" and ExecutionStatus = %d",
		workflows.MobileRunnerSemaphoreWorkflowName,
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
	)
	pageToken := []byte(nil)
	pageCount := 0
	runnerIDs := make(map[string]struct{})
	workflowPrefix := workflows.MobileRunnerSemaphoreWorkflowName + "/"

	for pageCount < semaphoreWorkflowPageCap {
		resp, err := client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     workflowengine.MobileRunnerSemaphoreDefaultNamespace,
			Query:         query,
			PageSize:      semaphoreWorkflowPageSize,
			NextPageToken: pageToken,
		})
		if err != nil {
			return nil, err
		}

		for _, execution := range resp.GetExecutions() {
			if execution.GetExecution() == nil {
				continue
			}
			workflowID := execution.GetExecution().GetWorkflowId()
			if workflowID == "" {
				continue
			}
			runnerID := strings.TrimPrefix(workflowID, workflowPrefix)
			if runnerID == workflowID {
				continue
			}
			runnerIDs[runnerID] = struct{}{}
		}

		if len(resp.GetNextPageToken()) == 0 {
			break
		}
		pageToken = resp.GetNextPageToken()
		pageCount++
	}

	result := make([]string, 0, len(runnerIDs))
	for runnerID := range runnerIDs {
		result = append(result, runnerID)
	}
	sort.Strings(result)

	return result, nil
}

func queryMobileRunnerSemaphoreQueuedRunsTemporal(
	ctx context.Context,
	runnerID string,
	ownerNamespace string,
) ([]workflows.MobileRunnerSemaphoreQueuedRunView, error) {
	client, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return nil, err
	}

	workflowID := workflows.MobileRunnerSemaphoreWorkflowID(runnerID)
	encoded, err := client.QueryWorkflow(
		ctx,
		workflowID,
		"",
		workflows.MobileRunnerSemaphoreListQueuedRunsQuery,
		ownerNamespace,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil, nil
		}
		return nil, err
	}

	var queued []workflows.MobileRunnerSemaphoreQueuedRunView
	if err := encoded.Get(&queued); err != nil {
		return nil, err
	}
	return queued, nil
}
