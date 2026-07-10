// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestFCAFAssessmentActivityExecutesGeneratedTest(t *testing.T) {
	act := NewFCAFAssessmentActivity()

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: FCAFAssessmentActivityInput{
			TestIDs: []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
			Evidence: evidence.Bundle{
				PipelineOutputs: map[string]any{
					"pipeline.pid.presentation.sdjwt.all-claims": map[string]any{
						"output": map[string]any{
							"capture-issuer-pid-dc-sd-jwt-0002": map[string]any{
								"outputs": "offer",
							},
							"getcredential-pid-sdjwt-all-claims-0003": map[string]any{
								"outputs": "issued",
							},
							"fake-verifier-pid-sd-jwt-credentials-all-claims-0006": map[string]any{
								"outputs": "request",
							},
							"verifycredential-pid-formeu-issuer-eudiw-dev-0007": map[string]any{
								"outputs": "presented",
							},
							"http-get-verifier-backend.eudiw.dev-0008": map[string]any{
								"outputs": map[string]any{
									"body": map[string]any{
										"observed": map[string]any{
											"wallet_response": map[string]any{
												"value": map[string]any{
													"vp_token": `{"query_0":["` + testPIDSDJWTPresentation(
														t,
													) + `"]}`,
												},
											},
										},
									},
								},
							},
						},
						"workflowId":    "wf-1",
						"workflowRunId": "run-1",
					},
				},
			},
		},
	})

	require.NoError(t, err)
	output, ok := result.Output.(FCAFAssessmentActivityOutput)
	require.True(t, ok)
	require.Empty(t, output.Report.Tests)
	require.Equal(t, "passed", output.Report.Status)
	require.Len(t, output.Report.ExecutedTests, 1)
	require.Equal(t, "passed", output.Report.ExecutedTests[0].Status)
	require.Contains(t, output.Report.Evidence, "pid_sdjwt")

	encoded, err := json.Marshal(output)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"tests"`)
}

func TestFCAFAssessmentActivityAllowsAllApplicableTests(t *testing.T) {
	act := NewFCAFAssessmentActivity()
	act.catalogLoader = func(root string) (*catalog.Catalog, error) {
		return &catalog.Catalog{
			Tests:         map[string]dsl.TestDefinition{},
			Preconditions: map[string]dsl.PreconditionDefinition{},
		}, nil
	}

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: FCAFAssessmentActivityInput{
			Suite: "wallet_solution/relying_party",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result.Output)
}

func testPIDSDJWTPresentation(t *testing.T) string {
	t.Helper()

	header := map[string]any{"alg": "none"}
	payload := map[string]any{
		"vct":               "urn:eudi:pid:1",
		"iss":               "https://issuer.example.test",
		"family_name":       "Trotter",
		"given_name":        "Filippo",
		"birthdate":         "1999-11-01",
		"place_of_birth":    map[string]any{"country": "IT"},
		"nationalities":     []string{"IT"},
		"date_of_expiry":    "2026-10-11",
		"issuing_authority": "GR Administrative authority",
		"issuing_country":   "GR",
		"email":             "person@example.test",
	}

	return encodeJWTLikeSegment(t, header) + "." + encodeJWTLikeSegment(t, payload) + ".~"
}

func encodeJWTLikeSegment(t *testing.T, value map[string]any) string {
	t.Helper()

	data, err := json.Marshal(value)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(data)
}
