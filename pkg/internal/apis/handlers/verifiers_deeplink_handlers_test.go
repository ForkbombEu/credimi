// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupYamlApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		DeepLinkVerifiers.Add(app)

		ver, _ := app.FindCollectionByNameOrId("verifiers")
		verRecord := core.NewRecord(ver)
		verRecord.Set("id", "verify123456789")
		verRecord.Set("name", "test verifier")
		verRecord.Set("url", "https://test-verifier.example.com")
		verRecord.Set("standard_and_version", "openid4vci_wallet/draft-15")
		verRecord.Set("owner", orgID)
		verRecord.Set("format", "mDOC")
		verRecord.Set("signing_algorithms", "EdDSA")
		verRecord.Set("cryptographic_binding_methods", "jwk")
		verRecord.Set("description", "des")

		require.NoError(t, app.Save(verRecord))

		verColl, _ := app.FindCollectionByNameOrId("use_cases_verifications")
		r := core.NewRecord(verColl)
		r.Set("owner", orgID)
		r.Set("name", "test use cases")
		r.Set("verifier", verRecord.Id)
		r.Set("yaml", "my yaml")
		require.NoError(t, app.Save(r))

		return app
	}
}

func TestGetVerificationDeeplink(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "get verification deeplink - missing id",
			Method:         http.MethodGet,
			URL:            "/api/verification/deeplink",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"error":"request"`,
				`"reason":"missing record id"`,
				`"message":"id parameter is required"`,
			},
			TestAppFactory: setupYamlApp(orgID),
		},
		{
			Name:   "get verification deeplink - invalid verification path",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/verification/deeplink?id=usera-s-organization/test-verifier-2/test-use-cases"
			}(),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"error":"resolve"`,
				`"reason":"failed to resolve verification path"`,
				`"message":"sql: no rows in result set"`,
			},
			TestAppFactory: setupYamlApp(orgID),
		},
		{
			Name:   "get verification deeplink with yaml - success",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases"
			}(),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`mock-deeplink-from-yaml`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupYamlApp(orgID)(t)

				mockServer := mockGetDeeplinkServer(
					t.(*testing.T),
					http.StatusOK,
					map[string]any{
						"deeplink": "mock-deeplink-from-yaml",
					},
				)
				t.Cleanup(mockServer.Close)
				app.Settings().Meta.AppURL = mockServer.URL
				ver, _ := app.FindCollectionByNameOrId("use_cases_verifications")
				r, err := app.FindFirstRecordByFilter(ver.Name, `name="test use cases"`)
				require.NoError(t, err)

				r.Set("yaml", "test: yaml content")
				err = app.Save(r)
				require.NoError(t, err)

				return app
			},
		},
		{
			Name:   "get verification deeplink - redirect",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases&redirect=true"
			}(),
			ExpectedStatus:  http.StatusMovedPermanently,
			ExpectedContent: []string{}, // redirect = no body
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupYamlApp(orgID)(t)

				mockServer := mockGetDeeplinkServer(
					t.(*testing.T),
					http.StatusOK,
					map[string]any{
						"deeplink": "mock-deeplink-from-yaml",
					},
				)
				t.Cleanup(mockServer.Close)

				app.Settings().Meta.AppURL = mockServer.URL

				ver, _ := app.FindCollectionByNameOrId("use_cases_verifications")
				r, err := app.FindFirstRecordByFilter(ver.Name, `name="test use cases"`)
				require.NoError(t, err)

				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				require.Equal(t.(*testing.T), "mock-deeplink-from-yaml", res.Header.Get("Location"))
			},
		},
		{
			Name:   "get verification deeplink with yaml - internal endpoint error",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"get-deeplink"`,
				`"reason":"internal endpoint returned an error"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupYamlApp(orgID)(t)

				mockServer := mockGetDeeplinkServer(
					t.(*testing.T),
					http.StatusInternalServerError,
					map[string]any{
						"error": "internal server error",
					},
				)
				t.Cleanup(mockServer.Close)

				app.Settings().Meta.AppURL = mockServer.URL

				ver, _ := app.FindCollectionByNameOrId("use_cases_verifications")
				r, _ := app.FindFirstRecordByFilter(ver.Name, `name="test use cases"`)
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get verification deeplink with yaml - invalid response",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"deeplink"`,
				`"reason":"deeplink missing in response"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupYamlApp(orgID)(t)

				mockServer := mockGetDeeplinkServer(
					t.(*testing.T),
					http.StatusOK,
					map[string]any{
						"wrong_field": "value",
					},
				)
				t.Cleanup(mockServer.Close)

				app.Settings().Meta.AppURL = mockServer.URL

				ver, _ := app.FindCollectionByNameOrId("use_cases_verifications")
				r, _ := app.FindFirstRecordByFilter(ver.Name, `name="test use cases"`)
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get verification deeplink - http call network error",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"request"`,
				`"failed to call internal /api/get-deeplink endpoint"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupYamlApp(orgID)(t)

				app.Settings().Meta.AppURL = "http://this-domain-does-not-exist-12345.local"

				ver, _ := app.FindCollectionByNameOrId("use_cases_verifications")
				r, _ := app.FindFirstRecordByFilter(ver.Name, `name="test use cases"`)
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get verification deeplink - json unmarshal error",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"json"`,
				`"failed to parse /api/get-deeplink response"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupYamlApp(orgID)(t)

				mockServer := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"deeplink": "value`))
					}),
				)
				t.Cleanup(mockServer.Close)

				app.Settings().Meta.AppURL = mockServer.URL

				ver, _ := app.FindCollectionByNameOrId("use_cases_verifications")
				r, _ := app.FindFirstRecordByFilter(ver.Name, `name="test use cases"`)
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
