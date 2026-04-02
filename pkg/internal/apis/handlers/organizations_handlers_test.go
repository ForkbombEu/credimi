// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupOrganizationApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	OrganizationRoutes.Add(app)
	return app
}

func setupOrganizationPublicApp(t testing.TB) *tests.TestApp {
    app, err := tests.NewTestApp(testDataDir)
    require.NoError(t, err)
    canonify.RegisterCanonifyHooks(app)
    OrganizationTemporalInternalRoutes.Add(app)
    seedInternalAdminKey(t, app)  
    return app
}

func getUserRecordFromName(name string) (*core.Record, error) {
	app, err := tests.NewTestApp(testDataDir)

	if err != nil {
		return nil, err
	}
	defer app.Cleanup()

	filter := fmt.Sprintf(`name="%s"`, name)

	record, err := app.FindFirstRecordByFilter("users", filter)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func TestOrganizationHandlers(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)
	scenarios := []tests.ApiScenario{
		{
			Name:   "get my organization info",
			Method: "GET",
			URL:    "/api/organizations/my",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"name",
				"userA's organization",
				"canonified_name",
				"usera-s-organization",
			},
			TestAppFactory: setupOrganizationApp,
		},
		{
			Name:           "get my organization info (unauthenticated)",
			Method:         "GET",
			URL:            "/api/organizations/my",
			ExpectedStatus: 401,
			ExpectedContent: []string{
				"authentication_required",
			},
			TestAppFactory: setupOrganizationApp,
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
func TestGetAllNamespaces(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:   "get all namespaces with API key",
			Method: "GET",
			URL:    "/api/organizations/namespaces",
			Headers: map[string]string{
				"Credimi-Api-Key": "internal-test-api-key",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"namespaces",
			},
			TestAppFactory: setupOrganizationPublicApp,
		},
		{
			Name:           "get all namespaces without API key",
			Method:         "GET",
			URL:            "/api/organizations/namespaces",
			ExpectedStatus: 401,
			ExpectedContent: []string{
				"api_key_required",
			},
			TestAppFactory: setupOrganizationPublicApp,
		},
		{
			Name:   "get all namespaces with wrong API key",
			Method: "GET",
			URL:    "/api/organizations/namespaces",
			Headers: map[string]string{
				"Credimi-Api-Key": "wrong-key",
			},
			ExpectedStatus: 401,
			ExpectedContent: []string{
				"invalid_api_key",
			},
			TestAppFactory: setupOrganizationPublicApp,
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
