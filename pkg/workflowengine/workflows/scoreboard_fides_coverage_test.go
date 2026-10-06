// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

func covWEScoreboardDetailsFor(workflowID string) any {
	return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
		payload, ok := input.Payload.(map[string]any)
		return ok && payload["workflow_id"] == workflowID
	})
}

func TestCovWEAggregateScoreboardWorkflowToleratesPartialFailures(t *testing.T) {
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	registerScoreboardActivities(env)
	mockNamespaces(env, []string{"ns-a", "ns-b", "ns-b", "ns-c"})
	env.OnActivity(GetNamespaceScoreboardActivityName, mock.Anything, scoreboardNamespacePayload("ns-a")).
		Return(workflowengine.ActivityResult{Output: map[string]any{"not": "a list"}}, nil)
	env.OnActivity(GetNamespaceScoreboardActivityName, mock.Anything, scoreboardNamespacePayload("ns-c")).
		Return(workflowengine.ActivityResult{}, covWENonRetryable("temporal unavailable"))
	mockNamespaceScoreboard(env, "ns-b", []map[string]any{
		{"pipeline_name": "no id"},
		{
			"pipeline_id":        "p1",
			"pipeline_name":      "Pipeline 1",
			"total_runs":         4.0,
			"total_successes":    1.0,
			"min_execution_time": "1m30s",
			"last_run":           "not-a-map",
		},
		{
			"pipeline_id":   "p2",
			"pipeline_name": "Pipeline 2",
			"last_run": map[string]any{
				"workflow_id": "wf-details-not-map",
				"run_id":      "run-2",
				"start_time":  "2026-04-01T10:00:00Z",
			},
		},
		{
			"pipeline_id":   "p3",
			"pipeline_name": "Pipeline 3",
			"last_run": map[string]any{
				"workflow_id": "wf-details-error",
				"run_id":      "run-3",
				"start_time":  "2026-04-01T10:00:00Z",
			},
		},
		{
			"pipeline_id":   "p4",
			"pipeline_name": "Pipeline 4",
			"last_run":      map[string]any{"workflow_id": "wf-incomplete"},
		},
	})
	env.OnActivity(GetScoreboardExecutionDetailsActivityName, mock.Anything, covWEScoreboardDetailsFor("wf-details-not-map")).
		Return(workflowengine.ActivityResult{Output: []any{"x"}}, nil).
		Once()
	env.OnActivity(GetScoreboardExecutionDetailsActivityName, mock.Anything, covWEScoreboardDetailsFor("wf-details-error")).
		Return(workflowengine.ActivityResult{}, covWENonRetryable("details missing")).
		Once()
	env.OnActivity(SaveScoreboardResultsActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, covWENonRetryable("cache table locked")).
		Once()

	env.ExecuteWorkflow(NewAggregateScoreboardWorkflow().Workflow, workflowengine.WorkflowInput{
		Config: map[string]any{"app_url": "https://credimi.test"},
	})

	require.True(t, env.IsWorkflowCompleted())
	env.AssertExpectations(t)
	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	raw, err := json.Marshal(result.Output)
	require.NoError(t, err)
	var output AggregateScoreboardWorkflowOutput
	require.NoError(t, json.Unmarshal(raw, &output))

	assert.Equal(t, 1, output.NamespacesProcessed)
	assert.Equal(t, 2, output.NamespacesFailed)
	assert.Equal(t, []string{"ns-a", "ns-c"}, output.FailedNamespaces)
	require.Len(t, output.AggregatedPipelines, 4)
	byID := map[string]AggregatedPipelineStats{}
	for _, p := range output.AggregatedPipelines {
		byID[p.PipelineID] = p
	}
	assert.InDelta(t, 25.0, byID["p1"].SuccessRate, 0.001)
	assert.Equal(t, "1m30s", byID["p1"].MinExecutionTime)
	assert.Equal(t, 90, byID["p1"].MinExecutionTimeSeconds)
	assert.Nil(t, byID["p1"].LastExecution)
	assert.Nil(t, byID["p2"].LastExecution)
	assert.Nil(t, byID["p3"].LastExecution)
	assert.Nil(t, byID["p4"].LastExecution)
}

func TestCovWEAggregateScoreboardWorkflowRejectsInvalidNamespaces(t *testing.T) {
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	registerScoreboardActivities(env)
	env.OnActivity(ListScoreboardNamespacesActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []any{"ok", 3}}, nil)

	env.ExecuteWorkflow(NewAggregateScoreboardWorkflow().Workflow, workflowengine.WorkflowInput{
		Config: map[string]any{"app_url": "https://credimi.test"},
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespaces output missing or invalid")
}

func TestCovWEScoreboardValueHelpers(t *testing.T) {
	t.Run("getString", func(t *testing.T) {
		assert.Empty(t, getString(nil, "k"))
		assert.Empty(t, getString(map[string]any{"k": 1}, "k"))
		assert.Equal(t, "v", getString(map[string]any{"k": "v"}, "k"))
	})

	t.Run("getStringSlice", func(t *testing.T) {
		tests := []struct {
			name string
			in   map[string]any
			want []string
		}{
			{name: "nil map", in: nil, want: nil},
			{
				name: "mixed any slice keeps strings",
				in:   map[string]any{"k": []any{"a", 1, "b"}},
				want: []string{"a", "b"},
			},
			{
				name: "string slice is copied",
				in:   map[string]any{"k": []string{"x"}},
				want: []string{"x"},
			},
			{name: "wrong type", in: map[string]any{"k": "x"}, want: nil},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.want, getStringSlice(tc.in, "k"))
			})
		}
	})

	t.Run("getRequiredStringSlice", func(t *testing.T) {
		tests := []struct {
			name   string
			in     map[string]any
			want   []string
			wantOK bool
		}{
			{name: "nil map", in: nil},
			{name: "missing key", in: map[string]any{}},
			{
				name:   "string slice",
				in:     map[string]any{"k": []string{"a"}},
				want:   []string{"a"},
				wantOK: true,
			},
			{
				name:   "any slice of strings",
				in:     map[string]any{"k": []any{"a", "b"}},
				want:   []string{"a", "b"},
				wantOK: true,
			},
			{name: "any slice with non string", in: map[string]any{"k": []any{"a", 2}}},
			{name: "wrong type", in: map[string]any{"k": 4}},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got, ok := getRequiredStringSlice(tc.in, "k")
				assert.Equal(t, tc.wantOK, ok)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("shouldReplaceMinExecutionTime", func(t *testing.T) {
		tests := []struct {
			name      string
			current   string
			candidate string
			want      bool
		}{
			{name: "empty current", current: "", candidate: "5s", want: true},
			{name: "shorter duration replaces", current: "10s", candidate: "5s", want: true},
			{name: "longer duration keeps", current: "5s", candidate: "10s", want: false},
			{
				name:      "parsable candidate beats unparsable current",
				current:   "fast",
				candidate: "5s",
				want:      true,
			},
			{
				name:      "unparsable candidate never beats parsable",
				current:   "5s",
				candidate: "fast",
				want:      false,
			},
			{name: "both unparsable compare lexically", current: "b", candidate: "a", want: true},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.want, shouldReplaceMinExecutionTime(tc.current, tc.candidate))
			})
		}
	})

	t.Run("jsonNumberAsInt", func(t *testing.T) {
		tests := []struct {
			in     any
			want   int
			wantOK bool
		}{
			{in: 2.6, want: 3, wantOK: true},
			{in: float32(1.4), want: 1, wantOK: true},
			{in: 7, want: 7, wantOK: true},
			{in: int64(9), want: 9, wantOK: true},
			{in: "9"},
		}
		for _, tc := range tests {
			got, ok := jsonNumberAsInt(tc.in)
			assert.Equal(t, tc.wantOK, ok, "%v", tc.in)
			assert.Equal(t, tc.want, got, "%v", tc.in)
		}
	})
}

func TestCovWEFidesCredentialIssuersWorkflowStart(t *testing.T) {
	orig := fidesCredentialIssuersStartWorkflowWithOptions
	t.Cleanup(func() { fidesCredentialIssuersStartWorkflowWithOptions = orig })

	var gotNamespace, gotName string
	var gotOptions client.StartWorkflowOptions
	fidesCredentialIssuersStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		_ workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		gotNamespace, gotOptions, gotName = namespace, options, name
		return workflowengine.WorkflowResult{WorkflowID: options.ID}, nil
	}

	w := NewFidesCredentialIssuersWorkflow()
	result, err := w.Start("org-ns", workflowengine.WorkflowInput{})
	require.NoError(t, err)
	assert.Equal(t, "org-ns", gotNamespace)
	assert.Equal(t, w.Name(), gotName)
	assert.Equal(t, FidesCredentialIssuersTaskQueue, gotOptions.TaskQueue)
	assert.Equal(t, 24*time.Hour, gotOptions.WorkflowExecutionTimeout)
	assert.True(t, strings.HasPrefix(result.WorkflowID, "Fides-Credential-Issuers-"))
}

func TestCovWEFidesCredentialIssuersWorkflowCatalogFailures(t *testing.T) {
	httpName := activities.NewHTTPActivity().Name()
	parseName := activities.NewParseFidesCredentialIssuersActivity().Name()
	unexpected := errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Code
	okBody := workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{}}}

	tests := []struct {
		name     string
		config   map[string]any
		setup    func(env *testsuite.TestWorkflowEnvironment)
		wantCode string
		wantText string
	}{
		{
			name:     "missing issuer schema",
			config:   map[string]any{"app_url": "https://credimi.test", "orgID": "org"},
			wantCode: errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
			wantText: "issuer_schema",
		},
		{
			name: "catalog request fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("fides down"))
			},
			wantText: "fides down",
		},
		{
			name: "catalog response without body",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200}}, nil)
			},
			wantCode: unexpected,
			wantText: ": body",
		},
		{
			name: "catalog parsing fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, mock.Anything).Return(okBody, nil)
				env.OnActivity(parseName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("bad catalog"))
			},
			wantText: "bad catalog",
		},
		{
			name: "catalog parse output is malformed",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, mock.Anything).Return(okBody, nil)
				env.OnActivity(parseName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: "garbage"}, nil)
			},
			wantCode: unexpected,
			wantText: ": output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			registerFidesWorkflowActivities(env)
			if tc.setup != nil {
				tc.setup(env)
			}
			config := tc.config
			if config == nil {
				config = map[string]any{
					"app_url":       "https://credimi.test",
					"issuer_schema": "{}",
					"orgID":         "org",
				}
			}
			env.ExecuteWorkflow(
				NewFidesCredentialIssuersWorkflow().Workflow,
				workflowengine.WorkflowInput{
					Config: config,
				},
			)

			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantCode)
			assert.Contains(t, err.Error(), tc.wantText)
		})
	}
}

func TestCovWEFidesCredentialIssuersWorkflowPaginatesAndCollectsIssuerErrors(t *testing.T) {
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	registerFidesWorkflowActivities(env)
	httpName := activities.NewHTTPActivity().Name()
	parseName := activities.NewParseFidesCredentialIssuersActivity().Name()
	checkName := activities.NewCheckCredentialsIssuerActivity().Name()
	jsonName := activities.NewJSONActivity(nil).Name()
	validateName := activities.NewSchemaValidationActivity().Name()

	env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(activities.FidesCredentialIssuersURL)).
		Return(workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{"p": 0}}}, nil).
		Once()
	env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(activities.FidesCredentialIssuersURL+"?page=1")).
		Return(workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{"p": 1}}}, nil).
		Once()
	env.OnActivity(parseName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: activities.ParseFidesCredentialIssuersActivityResponse{
			Issuers:    []string{"https://unreachable", "https://no-creds"},
			PageNumber: 0,
			TotalPages: 2,
		}}, nil).
		Once()
	env.OnActivity(parseName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: activities.ParseFidesCredentialIssuersActivityResponse{
			Issuers: []string{
				"https://store-fails",
				"https://no-id",
				"https://cred-store-fails",
			},
			PageNumber: 1,
			TotalPages: 2,
		}}, nil).
		Once()

	checkFor := func(url string) any {
		return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			p, err := workflowengine.DecodePayload[activities.CheckCredentialsIssuerActivityPayload](
				input.Payload,
			)
			return err == nil && p.BaseURL == url
		})
	}
	env.OnActivity(checkName, mock.Anything, checkFor("https://unreachable")).
		Return(workflowengine.ActivityResult{}, covWENonRetryable("well-known missing"))
	for _, url := range []string{"https://no-creds", "https://store-fails", "https://no-id", "https://cred-store-fails"} {
		env.OnActivity(checkName, mock.Anything, checkFor(url)).
			Return(workflowengine.ActivityResult{Output: map[string]any{"source": "custom", "rawJSON": url}}, nil)
	}
	jsonFor := func(url string) any {
		return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			p, err := workflowengine.DecodePayload[activities.JSONActivityPayload](input.Payload)
			return err == nil && p.RawJSON == url
		})
	}
	withCreds := map[string]any{
		"credential_configurations_supported": map[string]any{"c": map[string]any{}},
	}
	env.OnActivity(jsonName, mock.Anything, jsonFor("https://no-creds")).
		Return(workflowengine.ActivityResult{Output: map[string]any{"credential_configurations_supported": map[string]any{}}}, nil)
	for _, url := range []string{"https://store-fails", "https://no-id", "https://cred-store-fails"} {
		env.OnActivity(jsonName, mock.Anything, jsonFor(url)).
			Return(workflowengine.ActivityResult{Output: withCreds}, nil)
	}
	env.OnActivity(validateName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil)

	storeFor := func(url string) any {
		return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			p, err := workflowengine.DecodePayload[activities.StoreCredentialIssuerInput](
				input.Payload,
			)
			return err == nil && p.URL == url && p.OrgID == "org"
		})
	}
	env.OnActivity(activities.StoreCredentialIssuerActivityName, mock.Anything, storeFor("https://no-creds")).
		Return(workflowengine.ActivityResult{Output: map[string]any{"id": "issuer-no-creds"}}, nil)
	env.OnActivity(activities.StoreCredentialIssuerActivityName, mock.Anything, storeFor("https://store-fails")).
		Return(workflowengine.ActivityResult{}, covWENonRetryable("issuer save failed"))
	env.OnActivity(activities.StoreCredentialIssuerActivityName, mock.Anything, storeFor("https://no-id")).
		Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil)
	env.OnActivity(activities.StoreCredentialIssuerActivityName, mock.Anything, storeFor("https://cred-store-fails")).
		Return(workflowengine.ActivityResult{Output: map[string]any{"id": "issuer-cred"}}, nil)
	env.OnActivity(activities.StoreIssuerCredentialActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, covWENonRetryable("credential save failed"))

	env.ExecuteWorkflow(NewFidesCredentialIssuersWorkflow().Workflow, workflowengine.WorkflowInput{
		Config: map[string]any{
			"app_url":       "https://credimi.test",
			"issuer_schema": "{}",
			"orgID":         "org",
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, map[string]any{"issuers": []any{"https://no-creds"}}, result.Output)

	errs, ok := result.Errors.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, errs["https://unreachable"], "well-known missing")
	assert.Contains(t, errs["https://store-fails"], "issuer save failed")
	assert.Contains(t, errs["https://no-id"], activities.StoreCredentialIssuerActivityName+": id")
	assert.Contains(t, errs["https://cred-store-fails"], "credential save failed")

	logs, ok := result.Log.(map[string]any)
	require.True(t, ok)
	raw, err := json.Marshal(logs["https://no-creds"])
	require.NoError(t, err)
	assert.Contains(t, string(raw), "NoCredentialConfigurations")
}
