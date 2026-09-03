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

var listMobileDeviceSemaphoreWorkflows = listMobileDeviceSemaphoreWorkflowsTemporal
var queryMobileDeviceSemaphoreQueuedRuns = queryMobileDeviceSemaphoreQueuedRunsTemporal
var queuedRunsTemporalClient = temporalclient.GetTemporalClientWithNamespace

type QueuedPipelineRunAggregate struct {
	TicketID           string
	PipelineIdentifier string
	EnqueuedAt         time.Time
	LeaderDeviceID     string
	RequiredDeviceIDs  []string
	DeviceIDs          []string
	Status             workflows.MobileDeviceSemaphoreRunStatus
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

	deviceIDs, err := listMobileDeviceSemaphoreWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	aggregates := make(map[string]QueuedPipelineRunAggregate)
	statuses := make(map[string][]runqueue.DeviceStatus)

	for _, deviceID := range deviceIDs {
		runnerCtx, cancel := context.WithTimeout(ctx, semaphoreQueuedRunsTimeout)
		views, err := queryMobileDeviceSemaphoreQueuedRuns(runnerCtx, deviceID, orgNamespace)
		cancel()
		if err != nil {
			continue
		}

		for _, view := range views {
			if view.Status != workflowengine.MobileDeviceSemaphoreRunQueued {
				continue
			}

			statuses[view.TicketID] = append(
				statuses[view.TicketID],
				runqueue.DeviceStatus{
					DeviceID: deviceID,
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
					LeaderDeviceID:     view.LeaderDeviceID,
					RequiredDeviceIDs:  copyStringSlice(view.RequiredDeviceIDs),
					DeviceIDs:          copyStringSlice(view.RequiredDeviceIDs),
				}
				continue
			}

			if agg.PipelineIdentifier == "" {
				agg.PipelineIdentifier = view.PipelineIdentifier
			}
			if agg.EnqueuedAt.IsZero() {
				agg.EnqueuedAt = view.EnqueuedAt
			}
			if agg.LeaderDeviceID == "" {
				agg.LeaderDeviceID = view.LeaderDeviceID
			}
			if len(agg.RequiredDeviceIDs) == 0 && len(view.RequiredDeviceIDs) > 0 {
				agg.RequiredDeviceIDs = copyStringSlice(view.RequiredDeviceIDs)
				agg.DeviceIDs = copyStringSlice(view.RequiredDeviceIDs)
			}

			aggregates[view.TicketID] = agg
		}
	}

	for ticketID, runnerStatuses := range statuses {
		aggregateStatus := runqueue.AggregateDeviceStatuses(runnerStatuses)
		agg := aggregates[ticketID]
		agg.Status = aggregateStatus.Status
		agg.Position = aggregateStatus.Position
		agg.LineLen = aggregateStatus.LineLen
		aggregates[ticketID] = agg
	}

	return aggregates, nil
}

func listMobileDeviceSemaphoreWorkflowsTemporal(ctx context.Context) ([]string, error) {
	client, err := queuedRunsTemporalClient(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		"WorkflowType = \"%s\" and ExecutionStatus = %d",
		workflows.MobileDeviceSemaphoreWorkflowName,
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
	)
	pageToken := []byte(nil)
	pageCount := 0
	deviceIDs := make(map[string]struct{})
	workflowPrefix := workflows.MobileDeviceSemaphoreWorkflowName + "/"

	for pageCount < semaphoreWorkflowPageCap {
		resp, err := client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     workflowengine.MobileDeviceSemaphoreDefaultNamespace,
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
			deviceID := strings.TrimPrefix(workflowID, workflowPrefix)
			if deviceID == workflowID {
				continue
			}
			deviceIDs[deviceID] = struct{}{}
		}

		if len(resp.GetNextPageToken()) == 0 {
			break
		}
		pageToken = resp.GetNextPageToken()
		pageCount++
	}

	result := make([]string, 0, len(deviceIDs))
	for deviceID := range deviceIDs {
		result = append(result, deviceID)
	}
	sort.Strings(result)

	return result, nil
}

func queryMobileDeviceSemaphoreQueuedRunsTemporal(
	ctx context.Context,
	deviceID string,
	ownerNamespace string,
) ([]workflows.MobileDeviceSemaphoreQueuedRunView, error) {
	client, err := queuedRunsTemporalClient(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return nil, err
	}

	workflowID := workflows.MobileDeviceSemaphoreWorkflowID(deviceID)
	encoded, err := client.QueryWorkflow(
		ctx,
		workflowID,
		"",
		workflows.MobileDeviceSemaphoreListQueuedRunsQuery,
		ownerNamespace,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil, nil
		}
		return nil, err
	}

	var queued []workflows.MobileDeviceSemaphoreQueuedRunView
	if err := encoded.Get(&queued); err != nil {
		return nil, err
	}
	return queued, nil
}
