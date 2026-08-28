// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import "github.com/forkbombeu/credimi/pkg/fcaf/validators"

func AggregateVerdict(assertions []AssertionResult) validators.Status {
	if len(assertions) == 0 {
		return validators.StatusInconclusive
	}
	hasInconclusive := false
	hasNotApplicable := false
	for _, assertion := range assertions {
		switch assertion.Status {
		case validators.StatusError:
			return validators.StatusError
		case validators.StatusFail:
			return validators.StatusFail
		case validators.StatusBlocked:
			return validators.StatusBlocked
		case validators.StatusInconclusive:
			hasInconclusive = true
		case validators.StatusNotApplicable:
			hasNotApplicable = true
		}
	}
	if hasInconclusive {
		return validators.StatusInconclusive
	}
	if hasNotApplicable {
		return validators.StatusNotApplicable
	}
	return validators.StatusPass
}

func verdictMessage(status validators.Status) string {
	switch status {
	case validators.StatusPass:
		return "all assertions passed"
	case validators.StatusFail:
		return "one or more assertions failed"
	case validators.StatusBlocked:
		return "one or more dependencies were blocked"
	case validators.StatusInconclusive:
		return "one or more assertions could not be evaluated"
	case validators.StatusNotApplicable:
		return "test is not applicable"
	default:
		return "one or more assertions returned an error"
	}
}
