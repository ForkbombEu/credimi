// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import "sync"

// pipelineResultLocks serializes file appends to one pipeline_results record:
// each upload reloads the record and appends under the lock, so concurrent
// uploads of a run's devices never drop each other's files. A lock, not a
// database transaction, because PocketBase stores the file during Save and a
// transaction would hold the app's single write connection for the whole copy.
// The map keeps one tiny entry per record that received an upload; it is not
// evicted because the process restarts on deploy.
var pipelineResultLocks sync.Map // record ID -> *sync.Mutex

func lockPipelineResult(recordID string) func() {
	m, _ := pipelineResultLocks.LoadOrStore(recordID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
