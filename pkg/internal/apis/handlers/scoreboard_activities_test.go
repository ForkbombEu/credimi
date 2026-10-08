// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// runScoreboardActivity executes act in a Temporal test activity environment
// and decodes its output into T.
func runScoreboardActivity[T any](
	t *testing.T,
	act workflowengine.ExecutableActivity,
	payload any,
) (T, error) {
	t.Helper()
	var out T
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	value, err := env.ExecuteActivity(act.Name(), workflowengine.ActivityInput{Payload: payload})
	if err != nil {
		return out, err
	}
	var result workflowengine.ActivityResult
	require.NoError(t, value.Get(&result))
	raw, err := json.Marshal(result.Output)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &out))
	return out, nil
}

func requireActivityErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected application error, got %v", err)
	require.Equal(t, errorcodes.Codes[code].Code, appErr.Type())
}

func stubScoreboardTemporalClient(t *testing.T, c client.Client) {
	t.Helper()
	originalClient := pipelineResultsTemporalClient
	t.Cleanup(func() { pipelineResultsTemporalClient = originalClient })
	pipelineResultsTemporalClient = func(_ string) (client.Client, error) {
		return c, nil
	}
}

func TestListScoreboardNamespacesActivity(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	expected, err := allOrganizationNamespaces(app)
	require.NoError(t, err)
	require.Contains(t, expected, "usera-s-organization")

	namespaces, err := runScoreboardActivity[[]string](
		t,
		NewListScoreboardNamespacesActivity(app),
		nil,
	)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, namespaces)
}

func TestGetNamespaceScoreboardActivity(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	createRunnerRecord(t, app, orgID, "runner-android")
	createRunnerRecord(t, app, orgID, "runner-ios")
	createRunnerRecord(t, app, orgID, "runner-default")

	pipeline1 := createPipelineRecord(t, app, orgID, "Android E2E Tests")
	pipeline2 := createPipelineRecord(t, app, orgID, "iOS E2E Tests")
	pipeline3 := createPipelineRecord(t, app, orgID, "iOS E3E Tests")

	pipeline1.Set("published", true)
	require.NoError(t, app.Save(pipeline1))
	pipeline2.Set("published", true)
	require.NoError(t, app.Save(pipeline2))
	pipeline3.Set("published", false)
	require.NoError(t, app.Save(pipeline3))

	pipeline1Canonified := pipeline1.GetString("canonified_name")
	pipeline2Canonified := pipeline2.GetString("canonified_name")
	pipeline3Canonified := pipeline3.GetString("canonified_name")

	namespace := "usera-s-organization"

	mockClient := &temporalmocks.Client{}

	now := time.Now()
	exec1 := buildPipelineExecutionInfoWithRunner(
		t,
		"Pipeline-Sched-wf-1",
		"run-1",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-android"},
		now.Add(-2*time.Hour).Add(-2*time.Minute).Add(-33*time.Second),
		now.Add(-2*time.Hour),
	)
	exec1.Info.Memo = nil
	exec2 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-2",
		"run-2",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-android"},
		now.Add(-1*time.Hour).Add(-1*time.Minute).Add(-45*time.Second),
		now.Add(-1*time.Hour),
	)
	exec3 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-3",
		"run-3",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Failed",
		[]string{"usera-s-organization/runner-ios"},
		now.Add(-30*time.Minute).Add(-30*time.Second),
		now.Add(-30*time.Minute),
	)
	exec4 := buildPipelineExecutionInfoWithRunner(
		t,
		"Pipeline-Sched-wf-4",
		"run-4",
		fmt.Sprintf("%s/%s", namespace, pipeline2Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		now.Add(-5*time.Minute).Add(-10*time.Second),
		now.Add(-1*time.Minute),
	)
	exec5 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-5",
		"run-5",
		fmt.Sprintf("%s/%s", namespace, pipeline3Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		now.Add(-2*time.Hour).Add(-5*time.Minute).Add(-10*time.Second),
		now.Add(-1*time.Minute),
	)
	exec6 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-6",
		"run-6",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-default"},
		now.Add(-10*time.Minute).Add(-5*time.Second),
		now.Add(-10*time.Minute),
	)
	exec7 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-7",
		"run-7",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Canceled",
		[]string{"usera-s-organization/runner-default"},
		now.Add(-9*time.Minute).Add(-5*time.Second),
		now.Add(-9*time.Minute),
	)
	exec8 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-8",
		"run-8",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Terminated",
		[]string{"usera-s-organization/runner-default"},
		now.Add(-8*time.Minute).Add(-5*time.Second),
		now.Add(-8*time.Minute),
	)
	exec9 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-9",
		"run-9",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Running",
		[]string{"usera-s-organization/runner-default"},
		now.Add(-7*time.Minute),
		now.Add(-7*time.Minute),
	)
	createPipelineResultWithType(t, app, orgID, pipeline1.Id, "Pipeline-Sched-wf-1", "run-1",
		pipelineinternal.RunTypeManual)
	createPipelineResultWithType(t, app, orgID, pipeline1.Id, "wf-2", "run-2",
		pipelineinternal.RunTypeScheduled)
	createPipelineResultWithType(t, app, orgID, pipeline1.Id, "wf-3", "run-3",
		pipelineinternal.RunTypeCI)
	createPipelineResultWithType(t, app, orgID, pipeline2.Id, "Pipeline-Sched-wf-4", "run-4",
		pipelineinternal.RunTypeCI)
	createPipelineResultWithType(t, app, orgID, pipeline3.Id, "wf-5", "run-5",
		pipelineinternal.RunTypeManual)
	mockClient.
		On("ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return !strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				exec1.Info, exec2.Info, exec3.Info, exec4.Info, exec5.Info,
				exec6.Info, exec7.Info, exec8.Info, exec9.Info,
			},
		}, nil).
		Once()
	stubScoreboardTemporalClient(t, mockClient)

	response, err := runScoreboardActivity[[]PipelineStatsResponse](
		t,
		NewGetNamespaceScoreboardActivity(app),
		workflows.ScoreboardNamespaceInput{Namespace: namespace},
	)
	require.NoError(t, err)
	require.Len(t, response, 2)

	var stats1, stats2, stats3 *PipelineStatsResponse
	for i := range response {
		switch response[i].PipelineName {
		case "Android E2E Tests":
			stats1 = &response[i]
		case "iOS E2E Tests":
			stats2 = &response[i]
		case "iOS E3E Tests":
			stats3 = &response[i]
		}
	}

	require.NotNil(t, stats1)
	require.Equal(t, 4, stats1.TotalRuns)
	require.Equal(t, 3, stats1.TotalSuccesses)
	require.Equal(t, 1, stats1.ScheduledExecutions)
	require.Equal(t, 2, stats1.ManualExecutions)
	require.Equal(t, 1, stats1.CIExecutions)
	require.ElementsMatch(
		t,
		[]string{
			"usera-s-organization/runner-android",
			"usera-s-organization/runner-ios",
			"usera-s-organization/runner-default",
		},
		stats1.DeviceIDs,
	)
	require.Equal(t, "5s", stats1.MinExecutionTime)
	require.Equal(t, 5, stats1.MinExecutionTimeSeconds)
	actualFirstTime, err := time.Parse(time.RFC3339Nano, stats1.FirstExecutionDate)
	require.NoError(t, err)
	require.WithinDuration(t, exec1.Info.GetStartTime().AsTime(), actualFirstTime, time.Second)
	actualLastTime, err := time.Parse(time.RFC3339Nano, stats1.LastExecutionDate)
	require.NoError(t, err)
	require.WithinDuration(t, exec6.Info.GetStartTime().AsTime(), actualLastTime, time.Second)
	require.Equal(t, 75.00, stats1.SuccessRate)
	require.NotNil(t, stats1.LastRun)
	require.Equal(t, "wf-6", stats1.LastRun.WorkflowID)
	require.Equal(t, "run-6", stats1.LastRun.RunID)

	require.NotNil(t, stats2)
	require.Equal(t, 1, stats2.TotalRuns)
	require.Equal(t, 1, stats2.TotalSuccesses)
	require.Equal(t, 0, stats2.ScheduledExecutions)
	require.Equal(t, 0, stats2.ManualExecutions)
	require.Equal(t, 1, stats2.CIExecutions)
	require.ElementsMatch(
		t,
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		stats2.DeviceIDs,
	)
	require.Equal(t, "4m10s", stats2.MinExecutionTime)
	require.Equal(t, 250, stats2.MinExecutionTimeSeconds)
	actualTime2, err := time.Parse(time.RFC3339Nano, stats2.FirstExecutionDate)
	require.NoError(t, err)
	require.WithinDuration(t, exec4.Info.GetStartTime().AsTime(), actualTime2, time.Second)
	require.Equal(t, stats2.FirstExecutionDate, stats2.LastExecutionDate)
	require.Equal(t, 100.00, stats2.SuccessRate)
	require.NotNil(t, stats2.LastRun)
	require.Equal(t, "Pipeline-Sched-wf-4", stats2.LastRun.WorkflowID)
	require.Equal(t, "run-4", stats2.LastRun.RunID)

	require.Nil(t, stats3, "unpublished pipelines must not appear on the scoreboard")

	mockClient.AssertExpectations(t)
}

func TestGetNamespaceScoreboardActivityErrors(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	createPipelineRecord(t, app, orgID, "Any Pipeline")

	mockClient := &temporalmocks.Client{}
	mockClient.On("ListWorkflow", mock.Anything, mock.Anything).
		Return(nil, errors.New("temporal down"))
	stubScoreboardTemporalClient(t, mockClient)

	tests := []struct {
		name    string
		payload any
		code    string
	}{
		{
			name:    "missing namespace",
			payload: map[string]any{},
			code:    errorcodes.MissingOrInvalidPayload,
		},
		{
			name:    "listing failure",
			payload: workflows.ScoreboardNamespaceInput{Namespace: "usera-s-organization"},
			code:    errorcodes.DatabaseOperationFailed,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runScoreboardActivity[[]PipelineStatsResponse](
				t,
				NewGetNamespaceScoreboardActivity(app),
				tc.payload,
			)
			requireActivityErrorCode(t, err, tc.code)
		})
	}
}

func TestGetScoreboardExecutionDetailsActivity(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	createRunnerRecord(t, app, orgID, "runner-android")
	createRunnerRecord(t, app, orgID, "runner-ios")
	createRunnerRecord(t, app, orgID, "runner-default")

	pipeline1 := createPipelineRecord(t, app, orgID, "Android E2E Tests")
	pipeline2 := createPipelineRecord(t, app, orgID, "iOS E2E Tests")

	pipeline1.Set("published", true)
	require.NoError(t, app.Save(pipeline1))
	pipeline2.Set("published", true)
	require.NoError(t, app.Save(pipeline2))

	namespace := "usera-s-organization"
	now := time.Now()

	exec2 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-2",
		"run-2",
		fmt.Sprintf("%s/%s", namespace, pipeline1.GetString("canonified_name")),
		"Completed",
		[]string{"usera-s-organization/runner-android"},
		now.Add(-1*time.Hour).Add(-1*time.Minute).Add(-45*time.Second),
		now.Add(-1*time.Hour),
	)
	addEntitySearchAttributes(exec2.Info, map[string]any{
		workflowengine.VersionsSearchAttribute: "installed_from_external_source",
		workflowengine.ActionsSearchAttribute: []string{
			"org/wallet/maestro-1",
			"org/action/maestro-2",
		},
		workflowengine.CredentialsSearchAttribute: []string{
			"org/issuer/credential-1",
			"org/issuer/credential-2",
		},
		workflowengine.UseCaseSearchAttribute: []string{
			"org/verifier/uc-1",
			"org/verifier/uc-2",
		},
		workflowengine.ConformanceCheckSearchAttribute: []string{"conformance/check-1"},
		workflowengine.CustomCheckSearchAttribute:      []string{"custom/check-1"},
	})

	exec4 := buildPipelineExecutionInfoWithRunner(
		t,
		"Pipeline-Sched-wf-4",
		"run-4",
		fmt.Sprintf("%s/%s", namespace, pipeline2.GetString("canonified_name")),
		"Completed",
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		now.Add(-5*time.Minute).Add(-10*time.Second),
		now.Add(-1*time.Minute),
	)
	addEntitySearchAttributes(exec4.Info, map[string]any{
		workflowengine.VersionsSearchAttribute: []string{
			"org/wallet/v2-0-0",
			"org/wallet/v3-0-0",
		},
		workflowengine.CredentialsSearchAttribute: []string{"org/issuer/credential-3"},
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-2", "run-2").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: exec2.Info,
		}, nil).
		Once()
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "Pipeline-Sched-wf-4", "run-4").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: exec4.Info,
		}, nil).
		Once()
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "non-existent", "non-existent").
		Return(nil, fmt.Errorf("workflow not found")).
		Once()
	stubScoreboardTemporalClient(t, mockClient)

	t.Run("execution details for exec2", func(t *testing.T) {
		details, err := runScoreboardActivity[LastExecutionDetails](
			t,
			NewGetScoreboardExecutionDetailsActivity(),
			workflows.ScoreboardExecutionInput{
				Namespace:  namespace,
				WorkflowID: "wf-2",
				RunID:      "run-2",
			},
		)
		require.NoError(t, err)

		require.Equal(t, "wf-2", details.WorkflowID)
		require.Equal(t, "run-2", details.RunID)
		require.ElementsMatch(t, []string{"org/wallet", "org/action"}, details.WalletUsed)
		require.Empty(t, details.WalletVersionUsed)
		require.ElementsMatch(
			t,
			[]string{"org/wallet/maestro-1", "org/action/maestro-2"},
			details.MaestroScripts,
		)
		require.ElementsMatch(
			t,
			[]string{"org/issuer/credential-1", "org/issuer/credential-2"},
			details.Credentials,
		)
		require.ElementsMatch(t, []string{"org/issuer"}, details.Issuers)
		require.ElementsMatch(
			t,
			[]string{"org/verifier/uc-1", "org/verifier/uc-2"},
			details.UseCaseVerifications,
		)
		require.ElementsMatch(t, []string{"org/verifier"}, details.Verifiers)
		require.ElementsMatch(t, []string{"conformance/check-1"}, details.ConformanceTests)
		require.ElementsMatch(t, []string{"custom/check-1"}, details.CustomChecks)
	})

	t.Run("execution details for exec4", func(t *testing.T) {
		details, err := runScoreboardActivity[LastExecutionDetails](
			t,
			NewGetScoreboardExecutionDetailsActivity(),
			workflows.ScoreboardExecutionInput{
				Namespace:  namespace,
				WorkflowID: "Pipeline-Sched-wf-4",
				RunID:      "run-4",
			},
		)
		require.NoError(t, err)

		require.Equal(t, "Pipeline-Sched-wf-4", details.WorkflowID)
		require.ElementsMatch(t, []string{"org/wallet"}, details.WalletUsed)
		require.ElementsMatch(
			t,
			[]string{"org/wallet/v2-0-0", "org/wallet/v3-0-0"},
			details.WalletVersionUsed,
		)
		require.ElementsMatch(t, []string{"org/issuer/credential-3"}, details.Credentials)
		require.ElementsMatch(t, []string{"org/issuer"}, details.Issuers)
	})

	t.Run("missing run id", func(t *testing.T) {
		_, err := runScoreboardActivity[LastExecutionDetails](
			t,
			NewGetScoreboardExecutionDetailsActivity(),
			map[string]any{"namespace": namespace, "workflow_id": "wf-2"},
		)
		requireActivityErrorCode(t, err, errorcodes.MissingOrInvalidPayload)
	})

	t.Run("workflow not found", func(t *testing.T) {
		_, err := runScoreboardActivity[LastExecutionDetails](
			t,
			NewGetScoreboardExecutionDetailsActivity(),
			workflows.ScoreboardExecutionInput{
				Namespace:  namespace,
				WorkflowID: "non-existent",
				RunID:      "non-existent",
			},
		)
		requireActivityErrorCode(t, err, errorcodes.DatabaseOperationFailed)
	})

	mockClient.AssertExpectations(t)
}

func TestSaveScoreboardResults(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipeline := createPipelineRecord(t, app, orgID, "Test Pipeline")
	pipeline.Set("published", true)
	require.NoError(t, app.Save(pipeline))

	runner := createRunnerRecord(t, app, orgID, "test-runner")
	createDeviceRecord(t, app, orgID, runner.Id, "test-device")
	createPipelineResult(t, app, orgID, pipeline.Id, "wf-new", "run-new")
	publicWallet := createWalletRecord(t, app, orgID, "my-wallet")
	walletLogo := NewTestFile("wallet-logo.png", []byte("\x89PNG\r\n\x1a\n"))
	publicWallet.Set("logo", []*filesystem.File{walletLogo})
	require.NoError(t, app.Save(publicWallet))
	privateWallet := createWalletRecord(t, app, orgID, "private-wallet")
	privateWallet.Set("published", false)
	require.NoError(t, app.Save(privateWallet))
	createVerifierRecord(t, app, orgID, "my-verifier")
	createIssuerRecord(t, app, orgID, "my-issuer-1")
	createIssuerRecord(t, app, orgID, "my-issuer-2")
	createCustomCheckRecord(t, app, orgID, "my-check")

	cacheRecords := func(t *testing.T) []*core.Record {
		t.Helper()
		records, err := app.FindRecordsByFilter("pipeline_scoreboard_cache", "", "", -1, 0)
		require.NoError(t, err)
		return records
	}

	t.Run("success - saves results correctly", func(t *testing.T) {
		out, err := runScoreboardActivity[ScoreboardSaveOutput](
			t,
			NewSaveScoreboardResultsActivity(app),
			workflows.AggregateScoreboardWorkflowOutput{
				AggregatedPipelines: []workflows.AggregatedPipelineStats{
					{
						PipelineID:   pipeline.Id,
						PipelineName: "Test Pipeline",
						DeviceTypes:  []string{},
						DeviceIDs: []string{
							"usera-s-organization/test-runner/test-device",
						},
						TotalRuns:               10,
						TotalSuccesses:          8,
						SuccessRate:             80.0,
						ManualExecutions:        5,
						ScheduledExecutions:     5,
						CIExecutions:            2,
						MinExecutionTime:        "1m30s",
						MinExecutionTimeSeconds: 90,
						FirstExecutionDate:      "2024-01-01T00:00:00Z",
						LastExecutionDate:       "2024-01-02T00:00:00Z",
						LastExecution: &workflows.LatestExecutionDetails{
							WorkflowID: "wf-new",
							RunID:      "run-new",
							WalletUsed: []string{
								"usera-s-organization/my-wallet",
								"usera-s-organization/private-wallet",
							},
							Verifiers: []string{"usera-s-organization/my-verifier"},
							Issuers: []string{
								"usera-s-organization/my-issuer-1",
								"usera-s-organization/my-issuer-2",
							},
							WalletVersionUsed: []string{
								"installed_from_external_source",
								"usera-s-organization/my-wallet/1-0-0",
								"usera-s-organization/private-wallet/1-0-0",
							},
							MaestroScripts: []string{
								"usera-s-organization/my-wallet/my-action",
							},
							Credentials: []string{
								"usera-s-organization/my-issuer-1/cred-3",
							},
							UseCaseVerifications: []string{
								"usera-s-organization/my-verifier/usecase123",
							},
							CustomChecks:     []string{"usera-s-organization/my-check"},
							ConformanceTests: []string{"conformance-test-1"},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		require.Equal(t, 1, out.RecordsCount)

		records := cacheRecords(t)
		require.Len(t, records, 1)

		record := records[0]
		require.Equal(t, pipeline.Id, record.GetString("pipeline"))
		require.Equal(t, 10, record.GetInt("total_runs"))
		require.Equal(t, 8, record.GetInt("total_successes"))
		require.Equal(t, 80.0, record.GetFloat("success_rate"))
		require.Equal(t, 5, record.GetInt("manually_executed_runs"))
		require.Equal(t, 5, record.GetInt("scheduled_runs"))
		require.Equal(t, 2, record.GetInt("CI_runs"))
		require.Equal(t, "1m30s", record.GetString("minimum_running_time"))
		require.Equal(t, 90, record.GetInt("minimum_running_time_seconds"))

		deviceIDs := record.GetStringSlice("mobile_devices")
		require.Len(t, deviceIDs, 1)
		deviceRecord, err := app.FindRecordById("mobile_devices", deviceIDs[0])
		require.NoError(t, err)
		require.Equal(t, "test-device", deviceRecord.GetString("name"))

		var expandedData ScoreboardExpandedData
		require.NoError(t, json.Unmarshal([]byte(record.GetString("expanded_data")), &expandedData))
		require.NotNil(t, expandedData.Pipeline)
		require.Equal(t, pipeline.Id, expandedData.Pipeline.ID)
		require.Len(t, expandedData.Wallets, 1)
		require.Equal(
			t,
			"https://credimi.test/api/files/wallets/"+publicWallet.Id+"/wallet-logo.png",
			expandedData.Wallets[0].LogoURL,
		)
		require.Equal(t, publicWallet.Id, expandedData.Wallets[0].ID)
		require.NotEqual(t, privateWallet.Id, expandedData.Wallets[0].ID)
		require.Len(t, expandedData.MobileDevices, 1)
		require.Equal(t, deviceIDs[0], expandedData.MobileDevices[0].ID)
		require.Equal(
			t,
			"usera-s-organization/test-runner/test-device",
			expandedData.MobileDevices[0].DeviceID,
		)
		require.Equal(t, "test-runner", expandedData.MobileDevices[0].RunnerName)
		require.NotNil(t, expandedData.LatestExecution)
		require.Len(t, expandedData.WalletVersions, 1)
		require.Equal(t, publicWallet.Id, expandedData.WalletVersions[0].Wallet)

		executionRecord, err := app.FindRecordById(
			"pipeline_results",
			record.GetString("latest_execution"),
		)
		require.NoError(t, err)
		require.Equal(t, "wf-new", executionRecord.GetString("workflow_id"))
		require.Equal(t, "run-new", executionRecord.GetString("run_id"))

		relations := []struct {
			field      string
			collection string
			index      int
			column     string
			want       string
		}{
			{"wallets", "wallets", 0, "canonified_name", "my-wallet"},
			{"verifiers", "verifiers", 0, "canonified_name", "my-verifier"},
			{"issuers", "credential_issuers", 1, "canonified_name", "my-issuer-2"},
			{"wallet_versions", "wallet_versions", 0, "tag", "1.0.0"},
			{"wallet_actions", "wallet_actions", 0, "category", "onboarding"},
			{"credentials", "credentials", 0, "canonified_name", "cred-3"},
			{
				"use_case_verifications",
				"use_cases_verifications",
				0,
				"canonified_name",
				"usecase123",
			},
			{"custom_integrations", "custom_checks", 0, "canonified_name", "my-check"},
		}
		for _, rel := range relations {
			ids := record.GetStringSlice(rel.field)
			require.Greater(t, len(ids), rel.index, "%s should not be empty", rel.field)
			related, err := app.FindRecordById(rel.collection, ids[rel.index])
			require.NoError(t, err)
			require.Equal(t, rel.want, related.GetString(rel.column))
		}
		require.NotEmpty(t, record.GetStringSlice("conformance_checks"))
	})

	t.Run("fail - empty aggregated pipelines", func(t *testing.T) {
		_, err := runScoreboardActivity[ScoreboardSaveOutput](
			t,
			NewSaveScoreboardResultsActivity(app),
			workflows.AggregateScoreboardWorkflowOutput{},
		)
		requireActivityErrorCode(t, err, errorcodes.MissingOrInvalidPayload)
	})

	t.Run("partial - missing runners are skipped", func(t *testing.T) {
		out, err := runScoreboardActivity[ScoreboardSaveOutput](
			t,
			NewSaveScoreboardResultsActivity(app),
			workflows.AggregateScoreboardWorkflowOutput{
				AggregatedPipelines: []workflows.AggregatedPipelineStats{{
					PipelineID:   pipeline.Id,
					PipelineName: "Test Pipeline",
					DeviceIDs: []string{
						"usera-s-organization/test-runner/test-device",
						"usera-s-organization/test-runner/missing-device",
					},
					TotalRuns:          10,
					FirstExecutionDate: "2024-01-01T00:00:00Z",
					LastExecutionDate:  "2024-01-02T00:00:00Z",
				}},
			},
		)
		require.NoError(t, err)
		require.Equal(t, 1, out.RecordsCount)
		require.Contains(t, strings.Join(out.Errors, "\n"), "missing-device")

		records := cacheRecords(t)
		require.Len(t, records, 1)
		deviceIDs := records[0].GetStringSlice("mobile_devices")
		require.Len(t, deviceIDs, 1)
		deviceRecord, err := app.FindRecordById("mobile_devices", deviceIDs[0])
		require.NoError(t, err)
		require.Equal(t, "test-device", deviceRecord.GetString("name"))
	})

	t.Run("partial - missing last execution relations are skipped", func(t *testing.T) {
		count, saveErrors, err := saveScoreboardResults(
			app,
			&workflows.AggregateScoreboardWorkflowOutput{
				AggregatedPipelines: []workflows.AggregatedPipelineStats{{
					PipelineID:          pipeline.Id,
					PipelineName:        "Test Pipeline",
					TotalRuns:           10,
					FirstExecutionDate:  "2024-01-01T00:00:00Z",
					LastExecutionDate:   "2024-01-02T00:00:00Z",
					ManualExecutions:    5,
					ScheduledExecutions: 5,
					LastExecution: &workflows.LatestExecutionDetails{
						WorkflowID: "wf-new",
						RunID:      "run-new",
						WalletUsed: []string{
							"usera-s-organization/my-wallet",
							"usera-s-organization/missing-wallet",
						},
						Verifiers: []string{"usera-s-organization/my-verifier"},
						Issuers: []string{
							"usera-s-organization/my-issuer-1",
							"usera-s-organization/missing-issuer",
						},
						WalletVersionUsed: []string{
							"usera-s-organization/my-wallet/1-0-0",
							"usera-s-organization/missing-wallet/1-0-0",
						},
						MaestroScripts: []string{
							"usera-s-organization/my-wallet/my-action",
							"usera-s-organization/my-wallet/missing-action",
						},
						Credentials: []string{
							"usera-s-organization/my-issuer-1/cred-3",
							"usera-s-organization/my-issuer-1/missing-credential",
						},
						UseCaseVerifications: []string{
							"usera-s-organization/my-verifier/usecase123",
							"usera-s-organization/my-verifier/missing-use-case",
						},
						CustomChecks: []string{
							"usera-s-organization/my-check",
							"usera-s-organization/missing-check",
						},
					},
				}},
			},
		)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		joined := errors.Join(saveErrors...).Error()
		require.Contains(t, joined, "missing-wallet")
		require.Contains(t, joined, "missing-issuer")
		require.Contains(t, joined, "missing-action")

		records := cacheRecords(t)
		require.Len(t, records, 1)
		for _, field := range []string{
			"wallets", "issuers", "verifiers", "wallet_actions", "wallet_versions",
			"credentials", "use_case_verifications", "custom_integrations",
		} {
			require.Len(t, records[0].GetStringSlice(field), 1, field)
		}
	})

	t.Run("fail - pipeline not found", func(t *testing.T) {
		_, err := runScoreboardActivity[ScoreboardSaveOutput](
			t,
			NewSaveScoreboardResultsActivity(app),
			workflows.AggregateScoreboardWorkflowOutput{
				AggregatedPipelines: []workflows.AggregatedPipelineStats{{
					PipelineID:   "non-existent-pipeline-id",
					PipelineName: "Non Existent Pipeline",
					TotalRuns:    10,
				}},
			},
		)
		requireActivityErrorCode(t, err, errorcodes.DatabaseOperationFailed)
	})
}

func TestSaveScoreboardResultsSkipsUnpublishedPipelines(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	publishedPipeline := createPipelineRecord(t, app, orgID, "Published Pipeline")
	publishedPipeline.Set("published", true)
	require.NoError(t, app.Save(publishedPipeline))

	privatePipeline := createPipelineRecord(t, app, orgID, "Private Pipeline")
	privatePipeline.Set("published", false)
	require.NoError(t, app.Save(privatePipeline))

	count, _, err := saveScoreboardResults(app, &workflows.AggregateScoreboardWorkflowOutput{
		AggregatedPipelines: []workflows.AggregatedPipelineStats{
			{
				PipelineID:         publishedPipeline.Id,
				PipelineName:       "Published Pipeline",
				TotalRuns:          10,
				FirstExecutionDate: "2024-01-01T00:00:00Z",
				LastExecutionDate:  "2024-01-02T00:00:00Z",
			},
			{
				PipelineID:         privatePipeline.Id,
				PipelineName:       "Private Pipeline",
				TotalRuns:          5,
				FirstExecutionDate: "2024-01-01T00:00:00Z",
				LastExecutionDate:  "2024-01-02T00:00:00Z",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	records, err := app.FindRecordsByFilter("pipeline_scoreboard_cache", "", "", -1, 0)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, publishedPipeline.Id, records[0].GetString("pipeline"))
}
