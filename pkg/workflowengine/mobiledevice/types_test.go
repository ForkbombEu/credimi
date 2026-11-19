// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobiledevice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceSummaries(t *testing.T) {
	for _, deviceID := range []string{
		"forkbomb-bv-andrea/local-runner/pixel-7a",
		"/forkbomb-bv-andrea/local-runner/pixel-7a",
	} {
		require.Equal(t, "Prepare a mobile device (pixel-7a)", PrepareSummary(deviceID))
		require.Equal(t, "Clean up a mobile device (pixel-7a)", CleanUpSummary(deviceID))
	}
}
