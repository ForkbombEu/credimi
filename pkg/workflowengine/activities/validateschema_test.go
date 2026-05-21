// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestSchemaValidationActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	act := NewSchemaValidationActivity()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
		Name: act.Name(),
	})

	tests := []struct {
		name            string
		payload         SchemaValidationActivityPayload
		expectErr       bool
		expectedErrCode errorcodes.Code
		expectedErrMsg  string
	}{
		{
			name: "Success - valid schema and data",
			payload: SchemaValidationActivityPayload{
				Schema: `{
					"type": "object",
					"properties": {
						"name": { "type": "string" }
					},
					"required": ["name"]
				}`,
				Data: map[string]any{
					"name": "Credimi",
				},
			},
			expectErr: false,
		},
		{
			name: "Success - valid schema with subschema",
			payload: SchemaValidationActivityPayload{
				Schema: `{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
						"person": { "$ref": "subschema1.json" }
					},
					"required": ["person"]
				}`,
				SubSchema: []any{
					`{
						"$id": "person.json",
						"type": "object",
						"properties": {
							"name": { "type": "string" },
							"age": { "type": "integer" }
						},
						"required": ["name", "age"]
					}`,
				},
				Data: map[string]any{
					"person": map[string]any{
						"name": "Alice",
						"age":  30,
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Failure - missing schema",
			payload: SchemaValidationActivityPayload{
				Data: map[string]any{
					"name": "Credimi",
				},
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure - missing data",
			payload: SchemaValidationActivityPayload{
				Schema: `{"type":"object"}`,
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},

		{
			name: "Failure - invalid schema JSON",
			payload: SchemaValidationActivityPayload{
				Schema: `{"type":`,
				Data: map[string]any{
					"name": "Credimi",
				},
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.JSONUnmarshalFailed],
		},
		{
			name: "Failure - schema validation fails",
			payload: SchemaValidationActivityPayload{
				Schema: `{
					"type": "object",
					"properties": {
						"age": { "type": "integer" }
					},
					"required": ["age"]
				}`,
				Data: map[string]any{
					"age": "not-an-integer",
				},
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.SchemaValidationFailed],
			expectedErrMsg:  "jsonschema validation failed with 'file:///schema.json#'\n- at '/age' [S#/properties/age/type]: got string, want integer",
		},
		{
			name: "Failure - invalid subschema type",
			payload: SchemaValidationActivityPayload{
				Schema:    `{"type":"object"}`,
				Data:      map[string]any{},
				SubSchema: 123, // invalid
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workflowengine.ActivityInput{
				Payload: tt.payload,
			}
			future, err := env.ExecuteActivity(act.Execute, input)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrCode.Code)
				if tt.expectedErrMsg == "" {
					require.Contains(t, err.Error(), tt.expectedErrCode.Description)
				}
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				require.NoError(t, future.Get(&result))
			}
		})
	}
}

func TestCredentialIssuerSchemaOID4VCI10(t *testing.T) {
	schema, err := os.ReadFile(filepath.Join(
		"..",
		"..",
		"..",
		"schemas",
		"credentialissuer",
		"openid-credential-issuer.schema.json",
	))
	require.NoError(t, err)

	act := NewSchemaValidationActivity()
	validIssuer := map[string]any{
		"credential_issuer":   "https://issuer.example.com",
		"credential_endpoint": "https://issuer.example.com/credential",
		"batch_credential_issuance": map[string]any{
			"batch_size": 2,
		},
		"credential_request_encryption": map[string]any{
			"jwks": map[string]any{
				"keys": []any{
					map[string]any{
						"kty": "RSA",
						"use": "enc",
						"kid": "key-1",
						"n":   "abc",
						"e":   "AQAB",
					},
				},
			},
			"enc_values_supported": []any{"A256GCM"},
			"encryption_required":  true,
			"zip_values_supported": []any{"DEF"},
		},
		"credential_configurations_supported": map[string]any{
			"pid": map[string]any{
				"format": "dc+sd-jwt",
				"vct":    "https://issuer.example.com/pid",
				"credential_metadata": map[string]any{
					"display": []any{
						map[string]any{
							"name":        "PID",
							"description": "Person identification data",
							"logo": map[string]any{
								"uri": "https://issuer.example.com/logo.png",
							},
						},
					},
					"claims": []any{
						map[string]any{
							"path":      []any{"given_name"},
							"mandatory": true,
						},
						map[string]any{
							"path": []any{"addresses", 0, "street_address"},
						},
						map[string]any{
							"path": []any{"nationalities", nil},
						},
					},
				},
			},
			"jwt_vc": map[string]any{
				"format": "jwt_vc_json",
				"credential_definition": map[string]any{
					"type": []any{"VerifiableCredential", "UniversityDegreeCredential"},
				},
			},
			"ldp_vc": map[string]any{
				"format": "ldp_vc",
				"credential_definition": map[string]any{
					"@context": []any{"https://www.w3.org/2018/credentials/v1"},
					"type":     []any{"VerifiableCredential", "UniversityDegreeCredential"},
				},
			},
			"mdl": map[string]any{
				"format":  "mso_mdoc",
				"doctype": "org.iso.18013.5.1.mDL",
				"credential_signing_alg_values_supported": []any{-7, -9},
			},
		},
	}

	_, err = act.Execute(t.Context(), workflowengine.ActivityInput{
		Payload: SchemaValidationActivityPayload{
			Schema: string(schema),
			Data:   validIssuer,
		},
	})
	require.NoError(t, err)

	invalidIssuer := map[string]any{
		"credential_issuer":   "https://issuer.example.com",
		"credential_endpoint": "https://issuer.example.com/credential",
		"credential_configurations_supported": map[string]any{
			"pid": map[string]any{
				"format": "dc+sd-jwt",
			},
		},
	}
	_, err = act.Execute(t.Context(), workflowengine.ActivityInput{
		Payload: SchemaValidationActivityPayload{
			Schema: string(schema),
			Data:   invalidIssuer,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.SchemaValidationFailed].Code)
}
