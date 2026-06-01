// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

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

func resolveAxisCoords(app core.App, record *core.Record, axis interopAxis) ([]interopAxisCoord, error) {
	seen := map[interopAxisCoord]struct{}{}
	add := func(tier interopTier, key string) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		seen[interopAxisCoord{Tier: tier, Key: key}] = struct{}{}
	}

	if !axis.Tiered() {
		if axis.PathBased {
			var rawIDs []string
			if err := record.UnmarshalJSONField(axis.CacheField, &rawIDs); err != nil {
				return nil, err
			}
			for _, id := range rawIDs {
				add(interopTierLeaf, id)
			}
		} else {
			for _, id := range record.GetStringSlice(axis.CacheField) {
				add(interopTierLeaf, id)
			}
		}
		return interopAxisCoordsFromSet(seen), nil
	}

	if axis.PathBased {
		var paths []string
		if err := record.UnmarshalJSONField(axis.Tier.LeafCacheField, &paths); err != nil {
			return nil, err
		}
		for _, path := range paths {
			if strings.TrimSpace(path) == "" {
				continue
			}
			group, _, err := interopSuiteGroupFromPath(path)
			if err != nil {
				add(interopTierLeaf, path)
				continue
			}
			add(interopTierGroup, group.ID)
			add(interopTierLeaf, path)
		}
		return interopAxisCoordsFromSet(seen), nil
	}

	leafIDs := record.GetStringSlice(axis.Tier.LeafCacheField)
	leafIDSet := map[string]struct{}{}
	for _, leafID := range leafIDs {
		if leafID = strings.TrimSpace(leafID); leafID != "" {
			leafIDSet[leafID] = struct{}{}
		}
	}

	leafRecords, err := findRecordsByIDs(app, axis.Tier.LeafCollection, interopUniqueIDs(leafIDSet))
	if err != nil {
		return nil, err
	}

	parentsWithLeaf := map[string]struct{}{}
	for leafID := range leafIDSet {
		add(interopTierLeaf, leafID)
		leaf := leafRecords[leafID]
		if leaf == nil {
			continue
		}
		parentID := strings.TrimSpace(leaf.GetString(axis.Tier.LeafParentField))
		if parentID == "" {
			continue
		}
		add(interopTierGroup, parentID)
		parentsWithLeaf[parentID] = struct{}{}
	}

	for _, parentID := range record.GetStringSlice(axis.CacheField) {
		parentID = strings.TrimSpace(parentID)
		if parentID == "" {
			continue
		}
		add(interopTierGroup, parentID)
		if _, ok := parentsWithLeaf[parentID]; !ok {
			add(interopTierLeaf, fmt.Sprintf("%s::%s", parentID, axis.Tier.NoLeafSentinel))
		}
	}

	return interopAxisCoordsFromSet(seen), nil
}

func interopAxisCoordsFromSet(seen map[interopAxisCoord]struct{}) []interopAxisCoord {
	out := make([]interopAxisCoord, 0, len(seen))
	for coord := range seen {
		out = append(out, coord)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tier != out[j].Tier {
			return out[i].Tier < out[j].Tier
		}
		return out[i].Key < out[j].Key
	})
	return out
}
