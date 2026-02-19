// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestParsePaginationParams(t *testing.T) {
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
			req := httptest.NewRequest("GET", tt.rawURL, nil)
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
	require.Equal(t, "View logs workflow", computeChildDisplayName("OpenIDNetCheckWorkflow_123"))
	require.Equal(t, "View logs workflow", computeChildDisplayName("EWCWorkflow_123"))

	workflowID := "prefix_123e4567-e89b-12d3-a456-426614174000_child-name"
	require.Equal(t, "child-name", computeChildDisplayName(workflowID))

	require.Equal(t, "plain-workflow-id", computeChildDisplayName("plain-workflow-id"))
}

func TestSortExecutionSummaries(t *testing.T) {
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

	got := computePipelineResultsFromRecord(app, record)
	require.Len(t, got, 1)
	require.Contains(t, got[0].Video, "https://app.test")
	require.Contains(t, got[0].Video, "abc_result_video_main.mp4")
	require.Contains(t, got[0].Screenshot, "abc_screenshot_main.png")

	require.Nil(t, computePipelineResultsFromRecord(nil, record))
	require.Nil(t, computePipelineResultsFromRecord(app, nil))
}
