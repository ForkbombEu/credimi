// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/avdpool"
	"github.com/spf13/cobra"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

var (
	debugNamespace string
	debugRunID     string
	debugWait      time.Duration
)

// NewDebugCmd creates the "debug" command group.
func NewDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug Temporal workflows",
	}
	cmd.PersistentFlags().StringVar(&debugNamespace, "namespace", "default", "Temporal namespace")

	cmd.AddCommand(newDebugPipelineCmd())
	cmd.AddCommand(newDebugPoolCmd())
	cmd.AddCommand(newDebugCleanupCmd())
	return cmd
}

func newDebugPipelineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline <workflow_id>",
		Short: "Inspect pipeline workflow state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID := args[0]
			c, err := temporalclient.GetTemporalDebugClient(debugNamespace)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx := cmd.Context()
			state, stateErr := queryPipelineState(ctx, c, workflowID, debugRunID)
			usage, usageErr := queryResourceUsage(ctx, c, workflowID, debugRunID)
			activities, historyErr := fetchRecentActivities(ctx, c, workflowID, debugRunID, 10)

			output := map[string]any{
				"workflow_id":       workflowID,
				"run_id":            debugRunID,
				"pipeline_state":    state,
				"resource_usage":    usage,
				"recent_activities": activities,
			}
			if stateErr != nil {
				output["pipeline_state_error"] = stateErr.Error()
			}
			if usageErr != nil {
				output["resource_usage_error"] = usageErr.Error()
			}
			if historyErr != nil {
				output["history_error"] = historyErr.Error()
			}
			return printJSON(output)
		},
	}
	cmd.Flags().StringVar(&debugRunID, "run-id", "", "Temporal run ID (optional)")
	return cmd
}

func newDebugPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool",
		Short: "Inspect AVD pool status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := temporalclient.GetTemporalDebugClient(debugNamespace)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx := cmd.Context()
			queryResp, err := c.QueryWorkflow(ctx, avdpool.DefaultPoolWorkflowID, "", avdpool.PoolStatusQuery)
			if err != nil {
				return err
			}
			var status avdpool.PoolStatus
			if err := queryResp.Get(&status); err != nil {
				return err
			}
			return printJSON(status)
		},
	}
	return cmd
}

func newDebugCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup <workflow_id>",
		Short: "Signal a workflow to force cleanup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID := args[0]
			c, err := temporalclient.GetTemporalDebugClient(debugNamespace)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx := cmd.Context()
			if err := c.SignalWorkflow(ctx, workflowID, debugRunID, workflowengine.ForceCleanupSignal, struct{}{}); err != nil {
				return err
			}

			output := map[string]any{
				"workflow_id": workflowID,
				"run_id":      debugRunID,
				"signal":      workflowengine.ForceCleanupSignal,
				"signal_sent": true,
			}

			if debugWait > 0 {
				waitCtx, cancel := context.WithTimeout(ctx, debugWait)
				defer cancel()
				we := c.GetWorkflow(waitCtx, workflowID, debugRunID)
				if err := we.Get(waitCtx, nil); err != nil {
					output["wait_error"] = err.Error()
				} else {
					output["completed"] = true
				}
			}

			return printJSON(output)
		},
	}
	cmd.Flags().StringVar(&debugRunID, "run-id", "", "Temporal run ID (optional)")
	cmd.Flags().DurationVar(&debugWait, "wait", 10*time.Second, "Time to wait for workflow completion")
	return cmd
}

type activityEvent struct {
	Name      string    `json:"name"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

func fetchRecentActivities(
	ctx context.Context,
	c client.Client,
	workflowID string,
	runID string,
	limit int,
) ([]activityEvent, error) {
	iter := c.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	if iter == nil {
		return nil, fmt.Errorf("unable to create history iterator")
	}

	events := make([]activityEvent, 0, limit)
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if event.GetEventType() != enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED {
			continue
		}
		attr := event.GetActivityTaskScheduledEventAttributes()
		if attr == nil || attr.GetActivityType() == nil {
			continue
		}
		events = append(events, activityEvent{
			Name:      attr.GetActivityType().GetName(),
			EventType: "scheduled",
			Timestamp: event.GetEventTime().AsTime(),
		})
	}

	if len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}

func queryPipelineState(
	ctx context.Context,
	c client.Client,
	workflowID string,
	runID string,
) (workflowengine.PipelineState, error) {
	queryResp, err := c.QueryWorkflow(ctx, workflowID, runID, workflowengine.PipelineStateQuery)
	if err != nil {
		return workflowengine.PipelineState{}, err
	}
	var state workflowengine.PipelineState
	if err := queryResp.Get(&state); err != nil {
		return workflowengine.PipelineState{}, err
	}
	return state, nil
}

func queryResourceUsage(
	ctx context.Context,
	c client.Client,
	workflowID string,
	runID string,
) (workflowengine.ResourceUsage, error) {
	queryResp, err := c.QueryWorkflow(ctx, workflowID, runID, workflowengine.ResourceUsageQuery)
	if err != nil {
		return workflowengine.ResourceUsage{}, err
	}
	var usage workflowengine.ResourceUsage
	if err := queryResp.Get(&usage); err != nil {
		return workflowengine.ResourceUsage{}, err
	}
	return usage, nil
}
