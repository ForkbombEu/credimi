// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestParsePageParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		rawURL       string
		defaultLimit int
		defaultPage  int
		wantLimit    int
		wantPage     int
	}{
		{
			name:         "uses defaults when missing",
			rawURL:       "/api/results",
			defaultLimit: 20,
			defaultPage:  0,
			wantLimit:    20,
			wantPage:     0,
		},
		{
			name:         "uses valid query params",
			rawURL:       "/api/results?limit=50&page=3",
			defaultLimit: 20,
			defaultPage:  0,
			wantLimit:    50,
			wantPage:     3,
		},
		{
			name:         "falls back on invalid limit and offset",
			rawURL:       "/api/results?limit=0&offset=-1",
			defaultLimit: 20,
			defaultPage:  2,
			wantLimit:    20,
			wantPage:     2,
		},
		{
			name:         "falls back on non numeric values",
			rawURL:       "/api/results?limit=abc&offset=xyz",
			defaultLimit: 10,
			defaultPage:  4,
			wantLimit:    10,
			wantPage:     4,
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

			limit, page := parsePageParams(e, tt.defaultLimit, tt.defaultPage)
			require.Equal(t, tt.wantLimit, limit)
			require.Equal(t, tt.wantPage, page)
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
				Children: []*WorkflowExecutionSummary{
					{StartTime: "2025-01-02T08:20:00Z"},
					{StartTime: "2025-01-02T08:10:00Z"},
				},
			},
		},
	}
	rootB := &WorkflowExecutionSummary{
		StartTime: "2025-01-03T10:00:00Z",
		EndTime:   "2025-01-03T10:30:00Z",
	}

	list := []*WorkflowExecutionSummary{rootA, rootB}
	sortExecutionSummaries(list, false)

	require.Equal(t, "2025-01-03T10:00:00Z", list[0].StartTime)
	require.Equal(t, "2025-01-02T10:00:00Z", list[1].StartTime)
	require.Equal(t, "2025-01-02T08:00:00Z", list[1].Children[0].StartTime)
	require.Equal(t, "2025-01-02T09:00:00Z", list[1].Children[1].StartTime)
	// Every nested level is chronological, whatever the roots' order.
	require.Equal(t, "2025-01-02T08:10:00Z", list[1].Children[0].Children[0].StartTime)
	require.Equal(t, "2025-01-02T08:20:00Z", list[1].Children[0].Children[1].StartTime)
}

func TestLinkedInnerDisplayName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		blockID string
		innerID string
		want    string
	}{
		{"step with attempt", "wf-1-offer", "wf-1-offer-offer-a1", "offer"},
		{"multi digit attempt", "wf-1-offer", "wf-1-offer-wallet-step-a12", "wallet-step"},
		{"no attempt suffix", "wf-1-offer", "wf-1-offer-wallet", "wallet"},
		{"foreign id", "wf-1-offer", "other-a3", "other"},
		{"only suffix kept", "wf-1-offer", "wf-1-offer-a1", "a1"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, linkedInnerDisplayName(tt.blockID, tt.innerID))
		})
	}
}

func TestFilterNonPipelineExecutionsDropsLinkedSteps(t *testing.T) {
	t.Parallel()

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "block-unlisted", "run-unlisted").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Type: &common.WorkflowType{Name: pipelineinternal.LinkedStepsWorkflowName},
			},
		}, nil).
		Once()

	check := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "check", RunID: "run-check"},
		Type:      WorkflowType{Name: "Mobile Automation"},
	}
	block := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "block", RunID: "run-block"},
		Type:      WorkflowType{Name: pipelineinternal.LinkedStepsWorkflowName},
	}
	innerListed := &WorkflowExecution{
		Execution:       &WorkflowIdentifier{WorkflowID: "block-wallet-a1", RunID: "run-inner"},
		Type:            WorkflowType{Name: "Mobile Automation"},
		ParentExecution: &WorkflowIdentifier{WorkflowID: "block", RunID: "run-block"},
	}
	innerUnlisted := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "block-offer-a1", RunID: "run-inner-2"},
		Type:      WorkflowType{Name: "Credentials Issuance"},
		ParentExecution: &WorkflowIdentifier{
			WorkflowID: "block-unlisted",
			RunID:      "run-unlisted",
		},
	}

	filtered, err := filterNonPipelineExecutions(
		context.Background(),
		mockClient,
		[]*WorkflowExecution{check, block, innerListed, innerUnlisted},
	)
	require.NoError(t, err)
	require.Equal(t, []*WorkflowExecution{check}, filtered)
	mockClient.AssertExpectations(t)
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

func TestComputePipelineResultsFromRecordWithoutVideo(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	// Device "abc" ran with video, "nov" with no_video (screenshot and log only).
	record := core.NewRecord(coll)
	record.Set("video_results", []string{"abc_result_video_main.mp4"})
	record.Set("screenshots", []string{"abc_screenshot_main.png", "nov_screenshot_x.png"})
	record.Set("logcats", []string{"abc_logfile_main.zip", "nov_logfile_x.zip"})
	record.Set("maestro_timelines", []string{"nov_timeline_x.jsonl"})

	got := pipelineresults.ComputePipelineResultsFromRecord(app, record)
	require.Len(t, got, 2)
	require.Contains(t, got[0].Video, "abc_result_video_main.mp4")
	require.Empty(t, got[1].Video)
	require.Contains(t, got[1].Screenshot, "nov_screenshot_x.png")
	require.Contains(t, got[1].Log, "nov_logfile_x.zip")
	require.Empty(t, got[1].Timeline, "a timeline belongs to a video")
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
	record.Set("fcaf_report", []string{"fcaf_assessment.json"})
	record.Set("fcaf_report_pdf", []string{"fcaf_assessment.pdf"})

	got := pipelineresults.BuildPipelineExecutionArtifacts(app, record)
	require.Len(t, got.Results, 1)
	require.Contains(t, got.Results[0].Log, "abc_logfile_main.zip")
	require.Equal(t, "https://app.test/api/files/pipeline_results/rec123/run_report.md", got.Report)
	require.Equal(
		t,
		"https://app.test/api/files/pipeline_results/rec123/fcaf_assessment.json",
		got.FCAFReport,
	)
	require.Equal(
		t,
		"https://app.test/api/files/pipeline_results/rec123/fcaf_assessment.pdf",
		got.FCAFReportPDF,
	)

	require.Equal(
		t,
		pipelineresults.PipelineExecutionArtifacts{Results: []pipelineresults.PipelineResults{}},
		pipelineresults.BuildPipelineExecutionArtifacts(nil, record),
	)
	require.Equal(
		t,
		pipelineresults.PipelineExecutionArtifacts{Results: []pipelineresults.PipelineResults{}},
		pipelineresults.BuildPipelineExecutionArtifacts(app, nil),
	)
}
