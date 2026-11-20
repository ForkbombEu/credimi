// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
)

func assertionResults(statuses ...validators.Status) []AssertionResult {
	results := make([]AssertionResult, 0, len(statuses))
	for _, status := range statuses {
		results = append(results, AssertionResult{
			ID:     "assertion",
			Status: status,
		})
	}
	return results
}

// The verdict is a ranking of the statuses, so the assertion order cannot
// change it and a blocked harness gap outranks a fail recorded beside it.
func TestAggregateVerdictIsOrderIndependent(t *testing.T) {
	tests := []struct {
		name     string
		statuses []validators.Status
		want     validators.Status
	}{
		{
			name:     "blocked beats a fail in either order",
			statuses: []validators.Status{validators.StatusFail, validators.StatusBlocked},
			want:     validators.StatusBlocked,
		},
		{
			name:     "blocked beats a fail whichever assertion comes first",
			statuses: []validators.Status{validators.StatusBlocked, validators.StatusFail},
			want:     validators.StatusBlocked,
		},
		{
			name: "error beats blocked and fail",
			statuses: []validators.Status{
				validators.StatusBlocked,
				validators.StatusFail,
				validators.StatusError,
			},
			want: validators.StatusError,
		},
		{
			name:     "fail beats a pass",
			statuses: []validators.Status{validators.StatusPass, validators.StatusFail},
			want:     validators.StatusFail,
		},
		{
			name:     "not applicable beats a pass",
			statuses: []validators.Status{validators.StatusPass, validators.StatusNotApplicable},
			want:     validators.StatusNotApplicable,
		},
		{
			// A skip states the test's setup cannot be built, so it outranks an
			// inconclusive assertion, matching the ranking AggregateVerdict
			// applied before the ranking was made explicit.
			name: "skip outranks inconclusive",
			statuses: []validators.Status{
				validators.StatusInconclusive,
				validators.StatusSkipped,
			},
			want: validators.StatusSkipped,
		},
		{
			name:     "all passes",
			statuses: []validators.Status{validators.StatusPass, validators.StatusPass},
			want:     validators.StatusPass,
		},
		{
			name: "an unknown status is an error, never a pass",
			statuses: []validators.Status{
				validators.StatusPass,
				validators.Status("finished"),
			},
			want: validators.StatusError,
		},
		{
			name:     "no assertions is inconclusive",
			statuses: nil,
			want:     validators.StatusInconclusive,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, AggregateVerdict(assertionResults(test.statuses...)))
		})
	}
}

// mergeStatus builds the test verdict as assertions are evaluated, so it must
// apply exactly the ranking AggregateVerdict applies.
func TestMergeStatusUsesTheAggregateRanking(t *testing.T) {
	tests := []struct {
		name          string
		current, next validators.Status
		want          validators.Status
	}{
		{
			name:    "blocked upgrades a fail",
			current: validators.StatusFail,
			next:    validators.StatusBlocked,
			want:    validators.StatusBlocked,
		},
		{
			name:    "fail does not downgrade a block",
			current: validators.StatusBlocked,
			next:    validators.StatusFail,
			want:    validators.StatusBlocked,
		},
		{
			name:    "pass is the identity",
			current: validators.StatusPass,
			next:    validators.StatusPass,
			want:    validators.StatusPass,
		},
		{
			name:    "error upgrades everything",
			current: validators.StatusBlocked,
			next:    validators.StatusError,
			want:    validators.StatusError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, mergeStatus(test.current, test.next))
			require.Equal(
				t,
				test.want,
				AggregateVerdict(assertionResults(test.current, test.next)),
			)
		})
	}
}
