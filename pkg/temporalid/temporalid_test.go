// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalid

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestBounded(t *testing.T) {
	atLimit := strings.Repeat("a", MaxLength)
	require.Equal(t, "short-id", Bounded("short-id"))
	require.Equal(t, atLimit, Bounded(atLimit))

	// The ID of the reported failure: a step of a linked block in a long FCAF pipeline.
	long := "Pipeline-fcaf-wallet-attestation-provider-complete-validation-" +
		"3b3a6ffc-43eb-41e2-be81-25864385330d-jwt-key-attestation-issuance-d607cf80-" +
		"create-jwt-key-attestation-issuance-session-jwt-key-attestation-issuance-d607cf80-" +
		"obtain-jwt-key-attestation-credential-a1"
	bounded := Bounded(long)
	require.Len(t, bounded, MaxLength)
	require.True(t, strings.HasPrefix(bounded, long[:MaxLength-hashLength-1]+"-"))
	require.Equal(t, bounded, Bounded(long), "the same ID must give the same result")

	// IDs that differ only past the cut stay different.
	require.NotEqual(t, bounded, Bounded(strings.TrimSuffix(long, "-a1")+"-a2"))
}

func TestBoundedScheduleIDsLeaveRoomForTemporalAffixes(t *testing.T) {
	long := strings.Repeat("s", 400)
	require.Len(t, scheduleWorkflowPrefix+BoundedSchedule(long), MaxLength)
	require.Len(t, BoundedScheduledWorkflow(long)+scheduledRunSuffix, MaxLength)
	require.Equal(t, "Schedule_ID-x", BoundedSchedule("Schedule_ID-x"))
	require.Equal(t, "Scheduled-x", BoundedScheduledWorkflow("Scheduled-x"))
}

func TestRunnerTaskQueue(t *testing.T) {
	require.Equal(t, "org/runner-TaskQueue", RunnerTaskQueue("org/runner"))
	require.Len(t, RunnerTaskQueue(strings.Repeat("r", 300)), MaxLength)
}

func TestBoundedKeepsValidUTF8(t *testing.T) {
	// A child pipeline ID ends in the child pipeline's raw name, which may hold
	// multi-byte characters across the cut.
	for offset := range 4 {
		id := strings.Repeat("a", offset) + strings.Repeat("è€😀", 60)
		bounded := Bounded(id)
		require.True(t, utf8.ValidString(bounded), "offset %d", offset)
		require.LessOrEqual(t, len(bounded), MaxLength)
		require.Greater(t, len(bounded), MaxLength-4)
	}
}
