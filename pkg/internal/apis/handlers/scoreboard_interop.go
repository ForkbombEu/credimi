// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

type interopStatus string

const (
	interopStatusStable  interopStatus = "stable"
	interopStatusFlaky   interopStatus = "flaky"
	interopStatusFailing interopStatus = "failing"
	interopStatusBroken  interopStatus = "broken"
)

type interopMode string

const interopModeWalletsIssuers interopMode = "wallets_issuers"

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
