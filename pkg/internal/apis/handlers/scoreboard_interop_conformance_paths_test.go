// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInteropSuiteGroupFromPath_Valid(t *testing.T) {
	t.Parallel()

	const path = "eu/a1/suite1/check1"

	group, leaf, err := interopSuiteGroupFromPath(path)
	require.NoError(t, err)
	require.Equal(t, interopConformanceSuiteGroup{
		ID:    "eu/a1/suite1",
		Title: "eu • a1 • suite1",
	}, group)
	require.Equal(t, path, leaf)
}

func TestInteropSuiteGroupFromPath_Invalid(t *testing.T) {
	t.Parallel()

	_, _, err := interopSuiteGroupFromPath("bad/path")
	require.Error(t, err)
}
