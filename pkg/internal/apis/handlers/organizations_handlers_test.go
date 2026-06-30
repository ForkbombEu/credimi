// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupOrganizationApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	ensureOrganizationPublishedField(t, app)
	canonify.RegisterCanonifyHooks(app)
	OrganizationRoutes.Add(app)
	return app
}

func setupOrganizationPublicApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	ensureOrganizationPublishedField(t, app)
	canonify.RegisterCanonifyHooks(app)
	OrganizationTemporalInternalRoutes.Add(app)
	seedInternalAdminKey(t, app)
	return app
}

func ensureOrganizationPublishedField(t testing.TB, app *tests.TestApp) {
	t.Helper()

	organizations, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	if organizations.Fields.GetByName("published") == nil {
		organizations.Fields.Add(&core.BoolField{Name: "published"})
	}
	require.NoError(t, app.Save(organizations))
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
			Method: http.MethodGet,
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
			Name:   "get visible organization namespaces",
			Method: http.MethodGet,
			URL:    "/api/organizations/visible-namespaces",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"namespaces",
				"usera-s-organization",
			},
			TestAppFactory: setupOrganizationApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				require.NotContains(t, string(body), "my_namespace")

				var payload struct {
					Namespaces []string `json:"namespaces"`
				}
				require.NoError(t, json.Unmarshal(body, &payload))
				require.Contains(t, payload.Namespaces, "usera-s-organization")
				require.NotEmpty(t, payload.Namespaces)
			},
		},
		{
			Name:           "get my organization info (unauthenticated)",
			Method:         http.MethodGet,
			URL:            "/api/organizations/my",
			ExpectedStatus: 401,
			ExpectedContent: []string{
				"authentication_required",
			},
			TestAppFactory: setupOrganizationApp,
		},
		{
			Name:           "get visible organization namespaces (unauthenticated)",
			Method:         http.MethodGet,
			URL:            "/api/organizations/visible-namespaces",
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
			Method: http.MethodGet,
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
			Method:         http.MethodGet,
			URL:            "/api/organizations/namespaces",
			ExpectedStatus: 401,
			ExpectedContent: []string{
				"api_key_required",
			},
			TestAppFactory: setupOrganizationPublicApp,
		},
		{
			Name:   "get all namespaces with wrong API key",
			Method: http.MethodGet,
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
