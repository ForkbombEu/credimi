// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

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

func getTestOrgID() (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := `name="userA's organization"`

	record, err := app.FindFirstRecordByFilter("organizations", filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func setupApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		CloneRecord.Add(app)

		coll, _ := app.FindCollectionByNameOrId("credential_issuers")
		issuerRecord := core.NewRecord(coll)
		issuerRecord.Set("id", "tikklnj1uh32237")
		issuerRecord.Set("name", "test issuer")
		issuerRecord.Set("url", "https://test-issuer.example.com")
		issuerRecord.Set("owner", orgID)
		require.NoError(t, app.Save(issuerRecord))

		credColl, _ := app.FindCollectionByNameOrId("credentials")
		r := core.NewRecord(credColl)
		r.Set("id", "crede1234567890")
		r.Set("owner", orgID)
		r.Set("name", "test credential")
		r.Set("credential_issuer", issuerRecord.Id)
		require.NoError(t, app.Save(r))
		return app
	}
}

func TestGetCloneRecord(t *testing.T) {
	orgID, err := getTestOrgID()
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)
	userRecordNotAuth, err := getUserRecordFromName("userB")
	require.NoError(t, err)
	tokenUserNotAuth, err := userRecordNotAuth.NewAuthToken()
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "get clone-record success",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials"
			}`),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"cloned_record"`,
				`"message"`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with invalid JSON",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials",`), // JSON malformato
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Invalid JSON."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with empty collection",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": ""
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Id and collection are required."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with not supported collection",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "coll"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Collection 'coll' not supported for cloning."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "record not found in collection",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": token,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567891",
				"collection": "credentials"
			}`),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"reason":"Record 'crede1234567891' not found in collection 'credentials'"`,
				`"message":"sql: no rows in result set"`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone without authentication",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Authentication required."`,
			},
			TestAppFactory: setupApp(orgID),
		},
		{
			Name:   "clone with unauthorized user",
			Method: "POST",
			URL:    "/api/clone-record",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": tokenUserNotAuth,
			},
			Body: strings.NewReader(`{
				"id": "crede1234567890",
				"collection": "credentials"
			}`),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Not authorized for this organization."`,
			},
			TestAppFactory: setupApp(orgID),
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
