// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
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
				httpAct := activities.NewInternalHTTPActivity()

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
			name: "Failure: missing app_url config",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
				Payload: CredentialsIssuersWorkflowPayload{
					IssuerID: "issuer123",
					BaseURL:  "baseurl",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
		},
		{
			name: "Failure: missing issuer_schema config",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://example.com",
					"orgID":   "org123",
				},
				Payload: CredentialsIssuersWorkflowPayload{
					IssuerID: "issuer123",
					BaseURL:  "baseurl",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
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
		{
			name: "Failure: parse JSON output not map",
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
				env.RegisterActivityWithOptions(
					checkAct.Execute,
					activity.RegisterOptions{Name: checkAct.Name()},
				)
				env.RegisterActivityWithOptions(
					jsonAct.Execute,
					activity.RegisterOptions{Name: jsonAct.Name()},
				)
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": rawJSON,
						"source":  "testsource",
					}}, nil)
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: "not-map"}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
		{
			name: "Failure: missing credential_configurations_supported",
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
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": rawJSON,
						"source":  "testsource",
					}}, nil)
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer": "testissuer",
					}}, nil)
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
		{
			name: "Failure: missing orgID config",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
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
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": rawJSON,
						"source":  "testsource",
					}}, nil)
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer": "testissuer",
						"credential_configurations_supported": map[string]any{
							"cred1": map[string]any{},
						},
					}}, nil)
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
		},
		{
			name: "Failure: no credential configurations",
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
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": rawJSON,
						"source":  "testsource",
					}}, nil)
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer":                   "testissuer",
						"credential_configurations_supported": map[string]any{},
					}}, nil)
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
		{
			name: "Failure: store response missing key",
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
				httpAct := activities.NewInternalHTTPActivity()
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
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": rawJSON,
						"source":  "testsource",
					}}, nil)
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer": "testissuer",
						"credential_configurations_supported": map[string]any{
							"cred1": map[string]any{},
						},
					}}, nil)
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{},
					}}, nil)
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

			wf := NewCredentialsIssuersWorkflow()
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
				require.Contains(t, result.Message, "Successfully retrieved, stored, and updated")
				require.NotEmpty(t, result.Log.(map[string]any)["StoredCredentials"])
			}
		})
	}
}

func TestCredentialsIssuersWorkflowStart(t *testing.T) {
	origStart := credentialsStartWorkflowWithOptions
	t.Cleanup(func() {
		credentialsStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	credentialsStartWorkflowWithOptions = func(
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

	w := NewCredentialsIssuersWorkflow()
	input := workflowengine.WorkflowInput{
		Payload: CredentialsIssuersWorkflowPayload{IssuerID: "issuer"},
	}
	result, err := w.Start("ns-1", input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, CredentialsTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "Credentials-Workflow-"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}

func TestInvalidCredentialsFromSchemaValidationIssues(t *testing.T) {
	issues := []activities.SchemaValidationIssue{
		{
			Scope:        "credential",
			CredentialID: "cred-1",
			Field:        "claims",
			Message:      "claims got object, expected array",
		},
		{
			Scope:   "issuer",
			Field:   "credential_endpoint",
			Message: "credential_endpoint is missing",
		},
	}

	require.Equal(
		t,
		map[string]bool{"cred-1": true},
		invalidCredentialsFromSchemaValidationIssues(issues),
	)
}

func TestHasIssuerLevelValidationIssues(t *testing.T) {
	tests := []struct {
		name   string
		issues []activities.SchemaValidationIssue
		want   bool
	}{
		{
			name: "credential configuration errors are non-fatal",
			issues: []activities.SchemaValidationIssue{
				{Scope: "credential", CredentialID: "cred-1"},
			},
			want: false,
		},
		{
			name: "top-level error is fatal",
			issues: []activities.SchemaValidationIssue{
				{Scope: "issuer", Field: "credential_endpoint"},
			},
			want: true,
		},
		{
			name: "root required error is fatal",
			issues: []activities.SchemaValidationIssue{
				{Scope: "issuer"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, hasIssuerLevelValidationIssues(tt.issues))
		})
	}
}

func TestCredentialConfigurationsFromIssuerData(t *testing.T) {
	t.Run("uses credential_configurations_supported when present", func(t *testing.T) {
		errs := map[string]any{}
		configs, invalid := credentialConfigurationsFromIssuerData(
			map[string]any{
				"credential_configurations_supported": map[string]any{
					"cred-1": map[string]any{"format": "dc+sd-jwt"},
				},
				"credentials_supported": []any{
					map[string]any{"id": "legacy-1", "format": "ldp_vc"},
				},
			},
			errs,
		)

		require.Equal(t, map[string]any{"cred-1": map[string]any{"format": "dc+sd-jwt"}}, configs)
		require.Empty(t, invalid)
		require.Empty(t, errs)
	})

	t.Run("falls back to legacy credentials_supported as non-conformant", func(t *testing.T) {
		errs := map[string]any{}
		configs, invalid := credentialConfigurationsFromIssuerData(
			map[string]any{
				"credential_configurations_supported": nil,
				"credentials_supported": []any{
					map[string]any{
						"id":     "legacy-id",
						"format": "ldp_vc",
						"types":  []any{"VerifiableCredential", "LegacyCredential"},
					},
					map[string]any{
						"format": "jwt_vc_json",
						"types":  []any{"VerifiableCredential", "TypeBasedCredential"},
					},
				},
			},
			errs,
		)

		require.Len(t, configs, 2)
		require.Contains(t, configs, "legacy-id")
		require.Contains(t, configs, "TypeBasedCredential")
		require.Equal(t, map[string]bool{
			"legacy-id":           true,
			"TypeBasedCredential": true,
		}, invalid)
		require.Contains(t, errs, "LegacyCredentialsSupportedFallback")
	})

	t.Run("empty configurations without fallback returns warning", func(t *testing.T) {
		errs := map[string]any{}
		configs, invalid := credentialConfigurationsFromIssuerData(
			map[string]any{
				"credential_configurations_supported": map[string]any{},
			},
			errs,
		)

		require.Empty(t, configs)
		require.Empty(t, invalid)
		require.Contains(t, errs, "NoCredentialConfigurations")
	})
}

func TestCredentialIssuerIdentifierFromInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "generic issuer URL",
			input: "issuer.example.com/tenant",
			want:  "https://issuer.example.com/tenant",
		},
		{
			name:  "path-based well-known URL",
			input: "https://issuer.example.com/.well-known/openid-credential-issuer/tenant",
			want:  "https://issuer.example.com/tenant",
		},
		{
			name:  "well-known URL without path",
			input: "https://issuer.example.com/.well-known/openid-credential-issuer",
			want:  "https://issuer.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, credentialIssuerIdentifierFromInput(tt.input))
		})
	}
}

func TestValidateCredentialIssuerIdentifier(t *testing.T) {
	err := validateCredentialIssuerIdentifier(
		map[string]any{"credential_issuer": "https://issuer.example.com/tenant"},
		"https://issuer.example.com/.well-known/openid-credential-issuer/tenant",
	)
	require.NoError(t, err)

	err = validateCredentialIssuerIdentifier(
		map[string]any{"credential_issuer": "https://other.example.com/tenant"},
		"https://issuer.example.com/.well-known/openid-credential-issuer/tenant",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.SchemaValidationFailed].Code)
}

func TestExtractAppErrorDetailsFromApplicationError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func() error {
			return temporal.NewApplicationError(
				"boom",
				"type",
				[]any{map[string]any{"key": "value"}},
			)
		},
		activity.RegisterOptions{Name: "app-error"},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) ([]any, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 1,
				},
			})
			err := workflow.ExecuteActivity(ctx, "app-error").Get(ctx, nil)
			return extractAppErrorDetails(err)
		},
		workflow.RegisterOptions{Name: "extract-app-error-details"},
	)

	env.ExecuteWorkflow("extract-app-error-details")
	require.NoError(t, env.GetWorkflowError())

	var details []any
	require.NoError(t, env.GetWorkflowResult(&details))
	require.Len(t, details, 1)
}

func TestExtractAppErrorDetailsFromNonApplicationError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func() error {
			return temporal.NewCanceledError("canceled")
		},
		activity.RegisterOptions{Name: "cancel-error"},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) ([]any, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 1,
				},
			})
			err := workflow.ExecuteActivity(ctx, "cancel-error").Get(ctx, nil)
			return extractAppErrorDetails(err)
		},
		workflow.RegisterOptions{Name: "extract-non-app-error-details"},
	)

	env.ExecuteWorkflow("extract-non-app-error-details")
	require.Error(t, env.GetWorkflowError())
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
				httpAct := activities.NewInternalHTTPActivity()
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
				httpAct := activities.NewInternalHTTPActivity()
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
							"body": map[string]any{
								"dynamic": true,
								"code":    "yaml-content",
							},
						},
						Secrets: map[string]any{"token": "credential-secret"},
					}, nil)
				env.OnActivity(
					stepCIAct.Name(),
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, ok := input.Payload.(map[string]any)
						return ok &&
							payload["yaml"] == "yaml-content" &&
							requireSecrets(t, input.Secrets, map[string]string{
								"token": "credential-secret",
							})
					}),
				).
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
				httpAct := activities.NewInternalHTTPActivity()
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
			name: "Failure: invalid HTTP output (body not a map)",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{"app_url": "https://example.com"},
				Payload: GetCredentialOfferWorkflowPayload{CredentialID: "test_cred"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewInternalHTTPActivity()
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
			wf := NewGetCredentialOfferWorkflow()
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
