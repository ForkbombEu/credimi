// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// covWENonRetryable returns an activity failure the test environment does not retry.
func covWENonRetryable(msg string) error {
	return temporal.NewNonRetryableApplicationError(msg, "CovWETestFailure", nil)
}

func TestCovWEGetCredentialOfferWorkflowRejectsMalformedOutputs(t *testing.T) {
	unexpected := errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Code
	tests := []struct {
		name        string
		offer       map[string]any
		stepCI      map[string]any
		wantMessage string
	}{
		{
			name:        "dynamic flag is not a bool",
			offer:       map[string]any{"dynamic": "yes"},
			wantMessage: "dynamic is not a bool",
		},
		{
			name:        "static offer without credential_offer string",
			offer:       map[string]any{"dynamic": false, "credential_offer": 42},
			wantMessage: "credential_offer is not a string",
		},
		{
			name:        "dynamic offer without yaml code",
			offer:       map[string]any{"dynamic": true},
			wantMessage: "yaml code is not a string",
		},
		{
			name:        "StepCI output without captures",
			offer:       map[string]any{"dynamic": true, "code": "yaml"},
			stepCI:      map[string]any{"other": "value"},
			wantMessage: "captures is not a map",
		},
		{
			name:        "StepCI captures without deeplink",
			offer:       map[string]any{"dynamic": true, "code": "yaml"},
			stepCI:      map[string]any{"captures": map[string]any{"deeplink": 7}},
			wantMessage: "deeplink missing or invalid from captures",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			registerGetCredentialOfferActivity(env)
			stepCIAct := activities.NewStepCIWorkflowActivity()
			env.RegisterActivityWithOptions(
				stepCIAct.Execute,
				activity.RegisterOptions{Name: stepCIAct.Name()},
			)
			env.OnActivity(activities.GetCredentialOfferActivityName, mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: tc.offer}, nil)
			if tc.stepCI != nil {
				env.OnActivity(stepCIAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: tc.stepCI}, nil)
			}

			env.ExecuteWorkflow(
				NewGetCredentialOfferWorkflow().Workflow,
				workflowengine.WorkflowInput{
					Payload:         GetCredentialOfferWorkflowPayload{CredentialID: "cred"},
					ActivityOptions: &DefaultActivityOptions,
					Config:          map[string]any{"app_url": "https://credimi.test"},
				},
			)

			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			assert.Contains(t, err.Error(), unexpected)
			assert.Contains(t, err.Error(), tc.wantMessage)
		})
	}
}

// covWECredentialIssuerMocks configures the activities used by the credential issuer workflow.
type covWECredentialIssuerMocks struct {
	check      *workflowengine.ActivityResult
	checkErr   error
	parsed     *workflowengine.ActivityResult
	parseErr   error
	validate   error
	store      *workflowengine.ActivityResult
	storeErr   error
	skipParse  bool
	skipStore  bool
	storeCalls *int
}

func covWERunCredentialIssuersWorkflow(
	t *testing.T,
	m covWECredentialIssuerMocks,
) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	checkAct := activities.NewCheckCredentialsIssuerActivity()
	jsonAct := activities.NewJSONActivity(nil)
	validateAct := activities.NewSchemaValidationActivity()
	env.RegisterActivityWithOptions(
		checkAct.Execute,
		activity.RegisterOptions{Name: checkAct.Name()},
	)
	env.RegisterActivityWithOptions(jsonAct.Execute, activity.RegisterOptions{Name: jsonAct.Name()})
	env.RegisterActivityWithOptions(
		validateAct.Execute,
		activity.RegisterOptions{Name: validateAct.Name()},
	)
	env.RegisterActivityWithOptions(
		activities.NewStoreIssuerCredentialActivity(nil).Execute,
		activity.RegisterOptions{Name: activities.StoreIssuerCredentialActivityName},
	)

	check := workflowengine.ActivityResult{}
	if m.check != nil {
		check = *m.check
	}
	env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).Return(check, m.checkErr)
	if !m.skipParse {
		parsed := workflowengine.ActivityResult{}
		if m.parsed != nil {
			parsed = *m.parsed
		}
		env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).Return(parsed, m.parseErr)
		env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
			Return(workflowengine.ActivityResult{}, m.validate)
	}
	if !m.skipStore {
		store := workflowengine.ActivityResult{}
		if m.store != nil {
			store = *m.store
		}
		env.OnActivity(activities.StoreIssuerCredentialActivityName, mock.Anything, mock.Anything).
			Run(func(mock.Arguments) {
				if m.storeCalls != nil {
					*m.storeCalls++
				}
			}).
			Return(store, m.storeErr)
	}

	env.ExecuteWorkflow(NewCredentialsIssuersWorkflow().Workflow, workflowengine.WorkflowInput{
		Config: map[string]any{
			"app_url":       "https://credimi.test",
			"issuer_schema": "{}",
			"orgID":         "org123",
		},
		Payload: CredentialsIssuersWorkflowPayload{
			IssuerID: "issuer123",
			BaseURL:  "https://issuer.example.com",
		},
	})
	require.True(t, env.IsWorkflowCompleted())
	return env
}

func TestCovWECredentialsIssuersWorkflowRejectsMalformedActivityOutputs(t *testing.T) {
	unexpected := errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Code
	validIssuer := map[string]any{
		"credential_configurations_supported": map[string]any{"cred1": map[string]any{}},
	}
	tests := []struct {
		name        string
		mocks       covWECredentialIssuerMocks
		wantCode    string
		wantMessage string
	}{
		{
			name: "check output without source",
			mocks: covWECredentialIssuerMocks{
				check:     &workflowengine.ActivityResult{Output: map[string]any{"rawJSON": "{}"}},
				skipParse: true,
				skipStore: true,
			},
			wantCode:    unexpected,
			wantMessage: ": source",
		},
		{
			name: "check output without rawJSON",
			mocks: covWECredentialIssuerMocks{
				check:     &workflowengine.ActivityResult{Output: map[string]any{"source": "s"}},
				skipParse: true,
				skipStore: true,
			},
			wantCode:    unexpected,
			wantMessage: ": rawJSON",
		},
		{
			name: "JSON parsing fails",
			mocks: covWECredentialIssuerMocks{
				check: &workflowengine.ActivityResult{
					Output: map[string]any{"source": "s", "rawJSON": "{"},
				},
				parseErr:  covWENonRetryable("bad json"),
				skipStore: true,
			},
			wantMessage: "bad json",
		},
		{
			name: "parsed JSON is not an object",
			mocks: covWECredentialIssuerMocks{
				check: &workflowengine.ActivityResult{
					Output: map[string]any{"source": "s", "rawJSON": "[]"},
				},
				parsed:    &workflowengine.ActivityResult{Output: []any{"x"}},
				skipStore: true,
			},
			wantCode:    unexpected,
			wantMessage: ": output",
		},
		{
			name: "stored credential without key",
			mocks: covWECredentialIssuerMocks{
				check: &workflowengine.ActivityResult{
					Output: map[string]any{"source": "s", "rawJSON": "{}"},
				},
				parsed: &workflowengine.ActivityResult{Output: validIssuer},
				store:  &workflowengine.ActivityResult{Output: map[string]any{"id": "x"}},
			},
			wantCode:    unexpected,
			wantMessage: activities.StoreIssuerCredentialActivityName + ": key",
		},
		{
			name: "credential store fails",
			mocks: covWECredentialIssuerMocks{
				check: &workflowengine.ActivityResult{
					Output: map[string]any{"source": "s", "rawJSON": "{}"},
				},
				parsed:   &workflowengine.ActivityResult{Output: validIssuer},
				storeErr: covWENonRetryable("store exploded"),
			},
			wantMessage: "store exploded",
		},
		{
			name: "schema validation error without details",
			mocks: covWECredentialIssuerMocks{
				check: &workflowengine.ActivityResult{
					Output: map[string]any{"source": "s", "rawJSON": "{}"},
				},
				parsed:    &workflowengine.ActivityResult{Output: validIssuer},
				validate:  covWENonRetryable("schema down"),
				skipStore: true,
			},
			wantCode:    errorcodes.Codes[errorcodes.UnexpectedActivityErrorDetails].Code,
			wantMessage: "schema validation details are empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := covWERunCredentialIssuersWorkflow(t, tc.mocks)
			err := env.GetWorkflowError()
			require.Error(t, err)
			if tc.wantCode != "" {
				assert.Contains(t, err.Error(), tc.wantCode)
			}
			assert.Contains(t, err.Error(), tc.wantMessage)
		})
	}
}

func TestCovWECredentialsIssuersWorkflowExposesIssuerDataAndLogoURL(t *testing.T) {
	storeCalls := 0
	env := covWERunCredentialIssuersWorkflow(t, covWECredentialIssuerMocks{
		check: &workflowengine.ActivityResult{
			Output: map[string]any{"source": "custom", "rawJSON": "{}"},
		},
		parsed: &workflowengine.ActivityResult{Output: map[string]any{
			"display": []any{map[string]any{
				"name": "Issuer With URL Logo",
				"logo": map[string]any{"url": "https://logo.example/l.png"},
			}},
			"credential_configurations_supported": map[string]any{
				"a": map[string]any{},
				"b": map[string]any{},
			},
		}},
		store:      &workflowengine.ActivityResult{Output: map[string]any{"key": "k"}},
		storeCalls: &storeCalls,
	})
	require.NoError(t, env.GetWorkflowError())

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Issuer With URL Logo", output["issuerName"])
	assert.Equal(t, "https://logo.example/l.png", output["logo"])
	assert.Equal(t, 2, storeCalls)

	encoded, err := env.QueryWorkflow(CredentialsIssuerDataQuery)
	require.NoError(t, err)
	var data map[string]any
	require.NoError(t, encoded.Get(&data))
	assert.Equal(t, "Issuer With URL Logo", data["issuerName"])
	assert.Equal(t, "https://logo.example/l.png", data["logo"])
	assert.InDelta(t, 2, data["credentialsNumber"], 0)
}
