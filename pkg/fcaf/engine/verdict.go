// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import "github.com/forkbombeu/credimi/pkg/fcaf/validators"

// Status severity. The values rank the statuses from least to most serious:
// pass, not_applicable, inconclusive, skipped, fail, blocked, error. The order
// is the single ranking both the per-assertion merge and AggregateVerdict apply,
// so a test's verdict does not depend on the order its assertions appear in.
//
// blocked outranks fail: a blocked assertion states that the evidence needed to
// judge the Wallet is unusable (the flow stored no screenshot, the producing
// step captured nothing, the Wallet never received the request), so a fail
// recorded beside it cannot be attributed to the Wallet. error outranks both
// because a broken definition or validator means nothing in the test was
// evaluated as specified. Skipped and not_applicable are statements about the
// test's setup, not about the Wallet: a skip outranks an inconclusive
// assertion, and both lose to any evaluated status.
const (
	severityPass = iota + 1
	severityNotApplicable
	severityInconclusive
	severitySkipped
	severityFail
	severityBlocked
	severityError
)

// normalizeStatus maps an unknown status string to StatusError rather than
// letting it rank as a pass.
func normalizeStatus(status validators.Status) validators.Status {
	switch status {
	case validators.StatusPass,
		validators.StatusFail,
		validators.StatusBlocked,
		validators.StatusSkipped,
		validators.StatusNotApplicable,
		validators.StatusInconclusive,
		validators.StatusError:
		return status
	default:
		return validators.StatusError
	}
}

// statusSeverity returns the rank of a status in the precedence above.
func statusSeverity(status validators.Status) int {
	switch normalizeStatus(status) {
	case validators.StatusError:
		return severityError
	case validators.StatusBlocked:
		return severityBlocked
	case validators.StatusFail:
		return severityFail
	case validators.StatusInconclusive:
		return severityInconclusive
	case validators.StatusSkipped:
		return severitySkipped
	case validators.StatusNotApplicable:
		return severityNotApplicable
	default:
		return severityPass
	}
}

// worseStatus returns the status of the two that ranks higher.
func worseStatus(current, next validators.Status) validators.Status {
	current = normalizeStatus(current)
	next = normalizeStatus(next)
	if statusSeverity(next) > statusSeverity(current) {
		return next
	}
	return current
}

func AggregateVerdict(assertions []AssertionResult) validators.Status {
	if len(assertions) == 0 {
		return validators.StatusInconclusive
	}
	verdict := validators.StatusPass
	for _, assertion := range assertions {
		verdict = worseStatus(verdict, assertion.Status)
	}
	return verdict
}

func verdictMessage(status validators.Status) string {
	switch status {
	case validators.StatusPass:
		return "all assertions passed"
	case validators.StatusFail:
		return "one or more assertions failed"
	case validators.StatusBlocked:
		return "one or more assertions were blocked"
	case validators.StatusSkipped:
		return "test was skipped"
	case validators.StatusInconclusive:
		return "one or more assertions could not be evaluated"
	case validators.StatusNotApplicable:
		return "test is not applicable"
	default:
		return "one or more assertions returned an error"
	}
}
