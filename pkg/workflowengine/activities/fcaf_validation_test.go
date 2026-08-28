// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestFCAFValidationActivityExecutesGeneratedTestFromAggregateOutput(t *testing.T) {
	act := NewFCAFValidationActivity()

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{
			TestIDs: []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
			Pipeline: map[string]any{
				"pipeline.pid.presentation.sdjwt.all-claims": map[string]any{
					"output": map[string]any{
						"pid_sdjwt": `{"query_0":["` + testPIDSDJWTPresentation(t) + `"]}`,
					},
				},
			},
		},
	})

	require.NoError(t, err)
	output, ok := result.Output.(FCAFValidationActivityOutput)
	require.True(t, ok)
	require.Empty(t, output.Report.Tests)
	require.Equal(t, "passed", output.Report.Status)
	require.Len(t, output.Report.ExecutedTests, 1)
	require.Equal(t, "passed", output.Report.ExecutedTests[0].Status)
	require.Contains(t, output.Report.Evidence, "pid_sdjwt")

	encoded, err := json.Marshal(output)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"tests"`)
	require.NotContains(t, string(encoded), `"preconditions"`)
}

func TestNormalizeValidationTestIDsSupportsBatchAndLegacyInputs(t *testing.T) {
	ids := normalizeValidationTestIDs(FCAFValidationActivityInput{
		TestID:  " test-legacy ",
		TestIDs: []string{"test-one", "test-one", " ", "test-two"},
	})

	require.Equal(t, []string{"test-one", "test-two", "test-legacy"}, ids)
}

func TestNormalizeValidationTestIDsRejectsEmptyInputs(t *testing.T) {
	require.Empty(t, normalizeValidationTestIDs(FCAFValidationActivityInput{}))
}

func TestFCAFValidationActivityRequiresAggregateOutput(t *testing.T) {
	act := NewFCAFValidationActivity()

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{TestID: "test-one"},
	})

	require.Error(t, err)
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
