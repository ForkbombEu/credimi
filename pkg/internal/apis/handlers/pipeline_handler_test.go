// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
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
