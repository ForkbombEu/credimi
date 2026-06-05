// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestParsePaginationParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rawURL        string
		defaultLimit  int
		defaultOffset int
		wantLimit     int
		wantOffset    int
	}{
		{
			name:          "uses defaults when missing",
			rawURL:        "/api/results",
			defaultLimit:  20,
			defaultOffset: 0,
			wantLimit:     20,
			wantOffset:    0,
		},
		{
			name:          "uses valid query params",
			rawURL:        "/api/results?limit=50&offset=3",
			defaultLimit:  20,
			defaultOffset: 0,
			wantLimit:     50,
			wantOffset:    3,
		},
		{
			name:          "falls back on invalid limit and offset",
			rawURL:        "/api/results?limit=0&offset=-1",
			defaultLimit:  20,
			defaultOffset: 2,
			wantLimit:     20,
			wantOffset:    2,
		},
		{
			name:          "falls back on non numeric values",
			rawURL:        "/api/results?limit=abc&offset=xyz",
			defaultLimit:  10,
			defaultOffset: 4,
			wantLimit:     10,
			wantOffset:    4,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.rawURL, nil)
			e := &core.RequestEvent{
				Event: router.Event{
					Request: req,
				},
			}

			limit, offset := parsePaginationParams(e, tt.defaultLimit, tt.defaultOffset)
			require.Equal(t, tt.wantLimit, limit)
			require.Equal(t, tt.wantOffset, offset)
		})
	}
}

func TestComputeChildDisplayName(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"View logs workflow",
		computeChildDisplayName("OpenID4VPWalletCheckWorkflow_123"),
	)
	require.Equal(t, "View logs workflow", computeChildDisplayName("EWCWorkflow_123"))

	workflowID := "prefix_123e4567-e89b-12d3-a456-426614174000_child-name"
	require.Equal(t, "child-name", computeChildDisplayName(workflowID))

	require.Equal(t, "plain-workflow-id", computeChildDisplayName("plain-workflow-id"))
}

func TestSortExecutionSummaries(t *testing.T) {
	t.Parallel()

	loc := time.UTC
	rootA := &WorkflowExecutionSummary{
		StartTime: "2025-01-02T10:00:00Z",
		EndTime:   "2025-01-02T10:30:00Z",
		Children: []*WorkflowExecutionSummary{
			{
				StartTime: "2025-01-02T09:00:00Z",
				EndTime:   "2025-01-02T09:30:00Z",
			},
			{
				StartTime: "2025-01-02T08:00:00Z",
				EndTime:   "2025-01-02T08:30:00Z",
			},
		},
	}
	rootB := &WorkflowExecutionSummary{
		StartTime: "2025-01-03T10:00:00Z",
		EndTime:   "2025-01-03T10:30:00Z",
	}

	list := []*WorkflowExecutionSummary{rootA, rootB}
	sortExecutionSummaries(list, loc, false)

	require.Equal(t, "03/01/2025, 10:00:00", list[0].StartTime)
	require.Equal(t, "02/01/2025, 10:00:00", list[1].StartTime)
	require.Equal(t, "02/01/2025, 08:00:00", list[1].Children[0].StartTime)
	require.Equal(t, "02/01/2025, 09:00:00", list[1].Children[1].StartTime)
}

func TestBaseKey(t *testing.T) {
	t.Parallel()

	key, ok := baseKey("abc_result_video_main.mp4", "_result_video_")
	require.True(t, ok)
	require.Equal(t, "abc", key)

	key, ok = baseKey("abc_without_marker.mp4", "_result_video_")
	require.False(t, ok)
	require.Equal(t, "", key)
}

func TestComputePipelineResultsFromRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.Settings().Meta.AppURL = "https://app.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("video_results", []string{"abc_result_video_main.mp4"})
	record.Set("screenshots", []string{"abc_screenshot_main.png"})
	record.Set("logcats", []string{"abc_logfile_main.zip"})

	got := pipelineresults.ComputePipelineResultsFromRecord(app, record)
	require.Len(t, got, 1)
	require.Contains(t, got[0].Video, "https://app.test")
	require.Contains(t, got[0].Video, "abc_result_video_main.mp4")
	require.Contains(t, got[0].Screenshot, "abc_screenshot_main.png")
	require.Contains(t, got[0].Log, "abc_logfile_main.zip")

	require.Nil(t, pipelineresults.ComputePipelineResultsFromRecord(nil, record))
	require.Nil(t, pipelineresults.ComputePipelineResultsFromRecord(app, nil))
}

func TestComputePipelineReportURLFromRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.Settings().Meta.AppURL = "https://app.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Id = "rec123"
	record.Set("report", []string{"run_report.md"})

	got := pipelineresults.ComputePipelineReportURLFromRecord(app, record)
	require.Equal(t, "https://app.test/api/files/pipeline_results/rec123/run_report.md", got)

	record.Set("report", []string{})
	require.Equal(t, "", pipelineresults.ComputePipelineReportURLFromRecord(app, record))
	require.Equal(t, "", pipelineresults.ComputePipelineReportURLFromRecord(nil, record))
	require.Equal(t, "", pipelineresults.ComputePipelineReportURLFromRecord(app, nil))
}

func TestBuildPipelineExecutionArtifacts(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.Settings().Meta.AppURL = "https://app.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Id = "rec123"
	record.Set("video_results", []string{"abc_result_video_main.mp4"})
	record.Set("screenshots", []string{"abc_screenshot_main.png"})
	record.Set("logcats", []string{"abc_logfile_main.zip"})
	record.Set("report", []string{"run_report.md"})

	got := pipelineresults.BuildPipelineExecutionArtifacts(app, record)
	require.Len(t, got.Results, 1)
	require.Contains(t, got.Results[0].Log, "abc_logfile_main.zip")
	require.Equal(t, "https://app.test/api/files/pipeline_results/rec123/run_report.md", got.Report)

	require.Equal(
		t,
		PipelineExecutionArtifacts{Results: []PipelineResults{}},
		pipelineresults.BuildPipelineExecutionArtifacts(nil, record),
	)
	require.Equal(
		t,
		PipelineExecutionArtifacts{Results: []PipelineResults{}},
		pipelineresults.BuildPipelineExecutionArtifacts(app, nil),
	)
}
