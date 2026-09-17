// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunnerlifecycle

import "github.com/pocketbase/pocketbase/core"

// EligibleForWorkerStart reports whether a mobile runner record should receive
// worker-manager starts.
//
// Disabled runners are administratively excluded, and offline runners cannot
// answer the runner HTTP contract, so starting a worker manager for them only
// burns activity retries. A runner registers its own workers for every visible
// namespace when it boots (credimi-runner StartExistingWorkers), and the
// mobile_runners update hook covers runners that become enabled or come back
// online without restarting.
func EligibleForWorkerStart(record *core.Record) bool {
	if record == nil {
		return false
	}

	return !record.GetBool("disabled") && record.GetBool("online")
}
