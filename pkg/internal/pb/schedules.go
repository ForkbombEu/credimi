// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/runners"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

type ScheduleStatus struct {
	DisplayName    string           `json:"display_name,omitempty"`
	NextActionTime string           `json:"next_action_time,omitempty"`
	Paused         bool             `json:"paused"`
	Runners        []map[string]any `json:"runners"`
}

func RegisterSchedulesHooks(app core.App) {
	app.OnRecordEnrich("schedules").BindFunc(func(e *core.RecordEnrichEvent) error {
		ownerID := e.Record.GetString("owner")

		owner, err := e.App.FindRecordById("organizations", ownerID)
		if err != nil {
			return fmt.Errorf("failed to fetch owner organization: %w", err)
		}

		namespace := owner.GetString("canonified_name")

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return fmt.Errorf(
				"unable to create Temporal client for namespace %q: %w",
				namespace,
				err,
			)
		}
		ctx := context.Background()
		handle := c.ScheduleClient().GetHandle(ctx, e.Record.GetString("temporal_schedule_id"))
		desc, err := handle.Describe(ctx)
		if err != nil {
			var notFound *serviceerror.NotFound
			if errors.As(err, &notFound) {
				// Schedule no longer exists in Temporal; enrich with fallback status so the record still loads
				log.Printf("schedule not found in Temporal (temporal_schedule_id=%s): %v", e.Record.GetString("temporal_schedule_id"), err)
				runnerRecords, _ := resolveScheduleRunnerRecords(
					e.App,
					e.Record.GetString("pipeline"),
					nil,
				)
				if runnerRecords == nil {
					runnerRecords = []map[string]any{}
				}
				status := ScheduleStatus{
					DisplayName:    "",
					NextActionTime: "",
					Paused:         false,
					Runners:        runnerRecords,
				}
				e.Record.WithCustomData(true)
				e.Record.Set("__schedule_status__", status)
				return e.Next()
			}
			return fmt.Errorf("failed to describe schedule: %w", err)
		}
		var displayName string
		if desc.Memo != nil {
			if field, ok := desc.Memo.Fields["test"]; ok {
				displayName = handlers.DecodeFromTemporalPayload(string(field.Data))
			}
		}

		// Parse runners from pipeline yaml
		runnerRecords, err := resolveScheduleRunnerRecords(
			e.App,
			e.Record.GetString("pipeline"),
			desc,
		)
		if err != nil {
			// Log error but don't fail the enrichment
			log.Printf("failed to parse runners from pipeline: %v\n", err)
			runnerRecords = []map[string]any{}
		}

		nextActionTime := ""
		if len(desc.Info.NextActionTimes) > 0 {
			nextActionTime = desc.Info.NextActionTimes[0].Format("02/01/2006, 15:04:05")
		}
		status := ScheduleStatus{
			DisplayName:    displayName,
			NextActionTime: nextActionTime,
			Paused:         desc.Schedule.State.Paused,
			Runners:        runnerRecords,
		}
		e.Record.WithCustomData(true)
		e.Record.Set("__schedule_status__", status)

		return e.Next()

	})

}

func resolveScheduleRunnerRecords(
	app core.App,
	pipelineID string,
	desc *client.ScheduleDescription,
) ([]map[string]any, error) {
	pipelineRec, err := app.FindRecordById("pipelines", pipelineID)
	if err != nil {
		pipelineRec, err = canonify.Resolve(app, pipelineID)
		if err != nil {
			return nil, fmt.Errorf("failed to load pipeline: %w", err)
		}
	}

	info, err := runners.ParsePipelineRunnerInfo(pipelineRec.GetString("yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipeline yaml: %w", err)
	}

	globalRunnerID := readGlobalRunnerIDFromScheduleDescription(desc)
	runnerIDs := runners.RunnerIDsWithGlobal(info, globalRunnerID)

	return runners.ResolveRunnerRecords(app, runnerIDs, nil), nil
}

func readGlobalRunnerIDFromScheduleDescription(
	desc *client.ScheduleDescription,
) string {
	if desc == nil || desc.Schedule.Action == nil {
		return ""
	}

	action, ok := desc.Schedule.Action.(*client.ScheduleWorkflowAction)
	if !ok || len(action.Args) == 0 {
		return ""
	}

	switch arg := action.Args[0].(type) {
	case workflows.ScheduledPipelineEnqueueWorkflowInput:
		return globalRunnerIDFromScheduledInput(arg, nil)
	case *workflows.ScheduledPipelineEnqueueWorkflowInput:
		if arg == nil {
			return ""
		}
		return globalRunnerIDFromScheduledInput(*arg, nil)
	case workflowengine.WorkflowInput:
		return globalRunnerIDFromWorkflowInput(arg)
	case *workflowengine.WorkflowInput:
		if arg == nil {
			return ""
		}
		return globalRunnerIDFromWorkflowInput(*arg)
	case pipeline.PipelineWorkflowInput:
		return runners.GlobalRunnerIDFromConfig(arg.WorkflowInput.Config)
	case *pipeline.PipelineWorkflowInput:
		if arg == nil {
			return ""
		}
		return runners.GlobalRunnerIDFromConfig(arg.WorkflowInput.Config)
	case *commonpb.Payload:
		return globalRunnerIDFromPayload(arg)
	case commonpb.Payload:
		return globalRunnerIDFromPayload(&arg)
	default:
		return ""
	}
}

func globalRunnerIDFromPayload(payload *commonpb.Payload) string {
	if payload == nil {
		return ""
	}

	dc := converter.GetDefaultDataConverter()
	var scheduledInput workflows.ScheduledPipelineEnqueueWorkflowInput
	if err := dc.FromPayload(payload, &scheduledInput); err == nil {
		if globalRunnerID := globalRunnerIDFromScheduledInput(scheduledInput, nil); globalRunnerID != "" {
			return globalRunnerID
		}
	}

	var workflowInput workflowengine.WorkflowInput
	if err := dc.FromPayload(payload, &workflowInput); err == nil {
		if globalRunnerID := globalRunnerIDFromWorkflowInput(workflowInput); globalRunnerID != "" {
			return globalRunnerID
		}
	}

	var input pipeline.PipelineWorkflowInput
	if err := dc.FromPayload(payload, &input); err != nil {
		return ""
	}

	return runners.GlobalRunnerIDFromConfig(input.WorkflowInput.Config)
}

// globalRunnerIDFromWorkflowInput extracts a global runner ID from a workflow wrapper input.
func globalRunnerIDFromWorkflowInput(input workflowengine.WorkflowInput) string {
	switch payload := input.Payload.(type) {
	case workflows.ScheduledPipelineEnqueueWorkflowInput:
		return globalRunnerIDFromScheduledInput(payload, input.Config)
	case *workflows.ScheduledPipelineEnqueueWorkflowInput:
		if payload == nil {
			return ""
		}
		return globalRunnerIDFromScheduledInput(*payload, input.Config)
	default:
		return runners.GlobalRunnerIDFromConfig(input.Config)
	}
}

// globalRunnerIDFromScheduledInput extracts a global runner ID from scheduled enqueue inputs.
func globalRunnerIDFromScheduledInput(
	input workflows.ScheduledPipelineEnqueueWorkflowInput,
	config map[string]any,
) string {
	if input.GlobalRunnerID != "" {
		return input.GlobalRunnerID
	}
	if config != nil {
		if globalRunnerID := runners.GlobalRunnerIDFromConfig(config); globalRunnerID != "" {
			return globalRunnerID
		}
	}
	return runners.GlobalRunnerIDFromConfig(input.PipelineConfig)
}
