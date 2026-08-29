// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	workflow "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRunnerSetsIntersect(t *testing.T) {
	require.True(t, runnerSetsIntersect([]string{"a", "b"}, []string{"b"}))
	require.True(t, runnerSetsIntersect([]string{" a "}, []string{"a"}))
	require.False(t, runnerSetsIntersect([]string{"a"}, []string{"b"}))
	require.False(t, runnerSetsIntersect(nil, []string{"b"}))
}

func TestRunnerIDsFromSearchAttributes(t *testing.T) {
	require.Nil(t, runnerIDsFromSearchAttributes(nil))

	stringList := DecodedWorkflowSearchAttributes{
		"RunnerIdentifiers": []string{"runner-1", "runner-2"},
	}
	require.Equal(
		t,
		[]string{"runner-1", "runner-2"},
		runnerIDsFromSearchAttributes(&stringList),
	)

	anyList := DecodedWorkflowSearchAttributes{
		"RunnerIdentifiers": []any{"runner-1", "", 42},
	}
	require.Equal(t, []string{"runner-1"}, runnerIDsFromSearchAttributes(&anyList))

	missing := DecodedWorkflowSearchAttributes{"Other": "x"}
	require.Nil(t, runnerIDsFromSearchAttributes(&missing))
}

func progressHistoryClient(
	t *testing.T,
	executions []*workflow.WorkflowExecutionInfo,
) *temporalmocks.Client {
	t.Helper()
	mockClient := &temporalmocks.Client{}
	mockClient.
		On("ListWorkflow", mock.Anything, mock.Anything).
		Return(&workflowservice.ListWorkflowExecutionsResponse{Executions: executions}, nil)
	return mockClient
}

func TestExpectedPipelineDurationMedian(t *testing.T) {
	now := time.Now()
	executions := make([]*workflow.WorkflowExecutionInfo, 0, 3)
	for _, seconds := range []float64{60, 120, 300} {
		executions = append(executions, &workflow.WorkflowExecutionInfo{
			StartTime:     timestamppb.New(now.Add(-2 * time.Hour)),
			ExecutionTime: timestamppb.New(now.Add(-2 * time.Hour)),
			CloseTime: timestamppb.New(
				now.Add(-2*time.Hour + time.Duration(seconds)*time.Second),
			),
			Status: 1,
		})
	}

	duration, samples := expectedPipelineDuration(
		context.Background(),
		progressHistoryClient(t, executions),
		"ns",
		"tenant/pipeline",
		nil,
	)
	require.Equal(t, 3, samples)
	require.InDelta(t, 120.0, duration, 0.001)
}

func TestExpectedPipelineDurationFiltersByRunner(t *testing.T) {
	converter := temporalcrypto.DataConverter()
	runnerPayloads := map[string]*commonpb.Payload{}
	for _, runner := range []string{"runner-1", "runner-2"} {
		payload, err := converter.ToPayloads([]string{runner})
		require.NoError(t, err)
		runnerPayloads[runner] = payload.GetPayloads()[0]
	}

	now := time.Now()
	newExecution := func(runners string, durationSeconds float64) *workflow.WorkflowExecutionInfo {
		info := &workflow.WorkflowExecutionInfo{
			StartTime:     timestamppb.New(now.Add(-2 * time.Hour)),
			ExecutionTime: timestamppb.New(now.Add(-2 * time.Hour)),
			CloseTime: timestamppb.New(
				now.Add(-2*time.Hour + time.Duration(durationSeconds)*time.Second),
			),
			Status: 1,
		}
		if runners != "" {
			info.SearchAttributes = &commonpb.SearchAttributes{
				IndexedFields: map[string]*commonpb.Payload{
					"RunnerIdentifiers": runnerPayloads[runners],
				},
			}
		}
		return info
	}

	// runner-2 runs take 120s; the runner-1 run takes 600s and must be excluded.
	executions := []*workflow.WorkflowExecutionInfo{
		newExecution("runner-2", 120),
		newExecution("runner-2", 120),
		newExecution("runner-1", 600),
	}

	duration, samples := expectedPipelineDuration(
		context.Background(),
		progressHistoryClient(t, executions),
		"ns",
		"tenant/pipeline",
		[]string{"runner-2"},
	)
	require.Equal(t, 2, samples)
	require.InDelta(t, 120.0, duration, 0.001)
}

func TestComputePipelineProgressGuards(t *testing.T) {
	require.Nil(t, computePipelineProgress(
		context.Background(), nil, "tenant/pipeline", nil, "",
	))

	builder := newPipelineExecutionSummaryBuilder(nil, progressHistoryClient(t, nil), "ns", "UTC")
	require.Nil(t, computePipelineProgress(
		context.Background(), builder, "", nil, "",
	))
	require.Nil(t, computePipelineProgress(
		context.Background(), builder, "tenant/pipeline", nil, "not-a-time",
	))

	// No history: no estimate.
	require.Nil(t, computePipelineProgress(
		context.Background(),
		builder,
		"tenant/pipeline",
		nil,
		time.Now().Add(-time.Minute).Format(time.RFC3339Nano),
	))
}

func TestComputePipelineProgressEstimates(t *testing.T) {
	now := time.Now()
	executions := []*workflow.WorkflowExecutionInfo{
		{
			StartTime:     timestamppb.New(now.Add(-3 * time.Hour)),
			ExecutionTime: timestamppb.New(now.Add(-3 * time.Hour)),
			CloseTime:     timestamppb.New(now.Add(-3*time.Hour + 120*time.Second)),
			Status:        1,
		},
	}

	progress := computePipelineProgress(
		context.Background(),
		newPipelineExecutionSummaryBuilder(nil, progressHistoryClient(t, executions), "ns", "UTC"),
		"tenant/pipeline",
		nil,
		now.Add(-30*time.Second).Format(time.RFC3339Nano),
	)
	require.NotNil(t, progress)
	require.Equal(t, 1, progress.SampleSize)
	require.InDelta(t, 120.0, progress.ExpectedDurationSeconds, 0.001)
	require.InDelta(t, 30.0, progress.ElapsedSeconds, 2.0)
	require.InDelta(t, 25.0, progress.Percent, 3.0)
	require.InDelta(t, 90.0, progress.EtaSeconds, 3.0)
}

func TestExpectedPipelineDurationListError(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.
		On("ListWorkflow", mock.Anything, mock.Anything).
		Return(nil, &serviceerror.NotFound{Message: "none"})

	duration, samples := expectedPipelineDuration(
		context.Background(),
		mockClient,
		"ns",
		"tenant/pipeline",
		nil,
	)
	require.Zero(t, duration)
	require.Zero(t, samples)
}
