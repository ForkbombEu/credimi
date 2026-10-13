// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package canonify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func generateToken(collectionNameOrID string, email string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	record, err := app.FindAuthRecordByEmail(collectionNameOrID, email)
	if err != nil {
		return "", err
	}

	return record.NewAuthToken()
}
func getUserIDFromEmail(collectionNameOrID string, email string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	record, err := app.FindAuthRecordByEmail(collectionNameOrID, email)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}
func getOrgIDfromName(collectionNameOrID string, name string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := fmt.Sprintf(`name="%s"`, name)

	record, err := app.FindFirstRecordByFilter(collectionNameOrID, filter)
	if err != nil {
		return "", err
	}
	return record.Id, nil
}

func jsonBody(data map[string]any) *bytes.Reader {
	b, _ := json.Marshal(data)
	return bytes.NewReader(b)
}

func TestCanonifyAPI(t *testing.T) {
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		RegisterCanonifyHooks(testApp)

		return testApp
	}
	authTokenA, _ := generateToken("users", "userA@example.org")
	authTokenB, _ := generateToken("users", "userB@example.org")
	userID, _ := getUserIDFromEmail("users", "userA@example.org")
	orgAID, _ := getOrgIDfromName("organizations", "userA's organization")
	orgBID, _ := getOrgIDfromName("organizations", "userB's organization")

	scenarios := []tests.ApiScenario{
		{
			Name:   "create user with auto canonify",
			Method: http.MethodPost,
			URL:    "/api/collections/users/records",
			Body: jsonBody(
				map[string]any{
					"name":            "Alice Test",
					"password":        "12345678",
					"passwordConfirm": "12345678",
				},
			),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"alice-test"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "create user cannot override canonified_name",
			Method: http.MethodPost,
			URL:    "/api/collections/users/records",
			Body: jsonBody(
				map[string]any{
					"name":            "Bob",
					"password":        "12345678",
					"passwordConfirm": "12345678",
					"canonified_name": "override",
				},
			),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"bob"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "update user name updates canonified_name",
			Method: http.MethodPatch,
			URL:    "/api/collections/users/records/" + userID,
			Body:   jsonBody(map[string]any{"name": "Alice 2"}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"alice-2"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "cannot update canonified_name manually",
			Method: http.MethodPatch,
			URL:    "/api/collections/users/records/" + userID,
			Body: jsonBody(map[string]any{
				"canonified_name": "change-name",
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"usera"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "update unrelated field keeps canonified_name",
			Method: http.MethodPatch,
			URL:    "/api/collections/users/records/" + userID,
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			Body:           jsonBody(map[string]any{"username": "users111111"}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"usera"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "create credentials issuer with auto canonify",
			Method: http.MethodPost,
			URL:    "/api/collections/credential_issuers/records",
			Body: jsonBody(map[string]any{
				"name":        "New Issuer Test 😀",
				"url":         "https://example.com",
				"description": "A simple credential issuer",
				"owner":       orgAID,
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"new-issuer-test"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "create credentials issuer cannot override canonified_name",
			Method: http.MethodPost,
			URL:    "/api/collections/credential_issuers/records",
			Body: jsonBody(map[string]any{
				"name":            "Issuer Override",
				"url":             "https://example.com",
				"description":     "Another issuer",
				"owner":           orgAID,
				"canonified_name": "force-this",
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"issuer-override"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "update credential issuer name updates canonified_name",
			Method: http.MethodPatch,
			URL:    "/api/collections/credential_issuers/records/10dsg8625060x12",
			Body: jsonBody(map[string]any{
				"name": "Updated Issuer",
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"updated-issuer"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "update credential issuer unrelated field keeps canonified_name",
			Method: http.MethodPatch,
			URL:    "/api/collections/credential_issuers/records/10dsg8625060x12",
			Body: jsonBody(map[string]any{
				"description": "Modified description only",
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"test-issuer"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "entities can have same canonified_name if different parents",
			Method: http.MethodPost,
			URL:    "/api/collections/credential_issuers/records",
			Body: jsonBody(map[string]any{
				"name":        "TEST ISSUER",
				"url":         "https://example.com",
				"description": "A simple credential issuer",
				"owner":       orgBID,
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenB,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"test-issuer"`,
			},
			TestAppFactory: setupTestApp,
		},
		{
			Name:   "entities cannot have same canonified_name if same parent",
			Method: http.MethodPost,
			URL:    "/api/collections/credential_issuers/records",
			Body: jsonBody(map[string]any{
				"name":        "TEST ISSUER",
				"url":         "https://example.com",
				"description": "A simple credential issuer",
				"owner":       orgAID,
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authTokenA,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"test-issuer-1"`,
			},
			TestAppFactory: setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestMakeExistsFunc(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	const orgID = "co35481b68u3zj3"
	const issuerID = "10dsg8625060x12"
	collection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	cases := []struct {
		name       string
		collection string
		owner      string
		excludeID  string
		candidate  string
		want       bool
		wantErr    string
	}{
		{name: "free name", collection: "credential_issuers", owner: orgID, candidate: "free-name"},
		{
			name:       "taken name",
			collection: "credential_issuers",
			owner:      orgID,
			candidate:  "test-issuer",
			want:       true,
		},
		{
			name:       "name taken by the excluded record",
			collection: "credential_issuers",
			owner:      orgID,
			excludeID:  issuerID,
			candidate:  "test-issuer",
		},
		{
			name:       "missing parent",
			collection: "credential_issuers",
			owner:      "missing-org",
			candidate:  "free-name",
			wantErr:    "build path for credential_issuers",
		},
		{
			name:       "unknown collection",
			collection: "not_canonified",
			owner:      orgID,
			candidate:  "free-name",
			wantErr:    `no path template for collection "not_canonified"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := core.NewRecord(collection)
			record.Set("owner", tc.owner)
			taken, err := MakeExistsFunc(app, tc.collection, record, tc.excludeID)(tc.candidate)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, taken)
		})
	}
}

func TestCanonifyHookFailsFastOnMissingParent(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	RegisterCanonifyHooks(app)

	collection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("name", "Orphan issuer")
	record.Set("url", "https://orphan.example.com")
	record.Set("owner", "missing-org")

	err = app.Save(record)
	require.ErrorContains(t, err, "build path for credential_issuers")
}
