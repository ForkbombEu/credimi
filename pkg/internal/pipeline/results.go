// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

const (
	// ResultTypeMemoKey carries the pipeline_results type through Temporal memo.
	ResultTypeMemoKey = "pipeline_result_type"

	// ResultTypeManual marks runs started directly by a user.
	ResultTypeManual = "manual"
	// ResultTypeScheduled marks runs started by a Temporal schedule.
	ResultTypeScheduled = "scheduled"
	// ResultTypeCI marks runs started by CI integrations.
	ResultTypeCI = "CI"
)

// ValidResultType reports whether value is accepted by the pipeline_results type field.
func ValidResultType(value string) bool {
	switch value {
	case ResultTypeManual, ResultTypeScheduled, ResultTypeCI:
		return true
	default:
		return false
	}
}
