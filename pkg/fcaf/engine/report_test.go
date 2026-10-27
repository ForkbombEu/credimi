// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
)

func TestEvidenceValuePreservesSDJWTKeyBindingMaterial(t *testing.T) {
	value := evidenceValue(&evidence.SDJWTPresentation{
		Raw:           "issuer~disclosure~kb-jwt",
		SDJWT:         "issuer~disclosure~",
		KeyBindingJWT: "kb-jwt",
	})

	serialized, ok := value.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "issuer~disclosure~kb-jwt", serialized["raw"])
	require.Equal(t, "issuer~disclosure~", serialized["sd_jwt"])
	require.Equal(t, "kb-jwt", serialized["key_binding_jwt"])
}

func TestEvidenceValuePreservesAllSDJWTPresentations(t *testing.T) {
	value := evidenceValue([]*evidence.SDJWTPresentation{
		{Raw: "first~kb-1", SDJWT: "first~", KeyBindingJWT: "kb-1"},
		{Raw: "second~kb-2", SDJWT: "second~", KeyBindingJWT: "kb-2"},
	})

	serialized, ok := value.([]any)
	require.True(t, ok)
	require.Len(t, serialized, 2)
	first, ok := serialized[0].(map[string]any)
	require.True(t, ok)
	second, ok := serialized[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "first~kb-1", first["raw"])
	require.Equal(t, "second~kb-2", second["raw"])
	require.Equal(t, "sdjwt.presentations", evidenceType([]*evidence.SDJWTPresentation{}))
}

func TestPopulateExecutedTestsPassesWithAssertions(t *testing.T) {
	report := Report{Tests: []TestResult{{
		ID:      "test-1",
		Title:   "Email present",
		Status:  validators.StatusPass,
		Message: "all assertions passed",
		Assertions: []AssertionResult{{
			ID:      "email_present",
			Status:  validators.StatusPass,
			Message: `claim "email" is present`,
		}},
	}}}

	report.PopulateDerivedViews()

	require.Empty(t, report.Failures)
	require.Len(t, report.ExecutedTests, 1)
	require.Equal(t, "passed", report.ExecutedTests[0].Status)
	require.Equal(t, "passed", report.ExecutedTests[0].Assertions[0].Status)
}

func TestPopulateExecutedTestsFailsWhenAssertionFails(t *testing.T) {
	report := Report{Tests: []TestResult{{
		ID:      "test-1",
		Title:   "Email present",
		Status:  validators.StatusFail,
		Message: "one or more assertions failed",
		Assertions: []AssertionResult{{
			ID:      "email_present",
			Status:  validators.StatusFail,
			Message: `claim "email" is missing`,
		}},
	}}}

	report.PopulateDerivedViews()

	require.Len(t, report.Failures, 1)
	require.Equal(t, "failed", report.ExecutedTests[0].Status)
	require.Equal(t, `claim "email" is missing`, report.ExecutedTests[0].Outcome.Reason)
}

func TestPopulateExecutedTestsKeepsBlockedAndInconclusiveApartFromFailed(t *testing.T) {
	report := Report{Tests: []TestResult{
		{
			ID:     "blocked-test",
			Status: validators.StatusBlocked,
			Assertions: []AssertionResult{{
				ID:      "wallet_requests_authorization",
				Status:  validators.StatusBlocked,
				Message: "the Wallet was not invoked",
			}},
		},
		{
			ID:         "inconclusive-test",
			Status:     validators.StatusInconclusive,
			Assertions: []AssertionResult{{ID: "a", Status: validators.StatusInconclusive}},
		},
		{
			ID:         "error-test",
			Status:     validators.StatusError,
			Assertions: []AssertionResult{{ID: "a", Status: validators.StatusError}},
		},
	}}

	report.PopulateDerivedViews()

	require.Equal(t, "blocked", report.ExecutedTests[0].Status)
	require.Equal(t, "blocked", report.ExecutedTests[0].Outcome.Status)
	require.Equal(t, "the Wallet was not invoked", report.ExecutedTests[0].Outcome.Reason)
	require.Equal(t, "inconclusive", report.ExecutedTests[1].Status)
	require.Equal(t, "failed", report.ExecutedTests[2].Status)
	require.Equal(t, []PresentationFilter{
		{Key: "failed", Label: "Failed", Count: 1},
		{Key: "blocked", Label: "Blocked", Count: 1},
		{Key: "inconclusive", Label: "Inconclusive", Count: 1},
	}, BuildPresentation(report, nil).SummaryFilters)
}

func TestPopulateExecutedTestsCarriesPerTestVisualEvidence(t *testing.T) {
	first := Report{Tests: []TestResult{{
		ID:     "cryptography-test",
		Status: validators.StatusPass,
		Evidence: []EvidenceResult{{
			Name:       "visual_evidence",
			SourceNode: "pipeline.dcql.cryptography",
			Value:      []any{"https://app.test/a/cryptography.png"},
		}},
	}}}
	second := Report{Tests: []TestResult{{
		ID:     "trust-test",
		Status: validators.StatusPass,
		Evidence: []EvidenceResult{{
			Name:       "visual_evidence",
			SourceNode: "pipeline.dcql.trust-mechanisms",
			Value:      []any{"https://app.test/a/trust.png"},
		}},
	}}}

	first.PopulateDerivedViews()
	second.PopulateDerivedViews()

	require.Len(t, first.ExecutedTests[0].Evidence, 1)
	require.Equal(t, "pipeline.dcql.cryptography", first.ExecutedTests[0].Evidence[0].SourceNode)
	require.Equal(
		t,
		[]string{"https://app.test/a/cryptography.png"},
		first.ExecutedTests[0].Evidence[0].Visual,
	)
	require.Equal(
		t,
		[]string{"https://app.test/a/trust.png"},
		second.ExecutedTests[0].Evidence[0].Visual,
	)
}
