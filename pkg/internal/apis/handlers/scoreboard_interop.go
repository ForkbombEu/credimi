// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
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

type interopMode string

const (
	interopModeWalletsIssuers     interopMode = "wallets_issuers"
	interopModeWalletsCredentials interopMode = "wallets_credentials"
)

type interopModeConfig struct {
	RowRelationField    string
	ColumnRelationField string
	RowAxis             string
	ColumnAxis          string
	RowCollection       string
	ColumnCollection    string
}

var interopModeConfigs = map[interopMode]interopModeConfig{
	interopModeWalletsIssuers: {
		RowRelationField:    "wallets",
		ColumnRelationField: "issuers",
		RowAxis:             "wallet",
		ColumnAxis:          "issuer",
		RowCollection:       "wallets",
		ColumnCollection:    "credential_issuers",
	},
	interopModeWalletsCredentials: {
		RowRelationField:    "wallets",
		ColumnRelationField: "credentials",
		RowAxis:             "wallet",
		ColumnAxis:          "credential",
		RowCollection:       "wallets",
		ColumnCollection:    "credentials",
	},
}

func getInteropModeConfig(mode interopMode) (interopModeConfig, bool) {
	cfg, ok := interopModeConfigs[mode]
	return cfg, ok
}

func isSupportedInteropMode(mode interopMode) bool {
	_, ok := getInteropModeConfig(mode)
	return ok
}

type interopCacheInput struct {
	PipelineID     string
	TotalRuns      int
	TotalSuccesses int
	RowIDs         []string // wallet IDs for wallets_issuers mode
	ColumnIDs      []string // issuer IDs
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

type InteropMatrixResponse struct {
	Mode       interopMode           `json:"mode"`
	RowAxis    string                `json:"row_axis"`
	ColumnAxis string                `json:"column_axis"`
	Rows       []InteropMatrixEntity `json:"rows"`
	Columns    []InteropMatrixEntity `json:"columns"`
	Cells      []InteropMatrixCell   `json:"cells"`
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

type unsupportedInteropModeError struct {
	mode interopMode
}

func (e unsupportedInteropModeError) Error() string {
	return fmt.Sprintf("interop mode %q is not implemented", e.mode)
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
		Mode:       interopModeWalletsIssuers,
		RowAxis:    "wallet",
		ColumnAxis: "issuer",
		Rows:       sortedInteropEntities(rowEntities, rowSeen),
		Columns:    sortedInteropEntities(columnEntities, colSeen),
		Cells:      cells,
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
			Summary:        "Wallet×issuer interoperability matrix",
			Description:    "Interoperability matrix from pipeline_scoreboard_cache",
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "mode",
					Required:    true,
					Description: "Matrix pair mode. v1 supports wallets_issuers.",
				},
			},
		},
	},
}

func HandleInteropMatrix() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		mode := interopMode(e.Request.URL.Query().Get("mode"))
		if !isSupportedInteropMode(mode) {
			return apierror.New(
				http.StatusBadRequest,
				"mode",
				"unsupported or missing mode",
				"use mode=wallets_issuers or mode=wallets_credentials",
			).JSON(e)
		}

		resp, err := loadInteropMatrixFromCache(e.App, mode)
		if err != nil {
			var unsupportedModeErr unsupportedInteropModeError
			if errors.As(err, &unsupportedModeErr) {
				return apierror.New(
					http.StatusBadRequest,
					"mode",
					"mode not implemented",
					unsupportedModeErr.Error(),
				).JSON(e)
			}

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

func loadInteropMatrixFromCache(app core.App, mode interopMode) (InteropMatrixResponse, error) {
	modeConfig, ok := getInteropModeConfig(mode)
	if !ok {
		return InteropMatrixResponse{}, unsupportedInteropModeError{mode: mode}
	}

	collection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	if err != nil {
		return InteropMatrixResponse{}, fmt.Errorf("find collection: %w", err)
	}

	records, err := app.FindRecordsByFilter(collection.Id, "", "", -1, 0)
	if err != nil {
		return InteropMatrixResponse{}, fmt.Errorf("list cache: %w", err)
	}

	var inputs []interopCacheInput
	rowEntities := map[string]InteropMatrixEntity{}
	columnEntities := map[string]InteropMatrixEntity{}

	for _, record := range records {
		rowIDs := record.GetStringSlice(modeConfig.RowRelationField)
		colIDs := record.GetStringSlice(modeConfig.ColumnRelationField)

		inputs = append(inputs, interopCacheInput{
			PipelineID:     record.GetString("pipeline"),
			TotalRuns:      record.GetInt("total_runs"),
			TotalSuccesses: record.GetInt("total_successes"),
			RowIDs:         rowIDs,
			ColumnIDs:      colIDs,
		})

		if err := mergeInteropEntities(app, modeConfig.RowCollection, record, rowIDs, rowEntities); err != nil {
			return InteropMatrixResponse{}, err
		}
		if err := mergeInteropEntities(app, modeConfig.ColumnCollection, nil, colIDs, columnEntities); err != nil {
			return InteropMatrixResponse{}, err
		}
	}

	resp := buildInteropMatrix(inputs, rowEntities, columnEntities)
	resp.Mode = mode
	resp.RowAxis = modeConfig.RowAxis
	resp.ColumnAxis = modeConfig.ColumnAxis

	return resp, nil
}

func mergeInteropEntities(
	app core.App,
	collectionName string,
	cacheRecord *core.Record,
	recordIDs []string,
	entities map[string]InteropMatrixEntity,
) error {
	for _, recordID := range recordIDs {
		if _, ok := entities[recordID]; ok {
			continue
		}
		record, err := app.FindRecordById(collectionName, recordID)
		if err != nil {
			continue
		}
		entity, err := interopEntityFromRecord(app, record, collectionName)
		if err != nil {
			return err
		}
		if collectionName == "wallets" {
			label := walletVersionLabelFor(app, cacheRecord, recordID)
			if label != nil {
				entity.VersionLabel = label
			}
		}
		entities[recordID] = entity
	}
	return nil
}

func interopEntityFromRecord(
	app core.App,
	record *core.Record,
	collection string,
) (InteropMatrixEntity, error) {
	tpl, ok := canonify.CanonifyPaths[collection]
	if !ok {
		return InteropMatrixEntity{}, fmt.Errorf("no canonify path for collection %s", collection)
	}
	path, err := canonify.BuildPath(app, record, tpl, "")
	if err != nil {
		return InteropMatrixEntity{}, err
	}
	return InteropMatrixEntity{
		ID:   record.Id,
		Name: record.GetString("name"),
		Path: path,
	}, nil
}

func walletVersionLabelFor(app core.App, cacheRecord *core.Record, walletID string) *string {
	for _, versionID := range cacheRecord.GetStringSlice("wallet_versions") {
		version, err := app.FindRecordById("wallet_versions", versionID)
		if err != nil {
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
