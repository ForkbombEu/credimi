// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunnerlifecycle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDurationsUseDefaults(t *testing.T) {
	t.Setenv(MonitorIntervalEnv, "")
	t.Setenv(HeartbeatTimeoutEnv, "")
	t.Setenv(ShutdownAfterEnv, "")

	require.Equal(t, DefaultMonitorInterval, MonitorInterval())
	require.Equal(t, DefaultHeartbeatTimeout, HeartbeatTimeout())
	require.Equal(t, DefaultShutdownAfter, ShutdownAfter())
}

func TestDurationsUseEnvironmentOverrides(t *testing.T) {
	t.Setenv(MonitorIntervalEnv, "15s")
	t.Setenv(HeartbeatTimeoutEnv, "5m")
	t.Setenv(ShutdownAfterEnv, "30m")

	require.Equal(t, 15*time.Second, MonitorInterval())
	require.Equal(t, 5*time.Minute, HeartbeatTimeout())
	require.Equal(t, 30*time.Minute, ShutdownAfter())
}
