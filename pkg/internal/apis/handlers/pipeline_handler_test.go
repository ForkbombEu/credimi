// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupPipelineApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineTemporalInternalRoutes.Add(app)

	return app
}

// setupPipelineStartApp builds a test app with pipeline start routes.
func setupPipelineStartApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	return app
}

func TestGetPipelineYAML(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing pipeline_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/pipeline/get-yaml",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"pipeline_identifier"`,
				`"pipeline_identifier is required"`,
			},
			TestAppFactory: setupPipelineApp,
		},
		{
			Name:           "nonexistent pipeline identifier",
			Method:         http.MethodGet,
			URL:            "/api/pipeline/get-yaml?pipeline_identifier=does-not-exist",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"pipeline not found"`,
			},
			TestAppFactory: setupPipelineApp,
		},
		{
			Name:           "valid pipeline identifier",
			Method:         http.MethodGet,
			URL:            "/api/pipeline/get-yaml?pipeline_identifier=usera-s-organization/pipeline123",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`example-yaml-content`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineApp(t)

				coll, err := app.FindCollectionByNameOrId("pipelines")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("name", "pipeline123")
				record.Set("description", "test-description")
				record.Set(
					"steps",
					map[string]any{"rest-chain": map[string]any{"yaml": "example-yaml-content"}},
				)
				record.Set("yaml", "example-yaml-content")
				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSetPipelineExecutionResults(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing request body",
			Method:         http.MethodPost,
			URL:            "/api/pipeline/pipeline-execution-results",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				"pipeline not found",
			},
			TestAppFactory: setupPipelineApp,
		},
		{
			Name:   "valid pipeline execution result",
			Method: http.MethodPost,
			URL:    "/api/pipeline/pipeline-execution-results",
			Body: jsonBody(map[string]any{
				"owner":       "usera-s-organization",
				"pipeline_id": "usera-s-organization/pipeline123",
				"workflow_id": "workflow-xyz",
				"run_id":      "run-001",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"owner"`,
				`"pipeline"`,
				`"workflow_id"`,
				`"run_id"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineApp(t)

				coll, err := app.FindCollectionByNameOrId("pipelines")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("id", "pipeline1234567")
				record.Set("owner", orgID)
				record.Set("name", "pipeline123")
				record.Set("description", "test-description")
				record.Set(
					"steps",
					map[string]any{"rest-chain": map[string]any{"yaml": "example-yaml-content"}},
				)
				record.Set("yaml", "example-yaml-content")
				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSetPipelineExecutionResultsIdempotent(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("id", "pipeline1234567")
	record.Set("owner", orgID)
	record.Set("name", "pipeline123")
	record.Set("description", "test-description")
	record.Set(
		"steps",
		map[string]any{"rest-chain": map[string]any{"yaml": "example-yaml-content"}},
	)
	record.Set("yaml", "example-yaml-content")
	require.NoError(t, app.Save(record))

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		body := `{"owner":"usera-s-organization","pipeline_id":"usera-s-organization/pipeline123","workflow_id":"workflow-xyz","run_id":"run-001"}`
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/pipeline/pipeline-execution-results",
				strings.NewReader(body),
			)
			req.Header.Set("content-type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)
		}

		records, err := app.FindRecordsByFilter(
			"pipeline_results",
			"workflow_id = {:workflow_id} && run_id = {:run_id}",
			"",
			-1,
			0,
			dbx.Params{
				"workflow_id": "workflow-xyz",
				"run_id":      "run-001",
			},
		)
		require.NoError(t, err)
		require.Len(t, records, 1)

		return nil
	})
	require.NoError(t, serveErr)
}

func TestBuildChildWorkflowParentQuery(t *testing.T) {
	require.Equal(t, "", buildChildWorkflowParentQuery(nil))

	query := buildChildWorkflowParentQuery([]workflowExecutionRef{
		{
			WorkflowID: "parent-workflow-1",
			RunID:      "run-1",
		},
		{
			WorkflowID: `parent"workflow\2`,
			RunID:      `run"2\`,
		},
	})

	require.Equal(
		t,
		`(ParentWorkflowId="parent-workflow-1" AND ParentRunId="run-1") OR `+
			`(ParentWorkflowId="parent\"workflow\\2" AND ParentRunId="run\"2\\")`,
		query,
	)
}
