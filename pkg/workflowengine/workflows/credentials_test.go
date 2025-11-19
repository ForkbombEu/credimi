// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func Test_CredentialsIssuersWorkflow(t *testing.T) {
	rawJSON := `{
		"credential_issuer": "testissuer",
		"display": [{"name": "Test Issuer", "logo": {"uri": "testlogo.png"}}],
		"credential_configurations_supported": {"cred1": {}}
	}`

	testCases := []struct {
		name           string
		input          workflowengine.WorkflowInput
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
		errorCode      errorcodes.Code
	}{
		{
			name: "Success: stores credentials correctly",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
				Payload: CredentialsIssuersWorkflowPayload{
					IssuerID: "issuer123",
					BaseURL:  "baseurl",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				checkAct := activities.NewCheckCredentialsIssuerActivity()
				jsonAct := activities.NewJSONActivity(nil)
				validateAct := activities.NewSchemaValidationActivity()
				httpAct := activities.NewHTTPActivity()

				env.RegisterActivityWithOptions(
					checkAct.Execute,
					activity.RegisterOptions{Name: checkAct.Name()},
				)
				env.RegisterActivityWithOptions(
					jsonAct.Execute,
					activity.RegisterOptions{Name: jsonAct.Name()},
				)
				env.RegisterActivityWithOptions(
					validateAct.Execute,
					activity.RegisterOptions{Name: validateAct.Name()},
				)
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)

				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"rawJSON": rawJSON, "source": "testsource"}}, nil)
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer": "testissuer",
						"display": []any{
							map[string]any{
								"name": "Test Issuer",
								"logo": map[string]any{"uri": "testlogo.png"},
							},
						},
						"credential_configurations_supported": map[string]any{
							"cred1": map[string]any{},
						},
					}}, nil)
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{"key": "cred1"}}}, nil)
			},
		},
		{
			name: "Failure: missing base_url payload",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
				Payload: CredentialsIssuersWorkflowPayload{},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {

			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure: invalid CheckCredentialsIssuer output",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
				Payload: CredentialsIssuersWorkflowPayload{
					IssuerID: "issuer123",
					BaseURL:  "baseurl",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				checkAct := activities.NewCheckCredentialsIssuerActivity()
				env.RegisterActivityWithOptions(
					checkAct.Execute,
					activity.RegisterOptions{Name: checkAct.Name()},
				)
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"unexpected": "field"}}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			var wf CredentialsIssuersWorkflow
			env.ExecuteWorkflow(wf.Workflow, tc.input)

			if tc.expectedErr {
				require.True(t, env.IsWorkflowCompleted())
				err := env.GetWorkflowError()
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorCode.Code)
			} else {
				require.True(t, env.IsWorkflowCompleted())
				require.NoError(t, env.GetWorkflowError())

				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result))
				require.Contains(t, result.Message, "Successfully retrieved and stored")
				require.NotEmpty(t, result.Log.(map[string]any)["StoredCredentials"])
			}
		})
	}
}

func Test_GetCredentialOfferWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		input          workflowengine.WorkflowInput
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
		expectedOutput string
		errorCode      errorcodes.Code
	}{
		{
			name: "Success: retrieves static credential offer",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://example.com",
				},
				Payload: GetCredentialOfferWorkflowPayload{
					CredentialID: "test_cred",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{
								"dynamic":          false,
								"credential_offer": "static-offer",
							},
						},
					}, nil)
			},
			expectedOutput: "static-offer",
		},
		{
			name: "Success: retrieves dynamic credential offer via StepCI",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{"app_url": "https://example.com"},
				Payload: GetCredentialOfferWorkflowPayload{CredentialID: "dynamic_cred"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				stepCIAct := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)
				env.RegisterActivityWithOptions(
					stepCIAct.Execute,
					activity.RegisterOptions{Name: stepCIAct.Name()},
				)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{"dynamic": true, "code": "yaml-content"},
						},
					}, nil)
				env.OnActivity(stepCIAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{"deeplink": "dynamic-deeplink"},
						},
					}, nil)
			},
			expectedOutput: "dynamic-deeplink",
		},
		{
			name: "Failure: StepCI activity fails",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{"app_url": "https://example.com"},
				Payload: GetCredentialOfferWorkflowPayload{CredentialID: "test_cred"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				stepCIAct := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)
				env.RegisterActivityWithOptions(
					stepCIAct.Execute,
					activity.RegisterOptions{Name: stepCIAct.Name()},
				)

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{"dynamic": true, "code": "valid-yaml"},
						},
					}, nil)

				env.OnActivity(stepCIAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, fmt.Errorf("CRE301: stepCI execution failed"))
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.CommandExecutionFailed],
		},
		{
			name: "Failure: missing credential_id",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{"app_url": "https://example.com"},
				Payload: GetCredentialOfferWorkflowPayload{},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure: missing app_url",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{},
				Payload: GetCredentialOfferWorkflowPayload{CredentialID: "test_cred"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
		},
		{
			name: "Failure: invalid HTTP output (body not a map)",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{"app_url": "https://example.com"},
				Payload: GetCredentialOfferWorkflowPayload{CredentialID: "test_cred"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"body": "not-a-map"}}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)
			tc.input.ActivityOptions = &DefaultActivityOptions
			var wf GetCredentialOfferWorkflow
			env.ExecuteWorkflow(wf.Workflow, tc.input)

			require.True(t, env.IsWorkflowCompleted())

			if tc.expectedErr {
				err := env.GetWorkflowError()
				require.Error(t, err)
				if tc.errorCode.Code != "" {
					require.Contains(t, err.Error(), tc.errorCode.Code)
				}
			} else {
				require.NoError(t, env.GetWorkflowError())

				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result))
				require.Equal(t, "Successfully retrieved credential offer", result.Message)
				require.Equal(t, tc.expectedOutput, result.Output)
			}
		})
	}
}
