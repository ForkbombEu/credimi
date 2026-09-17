// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"os"
	"strings"
)

// TemporalWorkersDisabledEnv disables Temporal namespace creation, worker registration
// and worker-manager workflow starts when set to a truthy value (1, true, yes).
// Intended for faster local API/UI boots that do not need pipelines or workflows.
const TemporalWorkersDisabledEnv = "CREDIMI_TEMPORAL_WORKERS_DISABLED"

// TemporalWorkersDisabled reports whether Temporal workers should stay offline.
func TemporalWorkersDisabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(TemporalWorkersDisabledEnv)))
	return value == "1" || value == "true" || value == "yes"
}
