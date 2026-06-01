// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestAggregateInteropCells_LeafLeafCartesian(t *testing.T) {
	t.Parallel()

	inputs := []interopTieredCacheInput{{
		PipelineID:     "p1",
		TotalRuns:      10,
		TotalSuccesses: 8,
		RowCoords: []interopAxisCoord{
			{Tier: interopTierLeaf, Key: "v1"},
			{Tier: interopTierLeaf, Key: "v2"},
		},
		ColumnCoords: []interopAxisCoord{
			{Tier: interopTierLeaf, Key: "c1"},
		},
	}}

	cells := aggregateInteropCells(inputs)

	v1, ok := cells[interopTieredMatrixCellKey{rowTier: interopTierLeaf, rowKey: "v1", colTier: interopTierLeaf, colKey: "c1"}]
	require.True(t, ok)
	require.Equal(t, 10, v1.totalRuns)
	require.Equal(t, 8, v1.totalSuccesses)
	require.Len(t, v1.pipelineIDs, 1)

	v2, ok := cells[interopTieredMatrixCellKey{rowTier: interopTierLeaf, rowKey: "v2", colTier: interopTierLeaf, colKey: "c1"}]
	require.True(t, ok)
	require.Equal(t, 10, v2.totalRuns)
	require.Equal(t, 8, v2.totalSuccesses)
	require.Len(t, v2.pipelineIDs, 1)
}

func TestAggregateInteropCells_GroupLeafExistentialOnce(t *testing.T) {
	t.Parallel()

	// One cache record, two leaves under same wallet group — group×leaf must NOT double-count.
	inputs := []interopTieredCacheInput{{
		PipelineID:     "p1",
		TotalRuns:      100,
		TotalSuccesses: 80,
		RowCoords: []interopAxisCoord{
			{Tier: interopTierGroup, Key: "w1"},
			{Tier: interopTierLeaf, Key: "v1"},
			{Tier: interopTierLeaf, Key: "v2"},
		},
		ColumnCoords: []interopAxisCoord{
			{Tier: interopTierLeaf, Key: "i1"},
		},
	}}

	cells := aggregateInteropCells(inputs)

	g, ok := cells[interopTieredMatrixCellKey{rowTier: interopTierGroup, rowKey: "w1", colTier: interopTierLeaf, colKey: "i1"}]
	require.True(t, ok)
	require.Equal(t, 100, g.totalRuns)
	// Summing leaf×leaf into group would yield 200 — must not happen.
}

func TestAggregateInteropCells_GroupGroupDoubleFold(t *testing.T) {
	t.Parallel()

	inputs := []interopTieredCacheInput{{
		PipelineID:     "p1",
		TotalRuns:      50,
		TotalSuccesses: 40,
		RowCoords: []interopAxisCoord{
			{Tier: interopTierGroup, Key: "w1"},
			{Tier: interopTierLeaf, Key: "v1"},
			{Tier: interopTierLeaf, Key: "v2"},
		},
		ColumnCoords: []interopAxisCoord{
			{Tier: interopTierGroup, Key: "issuer1"},
			{Tier: interopTierLeaf, Key: "cred1"},
			{Tier: interopTierLeaf, Key: "cred2"},
		},
	}}

	cells := aggregateInteropCells(inputs)

	g, ok := cells[interopTieredMatrixCellKey{rowTier: interopTierGroup, rowKey: "w1", colTier: interopTierGroup, colKey: "issuer1"}]
	require.True(t, ok)
	require.Equal(t, 50, g.totalRuns)
}

func TestResolveAxisCoords_WalletInclusiveOrphan(t *testing.T) {
	t.Parallel()

	rec := newInteropCacheTestRecord(t)
	rec.Set("wallets", []string{"w1"})
	rec.Set("wallet_versions", []string{})

	axis, ok := getInteropAxis("wallets")
	require.True(t, ok)

	coords, err := resolveAxisCoords(nil, rec, axis)
	require.NoError(t, err)
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierGroup, Key: "w1"})
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierLeaf, Key: "w1::__no_version__"})
}

func TestResolveAxisCoords_FlatCredentials(t *testing.T) {
	t.Parallel()

	rec := newInteropCacheTestRecord(t)
	rec.Set("credentials", []string{"c1"})

	axis, ok := getInteropAxis("credentials")
	require.True(t, ok)

	coords, err := resolveAxisCoords(nil, rec, axis)
	require.NoError(t, err)
	require.Equal(t, []interopAxisCoord{{Tier: interopTierLeaf, Key: "c1"}}, coords)
}

func TestResolveAxisCoords_ConformancePath(t *testing.T) {
	t.Parallel()

	const path = "std/ver/suite/check1"

	rec := newInteropCacheTestRecord(t)
	rec.Set("conformance_checks", []string{path})

	axis, ok := getInteropAxis("conformance-checks")
	require.True(t, ok)

	coords, err := resolveAxisCoords(nil, rec, axis)
	require.NoError(t, err)
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierGroup, Key: "std/ver/suite"})
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierLeaf, Key: path})
}

func TestResolveAxisCoords_WalletWithVersion(t *testing.T) {
	t.Parallel()

	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	walletsCollection, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsCollection)
	wallet.Set("owner", orgID)
	wallet.Set("name", "interop-wallet-with-version")
	require.NoError(t, app.Save(wallet))

	versionsCollection, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)

	version := core.NewRecord(versionsCollection)
	version.Set("wallet", wallet.Id)
	version.Set("tag", "1.0.0")
	version.Set("owner", orgID)
	require.NoError(t, app.Save(version))

	cacheCollection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)

	cacheRecord := core.NewRecord(cacheCollection)
	cacheRecord.Set("wallets", []string{wallet.Id})
	cacheRecord.Set("wallet_versions", []string{version.Id})

	axis, ok := getInteropAxis("wallets")
	require.True(t, ok)

	coords, err := resolveAxisCoords(app, cacheRecord, axis)
	require.NoError(t, err)
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierGroup, Key: wallet.Id})
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierLeaf, Key: version.Id})
	require.NotContains(
		t,
		coords,
		interopAxisCoord{Tier: interopTierLeaf, Key: wallet.Id + "::" + axis.Tier.NoLeafSentinel},
	)
}

func newInteropCacheTestRecord(t testing.TB) *core.Record {
	t.Helper()

	app := setupScoreboardInteropApp(t)
	t.Cleanup(app.Cleanup)

	cacheCollection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)

	return core.NewRecord(cacheCollection)
}
