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

func setupVerifierApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	VerifierTemporalInternalRoutes.Add(app)
	return app
}

func TestGetUseCaseVerificationDeeplink(t *testing.T) {
	orgID, err := getOrgIDfromName("organizations", "userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing use_case_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/verifier/get-use-case-verification-deeplink",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"use_case_identifier"`,
				`"use_case_identifier is required"`,
			},
			TestAppFactory: setupVerifierApp,
		},
		{
			Name:           "nonexistent use_case_identifier",
			Method:         http.MethodGet,
			URL:            "/api/verifier/get-use-case-verification-deeplink?use_case_identifier=nonexistent",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"use case verification not found"`,
			},
			TestAppFactory: setupVerifierApp,
		},
		{
			Name:           "valid use case verification Identifier",
			Method:         http.MethodGet,
			URL:            "/api/verifier/get-use-case-verification-deeplink?use_case_identifier=usera-s-organization/verifier-123/usecase123",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"code"`,
				`"example code"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupVerifierApp(t)

				verifierColl, err := app.FindCollectionByNameOrId("verifiers")
				require.NoError(t, err)
				verifier := core.NewRecord(verifierColl)
				verifier.Set("owner", orgID)
				verifier.Set("name", "Verifier 123")
				verifier.Set("url", "https://verifier.example")
				verifier.Set("standard_and_version", "testsuite/draft-01")
				verifier.Set("format", []string{"SD-JWT"})
				verifier.Set("signing_algorithms", []string{"ES256"})
				verifier.Set("cryptographic_binding_methods", []string{"jwk"})
				verifier.Set("description", "example description")
				require.NoError(t, app.Save(verifier))

				coll, err := app.FindCollectionByNameOrId("use_cases_verifications")
				require.NoError(t, err)
				record := core.NewRecord(coll)
				record.Set("name", "usecase123")
				record.Set("owner", orgID)
				record.Set("verifier", verifier.Id)
				record.Set("yaml", "example code")
				require.NoError(t, app.Save(record))

				return app
			},
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
