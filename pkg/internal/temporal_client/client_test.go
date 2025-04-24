// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import "testing"

func TestTemporalClientGet(t *testing.T) {
	t.Skip()
	have, err := GetTemporalClient()
	if err != nil {
		t.Error(err)
	}

	if have == nil {
		t.Error("Expected a non-nil temporal client, got nil")
	}
}
