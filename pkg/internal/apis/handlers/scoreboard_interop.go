// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type interopStatus string

const (
	interopStatusStable  interopStatus = "stable"
	interopStatusFlaky   interopStatus = "flaky"
	interopStatusFailing interopStatus = "failing"
	interopStatusBroken  interopStatus = "broken"
)

type interopCellAccumulator struct {
	pipelineIDs    map[string]struct{}
	totalRuns      int
	totalSuccesses int
}

type InteropMatrixEntity struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Subtitle     *string `json:"subtitle,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	Path         string  `json:"path"`
	VersionLabel *string `json:"version_label,omitempty"`
}

type InteropMatrixTier string

const (
	InteropMatrixTierGroup InteropMatrixTier = "group"
	InteropMatrixTierLeaf  InteropMatrixTier = "leaf"
)

type InteropMatrixGroup struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Path       string  `json:"path"`
	ChildCount int     `json:"child_count"`
	AvatarURL  *string `json:"avatar_url,omitempty"`
	Subtitle   *string `json:"subtitle,omitempty"`
}

type InteropMatrixLeaf struct {
	InteropMatrixEntity
	ParentID string `json:"parent_id,omitempty"`
}

type InteropMatrixCell struct {
	RowID          string            `json:"row_id"`
	ColumnID       string            `json:"column_id"`
	RowTier        InteropMatrixTier `json:"row_tier"`
	ColumnTier     InteropMatrixTier `json:"column_tier"`
	PipelineCount  int               `json:"pipeline_count"`
	TotalRuns      int               `json:"total_runs"`
	TotalSuccesses int               `json:"total_successes"`
	SuccessRate    float64           `json:"success_rate"`
	Status         interopStatus     `json:"status"`
}

// InteropAxis bundles an axis with the facts a caller needs to render it:
// the hub URL segment for its entities (e.g. "wallets", "conformance-checks"),
// whether IDs are conformance check paths from a JSON field rather than
// PocketBase relation record IDs, and whether the axis supports group/leaf tiers.
type InteropAxis struct {
	HubCollection string `json:"hub_collection"`
	PathBased     bool   `json:"path_based"`
	Tiered        bool   `json:"tiered"`
}

type InteropMatrixResponse struct {
	Row          InteropAxis          `json:"row"`
	Column       InteropAxis          `json:"column"`
	RowGroups    []InteropMatrixGroup `json:"row_groups"`
	RowLeaves    []InteropMatrixLeaf  `json:"row_leaves"`
	ColumnGroups []InteropMatrixGroup `json:"column_groups"`
	ColumnLeaves []InteropMatrixLeaf  `json:"column_leaves"`
	Cells        []InteropMatrixCell  `json:"cells"`
}

func interopStatusFromRate(rate float64) interopStatus {
	switch {
	case rate >= 90:
		return interopStatusStable
	case rate >= 70:
		return interopStatusFlaky
	case rate >= 50:
		return interopStatusFailing
	default:
		return interopStatusBroken
	}
}

const interopRecordIDFilterChunkSize = 50

type interopAxisKeys struct {
	groups           map[string]struct{}
	leaves           map[string]struct{}
	suiteGroupTitles map[string]string
}

func newInteropAxisKeys() interopAxisKeys {
	return interopAxisKeys{
		groups:           map[string]struct{}{},
		leaves:           map[string]struct{}{},
		suiteGroupTitles: map[string]string{},
	}
}

func collectInteropAxisKeys(keys *interopAxisKeys, coords []interopAxisCoord, axis interopAxis) {
	for _, coord := range coords {
		switch coord.Tier {
		case interopTierGroup:
			keys.groups[coord.Key] = struct{}{}
			if axis.PathBased {
				keys.suiteGroupTitles[coord.Key] = interopSuiteGroupTitleFromID(coord.Key)
			}
		case interopTierLeaf:
			keys.leaves[coord.Key] = struct{}{}
			if axis.PathBased {
				group, _, err := interopSuiteGroupFromPath(coord.Key)
				if err == nil {
					keys.groups[group.ID] = struct{}{}
					keys.suiteGroupTitles[group.ID] = group.Title
				}
			}
		}
	}
}

func interopSuiteGroupTitleFromID(groupID string) string {
	parts := strings.Split(groupID, "/")
	if len(parts) != 3 {
		return groupID
	}
	return fmt.Sprintf("%s • %s • %s", parts[0], parts[1], parts[2])
}

func interopAxisDTO(axis interopAxis) InteropAxis {
	return InteropAxis{
		HubCollection: axis.HubCollection,
		PathBased:     axis.PathBased,
		Tiered:        axis.Tiered(),
	}
}

func interopMatrixTierString(t interopTier) InteropMatrixTier {
	return InteropMatrixTier(t)
}

func buildInteropTieredMatrix(
	rowAxis InteropAxis,
	colAxis InteropAxis,
	rowGroups []InteropMatrixGroup,
	rowLeaves []InteropMatrixLeaf,
	columnGroups []InteropMatrixGroup,
	columnLeaves []InteropMatrixLeaf,
	cellsMap map[interopTieredMatrixCellKey]*interopCellAccumulator,
) InteropMatrixResponse {
	cells := make([]InteropMatrixCell, 0, len(cellsMap))
	for k, acc := range cellsMap {
		if acc.totalRuns <= 0 {
			continue
		}
		rate := float64(acc.totalSuccesses) / float64(acc.totalRuns) * 100
		cells = append(cells, InteropMatrixCell{
			RowID:          k.rowKey,
			ColumnID:       k.colKey,
			RowTier:        interopMatrixTierString(k.rowTier),
			ColumnTier:     interopMatrixTierString(k.colTier),
			PipelineCount:  len(acc.pipelineIDs),
			TotalRuns:      acc.totalRuns,
			TotalSuccesses: acc.totalSuccesses,
			SuccessRate:    rate,
			Status:         interopStatusFromRate(rate),
		})
	}

	return InteropMatrixResponse{
		Row:          rowAxis,
		Column:       colAxis,
		RowGroups:    rowGroups,
		RowLeaves:    rowLeaves,
		ColumnGroups: columnGroups,
		ColumnLeaves: columnLeaves,
		Cells:        cells,
	}
}

func sortedInteropMatrixGroups(groups []InteropMatrixGroup) []InteropMatrixGroup {
	sort.Slice(groups, func(i, j int) bool {
		ni := strings.ToLower(strings.TrimSpace(groups[i].Name))
		nj := strings.ToLower(strings.TrimSpace(groups[j].Name))
		if ni != nj {
			return ni < nj
		}
		return groups[i].ID < groups[j].ID
	})
	return groups
}

func sortedInteropMatrixLeaves(leaves []InteropMatrixLeaf) []InteropMatrixLeaf {
	sort.Slice(leaves, func(i, j int) bool {
		ni := strings.ToLower(strings.TrimSpace(leaves[i].Name))
		nj := strings.ToLower(strings.TrimSpace(leaves[j].Name))
		if ni != nj {
			return ni < nj
		}
		return leaves[i].ID < leaves[j].ID
	})
	return leaves
}

func interopChildCountByParent(leaves []InteropMatrixLeaf) map[string]int {
	counts := map[string]int{}
	for _, leaf := range leaves {
		parentID := strings.TrimSpace(leaf.ParentID)
		if parentID == "" {
			continue
		}
		counts[parentID]++
	}
	return counts
}

var ScoreboardInteropPublicRoutes = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/scoreboard/interop",
			OperationID:    "scoreboard.interop",
			Handler:        HandleInteropMatrix,
			ResponseSchema: InteropMatrixResponse{},
			Summary:     "Interoperability matrix",
			Description: "Interoperability matrix from pipeline_scoreboard_cache. " + interopHubsUsageHint(),
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "row",
					Required:    true,
					Description: "Row hub collection (e.g. wallets).",
				},
				{
					Name:        "column",
					Required:    true,
					Description: "Column hub collection (e.g. credentials).",
				},
			},
		},
	},
}

func HandleInteropMatrix() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		q := e.Request.URL.Query()
		if q.Has("mode") {
			return apierror.New(
				http.StatusBadRequest,
				"mode",
				"mode query param is no longer supported",
				interopHubsUsageHint(),
			).JSON(e)
		}
		rowHub := q.Get("row")
		colHub := q.Get("column")
		if rowHub == "" || colHub == "" {
			return apierror.New(
				http.StatusBadRequest,
				"row",
				"missing row or column hub collection",
				interopHubsUsageHint(),
			).JSON(e)
		}
		if rowHub == colHub {
			return apierror.New(
				http.StatusBadRequest,
				"row",
				"row and column must differ",
				interopHubsUsageHint(),
			).JSON(e)
		}
		rowAxis, ok := getInteropAxis(rowHub)
		if !ok {
			return apierror.New(
				http.StatusBadRequest,
				"row",
				"unknown row hub collection",
				interopHubsUsageHint(),
			).JSON(e)
		}
		colAxis, ok := getInteropAxis(colHub)
		if !ok {
			return apierror.New(
				http.StatusBadRequest,
				"column",
				"unknown column hub collection",
				interopHubsUsageHint(),
			).JSON(e)
		}

		resp, err := loadInteropMatrixFromCache(e.App, rowAxis, colAxis)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"scoreboard",
				"failed to build interop matrix",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, resp)
	}
}

func loadInteropMatrixFromCache(
	app core.App,
	rowAxis interopAxis,
	colAxis interopAxis,
) (InteropMatrixResponse, error) {
	collection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	if err != nil {
		return InteropMatrixResponse{}, fmt.Errorf("find collection: %w", err)
	}

	records, err := app.FindRecordsByFilter(collection.Id, "", "", -1, 0)
	if err != nil {
		return InteropMatrixResponse{}, fmt.Errorf("list cache: %w", err)
	}

	inputs := make([]interopTieredCacheInput, 0, len(records))
	rowKeys := newInteropAxisKeys()
	colKeys := newInteropAxisKeys()

	for _, record := range records {
		rowCoords, err := resolveAxisCoords(app, record, rowAxis)
		if err != nil {
			return InteropMatrixResponse{}, fmt.Errorf("resolve row coords: %w", err)
		}
		colCoords, err := resolveAxisCoords(app, record, colAxis)
		if err != nil {
			return InteropMatrixResponse{}, fmt.Errorf("resolve column coords: %w", err)
		}

		if len(rowCoords) == 0 || len(colCoords) == 0 {
			continue
		}

		collectInteropAxisKeys(&rowKeys, rowCoords, rowAxis)
		collectInteropAxisKeys(&colKeys, colCoords, colAxis)

		inputs = append(inputs, interopTieredCacheInput{
			PipelineID:     record.GetString("pipeline"),
			TotalRuns:      record.GetInt("total_runs"),
			TotalSuccesses: record.GetInt("total_successes"),
			RowCoords:      rowCoords,
			ColumnCoords:   colCoords,
		})
	}

	cellsMap := aggregateInteropCells(inputs)

	rowGroups, rowLeaves, err := loadInteropAxisPresentation(app, rowAxis, rowKeys, records)
	if err != nil {
		return InteropMatrixResponse{}, fmt.Errorf("load row axis: %w", err)
	}
	columnGroups, columnLeaves, err := loadInteropAxisPresentation(app, colAxis, colKeys, records)
	if err != nil {
		return InteropMatrixResponse{}, fmt.Errorf("load column axis: %w", err)
	}

	return buildInteropTieredMatrix(
		interopAxisDTO(rowAxis),
		interopAxisDTO(colAxis),
		rowGroups,
		rowLeaves,
		columnGroups,
		columnLeaves,
		cellsMap,
	), nil
}

func loadInteropAxisPresentation(
	app core.App,
	axis interopAxis,
	keys interopAxisKeys,
	cacheRecords []*core.Record,
) ([]InteropMatrixGroup, []InteropMatrixLeaf, error) {
	leaves, err := loadInteropLeaves(app, axis, keys.leaves, cacheRecords)
	if err != nil {
		return nil, nil, err
	}

	childCounts := interopChildCountByParent(leaves)

	var groups []InteropMatrixGroup
	if axis.Tiered() {
		groups, err = loadInteropGroups(app, axis, keys, childCounts)
		if err != nil {
			return nil, nil, err
		}
	}

	return sortedInteropMatrixGroups(groups), sortedInteropMatrixLeaves(leaves), nil
}

func loadInteropGroups(
	app core.App,
	axis interopAxis,
	keys interopAxisKeys,
	childCounts map[string]int,
) ([]InteropMatrixGroup, error) {
	groups := make([]InteropMatrixGroup, 0, len(keys.groups))
	if len(keys.groups) == 0 {
		return groups, nil
	}

	if axis.PathBased {
		for groupID := range keys.groups {
			title := keys.suiteGroupTitles[groupID]
			if title == "" {
				title = interopSuiteGroupTitleFromID(groupID)
			}
			groups = append(groups, InteropMatrixGroup{
				ID:         groupID,
				Name:       title,
				Path:       groupID,
				ChildCount: childCounts[groupID],
			})
		}
		return groups, nil
	}

	if axis.buildEntity == nil {
		return groups, nil
	}

	recordsByID, err := findRecordsByIDs(app, axis.Tier.GroupCollection, interopUniqueIDs(keys.groups))
	if err != nil {
		return nil, err
	}

	for groupID := range keys.groups {
		record := recordsByID[groupID]
		if record == nil {
			continue
		}
		entity, err := axis.buildEntity(app, record, nil)
		if err != nil {
			return nil, err
		}
		groups = append(groups, InteropMatrixGroup{
			ID:         entity.ID,
			Name:       entity.Name,
			Path:       entity.Path,
			ChildCount: childCounts[groupID],
			AvatarURL:  entity.AvatarURL,
			Subtitle:   entity.Subtitle,
		})
	}

	return groups, nil
}

func loadInteropLeaves(
	app core.App,
	axis interopAxis,
	leafIDs map[string]struct{},
	cacheRecords []*core.Record,
) ([]InteropMatrixLeaf, error) {
	leaves := make([]InteropMatrixLeaf, 0, len(leafIDs))
	if len(leafIDs) == 0 {
		return leaves, nil
	}

	cacheRecordByLeafID := interopFirstCacheRecordByLeafID(cacheRecords, axis)

	for leafID := range leafIDs {
		leaf, err := buildInteropLeaf(app, axis, leafID, cacheRecordByLeafID[leafID])
		if err != nil {
			return nil, err
		}
		if leaf.ID == "" {
			continue
		}
		leaves = append(leaves, leaf)
	}

	return leaves, nil
}

func buildInteropLeaf(
	app core.App,
	axis interopAxis,
	leafID string,
	cacheRecord *core.Record,
) (InteropMatrixLeaf, error) {
	if axis.PathBased {
		entity := InteropMatrixEntity{
			ID:   leafID,
			Name: conformanceCheckName(leafID),
			Path: leafID,
		}
		parentID := ""
		if group, _, err := interopSuiteGroupFromPath(leafID); err == nil {
			parentID = group.ID
		}
		return InteropMatrixLeaf{InteropMatrixEntity: entity, ParentID: parentID}, nil
	}

	if !axis.Tiered() {
		if axis.buildEntity == nil {
			return InteropMatrixLeaf{}, nil
		}
		record, err := findRecordsByIDs(app, axis.HubCollection, []string{leafID})
		if err != nil {
			return InteropMatrixLeaf{}, err
		}
		rec := record[leafID]
		if rec == nil {
			return InteropMatrixLeaf{}, nil
		}
		entity, err := axis.buildEntity(app, rec, cacheRecord)
		if err != nil {
			return InteropMatrixLeaf{}, err
		}
		return InteropMatrixLeaf{InteropMatrixEntity: entity}, nil
	}

	return buildInteropTieredPBLeaf(app, axis, leafID, cacheRecord)
}

func buildInteropTieredPBLeaf(
	app core.App,
	axis interopAxis,
	leafID string,
	cacheRecord *core.Record,
) (InteropMatrixLeaf, error) {
	sentinel := "::" + axis.Tier.NoLeafSentinel
	if strings.Contains(leafID, sentinel) {
		parentID := strings.TrimSuffix(leafID, sentinel)
		walletRecords, err := findRecordsByIDs(app, axis.Tier.GroupCollection, []string{parentID})
		if err != nil {
			return InteropMatrixLeaf{}, err
		}
		wallet := walletRecords[parentID]
		if wallet == nil || axis.buildEntity == nil {
			return InteropMatrixLeaf{}, nil
		}
		entity, err := axis.buildEntity(app, wallet, cacheRecord)
		if err != nil {
			return InteropMatrixLeaf{}, err
		}
		entity.ID = leafID
		entity.VersionLabel = nil
		return InteropMatrixLeaf{InteropMatrixEntity: entity, ParentID: parentID}, nil
	}

	leafRecords, err := findRecordsByIDs(app, axis.Tier.LeafCollection, []string{leafID})
	if err != nil {
		return InteropMatrixLeaf{}, err
	}
	leafRecord := leafRecords[leafID]
	if leafRecord == nil {
		return InteropMatrixLeaf{}, nil
	}

	parentID := strings.TrimSpace(leafRecord.GetString(axis.Tier.LeafParentField))

	switch axis.HubCollection {
	case "wallets":
		walletRecords, err := findRecordsByIDs(app, "wallets", []string{parentID})
		if err != nil {
			return InteropMatrixLeaf{}, err
		}
		wallet := walletRecords[parentID]
		if wallet == nil || axis.buildEntity == nil {
			return InteropMatrixLeaf{}, nil
		}
		entity, err := axis.buildEntity(app, wallet, cacheRecord)
		if err != nil {
			return InteropMatrixLeaf{}, err
		}
		entity.ID = leafID
		if entity.VersionLabel == nil {
			tag := strings.TrimSpace(leafRecord.GetString("tag"))
			if tag != "" {
				label := tag
				if !strings.HasPrefix(label, "v") {
					label = "v" + label
				}
				entity.VersionLabel = &label
			}
		}
		return InteropMatrixLeaf{InteropMatrixEntity: entity, ParentID: parentID}, nil
	default:
		buildEntity := axis.buildEntity
		switch axis.HubCollection {
		case "credential_issuers":
			if credAxis, ok := getInteropAxis("credentials"); ok {
				buildEntity = credAxis.buildEntity
			}
		case "verifiers":
			if ucAxis, ok := getInteropAxis("use_cases_verifications"); ok {
				buildEntity = ucAxis.buildEntity
			}
		}
		if buildEntity == nil {
			return InteropMatrixLeaf{}, nil
		}
		entity, err := buildEntity(app, leafRecord, cacheRecord)
		if err != nil {
			return InteropMatrixLeaf{}, err
		}
		return InteropMatrixLeaf{InteropMatrixEntity: entity, ParentID: parentID}, nil
	}
}

func interopFirstCacheRecordByLeafID(
	cacheRecords []*core.Record,
	axis interopAxis,
) map[string]*core.Record {
	if len(cacheRecords) == 0 {
		return nil
	}

	out := map[string]*core.Record{}
	assign := func(leafID string, cacheRecord *core.Record) {
		if leafID == "" {
			return
		}
		if _, ok := out[leafID]; ok {
			return
		}
		out[leafID] = cacheRecord
	}

	for _, cacheRecord := range cacheRecords {
		if !axis.Tiered() {
			for _, id := range cacheRecord.GetStringSlice(axis.CacheField) {
				assign(id, cacheRecord)
			}
			continue
		}

		if axis.PathBased {
			var paths []string
			if err := cacheRecord.UnmarshalJSONField(axis.Tier.LeafCacheField, &paths); err != nil {
				continue
			}
			for _, path := range paths {
				assign(strings.TrimSpace(path), cacheRecord)
			}
			continue
		}

		for _, leafID := range cacheRecord.GetStringSlice(axis.Tier.LeafCacheField) {
			assign(strings.TrimSpace(leafID), cacheRecord)
		}
		for _, parentID := range cacheRecord.GetStringSlice(axis.CacheField) {
			parentID = strings.TrimSpace(parentID)
			if parentID == "" {
				continue
			}
			assign(fmt.Sprintf("%s::%s", parentID, axis.Tier.NoLeafSentinel), cacheRecord)
		}
	}

	return out
}

func interopUniqueIDs(seen map[string]struct{}) []string {
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func buildRecordIDsFilter(ids []string) (string, dbx.Params) {
	clauses := make([]string, 0, len(ids))
	params := dbx.Params{}
	for idx, id := range ids {
		if id == "" {
			continue
		}
		paramKey := fmt.Sprintf("id_%d", idx)
		params[paramKey] = id
		clauses = append(clauses, fmt.Sprintf("id = {:%s}", paramKey))
	}
	return strings.Join(clauses, " || "), params
}

func findRecordsByIDs(
	app core.App,
	collectionName string,
	ids []string,
) (map[string]*core.Record, error) {
	recordsByID := make(map[string]*core.Record, len(ids))
	if len(ids) == 0 {
		return recordsByID, nil
	}

	for start := 0; start < len(ids); start += interopRecordIDFilterChunkSize {
		end := start + interopRecordIDFilterChunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		filter, params := buildRecordIDsFilter(chunk)
		if filter == "" {
			continue
		}

		records, err := app.FindRecordsByFilter(
			collectionName,
			filter,
			"",
			-1,
			0,
			params,
		)
		if err != nil {
			return nil, fmt.Errorf("find %s records: %w", collectionName, err)
		}
		for _, record := range records {
			recordsByID[record.Id] = record
		}
	}

	return recordsByID, nil
}

func buildEnrichedEntityMetadata(
	id string,
	name string,
	path string,
	entityAvatarURL *string,
	subtitle *string,
	fallbackAvatarURL *string,
) InteropMatrixEntity {
	avatar := entityAvatarURL
	if avatar == nil {
		avatar = fallbackAvatarURL
	}

	return InteropMatrixEntity{
		ID:        id,
		Name:      name,
		Subtitle:  subtitle,
		AvatarURL: avatar,
		Path:      path,
	}
}

func optionalTrimmedStringPtr(raw string) *string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return &value
}

func firstNonEmptyStringPtr(values ...string) *string {
	for _, raw := range values {
		if value := optionalTrimmedStringPtr(raw); value != nil {
			return value
		}
	}
	return nil
}

func conformanceCheckName(pathID string) string {
	parts := strings.Split(pathID, "/")
	last := parts[len(parts)-1]
	ext := strings.LastIndex(last, ".")
	if ext >= 0 {
		last = last[:ext]
	}
	name := strings.NewReplacer("-", " ", "_", " ").Replace(last)
	name = strings.TrimSpace(name)
	if name == "" {
		return pathID
	}
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func walletVersionLabelFromCacheRecord(
	cacheRecord *core.Record,
	walletID string,
	versionsByID map[string]*core.Record,
) *string {
	for _, versionID := range cacheRecord.GetStringSlice("wallet_versions") {
		version := versionsByID[versionID]
		if version == nil {
			continue
		}
		if version.GetString("wallet") != walletID {
			continue
		}
		tag := strings.TrimSpace(version.GetString("tag"))
		if tag == "" {
			continue
		}
		label := tag
		if !strings.HasPrefix(label, "v") {
			label = "v" + label
		}
		return &label
	}
	return nil
}
