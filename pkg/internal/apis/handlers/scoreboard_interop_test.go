// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func findInteropCell(resp InteropMatrixResponse, rowID, colID string) (InteropMatrixCell, bool) {
	for _, c := range resp.Cells {
		if c.RowID == rowID && c.ColumnID == colID {
			return c, true
		}
	}
	return InteropMatrixCell{}, false
}

func TestBuildInteropMatrix_CartesianAndSums(t *testing.T) {
	t.Parallel()

	const (
		w1 = "wallet1"
		w2 = "wallet2"
		i1 = "issuer1"
		p1 = "pipeline_one"
		p2 = "pipeline_two"
	)
	rowEntities := map[string]InteropMatrixEntity{
		w1: {ID: w1, Name: "Wallet B", Path: "/w/b"},
		w2: {ID: w2, Name: "Wallet A", Path: "/w/a"},
	}
	colEntities := map[string]InteropMatrixEntity{
		i1: {ID: i1, Name: "Issuer One", Path: "/issuers/one"},
	}
	inputs := []interopCacheInput{
		{PipelineID: p1, TotalRuns: 92, TotalSuccesses: 78, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
		{PipelineID: p2, TotalRuns: 92, TotalSuccesses: 78, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: 60, TotalSuccesses: 53, RowIDs: []string{w2}, ColumnIDs: []string{i1}},
	}

	got := buildInteropMatrix(inputs, rowEntities, colEntities)

	require.Equal(t, interopModeWalletsIssuers, got.Mode)
	require.Equal(t, "wallet", got.RowAxis)
	require.Equal(t, "issuer", got.ColumnAxis)
	require.Len(t, got.Cells, 2)

	require.Len(t, got.Rows, 2)
	require.Equal(t, w2, got.Rows[0].ID)
	require.Equal(t, w1, got.Rows[1].ID)

	require.Len(t, got.Columns, 1)
	require.Equal(t, i1, got.Columns[0].ID)

	cellW1I1, ok := findInteropCell(got, w1, i1)
	require.True(t, ok)
	require.Equal(t, 2, cellW1I1.PipelineCount)
	require.Equal(t, 184, cellW1I1.TotalRuns)
	require.Equal(t, 156, cellW1I1.TotalSuccesses)
	require.Equal(t, interopStatusFlaky, cellW1I1.Status)
	expectedRate := 156.0 / 184.0 * 100
	require.InDelta(t, expectedRate, cellW1I1.SuccessRate, 1e-9)

	cellW2I1, ok := findInteropCell(got, w2, i1)
	require.True(t, ok)
	require.Equal(t, 1, cellW2I1.PipelineCount)
	require.Equal(t, 60, cellW2I1.TotalRuns)
	require.Equal(t, 53, cellW2I1.TotalSuccesses)
	require.Equal(t, interopStatusFlaky, cellW2I1.Status)
	require.InDelta(t, 53.0/60.0*100, cellW2I1.SuccessRate, 1e-9)
}

func TestBuildInteropMatrix_SkipsEmptySides(t *testing.T) {
	t.Parallel()

	const (
		w1 = "wallet1"
		i1 = "issuer1"
		p1 = "pipeline_one"
	)
	rowEntities := map[string]InteropMatrixEntity{w1: {ID: w1, Name: "Wal", Path: "/w"}}
	colEntities := map[string]InteropMatrixEntity{i1: {ID: i1, Name: "Iss", Path: "/i"}}

	skippedRuns := math.MaxInt
	inputs := []interopCacheInput{
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: nil, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: []string{}, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: []string{w1}, ColumnIDs: nil},
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: []string{w1}, ColumnIDs: []string{}},
		{PipelineID: p1, TotalRuns: 0, TotalSuccesses: 0, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: 10, TotalSuccesses: 9, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
	}

	got := buildInteropMatrix(inputs, rowEntities, colEntities)

	require.Len(t, got.Cells, 1)
	c, ok := findInteropCell(got, w1, i1)
	require.True(t, ok)
	require.Equal(t, 1, c.PipelineCount)
	require.Equal(t, 10, c.TotalRuns)
	require.Equal(t, 9, c.TotalSuccesses)
}

func TestInteropStatusFromRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rate float64
		want interopStatus
	}{
		{rate: 90, want: interopStatusStable},
		{rate: 89.9, want: interopStatusFlaky},
		{rate: 70, want: interopStatusFlaky},
		{rate: 69.9, want: interopStatusFailing},
		{rate: 50, want: interopStatusFailing},
		{rate: 49.9, want: interopStatusBroken},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%g", tt.rate), func(t *testing.T) {
			t.Parallel()
			got := interopStatusFromRate(tt.rate)
			require.Equal(t, tt.want, got)
		})
	}
}
