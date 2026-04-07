// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestAggregateScoreboardWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		config         map[string]any
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectError    bool
		validateOutput func(t *testing.T, output AggregateScoreboardWorkflowOutput)
	}{
		{
			name: "Success: aggregates pipelines with full last execution details",
			config: map[string]any{
				"app_url": "https://example.com",
				"api_key": "test-api-key",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewInternalHTTPActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{
								"namespaces": []interface{}{"namespace-1", "namespace-2","namespace-2"},
							},
						},
					}, nil).Once()

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": []interface{}{
								map[string]any{
									"pipeline_id":          "pipe-1",
									"pipeline_name":        "Pipeline 1",
									"pipeline_identifier":  "namespace-1/pipe-1",
									"runner_types":         []interface{}{"android", "ios"},
									"runners":              []interface{}{"runner-1", "runner-2"},
									"total_runs":           10.0,
									"total_successes":      8.0,
									"success_rate":         80.0,
									"manual_executions":    3.0,
									"scheduled_executions": 7.0,
									"min_execution_time":   "1m30s",
									"first_execution_date": "2026-01-01T00:00:00Z",
									"last_execution_date":  "2026-04-01T00:00:00Z",
									"last_successful_run": map[string]any{
										"workflow_id":         "wf-1",
										"run_id":              "run-1",
										"start_time":          "2026-04-01T10:00:00Z",
									},
								},
							},
						},
					}, nil).Once()

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": []interface{}{
								map[string]any{
									"pipeline_id":          "pipe-1",
									"pipeline_name":        "Pipeline 1",
									"pipeline_identifier":  "namespace-2/pipe-1",
									"runner_types":         []interface{}{"android"},
									"runners":              []interface{}{"runner-3"},
									"total_runs":           5.0,
									"total_successes":      5.0,
									"success_rate":         100.0,
									"manual_executions":    1.0,
									"scheduled_executions": 4.0,
									"min_execution_time":   "2m",
									"first_execution_date": "2026-02-01T00:00:00Z",
									"last_execution_date":  "2026-04-03T00:00:00Z",
									"last_successful_run": map[string]any{
										"workflow_id":         "wf-2",
										"run_id":              "run-2",
										"start_time":          "2026-04-03T10:00:00Z",
									},
								},
							},
						},
					}, nil).Once()
				
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": []interface{}{
								map[string]any{
									"pipeline_id":          "pipe-2",
									"pipeline_name":        "Pipeline 2",
									"pipeline_identifier":  "namespace-2/pipe-2",
									"runner_types":         []interface{}{"android"},
									"runners":              []interface{}{"runner-3"},
									"total_runs":           5.0,
									"total_successes":      5.0,
									"success_rate":         100.0,
									"manual_executions":    1.0,
									"scheduled_executions": 4.0,
									"min_execution_time":   "2m",
									"first_execution_date": "2026-02-02T00:00:00Z",
									"last_execution_date":  "2026-02-03T00:00:00Z",
									"last_successful_run": map[string]any{
										"workflow_id":         "wf-3",
										"run_id":              "run-3",
										"start_time":          "2026-02-03T10:00:00Z",
									},
								},
							},
						},
					}, nil).Once()

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{
								"pipeline_name":          "Pipeline 1",
								"org_logo":               "https://example.com/logo.png",
								"video":                  "https://example.com/video.mp4",
								"screenshots":            "https://example.com/screenshot.png",
								"logs":                   "https://example.com/logs.txt",
								"wallet_used":            []interface{}{"org/wallet-a", "org/wallet-b"},
								"wallet_version_used":    []interface{}{"org/wallet-a/v1-0-0", "org/wallet-b/v2-0-0"},
								"maestro_scripts":        []interface{}{"org/action/maestro-1", "org/action/maestro-2"},
								"credentials":            []interface{}{"org/issuer/credential-1", "org/issuer/credential-2"},
								"issuers":                []interface{}{"org/issuer"},
								"use_case_verifications": []interface{}{"org/verifier/uc-1", "org/verifier/uc-2"},
								"verifiers":              []interface{}{"org/verifier"},
								"conformance_tests":      []interface{}{"conformance/check-1", "conformance/check-2"},
								"custom_checks":          []interface{}{"custom/check-1"},
							},
						},
					}, nil).Once()
			},
			expectError: false,
			validateOutput: func(t *testing.T, output AggregateScoreboardWorkflowOutput) {
				require.Equal(t, 3, output.NamespacesProcessed)
				require.Equal(t, 0, output.NamespacesFailed)
				require.Len(t, output.AggregatedPipelines, 2)

				var pipeline1, pipeline2 *AggregatedPipelineStats
    			for i := range output.AggregatedPipelines {
        			if output.AggregatedPipelines[i].PipelineID == "pipe-1" {
            			pipeline1 = &output.AggregatedPipelines[i]
        			} else if output.AggregatedPipelines[i].PipelineID == "pipe-2" {
        				pipeline2 = &output.AggregatedPipelines[i]
    				}
				}

				require.NotNil(t, output.GlobalLatestExecution)
    			require.Equal(t, "Pipeline 1", output.GlobalLatestExecution.PipelineName)

				lastExec := output.GlobalLatestExecution

				require.Equal(t, "Pipeline 1", lastExec.PipelineName)
				require.Equal(t, "https://example.com/logo.png", lastExec.OrgLogo)
				require.Equal(t, "https://example.com/video.mp4", lastExec.Video)
				require.Equal(t, "https://example.com/screenshot.png", lastExec.Screenshot)
				require.Equal(t, "https://example.com/logs.txt", lastExec.Logs)

				require.ElementsMatch(t, []string{"org/wallet-a", "org/wallet-b"}, lastExec.WalletUsed)
				require.ElementsMatch(t, []string{"org/wallet-a/v1-0-0", "org/wallet-b/v2-0-0"}, lastExec.WalletVersionUsed)

				require.ElementsMatch(t, []string{"org/action/maestro-1", "org/action/maestro-2"}, lastExec.MaestroScripts)

				require.ElementsMatch(t, []string{"org/issuer/credential-1", "org/issuer/credential-2"}, lastExec.Credentials)
				require.ElementsMatch(t, []string{"org/issuer"}, lastExec.Issuers)

				require.ElementsMatch(t, []string{"org/verifier/uc-1", "org/verifier/uc-2"}, lastExec.UseCaseVerifications)
				require.ElementsMatch(t, []string{"org/verifier"}, lastExec.Verifiers)

				require.ElementsMatch(t, []string{"conformance/check-1", "conformance/check-2"}, lastExec.ConformanceTests)
				require.ElementsMatch(t, []string{"custom/check-1"}, lastExec.CustomChecks)

				require.Equal(t, 15, pipeline1.TotalRuns)      
				require.Equal(t, 13, pipeline1.TotalSuccesses) 
				require.Equal(t, 86.67, pipeline1.SuccessRate) 
				require.Equal(t, "1m30s", pipeline1.MinExecutionTime)
				require.Equal(t, "2026-01-01T00:00:00Z", pipeline1.FirstExecutionDate)
				require.Equal(t, "2026-04-03T00:00:00Z", pipeline1.LastExecutionDate)
				require.Equal(t, 11, pipeline1.ScheduledExecutions)
				require.ElementsMatch(t, []string{"runner-1", "runner-2", "runner-3"}, pipeline1.Runners)
				require.ElementsMatch(t, []string{"android", "ios"}, pipeline1.RunnerTypes)

				require.NotNil(t, pipeline2)
    			require.Equal(t, 5, pipeline2.TotalRuns)
    			require.Equal(t, 5, pipeline2.TotalSuccesses)
    			require.ElementsMatch(t, []string{"runner-3"}, pipeline2.Runners)
			},
		},
		{
			name: "Success: no namespaces found",
			config: map[string]any{
				"app_url": "https://example.com",
				"api_key": "test-api-key",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewInternalHTTPActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{
								"namespaces": []interface{}{},
							},
						},
					}, nil)
			},
			expectError: false,
			validateOutput: func(t *testing.T, output AggregateScoreboardWorkflowOutput) {
				require.Equal(t, 0, output.NamespacesProcessed)
				require.Len(t, output.AggregatedPipelines, 0)
				require.Nil(t, output.GlobalLatestExecution)
			},
		},
		{
			name: "Failure: missing app_url",
			config: map[string]any{
				"api_key": "test-api-key",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectError:    true,
		},
		{
			name: "Failure: missing api_key",
			config: map[string]any{
				"app_url": "https://example.com",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectError:    true,
		},
		{
			name: "Failure: get namespaces activity fails",
			config: map[string]any{
				"app_url": "https://example.com",
				"api_key": "test-api-key",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewInternalHTTPActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, mock.Anything)
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			w := NewAggregateScoreboardWorkflow()
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Config: tc.config,
			})

			require.True(t, env.IsWorkflowCompleted())

			var result workflowengine.WorkflowResult
			err := env.GetWorkflowResult(&result)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				jsonBytes, err := json.Marshal(result.Output)
				require.NoError(t, err)

				var output AggregateScoreboardWorkflowOutput
				err = json.Unmarshal(jsonBytes, &output)
				require.NoError(t, err)

				if tc.validateOutput != nil {
					tc.validateOutput(t, output)
				}
			}
		})
	}
}
