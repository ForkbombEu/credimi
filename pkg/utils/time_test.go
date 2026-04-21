// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareTimeStrings(t *testing.T) {
	require.True(t, TimeStringAfter("2026-04-21T10:00:00.1Z", "2026-04-21T10:00:00Z"))
	require.True(t, TimeStringBefore("2026-04-21T09:59:59.999999999Z", "2026-04-21T10:00:00Z"))
	require.Zero(t, CompareTimeStrings("2026-04-21T10:00:00Z", "2026-04-21T10:00:00.000Z"))
	require.True(t, TimeStringAfter("2026-04-21T10:00:00Z", "not-a-time"))
	require.True(t, TimeStringBefore("not-a-time", "2026-04-21T10:00:00Z"))
}
