// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupDeeplinkApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		DeepLinkCredential.Add(app)

		coll, _ := app.FindCollectionByNameOrId("credential_issuers")
		issuerRecord := core.NewRecord(coll)
		issuerRecord.Set("id", "issuer123456789")
		issuerRecord.Set("name", "test issuer")
		issuerRecord.Set("url", "https://test-issuer.example.com")
		issuerRecord.Set("owner", orgID)
		require.NoError(t, app.Save(issuerRecord))

		credColl, _ := app.FindCollectionByNameOrId("credentials")
		r := core.NewRecord(credColl)
		r.Set("owner", orgID)
		r.Set("name", "test credential")
		r.Set("credential_issuer", issuerRecord.Id)
		r.Set("deeplink", "openid-credential-offer://...")
		require.NoError(t, app.Save(r))

		return app
	}
}

func TestGetCredentialDeeplink(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "get credential deeplink-success",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				"openid-credential-offer://",
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get credential deeplink - missing id",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"error":"request"`,
				`"reason":"missing credential id"`,
				`"message":"id parameter is required"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:   "get credential deeplink - invalid credential path",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-2/test-credential"
			}(),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"error":"resolve"`,
				`"reason":"failed to resolve credential path"`,
				`"message":"sql: no rows in result set"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:   "get credential deeplink - redirect",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential&redirect=true"
			}(),
			ExpectedStatus: http.StatusMovedPermanently,
			TestAppFactory: setupDeeplinkApp(orgID),

			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				require.Equal(
					t.(*testing.T),
					"openid-credential-offer://...",
					res.Header.Get("Location"),
				)
			},
		},
		{
			Name:   "get credential deeplink - empty deeplink",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"credential"`,
				`"reason":"deeplink not found"`,
				`"message":"field 'deeplink' is missing or empty"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)
				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)

				r.Set("deeplink", "")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get credential deeplink with yaml - success",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`mock-deeplink-from-yaml`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)

				responseBody, err := json.Marshal(map[string]any{
					"deeplink": "mock-deeplink-from-yaml",
				})
				require.NoError(t, err)
				mockDeeplinkTransport(t.(*testing.T), http.StatusOK, string(responseBody), nil)
				app.Settings().Meta.AppURL = "https://internal.test"

				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)
				r.Set("deeplink", "")
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get credential deeplink with yaml - internal endpoint error",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"get-deeplink"`,
				`"reason":"internal endpoint returned an error"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)

				responseBody, err := json.Marshal(map[string]any{
					"error": "internal server error",
				})
				require.NoError(t, err)
				mockDeeplinkTransport(t.(*testing.T), http.StatusInternalServerError, string(responseBody), nil)
				app.Settings().Meta.AppURL = "https://internal.test"

				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)
				r.Set("deeplink", "")
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get credential deeplink with yaml - invalid response",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"deeplink"`,
				`"reason":"deeplink missing in response"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)

				responseBody, err := json.Marshal(map[string]any{
					"wrong_field": "value",
				})
				require.NoError(t, err)
				mockDeeplinkTransport(t.(*testing.T), http.StatusOK, string(responseBody), nil)
				app.Settings().Meta.AppURL = "https://internal.test"

				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)
				r.Set("deeplink", "")
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get credential deeplink - http call network error",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"request"`,
				`"failed to call internal /api/get-deeplink endpoint"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)

				mockDeeplinkTransport(
					t.(*testing.T),
					http.StatusOK,
					`{}`,
					errors.New("network error"),
				)
				app.Settings().Meta.AppURL = "https://internal.test"

				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)
				r.Set("deeplink", "")
				r.Set("yaml", "test: yaml content")
				require.NoError(t, app.Save(r))

				return app
			},
		},
		{
			Name:   "get credential deeplink - json unmarshal error",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"error":"json"`,
				`"failed to parse /api/get-deeplink response"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)

				mockDeeplinkTransport(t.(*testing.T), http.StatusOK, `{"deeplink": "value`, nil)
				app.Settings().Meta.AppURL = "https://internal.test"

				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)
				r.Set("deeplink", "")
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
