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

type interopCacheInput struct {
	PipelineID     string
	TotalRuns      int
	TotalSuccesses int
	RowIDs         []string // row axis entity IDs from cache scan
	ColumnIDs      []string // column axis entity IDs from cache scan
}

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

type InteropMatrixCell struct {
	RowID          string        `json:"row_id"`
	ColumnID       string        `json:"column_id"`
	PipelineCount  int           `json:"pipeline_count"`
	TotalRuns      int           `json:"total_runs"`
	TotalSuccesses int           `json:"total_successes"`
	SuccessRate    float64       `json:"success_rate"`
	Status         interopStatus `json:"status"`
}

// InteropAxis bundles an axis with the facts a caller needs to render it:
// the hub URL segment for its entities (e.g. "wallets", "conformance-checks"),
// and whether IDs are conformance check paths from a JSON field rather than
// PocketBase relation record IDs.
type InteropAxis struct {
	HubCollection string `json:"hub_collection"`
	PathBased     bool   `json:"path_based"`
}

type InteropMatrixResponse struct {
	Row     InteropAxis           `json:"row"`
	Column  InteropAxis           `json:"column"`
	Rows    []InteropMatrixEntity `json:"rows"`
	Columns []InteropMatrixEntity `json:"columns"`
	Cells   []InteropMatrixCell   `json:"cells"`
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

type interopMatrixCellKey struct {
	row string
	col string
}

const interopRecordIDFilterChunkSize = 50

type interopRelatedRecords struct {
	byCollection map[string]map[string]*core.Record
}

func (r interopRelatedRecords) record(collection, id string) *core.Record {
	if r.byCollection == nil {
		return nil
	}
	records, ok := r.byCollection[collection]
	if !ok {
		return nil
	}
	return records[id]
}

type interopCacheScan struct {
	inputs              []interopCacheInput
	rowIDs              map[string]struct{}
	columnIDs           map[string]struct{}
	rowEntities         map[string]InteropMatrixEntity
	columnEntities      map[string]InteropMatrixEntity
	walletVersionLabels map[string]*string
}

func sortedInteropEntities(all map[string]InteropMatrixEntity, seen map[string]struct{}) []InteropMatrixEntity {
	out := make([]InteropMatrixEntity, 0, len(seen))
	for id := range seen {
		e, ok := all[id]
		if !ok {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		ni := strings.ToLower(strings.TrimSpace(out[i].Name))
		nj := strings.ToLower(strings.TrimSpace(out[j].Name))
		if ni != nj {
			return ni < nj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func buildInteropMatrix(
	inputs []interopCacheInput,
	rowAxis InteropAxis,
	colAxis InteropAxis,
	rowEntities map[string]InteropMatrixEntity,
	columnEntities map[string]InteropMatrixEntity,
) InteropMatrixResponse {
	cellsMap := map[interopMatrixCellKey]*interopCellAccumulator{}
	rowSeen := map[string]struct{}{}
	colSeen := map[string]struct{}{}

	for _, in := range inputs {
		if len(in.RowIDs) == 0 || len(in.ColumnIDs) == 0 || in.TotalRuns <= 0 {
			continue
		}
		for _, rowID := range in.RowIDs {
			for _, colID := range in.ColumnIDs {
				key := interopMatrixCellKey{row: rowID, col: colID}
				acc := cellsMap[key]
				if acc == nil {
					acc = &interopCellAccumulator{
						pipelineIDs: make(map[string]struct{}),
					}
					cellsMap[key] = acc
				}
				if in.PipelineID != "" {
					acc.pipelineIDs[in.PipelineID] = struct{}{}
				}
				acc.totalRuns += in.TotalRuns
				acc.totalSuccesses += in.TotalSuccesses
				rowSeen[rowID] = struct{}{}
				colSeen[colID] = struct{}{}
			}
		}
	}

	cells := make([]InteropMatrixCell, 0, len(cellsMap))
	for k, acc := range cellsMap {
		if acc.totalRuns <= 0 {
			continue
		}
		rate := float64(acc.totalSuccesses) / float64(acc.totalRuns) * 100
		cells = append(cells, InteropMatrixCell{
			RowID:          k.row,
			ColumnID:       k.col,
			PipelineCount:  len(acc.pipelineIDs),
			TotalRuns:      acc.totalRuns,
			TotalSuccesses: acc.totalSuccesses,
			SuccessRate:    rate,
			Status:         interopStatusFromRate(rate),
		})
	}

	return InteropMatrixResponse{
		Row:     rowAxis,
		Column:  colAxis,
		Rows:    sortedInteropEntities(rowEntities, rowSeen),
		Columns: sortedInteropEntities(columnEntities, colSeen),
		Cells:   cells,
	}
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

	scan := scanInteropCacheRecords(records, rowAxis, colAxis)

	if rowAxis.HubCollection == "wallets" {
		walletVersionsByID, err := loadWalletVersionsForCacheRecords(app, records)
		if err != nil {
			return InteropMatrixResponse{}, err
		}
		resolveWalletVersionLabels(scan, records, rowAxis, walletVersionsByID)
	}

	var rowEntities map[string]InteropMatrixEntity
	if rowAxis.PathBased {
		rowEntities = scan.rowEntities
	} else {
		var err error
		rowEntities, err = loadInteropEntitiesByIDs(
			app,
			rowAxis.HubCollection,
			scan.rowIDs,
			scan.walletVersionLabels,
		)
		if err != nil {
			return InteropMatrixResponse{}, err
		}
		for id, entity := range scan.rowEntities {
			rowEntities[id] = entity
		}
	}

	columnEntities := scan.columnEntities
	if !colAxis.PathBased {
		var err error
		columnEntities, err = loadInteropEntitiesByIDs(
			app,
			colAxis.HubCollection,
			scan.columnIDs,
			nil,
		)
		if err != nil {
			return InteropMatrixResponse{}, err
		}
	}

	return buildInteropMatrix(
		scan.inputs,
		InteropAxis{HubCollection: rowAxis.HubCollection, PathBased: rowAxis.PathBased},
		InteropAxis{HubCollection: colAxis.HubCollection, PathBased: colAxis.PathBased},
		rowEntities,
		columnEntities,
	), nil
}

func readAxisIDs(record *core.Record, axis interopAxis) ([]string, map[string]InteropMatrixEntity) {
	inline := map[string]InteropMatrixEntity{}
	if !axis.PathBased {
		ids := record.GetStringSlice(axis.CacheField)
		return ids, inline
	}
	var rawIDs []string
	if err := record.UnmarshalJSONField(axis.CacheField, &rawIDs); err != nil {
		return nil, inline
	}
	out := make([]string, 0, len(rawIDs))
	for _, id := range rawIDs {
		if id == "" {
			continue
		}
		out = append(out, id)
		if _, ok := inline[id]; ok {
			continue
		}
		inline[id] = InteropMatrixEntity{
			ID:   id,
			Name: conformanceCheckName(id),
			Path: id,
		}
	}
	return out, inline
}

func scanInteropCacheRecords(
	records []*core.Record,
	rowAxis interopAxis,
	colAxis interopAxis,
) interopCacheScan {
	scan := interopCacheScan{
		rowIDs:              map[string]struct{}{},
		columnIDs:           map[string]struct{}{},
		rowEntities:         map[string]InteropMatrixEntity{},
		columnEntities:      map[string]InteropMatrixEntity{},
		walletVersionLabels: map[string]*string{},
	}
	for _, record := range records {
		rowIDs, rowInline := readAxisIDs(record, rowAxis)
		colIDs, colInline := readAxisIDs(record, colAxis)
		for id, entity := range rowInline {
			scan.rowEntities[id] = entity
		}
		for id, entity := range colInline {
			scan.columnEntities[id] = entity
		}
		scan.inputs = append(scan.inputs, interopCacheInput{
			PipelineID:     record.GetString("pipeline"),
			TotalRuns:      record.GetInt("total_runs"),
			TotalSuccesses: record.GetInt("total_successes"),
			RowIDs:         rowIDs,
			ColumnIDs:      colIDs,
		})
		for _, rowID := range rowIDs {
			if !rowAxis.PathBased {
				scan.rowIDs[rowID] = struct{}{}
			}
		}
		for _, colID := range colIDs {
			if !colAxis.PathBased {
				scan.columnIDs[colID] = struct{}{}
			}
		}
	}
	return scan
}

func loadWalletVersionsForCacheRecords(
	app core.App,
	records []*core.Record,
) (map[string]*core.Record, error) {
	versionIDs := map[string]struct{}{}
	for _, record := range records {
		for _, versionID := range record.GetStringSlice("wallet_versions") {
			if versionID != "" {
				versionIDs[versionID] = struct{}{}
			}
		}
	}
	return findRecordsByIDs(app, "wallet_versions", interopUniqueIDs(versionIDs))
}

func resolveWalletVersionLabels(
	scan interopCacheScan,
	records []*core.Record,
	axis interopAxis,
	versionsByID map[string]*core.Record,
) {
	rowResolver, err := getInteropEntityResolver(axis.HubCollection)
	if err != nil || !rowResolver.SupportsVersionLabels() {
		return
	}

	for _, record := range records {
		for _, walletID := range record.GetStringSlice(axis.CacheField) {
			if walletID == "" {
				continue
			}
			if _, ok := scan.walletVersionLabels[walletID]; ok {
				continue
			}
			scan.walletVersionLabels[walletID] = walletVersionLabelFromCacheRecord(
				record,
				walletID,
				versionsByID,
			)
		}
	}
}

func loadInteropEntitiesByIDs(
	app core.App,
	collectionName string,
	ids map[string]struct{},
	walletVersionLabels map[string]*string,
) (map[string]InteropMatrixEntity, error) {
	entities := make(map[string]InteropMatrixEntity, len(ids))
	if len(ids) == 0 {
		return entities, nil
	}

	resolver, err := getInteropEntityResolver(collectionName)
	if err != nil {
		return nil, err
	}

	recordsByID, err := findRecordsByIDs(app, collectionName, interopUniqueIDs(ids))
	if err != nil {
		return nil, err
	}

	related, err := loadInteropRelatedRecords(app, resolver, recordsByID)
	if err != nil {
		return nil, err
	}

	for id := range ids {
		record, ok := recordsByID[id]
		if !ok {
			continue
		}
		entity, err := resolver.Entity(app, record, related)
		if err != nil {
			return nil, err
		}
		if walletVersionLabels != nil && resolver.SupportsVersionLabels() {
			if label, ok := walletVersionLabels[id]; ok && label != nil {
				entity.VersionLabel = label
			}
		}
		entities[id] = entity
	}

	return entities, nil
}

func loadInteropRelatedRecords(
	app core.App,
	resolver interopEntityResolver,
	recordsByID map[string]*core.Record,
) (interopRelatedRecords, error) {
	related := interopRelatedRecords{byCollection: map[string]map[string]*core.Record{}}

	for _, spec := range resolver.RelatedCollections() {
		relatedIDs := interopRelationIDs(recordsByID, spec.Field)
		records, err := findRecordsByIDs(app, spec.Collection, relatedIDs)
		if err != nil {
			return interopRelatedRecords{}, err
		}
		related.byCollection[spec.Collection] = records
	}

	return related, nil
}

func interopRelationIDs(recordsByID map[string]*core.Record, field string) []string {
	seen := map[string]struct{}{}
	for _, record := range recordsByID {
		id := strings.TrimSpace(record.GetString(field))
		if id == "" {
			continue
		}
		seen[id] = struct{}{}
	}
	return interopUniqueIDs(seen)
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
