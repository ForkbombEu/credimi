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

// TestPipelineStartRejectsRunnerPipelines ensures runner pipelines are redirected to the queue.
func TestPipelineStartRejectsRunnerPipelines(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	runnerYaml := "name: test\nsteps:\n  - id: step-1\n    use: mobile-automation\n    with:\n      runner_id: runner-1\n      action_id: action-1\n"

	scenario := tests.ApiScenario{
		Name:   "start rejects runner pipelines",
		Method: http.MethodPost,
		URL:    "/api/pipeline/start",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"yaml":                runnerYaml,
		}),
		ExpectedStatus: http.StatusConflict,
		ExpectedContent: []string{
			"mobile-runner pipelines must be started via queue/semaphore",
			"/api/pipeline/queue",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupPipelineStartApp(t)

			coll, err := app.FindCollectionByNameOrId("pipelines")
			require.NoError(t, err)

			record := core.NewRecord(coll)
			record.Set("owner", orgID)
			record.Set("name", "pipeline123")
			record.Set("description", "test-description")
			record.Set(
				"steps",
				map[string]any{"rest-chain": map[string]any{"yaml": runnerYaml}},
			)
			record.Set("yaml", runnerYaml)
			require.NoError(t, app.Save(record))

			return app
		},
	}

	scenario.Test(t)
}
