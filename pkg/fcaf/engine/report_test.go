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

func TestPopulateExecutedTestsPassesWithPreconditionsAndAssertions(t *testing.T) {
	report := Report{
		Tests: []TestResult{{
			ID:      "test-1",
			Title:   "Email present",
			Status:  validators.StatusPass,
			Message: "all assertions passed",
			Preconditions: []NodeResult{{
				ID:      "pipeline.pid.presentation.sdjwt.all-claims",
				Kind:    "pipeline",
				Status:  validators.StatusPass,
				Message: "pipeline outputs extracted",
			}},
			Assertions: []AssertionResult{{
				ID:      "email_present",
				Status:  validators.StatusPass,
				Message: `claim "email" is present`,
			}},
		}},
	}

	report.PopulateDerivedViews()

	require.Empty(t, report.Failures)
	require.Len(t, report.ExecutedTests, 1)
	require.Equal(t, "test-1", report.ExecutedTests[0].TestID)
	require.Equal(t, "passed", report.ExecutedTests[0].Status)
	require.Equal(t, "passed", report.ExecutedTests[0].Outcome.Status)
	require.Equal(t, "all assertions passed", report.ExecutedTests[0].Outcome.Reason)
	require.Equal(t, "passed", report.ExecutedTests[0].Preconditions[0].Status)
	require.Equal(t, "passed", report.ExecutedTests[0].Assertions[0].Status)
}

func TestPopulateExecutedTestsSkippedWhenPreconditionFails(t *testing.T) {
	report := Report{
		Tests: []TestResult{{
			ID:      "test-1",
			Title:   "Email present",
			Status:  validators.StatusFail,
			Message: "one or more assertions failed",
			Preconditions: []NodeResult{{
				ID:      "pipeline.pid.presentation.sdjwt.all-claims",
				Kind:    "pipeline",
				Status:  validators.StatusFail,
				Message: "missing key in path",
			}},
			Assertions: []AssertionResult{{
				ID:      "email_present",
				Status:  validators.StatusBlocked,
				Message: `input "evidence.pid_sdjwt" is missing`,
			}},
		}},
	}

	report.PopulateDerivedViews()

	require.Len(t, report.Failures, 1)
	require.Len(t, report.ExecutedTests, 1)
	require.Equal(t, "skipped", report.ExecutedTests[0].Status)
	require.Equal(t, "skipped", report.ExecutedTests[0].Outcome.Status)
	require.Equal(t, "missing key in path", report.ExecutedTests[0].Outcome.Reason)
	require.Equal(t, "failed", report.ExecutedTests[0].Preconditions[0].Status)
	require.Equal(t, "blocked", report.ExecutedTests[0].Assertions[0].Status)
}

func TestPopulateExecutedTestsFailsWhenAssertionFails(t *testing.T) {
	report := Report{
		Tests: []TestResult{{
			ID:      "test-1",
			Title:   "Email present",
			Status:  validators.StatusFail,
			Message: "one or more assertions failed",
			Preconditions: []NodeResult{{
				ID:      "pipeline.pid.presentation.sdjwt.all-claims",
				Kind:    "pipeline",
				Status:  validators.StatusPass,
				Message: "pipeline outputs extracted",
			}},
			Assertions: []AssertionResult{{
				ID:      "email_present",
				Status:  validators.StatusFail,
				Message: `claim "email" is missing`,
			}},
		}},
	}

	report.PopulateDerivedViews()

	require.Len(t, report.Failures, 1)
	require.Len(t, report.ExecutedTests, 1)
	require.Equal(t, "failed", report.ExecutedTests[0].Status)
	require.Equal(t, "failed", report.ExecutedTests[0].Outcome.Status)
	require.Equal(t, `claim "email" is missing`, report.ExecutedTests[0].Outcome.Reason)
	require.Equal(t, "passed", report.ExecutedTests[0].Preconditions[0].Status)
	require.Equal(t, "failed", report.ExecutedTests[0].Assertions[0].Status)
}
