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

func setupApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		CloneRecord.Add(app)

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
		require.NoError(t, app.Save(r))

		return app
	}
}

func TestGetCloneRecord(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "get clone-record success",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/clone-record?id=usera-s-organization/test-issuer-1/test-credential"
			}(),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"cloned_record"`,
				`"canonified_name":"test-credential-copy`,
				`"name":"test credential_copy`,
				`"conformant":false`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:           "get clone-record missing id",
			Method:         http.MethodGet,
			URL:            "/api/clone-record",
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Parameter 'id' is required."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:           "get clone-record invalid id",
			Method:         http.MethodGet,
			URL:            "/api/clone-record?id=usera-s-organization/test-issuer-1/test-credential/gddgjd",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"error":"resolve"`,
				`"message":"sql: no rows in result set"`,
				`"reason":"failed to resolve collection path"`,
			},
			TestAppFactory: setupApp(orgID),
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
