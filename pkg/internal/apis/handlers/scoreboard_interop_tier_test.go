// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

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
