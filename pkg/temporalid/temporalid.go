// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package temporalid keeps the Temporal IDs Credimi and credimi-runner build within the
// server's length limit. Both sides must build an ID the same way, so they share this package.
package temporalid

import (
	"crypto/sha256"
	"encoding/hex"
)

// MaxLength is the longest workflow, schedule or update ID Temporal accepts
// (limit.maxIDLength on the Credimi deployments).
const MaxLength = 255

const (
	// scheduleWorkflowPrefix is added by Temporal to a schedule ID to name the workflow
	// that runs the schedule.
	scheduleWorkflowPrefix = "temporal-sys-scheduler:"
	// scheduledRunSuffix is the shape of what Temporal appends to a schedule action's
	// workflow ID on every start: "-" and the nominal start time in RFC 3339, UTC.
	scheduledRunSuffix = "-2006-01-02T15:04:05Z"
	// hashLength is the number of hex digits of the hash that ends a shortened ID.
	hashLength = 12
)

// Bounded returns a workflow or update ID that fits in MaxLength. An id that fits is
// returned unchanged; a longer id is cut and ends in "-" and the first hashLength hex
// digits of the sha256 of the whole id. The result depends only on id, so workflow code may
// call it: replays and retries rebuild the same ID, and IDs that fit keep the value they
// always had. The hash covers the whole id, so ids that differ only past the cut, such as
// by their run or attempt suffix, stay different.
func Bounded(id string) string {
	return bounded(id, MaxLength)
}

// BoundedWithSuffix returns base+suffix within MaxLength, keeping suffix: callers that strip
// the suffix from a child workflow ID to find its parent still can. It is base+suffix
// whenever that fits; otherwise only base is shortened, so the stripped ID is no longer the
// parent's own ID, which happens only when the parent ID was too long to take the suffix.
func BoundedWithSuffix(base, suffix string) string {
	if len(base)+len(suffix) <= MaxLength {
		return base + suffix
	}
	return bounded(base, MaxLength-len(suffix)) + suffix
}

// BoundedSchedule returns a schedule ID that still fits once Temporal prefixes it to name
// the schedule's own workflow.
func BoundedSchedule(id string) string {
	return bounded(id, MaxLength-len(scheduleWorkflowPrefix))
}

// RunnerTaskQueue returns the task queue of the mobile runner at runnerPath (the canonified
// "<organization>/<runner>" path, without a leading slash). Credimi schedules runner
// activities on it and credimi-runner polls it, so both build it here.
func RunnerTaskQueue(runnerPath string) string {
	return Bounded(runnerPath + "-TaskQueue")
}

// BoundedScheduledWorkflow returns the workflow ID of a schedule action that still fits once
// Temporal appends each run's start time.
func BoundedScheduledWorkflow(id string) string {
	return bounded(id, MaxLength-len(scheduledRunSuffix))
}

func bounded(id string, limit int) string {
	if len(id) <= limit {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return id[:limit-hashLength-1] + "-" + hex.EncodeToString(sum[:])[:hashLength]
}
