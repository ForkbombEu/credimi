// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
)

func TestEngineConsumesDirectAggregatePipelineOutput(t *testing.T) {
	cat := testCatalog("pipeline.pid.outputs.value", "evidence.present")
	fcafEngine, err := New(nil)
	require.NoError(t, err)

	report, err := fcafEngine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"wallet_solution/relying_party",
		nil,
		evidence.Bundle{PipelineOutputs: map[string]any{
			"pipeline.pid": map[string]any{
				"output": map[string]any{"value": "direct aggregate evidence"},
			},
		}},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusPass, report.Tests[0].Status)
	require.Equal(t, "direct aggregate evidence", report.Tests[0].Evidence[0].Value)
}

func TestEngineDoesNotReuseUnrelatedAggregateOutput(t *testing.T) {
	cat := testCatalog("pipeline.original.outputs.value", "evidence.present")
	fcafEngine, err := New(nil)
	require.NoError(t, err)

	report, err := fcafEngine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		nil,
		evidence.Bundle{PipelineOutputs: map[string]any{
			"pipeline.unrelated": map[string]any{
				"output": map[string]any{"value": "wrong scenario"},
			},
		}},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusBlocked, report.Tests[0].Status)
	require.Empty(t, report.Tests[0].Evidence)
}

func TestEngineBlocksMissingDirectOutput(t *testing.T) {
	cat := testCatalog("pipeline.pid.outputs.missing", "evidence.present")
	fcafEngine, err := New(nil)
	require.NoError(t, err)

	report, err := fcafEngine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		nil,
		evidence.Bundle{PipelineOutputs: map[string]any{
			"pipeline.pid": map[string]any{"output": map[string]any{"other": true}},
		}},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusBlocked, report.Tests[0].Status)
	require.Equal(t, validators.StatusBlocked, report.Tests[0].Assertions[0].Status)
}

func TestEnginePassesRawVPTokenJSONToSDJWTValidator(t *testing.T) {
	cat := testCatalog("pipeline.pid.outputs.pid_sdjwt", "sdjwt.claim_present")
	test := cat.Tests["test-1"]
	test.Assertions[0].Params = map[string]any{"claim": "email"}
	cat.Tests["test-1"] = test
	fcafEngine, err := New(nil)
	require.NoError(t, err)

	report, err := fcafEngine.ExecuteCatalog(
		context.Background(),
		cat,
		[]string{"test-1"},
		"",
		nil,
		evidence.Bundle{PipelineOutputs: map[string]any{
			"pipeline.pid": map[string]any{"output": map[string]any{
				"pid_sdjwt": `{"query_0":["` + testSDJWTPresentation(t) + `"]}`,
			}},
		}},
	)

	require.NoError(t, err)
	require.Equal(t, validators.StatusPass, report.Tests[0].Status)
}

func testCatalog(binding string, validator string) *catalog.Catalog {
	return &catalog.Catalog{Tests: map[string]dsl.TestDefinition{
		"test-1": {
			ID:                  "test-1",
			Suite:               dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
			NormativeReferences: []dsl.NormativeReference{{Title: "reference"}},
			Evidence: map[string]dsl.EvidenceBinding{
				"value": {From: binding},
			},
			Assertions: []dsl.AssertionDefinition{{
				ID:        "check-value",
				Validator: validator,
				Input:     "evidence.value",
			}},
			Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
		},
	}}
}

func testSDJWTPresentation(t *testing.T) string {
	t.Helper()
	header := map[string]any{"alg": "none"}
	payload := map[string]any{"email": "person@example.test"}
	return encodeJWTLikeSegment(t, header) + "." + encodeJWTLikeSegment(t, payload) + ".~"
}

func encodeJWTLikeSegment(t *testing.T, value map[string]any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(data)
}
