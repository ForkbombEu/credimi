// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

const (
	// RunTypeMemoKey carries the pipeline run type through Temporal memo.
	RunTypeMemoKey = "pipeline_run_type"
	// PublishedMemoKey carries the pipeline published state at run start.
	PublishedMemoKey = "published"

	// RunTypeManual marks runs started directly by a user.
	RunTypeManual = "manual"
	// RunTypeScheduled marks runs started by a Temporal schedule.
	RunTypeScheduled = "scheduled"
	// RunTypeCI marks runs started by CI integrations.
	RunTypeCI = "CI"
)

// ValidRunType reports whether value is accepted by the pipeline_results type field.
func ValidRunType(value string) bool {
	switch value {
	case RunTypeManual, RunTypeScheduled, RunTypeCI:
		return true
	default:
		return false
	}
}
