// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build !unit
// +build !unit

package apis

import (
	"net/http"
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
			ExpectedContent: []string{"variants"},
			Timeout:         5 * time.Second,
			ExpectedStatus:  http.StatusOK,
			TestAppFactory:  setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
