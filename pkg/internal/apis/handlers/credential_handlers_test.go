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

func setupCredentialApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	CredentialTemporalInternalRoutes.Add(app)
	return app
}

func TestHandleGetCredentialOffer(t *testing.T) {
	orgID, err := getOrgIDfromName("organizations", "userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing credential_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/credential/get-credential-offer",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"credential_identifier"`,
				`"credential_identifier is required"`,
			},
			TestAppFactory: setupCredentialApp,
		},
		{
			Name:           "nonexistent credential_identifier",
			Method:         http.MethodGet,
			URL:            "/api/credential/get-credential-offer?credential_identifier=nonexistent",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"credential not found"`,
			},
			TestAppFactory: setupCredentialApp,
		},
		{
			Name:           "valid credential with deeplink",
			Method:         http.MethodGet,
			URL:            "/api/credential/get-credential-offer?credential_identifier=usera-s-organization/issuer-123/cred123",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"credential_offer"`,
				`"https://deeplink.example/offer"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupCredentialApp(t)

				issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
				require.NoError(t, err)
				issuer := core.NewRecord(issuerColl)
				issuer.Set("url", "https://issuer.example")
				issuer.Set("owner", orgID)
				issuer.Set("name", "Issuer 123")
				require.NoError(t, app.Save(issuer))

				coll, err := app.FindCollectionByNameOrId("credentials")
				require.NoError(t, err)
				record := core.NewRecord(coll)
				record.Set("name", "cred123")
				record.Set("credential_issuer", issuer.Id)
				record.Set("owner", orgID)
				record.Set("deeplink", "https://deeplink.example/offer")
				require.NoError(t, app.Save(record))

				return app
			},
		},
		{
			Name:           "valid credential without deeplink — builds credential offer URI",
			Method:         http.MethodGet,
			URL:            "/api/credential/get-credential-offer?credential_identifier=usera-s-organization/issuer-456/cred456",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"credential_offer"`,
				`"openid-credential-offer://?credential_offer=%7B%22credential_configuration_ids%22%3A%5B%22cred456%22%5D%2C%22credential_issuer%22%3A%22https%3A%2F%2Fissuer.example%22%7D"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupCredentialApp(t)

				issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
				require.NoError(t, err)
				issuer := core.NewRecord(issuerColl)
				issuer.Set("url", "https://issuer.example")
				issuer.Set("owner", orgID)
				issuer.Set("name", "Issuer 456")
				require.NoError(t, app.Save(issuer))

				coll, err := app.FindCollectionByNameOrId("credentials")
				require.NoError(t, err)
				record := core.NewRecord(coll)
				record.Set("name", "cred456")
				record.Set("owner", orgID)
				record.Set("credential_issuer", issuer.Id)
				require.NoError(t, app.Save(record))

				return app
			},
		},
		{
			Name:           "valid dynamic credential with code (yaml field present)",
			Method:         http.MethodGet,
			URL:            "/api/credential/get-credential-offer?credential_identifier=usera-s-organization/issuer-789/cred789",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"dynamic":true`,
				`"code"`,
				`"print('hello world')"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupCredentialApp(t)

				issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
				require.NoError(t, err)
				issuer := core.NewRecord(issuerColl)
				issuer.Set("url", "https://issuer.example")
				issuer.Set("owner", orgID)
				issuer.Set("name", "Issuer 789")
				require.NoError(t, app.Save(issuer))

				coll, err := app.FindCollectionByNameOrId("credentials")
				require.NoError(t, err)
				record := core.NewRecord(coll)
				record.Set("name", "cred789")
				record.Set("owner", orgID)
				record.Set("credential_issuer", issuer.Id)
				record.Set("yaml", "print('hello world')")
				require.NoError(t, app.Save(record))

				return app
			},
		},
		{
			Name:           "credential with empty code — should not be dynamic",
			Method:         http.MethodGet,
			URL:            "/api/credential/get-credential-offer?credential_identifier=usera-s-organization/issuer-987/cred987",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"credential_offer"`,
				`"openid-credential-offer://?credential_offer=%7B%22credential_configuration_ids%22%3A%5B%22cred987%22%5D%2C%22credential_issuer%22%3A%22https%3A%2F%2Fissuer.example%22%7D"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupCredentialApp(t)

				issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
				require.NoError(t, err)
				issuer := core.NewRecord(issuerColl)
				issuer.Set("url", "https://issuer.example")
				issuer.Set("owner", orgID)
				issuer.Set("name", "Issuer 987")
				require.NoError(t, app.Save(issuer))

				coll, err := app.FindCollectionByNameOrId("credentials")
				require.NoError(t, err)
				record := core.NewRecord(coll)
				record.Set("name", "cred987")
				record.Set("owner", orgID)
				record.Set("credential_issuer", issuer.Id)
				record.Set("yaml", "")
				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
