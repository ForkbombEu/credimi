// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package mobilerunner holds the rules shared by every surface that decides
// whether a mobile runner can be used and how it is reached over HTTP.
package mobilerunner

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const SelectorHeartbeatTTLEnv = "MOBILE_RUNNER_SELECTOR_HEARTBEAT_TTL"

// DefaultSelectorHeartbeatTTL tolerates two missed heartbeats. The runner
// heartbeats every 30s, so a runner that has not reported for a minute is
// treated as unavailable by catalog surfaces. The lifecycle heartbeat timeout
// stays an hour because it drives the durable `online` flag, which must not
// flap.
const DefaultSelectorHeartbeatTTL = 60 * time.Second

func SelectorHeartbeatTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv(SelectorHeartbeatTTLEnv))
	if raw == "" {
		return DefaultSelectorHeartbeatTTL
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil || ttl <= 0 {
		log.Printf(
			"[WARN] Invalid %s value %q (using %s)",
			SelectorHeartbeatTTLEnv,
			raw,
			DefaultSelectorHeartbeatTTL,
		)

		return DefaultSelectorHeartbeatTTL
	}

	return ttl
}

// RecentlyAlive reports whether a runner reported a heartbeat recently enough
// to be offered in a device selector. It answers "the runner was reaching
// Credimi a moment ago", which is what a catalog needs; whether the runner is
// reachable at execution time is decided by a live health check on the run
// path, not here.
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
