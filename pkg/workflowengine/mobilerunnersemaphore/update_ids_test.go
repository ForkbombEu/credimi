// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package mobilerunnersemaphore

import "testing"

func TestUpdateIDsAreDistinct(t *testing.T) {
	requestID := "lease-1"
	acquireID := AcquireUpdateID(requestID)
	releaseID := ReleaseUpdateID(requestID)

	if acquireID == releaseID {
		t.Fatalf("expected distinct update IDs, got %q", acquireID)
	}

	if acquireID != acquireUpdateIDPrefix+requestID {
		t.Fatalf("unexpected acquire update ID: %q", acquireID)
	}

	if releaseID != releaseUpdateIDPrefix+requestID {
		t.Fatalf("unexpected release update ID: %q", releaseID)
	}
}
