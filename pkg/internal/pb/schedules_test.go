// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

func TestReadGlobalRunnerIDFromScheduleDescription(t *testing.T) {
	t.Run("scheduled input value", func(t *testing.T) {
		desc := scheduleDescWithArg(workflows.ScheduledPipelineEnqueueWorkflowInput{
			GlobalRunnerID: "runner-1",
		})
		require.Equal(t, "runner-1", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("scheduled input pointer", func(t *testing.T) {
		desc := scheduleDescWithArg(&workflows.ScheduledPipelineEnqueueWorkflowInput{
			GlobalRunnerID: "runner-2",
		})
		require.Equal(t, "runner-2", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("workflow input value", func(t *testing.T) {
		desc := scheduleDescWithArg(workflowengine.WorkflowInput{
			Payload: workflows.ScheduledPipelineEnqueueWorkflowInput{GlobalRunnerID: "runner-3"},
		})
		require.Equal(t, "runner-3", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("workflow input pointer with map payload", func(t *testing.T) {
		desc := scheduleDescWithArg(&workflowengine.WorkflowInput{
			Payload: map[string]any{
				"global_runner_id": "runner-4",
			},
		})
		require.Equal(t, "runner-4", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("pipeline input value", func(t *testing.T) {
		desc := scheduleDescWithArg(pipeline.PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"global_runner_id": "runner-5"},
			},
		})
		require.Equal(t, "runner-5", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("pipeline input pointer", func(t *testing.T) {
		desc := scheduleDescWithArg(&pipeline.PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"global_runner_id": "runner-6"},
			},
		})
		require.Equal(t, "runner-6", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("payload scheduled input", func(t *testing.T) {
		payload := mustPayload(t, workflows.ScheduledPipelineEnqueueWorkflowInput{
			GlobalRunnerID: "runner-7",
		})
		desc := scheduleDescWithArg(payload)
		require.Equal(t, "runner-7", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("payload workflow input", func(t *testing.T) {
		payload := mustPayload(t, workflowengine.WorkflowInput{
			Config: map[string]any{"global_runner_id": "runner-8"},
		})
		desc := scheduleDescWithArg(payload)
		require.Equal(t, "runner-8", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("payload pipeline input", func(t *testing.T) {
		payload := mustPayload(t, pipeline.PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"global_runner_id": "runner-9"},
			},
		})
		desc := scheduleDescWithArg(payload)
		require.Equal(t, "runner-9", readGlobalRunnerIDFromScheduleDescription(desc))
	})
}

func TestDecodeScheduledEnqueueInput(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		input, err := decodeScheduledEnqueueInput(map[string]any{
			"pipeline_identifier": "pipeline-1",
			"global_runner_id":    "runner-1",
		})
		require.NoError(t, err)
		require.Equal(t, "pipeline-1", input.PipelineIdentifier)
		require.Equal(t, "runner-1", input.GlobalRunnerID)
	})

	t.Run("invalid payload types", func(t *testing.T) {
		_, err := decodeScheduledEnqueueInput(map[string]any{
			"pipeline_identifier": 123,
		})
		require.Error(t, err)
	})
}

func scheduleDescWithArg(arg any) *client.ScheduleDescription {
	return &client.ScheduleDescription{
		Schedule: client.Schedule{
			Action: &client.ScheduleWorkflowAction{
				Args: []any{arg},
			},
		},
	}
}

func mustPayload(t testing.TB, v any) *commonpb.Payload {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(v)
	require.NoError(t, err)
	return payload
}
