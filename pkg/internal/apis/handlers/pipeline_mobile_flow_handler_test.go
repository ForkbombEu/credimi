// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
)

type pipelineMobileFlowEncodedValue struct {
	devices map[string]any
}

func (v pipelineMobileFlowEncodedValue) HasValue() bool {
	return true
}

func (v pipelineMobileFlowEncodedValue) Get(valuePtr interface{}) error {
	target := valuePtr.(*map[string]any)
	*target = v.devices
	return nil
}

func TestPipelineMobileFlowDevice(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, []string{
		"tenant/runner-1",
	}), nil).Once()
	client.On(
		"QueryWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
		pipeline.PipelineMobileDevicesQuery,
	).Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{
		devices: map[string]any{
			"tenant/runner-1": map[string]any{
				"serial": "emulator-5554",
				"type":   "android_emulator",
			},
		},
	}), nil).Once()

	runnerID, device, apiErr := pipelineMobileFlowDevice(
		context.Background(),
		client,
		"pipeline-1",
		"run-1",
	)
	require.Nil(t, apiErr)
	require.Equal(t, "tenant/runner-1", runnerID)
	require.Equal(t, "emulator-5554", device["serial"])
}

func TestPipelineMobileFlowDeviceRejectsMultipleRunners(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, []string{
		"tenant/runner-1",
		"tenant/runner-2",
	}), nil).Once()

	_, _, apiErr := pipelineMobileFlowDevice(context.Background(), client, "pipeline-1", "run-1")
	require.NotNil(t, apiErr)
	require.Contains(t, apiErr.Error(), "pipeline must have exactly one reserved runner")
}

func TestPipelineMobileFlowDeviceRequiresInitializedDevice(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, []string{
		"tenant/runner-1",
	}), nil).Once()
	client.On(
		"QueryWorkflow",
		mock.Anything,
		"pipeline-1",
		"run-1",
		pipeline.PipelineMobileDevicesQuery,
	).Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{
		devices: map[string]any{},
	}), nil).Once()

	_, _, apiErr := pipelineMobileFlowDevice(context.Background(), client, "pipeline-1", "run-1")
	require.NotNil(t, apiErr)
	require.Contains(t, apiErr.Error(), "pipeline mobile device is not initialized")
}

func TestPipelineMobileFlowDeviceRequiresRunningPipeline(t *testing.T) {
	client := temporalmocks.NewClient(t)
	client.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, []string{
		"tenant/runner-1",
	}), nil).Once()

	_, _, apiErr := pipelineMobileFlowDevice(context.Background(), client, "pipeline-1", "run-1")
	require.NotNil(t, apiErr)
	require.Contains(t, apiErr.Error(), "pipeline workflow is not running")
}

func TestPipelineMobileFlowResponseIncludesOutput(t *testing.T) {
	response := pipelineMobileFlowResponse(workflowengine.WorkflowResult{
		Output: map[string]any{
			"status": "done",
		},
	}, nil)

	require.True(t, response.Success)
	require.Equal(t, map[string]any{"status": "done"}, response.Output)
	require.Nil(t, response.Error)
}

func TestPipelineMobileFlowResponseIncludesError(t *testing.T) {
	response := pipelineMobileFlowResponse(
		workflowengine.WorkflowResult{},
		errors.New("maestro flow failed"),
	)

	require.False(t, response.Success)
	require.Nil(t, response.Output)
	require.Equal(t, "maestro flow failed", response.Error)
}

func TestPipelineMobileFlowActivityOptions(t *testing.T) {
	options := pipelineMobileFlowActivityOptions()

	require.Equal(t, 20*time.Minute, options.ScheduleToCloseTimeout)
	require.Equal(t, 20*time.Minute, options.StartToCloseTimeout)
	require.NotNil(t, options.RetryPolicy)
	require.Equal(t, int32(1), options.RetryPolicy.MaximumAttempts)
}

func pipelineMobileFlowDescription(
	t *testing.T,
	status enums.WorkflowExecutionStatus,
	runnerIDs []string,
) *workflowservice.DescribeWorkflowExecutionResponse {
	t.Helper()

	runnerPayload, err := converter.GetDefaultDataConverter().ToPayload(runnerIDs)
	require.NoError(t, err)

	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
			Type:   &commonpb.WorkflowType{Name: "Dynamic Pipeline Workflow"},
			Status: status,
			SearchAttributes: &commonpb.SearchAttributes{
				IndexedFields: map[string]*commonpb.Payload{
					workflowengine.RunnerIdentifiersSearchAttribute: runnerPayload,
				},
			},
		},
	}
}
