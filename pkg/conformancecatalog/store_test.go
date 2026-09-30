// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testCatalogCache(t *testing.T, suffix string) *catalogCache {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	uri := fmt.Sprintf(
		"file:credimi_conformance_catalog_test_%s_%s?mode=memory&cache=shared",
		name,
		suffix,
	)
	c, err := openCatalogCache(uri)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestCatalogCachePrivateHandlesAreIsolated(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFixtureTree(t, root)
	loaded, err := LoadFromDir(root)
	require.NoError(t, err)
	require.NotEmpty(t, loaded.Checks)

	a := testCatalogCache(t, "a")
	b := testCatalogCache(t, "b")

	require.NoError(t, a.replaceEphemeralRows(loaded))
	nA, err := a.countChecks()
	require.NoError(t, err)
	require.Equal(t, len(loaded.Checks), nA)

	nB, err := b.countChecks()
	require.NoError(t, err)
	require.Equal(t, 0, nB, "private handle B must not see A's rows")

	require.NoError(t, b.replaceEphemeralRows(LoadedCatalog{}))
	nA, err = a.countChecks()
	require.NoError(t, err)
	require.Equal(t, len(loaded.Checks), nA, "clearing B must not clear A")
}
