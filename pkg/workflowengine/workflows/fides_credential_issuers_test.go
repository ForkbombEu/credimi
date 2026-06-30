// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestFidesCredentialIssuersWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		input          workflowengine.WorkflowInput
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
		errorCode      errorcodes.Code
	}{
		{
			name: "success",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerFidesWorkflowActivities(env)

				httpAct := activities.NewHTTPActivity()
				parseAct := activities.NewParseFidesCredentialIssuersActivity()
				internalAct := activities.NewInternalHTTPActivity()
				checkAct := activities.NewCheckCredentialsIssuerActivity()
				jsonAct := activities.NewJSONActivity(nil)
				validateAct := activities.NewSchemaValidationActivity()

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"content": []any{
								map[string]any{
									"issuanceProtocol":    "oid4vci",
									"credentialIssuerUrl": "https://issuer-1",
								},
							},
							"page": map[string]any{"number": 0, "totalPages": 0},
						},
					}}, nil).
					Once()
				env.OnActivity(parseAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: activities.ParseFidesCredentialIssuersActivityResponse{
							Issuers:    []string{"https://issuer-1"},
							PageNumber: 0,
							TotalPages: 0,
						},
					}, nil).
					Once()
				env.OnActivity(
					internalAct.Name(),
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, err := workflowengine.DecodePayload[activities.InternalHTTPActivityPayload](
							input.Payload,
						)
						if err != nil {
							return false
						}
						body, ok := payload.Body.(map[string]any)
						if !ok {
							return false
						}
						return body["url"] == "https://issuer-1" &&
							body["orgID"] == "org123" &&
							body["name"] == "Issuer One" &&
							body["logo"] == "https://issuer-1/logo.png"
					}),
				).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"record": map[string]any{"id": "issuer123"},
						},
					}}, nil).
					Once()
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": `{"credential_issuer":"https://issuer-1","credential_configurations_supported":{"cred1":{}}}`,
						"source":  ".well-known/openid-credential-issuer",
					}}, nil).
					Once()
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer": "https://issuer-1",
						"display": []any{
							map[string]any{
								"name": "Issuer One",
								"logo": map[string]any{"uri": "https://issuer-1/logo.png"},
							},
						},
						"credential_configurations_supported": map[string]any{
							"cred1": map[string]any{},
						},
					}}, nil).
					Once()
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil).
					Once()
				env.OnActivity(internalAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{"key": "cred1"},
					}}, nil).
					Once()
			},
		},
		{
			name: "missing org",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
		},
		{
			name: "imports issuer without credentials when configurations are null",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerFidesWorkflowActivities(env)

				httpAct := activities.NewHTTPActivity()
				parseAct := activities.NewParseFidesCredentialIssuersActivity()
				internalAct := activities.NewInternalHTTPActivity()
				checkAct := activities.NewCheckCredentialsIssuerActivity()
				jsonAct := activities.NewJSONActivity(nil)
				validateAct := activities.NewSchemaValidationActivity()

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"content": []any{
								map[string]any{
									"issuanceProtocol":    "oid4vci",
									"credentialIssuerUrl": "https://issuer-empty",
								},
							},
							"page": map[string]any{"number": 0, "totalPages": 0},
						},
					}}, nil).
					Once()
				env.OnActivity(parseAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: activities.ParseFidesCredentialIssuersActivityResponse{
							Issuers:    []string{"https://issuer-empty"},
							PageNumber: 0,
							TotalPages: 0,
						},
					}, nil).
					Once()
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": `{"credential_issuer":"https://issuer-empty","credential_configurations_supported":null}`,
						"source":  ".well-known/openid-credential-issuer",
					}}, nil).
					Once()
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer": "https://issuer-empty",
						"display": []any{
							map[string]any{
								"name": "Issuer Empty",
								"logo": map[string]any{"uri": "https://issuer-empty/logo.png"},
							},
						},
						"credential_configurations_supported": nil,
					}}, nil).
					Once()
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil).
					Once()
				env.OnActivity(
					internalAct.Name(),
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, err := workflowengine.DecodePayload[activities.InternalHTTPActivityPayload](
							input.Payload,
						)
						if err != nil {
							return false
						}
						return payload.URL == "https://example.com/api/credentials_issuers/store-or-update"
					}),
				).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"record": map[string]any{"id": "issuer-empty-id"},
						},
					}}, nil).
					Once()
			},
		},
		{
			name: "continues after one issuer import failure",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				registerFidesWorkflowActivities(env)

				httpAct := activities.NewHTTPActivity()
				parseAct := activities.NewParseFidesCredentialIssuersActivity()
				internalAct := activities.NewInternalHTTPActivity()
				checkAct := activities.NewCheckCredentialsIssuerActivity()
				jsonAct := activities.NewJSONActivity(nil)
				validateAct := activities.NewSchemaValidationActivity()

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"content": []any{
								map[string]any{
									"issuanceProtocol":    "oid4vci",
									"credentialIssuerUrl": "https://issuer-bad",
								},
								map[string]any{
									"issuanceProtocol":    "oid4vci",
									"credentialIssuerUrl": "https://issuer-good",
								},
							},
							"page": map[string]any{"number": 0, "totalPages": 0},
						},
					}}, nil).
					Once()
				env.OnActivity(parseAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: activities.ParseFidesCredentialIssuersActivityResponse{
							Issuers:    []string{"https://issuer-bad", "https://issuer-good"},
							PageNumber: 0,
							TotalPages: 0,
						},
					}, nil).
					Once()
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, errors.New("issuer-bad failed")).
					Once()
				env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"rawJSON": `{"credential_issuer":"https://issuer-good","credential_configurations_supported":{"cred1":{}}}`,
						"source":  ".well-known/openid-credential-issuer",
					}}, nil).
					Once()
				env.OnActivity(jsonAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"credential_issuer": "https://issuer-good",
						"credential_configurations_supported": map[string]any{
							"cred1": map[string]any{},
						},
					}}, nil).
					Once()
				env.OnActivity(validateAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil).
					Once()
				env.OnActivity(internalAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"record": map[string]any{"id": "issuer-good-id"},
						},
					}}, nil).
					Once()
				env.OnActivity(internalAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{"key": "cred1"},
					}}, nil).
					Once()
			},
		},
		{
			name: "no issuers",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":       "https://example.com",
					"issuer_schema": "{}",
					"orgID":         "org123",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				parseAct := activities.NewParseFidesCredentialIssuersActivity()
				env.RegisterActivityWithOptions(
					httpAct.Execute,
					activity.RegisterOptions{Name: httpAct.Name()},
				)
				env.RegisterActivityWithOptions(
					parseAct.Execute,
					activity.RegisterOptions{Name: parseAct.Name()},
				)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{"content": []any{}, "page": map[string]any{}},
					}}, nil)
				env.OnActivity(parseAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: activities.ParseFidesCredentialIssuersActivityResponse{},
					}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := &testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			tt.mockActivities(env)

			wf := NewFidesCredentialIssuersWorkflow()
			env.ExecuteWorkflow(wf.Workflow, tt.input)

			require.True(t, env.IsWorkflowCompleted())
			if tt.expectedErr {
				err := env.GetWorkflowError()
				require.Error(t, err)
				if tt.errorCode.Code != "" {
					require.Contains(t, err.Error(), tt.errorCode.Code)
				}
				return
			}

			require.NoError(t, env.GetWorkflowError())
			var result workflowengine.WorkflowResult
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Contains(t, result.Message, "Imported 1 credential issuers from Fides")
		})
	}
}

func registerFidesWorkflowActivities(env *testsuite.TestWorkflowEnvironment) {
	httpAct := activities.NewHTTPActivity()
	parseAct := activities.NewParseFidesCredentialIssuersActivity()
	internalAct := activities.NewInternalHTTPActivity()
	checkAct := activities.NewCheckCredentialsIssuerActivity()
	jsonAct := activities.NewJSONActivity(nil)
	validateAct := activities.NewSchemaValidationActivity()

	env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{Name: httpAct.Name()})
	env.RegisterActivityWithOptions(
		parseAct.Execute,
		activity.RegisterOptions{Name: parseAct.Name()},
	)
	env.RegisterActivityWithOptions(
		internalAct.Execute,
		activity.RegisterOptions{Name: internalAct.Name()},
	)
	env.RegisterActivityWithOptions(
		checkAct.Execute,
		activity.RegisterOptions{Name: checkAct.Name()},
	)
	env.RegisterActivityWithOptions(jsonAct.Execute, activity.RegisterOptions{Name: jsonAct.Name()})
	env.RegisterActivityWithOptions(
		validateAct.Execute,
		activity.RegisterOptions{Name: validateAct.Name()},
	)
}

func TestFidesCredentialIssuersWorkflowOptions(t *testing.T) {
	require.Equal(t, DefaultActivityOptions, NewFidesCredentialIssuersWorkflow().GetOptions())
}
