// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

type interopTier string

const (
	interopTierGroup interopTier = "group"
	interopTierLeaf  interopTier = "leaf"
)

type interopAxisCoord struct {
	Tier interopTier
	Key  string
}

type interopTieredCacheInput struct {
	PipelineID     string
	TotalRuns      int
	TotalSuccesses int
	RowCoords      []interopAxisCoord
	ColumnCoords   []interopAxisCoord
}

type interopTieredMatrixCellKey struct {
	rowTier interopTier
	rowKey  string
	colTier interopTier
	colKey  string
}

func aggregateInteropCells(inputs []interopTieredCacheInput) map[interopTieredMatrixCellKey]*interopCellAccumulator {
	out := map[interopTieredMatrixCellKey]*interopCellAccumulator{}

	for _, in := range inputs {
		if in.TotalRuns <= 0 {
			continue
		}

		seen := map[interopTieredMatrixCellKey]struct{}{}
		for _, r := range in.RowCoords {
			for _, c := range in.ColumnCoords {
				key := interopTieredMatrixCellKey{
					rowTier: r.Tier,
					rowKey:  r.Key,
					colTier: c.Tier,
					colKey:  c.Key,
				}
				if r.Tier == interopTierGroup || c.Tier == interopTierGroup {
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
				}

				acc := out[key]
				if acc == nil {
					acc = &interopCellAccumulator{pipelineIDs: map[string]struct{}{}}
					out[key] = acc
				}
				if in.PipelineID != "" {
					acc.pipelineIDs[in.PipelineID] = struct{}{}
				}
				acc.totalRuns += in.TotalRuns
				acc.totalSuccesses += in.TotalSuccesses
			}
		}
	}

	return out
}
