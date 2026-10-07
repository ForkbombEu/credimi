// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestAggregateScoreboardWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectError    bool
		validateOutput func(t *testing.T, output AggregateScoreboardWorkflowOutput)
	}{
		{
			name: "success aggregates pipelines and execution details",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerScoreboardActivities(env)

				mockNamespaces(env, []string{"namespace-1", "namespace-2"})
				mockNamespaceScoreboard(env, "namespace-1", []map[string]any{
					{
						"pipeline_id":                "pipe-1",
						"pipeline_name":              "Pipeline 1",
						"pipeline_identifier":        "namespace-1/pipe-1",
						"device_types":               []any{"android", "ios"},
						"device_ids":                 []any{"runner-1", "runner-2"},
						"total_runs":                 10.0,
						"total_successes":            8.0,
						"manual_executions":          3.0,
						"scheduled_executions":       7.0,
						"ci_executions":              2.0,
						"min_execution_time":         "10m0s",
						"min_execution_time_seconds": 600.0,
						"first_execution_date":       "2026-01-01T00:00:00Z",
						"last_execution_date":        "2026-04-01T00:00:00Z",
						"last_run": map[string]any{
							"workflow_id": "wf-1",
							"run_id":      "run-1",
							"start_time":  "2026-04-01T10:00:00Z",
						},
					},
				})
				mockNamespaceScoreboard(env, "namespace-2", []map[string]any{
					{
						"pipeline_id":                "pipe-1",
						"pipeline_name":              "Pipeline 1",
						"pipeline_identifier":        "namespace-2/pipe-1",
						"device_types":               []any{"android"},
						"device_ids":                 []any{"runner-3"},
						"total_runs":                 5.0,
						"total_successes":            5.0,
						"manual_executions":          1.0,
						"scheduled_executions":       4.0,
						"ci_executions":              3.0,
						"min_execution_time":         "2m0s",
						"min_execution_time_seconds": 120.0,
						"first_execution_date":       "2026-02-01T00:00:00Z",
						"last_execution_date":        "2026-04-03T00:00:00Z",
						"last_run": map[string]any{
							"workflow_id": "wf-2",
							"run_id":      "run-2",
							"start_time":  "2026-04-03T10:00:00Z",
						},
					},
					{
						"pipeline_id":                "pipe-2",
						"pipeline_name":              "Pipeline 2",
						"pipeline_identifier":        "namespace-2/pipe-2",
						"device_types":               []any{"android"},
						"device_ids":                 []any{"runner-3"},
						"total_runs":                 5.0,
						"total_successes":            5.0,
						"manual_executions":          1.0,
						"scheduled_executions":       4.0,
						"ci_executions":              5.0,
						"min_execution_time":         "2m0s",
						"min_execution_time_seconds": 120.0,
						"first_execution_date":       "2026-02-02T00:00:00Z",
						"last_execution_date":        "2026-02-03T00:00:00Z",
						"last_run": map[string]any{
							"workflow_id": "wf-3",
							"run_id":      "run-3",
							"start_time":  "2026-02-03T10:00:00Z",
						},
					},
				})
				mockExecutionDetailsForRun(env, "wf-2", "run-2", map[string]any{
					"pipeline_name": "Pipeline 1",
					"org_logo":      "https://example.com/logo.png",
					"video":         "https://example.com/video.mp4",
					"screenshots":   "https://example.com/screenshot.png",
					"logs":          "https://example.com/logs.txt",
					"wallet_used": []any{
						"org/wallet-a",
						"org/wallet-b",
					},
					"wallet_version_used": []any{
						"org/wallet-a/v1-0-0",
						"org/wallet-b/v2-0-0",
					},
					"maestro_scripts": []any{
						"org/action/maestro-1",
						"org/action/maestro-2",
					},
					"credentials": []any{
						"org/issuer/credential-1",
						"org/issuer/credential-2",
					},
					"issuers": []any{"org/issuer"},
					"use_case_verifications": []any{
						"org/verifier/uc-1",
						"org/verifier/uc-2",
					},
					"verifiers":         []any{"org/verifier"},
					"conformance_tests": []any{"conformance/check-1", "conformance/check-2"},
					"custom_checks":     []any{"custom/check-1"},
				})
				mockExecutionDetailsForRun(env, "wf-3", "run-3", map[string]any{
					"pipeline_name": "Pipeline 2",
				})
				mockSaveResults(env)
			},
			validateOutput: func(t *testing.T, output AggregateScoreboardWorkflowOutput) {
				require.Equal(t, 2, output.NamespacesProcessed)
				require.Equal(t, 0, output.NamespacesFailed)
				require.Len(t, output.AggregatedPipelines, 2)

				var pipeline1, pipeline2 *AggregatedPipelineStats
				for i := range output.AggregatedPipelines {
					switch output.AggregatedPipelines[i].PipelineID {
					case "pipe-1":
						pipeline1 = &output.AggregatedPipelines[i]
					case "pipe-2":
						pipeline2 = &output.AggregatedPipelines[i]
					}
				}

				require.NotNil(t, pipeline1)
				require.Equal(t, 15, pipeline1.TotalRuns)
				require.Equal(t, 13, pipeline1.TotalSuccesses)
				require.Equal(t, 86.67, pipeline1.SuccessRate)
				require.Equal(t, "2m0s", pipeline1.MinExecutionTime)
				require.Equal(t, 120, pipeline1.MinExecutionTimeSeconds)
				require.Equal(t, "2026-01-01T00:00:00Z", pipeline1.FirstExecutionDate)
				require.Equal(t, "2026-04-03T00:00:00Z", pipeline1.LastExecutionDate)
				require.Equal(t, 4, pipeline1.ManualExecutions)
				require.Equal(t, 11, pipeline1.ScheduledExecutions)
				require.Equal(t, 5, pipeline1.CIExecutions)
				require.ElementsMatch(
					t,
					[]string{"runner-1", "runner-2", "runner-3"},
					pipeline1.DeviceIDs,
				)
				require.ElementsMatch(t, []string{"android", "ios"}, pipeline1.DeviceTypes)
				require.NotNil(t, pipeline1.LastExecution)
				require.Equal(t, "Pipeline 1", pipeline1.LastExecution.PipelineName)
				require.Equal(t, "https://example.com/logo.png", pipeline1.LastExecution.OrgLogo)
				require.Equal(t, "https://example.com/video.mp4", pipeline1.LastExecution.Video)

				require.NotNil(t, pipeline2)
				require.Equal(t, 5, pipeline2.TotalRuns)
				require.Equal(t, 5, pipeline2.TotalSuccesses)
				require.Equal(t, 5, pipeline2.CIExecutions)
				require.ElementsMatch(t, []string{"runner-3"}, pipeline2.DeviceIDs)
				require.NotNil(t, pipeline2.LastExecution)
				require.Equal(t, "Pipeline 2", pipeline2.LastExecution.PipelineName)
			},
		},
		{
			name: "success keeps partial results when one namespace fetch fails",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerScoreboardActivities(env)

				mockNamespaces(env, []string{"namespace-1", "namespace-2"})
				mockNamespaceScoreboard(env, "namespace-1", []map[string]any{
					{
						"pipeline_id":                "pipe-1",
						"pipeline_name":              "Pipeline 1",
						"pipeline_identifier":        "namespace-1/pipe-1",
						"device_types":               []any{"android"},
						"device_ids":                 []any{"runner-1"},
						"total_runs":                 4.0,
						"total_successes":            3.0,
						"manual_executions":          1.0,
						"scheduled_executions":       3.0,
						"min_execution_time":         "45s",
						"min_execution_time_seconds": 45.0,
						"first_execution_date":       "2026-04-01T00:00:00Z",
						"last_execution_date":        "2026-04-02T00:00:00Z",
						"last_run": map[string]any{
							"workflow_id": "wf-1",
							"run_id":      "run-1",
							"start_time":  "2026-04-02T10:00:00Z",
						},
					},
				})
				env.OnActivity(
					GetNamespaceScoreboardActivityName,
					mock.Anything,
					scoreboardNamespacePayload("namespace-2"),
				).
					Return(workflowengine.ActivityResult{}, errors.New("boom")).
					Once()
				mockExecutionDetails(env, map[string]any{
					"pipeline_name": "Pipeline 1",
				})
				mockSaveResults(env)
			},
			validateOutput: func(t *testing.T, output AggregateScoreboardWorkflowOutput) {
				require.Equal(t, 1, output.NamespacesProcessed)
				require.Equal(t, 1, output.NamespacesFailed)
				require.Len(t, output.FailedNamespaces, 1)
				require.Equal(t, []string{"namespace-2"}, output.FailedNamespaces)
				require.Len(t, output.AggregatedPipelines, 1)
				require.NotNil(t, output.AggregatedPipelines[0].LastExecution)
			},
		},
		{
			name: "success returns empty output when no namespaces are found",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerScoreboardActivities(env)
				mockNamespaces(env, []string{})
			},
			validateOutput: func(t *testing.T, output AggregateScoreboardWorkflowOutput) {
				require.Equal(t, 0, output.NamespacesProcessed)
				require.Equal(t, 0, output.NamespacesFailed)
				require.Empty(t, output.AggregatedPipelines)
			},
		},
		{
			name: "failure when namespace list activity fails",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerScoreboardActivities(env)
				env.OnActivity(ListScoreboardNamespacesActivityName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, errors.New("boom")).
					Once()
			},
			expectError: true,
		},
		{
			name: "failure when namespaces output is malformed",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerScoreboardActivities(env)
				env.OnActivity(ListScoreboardNamespacesActivityName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: []any{"namespace-1", 10},
					}, nil).
					Once()
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			activityOptions := DefaultActivityOptions
			activityOptions.RetryPolicy = &temporal.RetryPolicy{
				MaximumAttempts: 1,
			}

			w := NewAggregateScoreboardWorkflow()
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				// BuildWorkflow reads app_url to build the Temporal UI link.
				Config:          map[string]any{"app_url": "https://example.com"},
				ActivityOptions: &activityOptions,
			})

			require.True(t, env.IsWorkflowCompleted())
			env.AssertExpectations(t)

			var result workflowengine.WorkflowResult
			err := env.GetWorkflowResult(&result)
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			jsonBytes, err := json.Marshal(result.Output)
			require.NoError(t, err)

			var output AggregateScoreboardWorkflowOutput
			require.NoError(t, json.Unmarshal(jsonBytes, &output))
			if tc.validateOutput != nil {
				tc.validateOutput(t, output)
			}
		})
	}
}

func TestAggregateScoreboardWorkflowStart(t *testing.T) {
	origStart := aggregateScoreboardStartWorkflowWithOptions
	t.Cleanup(func() {
		aggregateScoreboardStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	aggregateScoreboardStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		capturedInput = input
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}

	w := NewAggregateScoreboardWorkflow()
	input := workflowengine.WorkflowInput{
		Config: map[string]any{"app_url": "https://example.com"},
	}

	result, err := w.Start("default", input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "default", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, AggregateScoreboardTaskQueue, capturedOptions.TaskQueue)
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "aggregate-scoreboard-"))
}

func TestAggregateScoreboardWorkflowOrdersMixedTimestampPrecision(t *testing.T) {
	w := NewAggregateScoreboardWorkflow()
	stats := &AggregatedPipelineStats{}

	w.updateDates(stats, &namespacePipelineStats{
		FirstExecutionDate: "2026-04-21T10:00:00Z",
		LastExecutionDate:  "2026-04-21T10:00:00Z",
	})
	w.updateDates(stats, &namespacePipelineStats{
		FirstExecutionDate: "2026-04-21T09:59:59.999999999Z",
		LastExecutionDate:  "2026-04-21T10:00:00.1Z",
	})

	lastRunMap := map[string]*pipelineRunRef{}
	w.trackLastRun(
		&namespacePipelineStats{
			LastRun: &namespacePipelineLastRun{
				WorkflowID: "whole-second",
				RunID:      "run-1",
				StartTime:  "2026-04-21T10:00:00Z",
			},
		},
		"namespace-1",
		"pipe-1",
		lastRunMap,
	)
	w.trackLastRun(
		&namespacePipelineStats{
			LastRun: &namespacePipelineLastRun{
				WorkflowID: "fractional-second",
				RunID:      "run-2",
				StartTime:  "2026-04-21T10:00:00.1Z",
			},
		},
		"namespace-1",
		"pipe-1",
		lastRunMap,
	)

	require.Equal(t, "2026-04-21T09:59:59.999999999Z", stats.FirstExecutionDate)
	require.Equal(t, "2026-04-21T10:00:00.1Z", stats.LastExecutionDate)
	require.Equal(t, "fractional-second", lastRunMap["pipe-1"].WorkflowID)
}

func registerScoreboardActivities(env *testsuite.TestWorkflowEnvironment) {
	stub := func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
		return workflowengine.ActivityResult{}, nil
	}
	for _, name := range []string{
		ListScoreboardNamespacesActivityName,
		GetNamespaceScoreboardActivityName,
		GetScoreboardExecutionDetailsActivityName,
		SaveScoreboardResultsActivityName,
	} {
		env.RegisterActivityWithOptions(stub, activity.RegisterOptions{Name: name})
	}
}

func scoreboardNamespacePayload(namespace string) any {
	return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
		payload, ok := input.Payload.(map[string]any)
		return ok && payload["namespace"] == namespace
	})
}

func mockNamespaces(env *testsuite.TestWorkflowEnvironment, namespaces []string) {
	output := make([]any, 0, len(namespaces))
	for _, namespace := range namespaces {
		output = append(output, namespace)
	}

	env.OnActivity(ListScoreboardNamespacesActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: output}, nil).
		Once()
}

func mockNamespaceScoreboard(
	env *testsuite.TestWorkflowEnvironment,
	namespace string,
	pipelines []map[string]any,
) {
	output := make([]any, 0, len(pipelines))
	for _, pipeline := range pipelines {
		output = append(output, pipeline)
	}

	env.OnActivity(
		GetNamespaceScoreboardActivityName,
		mock.Anything,
		scoreboardNamespacePayload(namespace),
	).
		Return(workflowengine.ActivityResult{Output: output}, nil).
		Once()
}

func mockExecutionDetails(env *testsuite.TestWorkflowEnvironment, details map[string]any) {
	env.OnActivity(GetScoreboardExecutionDetailsActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: details}, nil).
		Once()
}

func mockExecutionDetailsForRun(
	env *testsuite.TestWorkflowEnvironment,
	workflowID string,
	runID string,
	details map[string]any,
) {
	env.OnActivity(
		GetScoreboardExecutionDetailsActivityName,
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return payload["workflow_id"] == workflowID && payload["run_id"] == runID &&
				strings.HasPrefix(workflowengine.AsString(payload["namespace"]), "namespace-")
		}),
	).
		Return(workflowengine.ActivityResult{Output: details}, nil).
		Once()
}

func mockSaveResults(env *testsuite.TestWorkflowEnvironment) {
	env.OnActivity(
		SaveScoreboardResultsActivityName,
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			pipelines, ok := payload["aggregated_pipelines"].([]any)
			return ok && len(pipelines) > 0
		}),
	).
		Return(workflowengine.ActivityResult{Output: map[string]any{"records_count": 1}}, nil).
		Once()
}
