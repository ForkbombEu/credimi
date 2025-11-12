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

func setupDeeplinkApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		DeepLinkCredential.Add(app)

		// Crea issuer e credential se necessario
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
	orgID, err := getOrgIDfromName("organizations", "userA's organization")
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
				`"deeplink":"openid-credential-offer://`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:            "get credential deeplink - missing id",
			Method:          http.MethodGet,
			URL:             "/api/credential/deeplink",
			ExpectedStatus:  400,
			ExpectedContent: []string{`"error":"request"`, `"reason":"missing credential id"`, `"message":"id parameter is required"`},
			TestAppFactory:  setupDeeplinkApp(orgID),
		},
		{
			Name:   "get credential deeplink - invalid credential path",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-2/test-credential"
			}(),
			ExpectedStatus:  404,
			ExpectedContent: []string{`"error":"resolve"`, `"reason":"failed to resolve credential path"`, `"message":"sql: no rows in result set"`},
			TestAppFactory:  setupDeeplinkApp(orgID),
		},
		{
			Name:   "get credential deeplink - empty deeplink",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus:  500,
			ExpectedContent: []string{`"error":"credential"`, `"reason":"deeplink not found"`, `"message":"field 'deeplink' is missing or empty"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)
				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)

				r.Set("deeplink", "")
				require.NoError(t, app.Save(r))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
