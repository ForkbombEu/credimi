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

func TestEngineReportsUnbuildablePrerequisiteAsSkippedWithReason(t *testing.T) {
	cat := testCatalog("pipeline.pid.outputs.value", "fcaf.skipped")
	test := cat.Tests["test-1"]
	test.Assertions[0].Params = map[string]any{"reason": "needs an RS384 request key"}
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
			"pipeline.pid": map[string]any{"output": map[string]any{"value": "session"}},
		}},
	)
	report.PopulateDerivedViews()

	require.NoError(t, err)
	require.Equal(t, validators.StatusSkipped, report.Tests[0].Status)
	require.Equal(t, 1, report.Summary.Skipped)
	require.Zero(t, report.Summary.Blocked)
	require.Equal(t, "skipped", report.ExecutedTests[0].Status)
	require.Equal(t, "skipped", report.ExecutedTests[0].Outcome.Status)
	require.Equal(t, "needs an RS384 request key", report.ExecutedTests[0].Outcome.Reason)
}

func TestEngineReportsForbiddenFeatureAsNotApplicableWithReason(t *testing.T) {
	cat := testCatalog("pipeline.pid.outputs.value", "fcaf.not_applicable")
	test := cat.Tests["test-1"]
	test.Assertions[0].Params = map[string]any{
		"reason": "source relevancy is EUDI_forbidden: an EUDI wallet must not support the feature",
	}
	test.Assertions = append(test.Assertions, dsl.AssertionDefinition{
		ID:        "request-shape-still-holds",
		Validator: "evidence.present",
		Input:     "evidence.value",
	})
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
			"pipeline.pid": map[string]any{"output": map[string]any{"value": "session"}},
		}},
	)
	report.PopulateDerivedViews()

	require.NoError(t, err)
	require.Equal(t, validators.StatusPass, report.Tests[0].Assertions[1].Status)
	require.Equal(t, validators.StatusNotApplicable, report.Tests[0].Status)
	require.Equal(t, 1, report.Summary.NotApplicable)
	require.Zero(t, report.Summary.Fail)
	require.Zero(t, report.Summary.Pass)
	// The derived views keep not_applicable distinct from skipped so the PDF's
	// "Not applicable" label is reachable for the test and for the assertion.
	require.Equal(t, "not_applicable", report.ExecutedTests[0].Status)
	require.Equal(t, "not_applicable", report.ExecutedTests[0].Outcome.Status)
	require.Equal(t, "not_applicable", report.ExecutedTests[0].Assertions[0].Status)
	require.Equal(t, "passed", report.ExecutedTests[0].Assertions[1].Status)
	require.Zero(t, report.Summary.Skipped)
	require.Equal(
		t,
		"source relevancy is EUDI_forbidden: an EUDI wallet must not support the feature",
		report.ExecutedTests[0].Outcome.Reason,
	)
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
