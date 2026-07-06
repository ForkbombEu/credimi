// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunnerlifecycle

import (
	"log"
	"os"
	"strings"
	"time"
)

const (
	MonitorIntervalEnv  = "MOBILE_RUNNER_LIFECYCLE_MONITOR_INTERVAL"
	HeartbeatTimeoutEnv = "MOBILE_RUNNER_LIFECYCLE_HEARTBEAT_TIMEOUT"
	ShutdownAfterEnv    = "MOBILE_RUNNER_LIFECYCLE_SHUTDOWN_AFTER"
)

const (
	DefaultMonitorInterval  = 30 * time.Second
	DefaultHeartbeatTimeout = time.Hour
	DefaultShutdownAfter    = 7 * 24 * time.Hour
)

func MonitorInterval() time.Duration {
	return durationFromEnv(MonitorIntervalEnv, DefaultMonitorInterval)
}

func HeartbeatTimeout() time.Duration {
	return durationFromEnv(HeartbeatTimeoutEnv, DefaultHeartbeatTimeout)
}

func ShutdownAfter() time.Duration {
	return durationFromEnv(ShutdownAfterEnv, DefaultShutdownAfter)
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("[WARN] Invalid %s value %q (using %s)", name, raw, fallback)
		return fallback
	}
	if duration <= 0 {
		log.Printf("[WARN] Non-positive %s value %q (using %s)", name, raw, fallback)
		return fallback
	}

	return duration
}
