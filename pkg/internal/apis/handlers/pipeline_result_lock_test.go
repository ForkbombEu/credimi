// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockPipelineResultSerializesOneRecord(t *testing.T) {
	unlockFirst := lockPipelineResult("lock-test-record")

	// The record's mutex is held, so a second lock on it must wait.
	m, ok := pipelineResultLocks.Load("lock-test-record")
	require.True(t, ok)
	if m.(*sync.Mutex).TryLock() {
		m.(*sync.Mutex).Unlock()
		require.FailNow(t, "record lock is not held after lockPipelineResult")
	}

	secondLocked := make(chan func())
	go func() {
		secondLocked <- lockPipelineResult("lock-test-record")
	}()

	// A different record is not blocked by the held lock.
	otherLocked := make(chan func())
	go func() {
		otherLocked <- lockPipelineResult("lock-test-other-record")
	}()
	unlockOther := <-otherLocked
	unlockOther()

	select {
	case unlockSecond := <-secondLocked:
		unlockSecond()
		require.FailNow(t, "second lock on the same record acquired while the first is held")
	default:
	}

	unlockFirst()
	unlockSecond := <-secondLocked
	unlockSecond()
}
