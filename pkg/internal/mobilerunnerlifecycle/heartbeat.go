// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunnerlifecycle

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const SelectorHeartbeatTTLEnv = "MOBILE_RUNNER_SELECTOR_HEARTBEAT_TTL"

// DefaultSelectorHeartbeatTTL tolerates two missed heartbeats. The runner
// heartbeats every 30s, so a runner that has not reported for a minute is
// treated as unavailable by catalog surfaces. HeartbeatTimeout stays an hour
// because it drives the durable `online` flag, which must not flap.
const DefaultSelectorHeartbeatTTL = 60 * time.Second

func SelectorHeartbeatTTL() time.Duration {
	return durationFromEnv(SelectorHeartbeatTTLEnv, DefaultSelectorHeartbeatTTL)
}

// RecentlyAlive reports whether a runner reported a heartbeat recently enough
// to be offered in a device selector. It answers "the runner was reaching
// Credimi a moment ago", which is what a catalog needs; whether the runner is
// reachable from Credimi at execution time is decided by a live health check
// on the run path, not here.
func RecentlyAlive(record *core.Record, now time.Time) bool {
	if record == nil || record.GetBool("disabled") || !record.GetBool("online") {
		return false
	}
	heartbeat := record.GetDateTime("last_heartbeat_at").Time()
	if heartbeat.IsZero() {
		return false
	}

	return now.Sub(heartbeat) <= SelectorHeartbeatTTL()
}
