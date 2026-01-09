// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build !unit

package apis

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestAddTemplatingRoutes(t *testing.T) {
	godotenv.Load("../../../.env")

	app, err := tests.NewTestApp(testDataDir)
	defer app.Cleanup()
	require.NoError(t, err)

	setupTestApp := func(t testing.TB) *tests.TestApp {
		rootDir := t.TempDir()
		templatesDir := filepath.Join(rootDir, "config_templates", "test-standard", "v1", "suite-a")
		require.NoError(t, os.MkdirAll(templatesDir, 0o755))

		require.NoError(
			t,
			os.WriteFile(
				filepath.Join(rootDir, "config_templates", "test-standard", "standard.yaml"),
				[]byte("uid: test-standard\nname: Test Standard\n"),
				0o644,
			),
		)
		require.NoError(
			t,
			os.WriteFile(
				filepath.Join(rootDir, "config_templates", "test-standard", "v1", "version.yaml"),
				[]byte("uid: v1\nname: Version 1\n"),
				0o644,
			),
		)
		require.NoError(
			t,
			os.WriteFile(
				filepath.Join(templatesDir, "metadata.yaml"),
				[]byte("uid: suite-a\nname: Suite A\nshow_in_pipeline_gui: true\n"),
				0o644,
			),
		)
		require.NoError(
			t,
			os.WriteFile(filepath.Join(templatesDir, "suite.json"), []byte("{}"), 0o644),
		)

		originalRoot := os.Getenv("ROOT_DIR")
		require.NoError(t, os.Setenv("ROOT_DIR", rootDir))
		t.Cleanup(func() {
			_ = os.Setenv("ROOT_DIR", originalRoot)
		})

		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		handlers.TemplateRoutes.Add(testApp)

		return testApp
	}
	authToken, _ := generateToken("users", "userA@example.org")

	scenarios := []tests.ApiScenario{
		{
			Name:   "Get configs templates",
			Method: http.MethodGet,
			URL:    "/api/template/blueprints",
			Headers: map[string]string{
				"Authorization": "Bearer " + authToken,
			},
			Delay:           0,
			ExpectedContent: []string{"suites"},
			Timeout:         5 * time.Second,
			ExpectedStatus:  http.StatusOK,
			TestAppFactory:  setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
