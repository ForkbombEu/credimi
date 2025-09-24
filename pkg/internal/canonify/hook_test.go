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
	authToken, _ := generateToken("users", "userA@example.org")
	userID, _ := getUserIDFromEmail("users", "userA@example.org")
	orgID, _ := getOrgIDfromName("organizations", "userA's organization")

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
			URL:    "/api/collections/users/records/" + userID, // replace 1 with a real test record ID
			Body:   jsonBody(map[string]any{"name": "Alice 2"}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authToken,
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
				"Authorization": authToken,
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
				"Authorization": authToken,
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
				"owner":       orgID,
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authToken,
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
				"owner":           orgID,
				"canonified_name": "force-this",
			}),
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": authToken,
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
				"Authorization": authToken,
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
				"Authorization": authToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"canonified_name":"test-issuer"`,
			},
			TestAppFactory: setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
