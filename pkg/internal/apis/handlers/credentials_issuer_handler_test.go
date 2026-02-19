// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	credential_workflow "github.com/forkbombeu/credimi/pkg/credential_issuer/workflow"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

type TestSetupData struct {
	IssuerID          string
	CreateIssuer      bool
	CreateCredentials []TestCredential
}

type TestCredential struct {
	Name        string
	IssuerID    string
	CredKey     string
	Format      string
	DisplayName string
	Locale      string
	Description string
	LogoURI     string
	Conformant  bool
}

func getOrgIDfromName(name string) (string, error) { //nolint
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := fmt.Sprintf(`name="%s"`, name)

	record, err := app.FindFirstRecordByFilter("organizations", filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func jsonBody(data map[string]any) *bytes.Reader {
	b, _ := json.Marshal(data)
	return bytes.NewReader(b)
}

func setupTestAppWithData(orgID string, setupData TestSetupData) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(testApp)
		IssuerTemporalInternalRoutes.Add(testApp)

		var issuerID string

		// Create issuer if needed
		if setupData.CreateIssuer {
			issuerCollection, err := testApp.FindCollectionByNameOrId("credential_issuers")
			require.NoError(t, err)

			issuerRecord := core.NewRecord(issuerCollection)
			if setupData.IssuerID != "" {
				issuerRecord.Set("id", setupData.IssuerID)
			}
			issuerRecord.Set("url", "https://test-issuer.example.com")
			issuerRecord.Set("name", "Test Issuer")
			issuerRecord.Set("owner", orgID)
			issuerRecord.Set("imported", true)
			err = testApp.Save(issuerRecord)
			require.NoError(t, err)
			issuerID = issuerRecord.Id
		}

		// Create credentials if needed
		if len(setupData.CreateCredentials) > 0 && issuerID != "" {
			credCollection, err := testApp.FindCollectionByNameOrId("credentials")
			require.NoError(t, err)

			for _, cred := range setupData.CreateCredentials {
				credRecord := core.NewRecord(credCollection)

				// Build credential JSON
				credJSON := map[string]any{
					"format": cred.Format,
					"display": []any{
						map[string]any{
							"name":        cred.DisplayName,
							"locale":      cred.Locale,
							"description": cred.Description,
						},
					},
				}

				// Add logo if provided
				if cred.LogoURI != "" {
					display := credJSON["display"].([]any)[0].(map[string]any)
					display["logo"] = map[string]any{
						"uri": cred.LogoURI,
					}
				}

				jsonBytes, _ := json.Marshal(credJSON)

				credRecord.Set("name", cred.CredKey)
				credRecord.Set("display_name", cred.DisplayName)
				credRecord.Set("locale", cred.Locale)
				credRecord.Set("description", cred.Description)
				credRecord.Set("logo_url", cred.LogoURI)
				credRecord.Set("format", cred.Format)
				credRecord.Set("json", string(jsonBytes))
				credRecord.Set("credential_issuer", issuerID)
				credRecord.Set("conformant", cred.Conformant)
				credRecord.Set("owner", orgID)

				err = testApp.Save(credRecord)
				require.NoError(t, err)
			}
		}

		return testApp
	}
}

func TestCredentialIssuersAPI(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "store new credential",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "university-degree",
				"credential": map[string]any{
					"format": "jwt_vc_json",
					"display": []any{
						map[string]any{
							"name":        "University Degree",
							"locale":      "en-US",
							"description": "A university degree credential",
							"logo": map[string]any{
								"uri": "https://example.com/logo.png",
							},
						},
					},
				},
				"conformant": true,
				"orgID":      orgID,
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"key":"university-degree"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
				IssuerID:     "issuerid1234567",
			}),
		},
		{
			Name:   "update existing credential",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "university-degree",
				"credential": map[string]any{
					"format": "jwt_vc_json",
					"display": []any{
						map[string]any{
							"name":        "University Degree",
							"locale":      "en-US",
							"description": "An updated university degree credential",
							"logo": map[string]any{
								"uri": "https://example.com/new-logo.png",
							},
						},
					},
				},
				"conformant": true,
				"orgID":      orgID,
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"key":"university-degree"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Description: "A university degree credential",
						LogoURI:     "https://example.com/logo.png",
						Conformant:  true,
					},
				},
			}),
		},
		{
			Name:   "store credential without logo",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "simple-credential",
				"credential": map[string]any{
					"format": "vc+sd-jwt",
					"display": []any{
						map[string]any{
							"name":   "Simple Credential",
							"locale": "en-US",
						},
					},
				},
				"conformant": false,
				"orgID":      orgID,
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"key":"simple-credential"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
			}),
		},
		{
			Name:   "cleanup credentials - delete stale ones",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/cleanup-credentials",
			Body: jsonBody(map[string]any{
				"issuerID":  "issuerid1234567",
				"validKeys": []string{"university-degree"},
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"deleted"`,
				`"simple-credential"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Conformant:  true,
					},
					{
						CredKey:     "simple-credential",
						Format:      "vc+sd-jwt",
						DisplayName: "Simple Credential",
						Locale:      "en-US",
						Conformant:  false,
					},
				},
			}),
		},
		{
			Name:   "cleanup credentials - no deletions when all valid",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/cleanup-credentials",
			Body: jsonBody(map[string]any{
				"issuerID":  "issuerid1234567",
				"validKeys": []string{"university-degree"},
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"deleted":null`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{IssuerID: "issuerid1234567",
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Conformant:  true,
					},
				},
			}),
		},
		{
			Name:   "cleanup credentials - delete all",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/cleanup-credentials",
			Body: jsonBody(map[string]any{
				"issuerID":  "issuerid1234567",
				"validKeys": []string{},
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"deleted"`,
				`"university-degree"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{
						IssuerID:    "issuerid1234567",
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Conformant:  true,
					},
				},
			}),
		},
		{
			Name:   "store credential with wrong orgID",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "invalid-org-credential",
				"credential": map[string]any{
					"format": "jwt_vc_json",
					"display": []any{
						map[string]any{
							"name":        "Invalid Org Credential",
							"locale":      "en-US",
							"description": "Should fail because of wrong orgID",
						},
					},
				},
				"conformant": true,
				"orgID":      "nonexistent-org-id", // <-- invalid orgID
			}),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"credentials"`,
				`"failed to save credentials"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
				IssuerID:     "issuerid1234567",
			}),
		},
		{
			Name:           "store credential with invalid JSON body",
			Method:         http.MethodPost,
			URL:            "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body:           bytes.NewReader([]byte(`{invalid json}`)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"reason":"Invalid JSON format for the expected type"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
			}),
		},
		{
			Name:           "cleanup credentials with invalid JSON body",
			Method:         http.MethodPost,
			URL:            "/api/credentials_issuers/cleanup-credentials",
			Body:           bytes.NewReader([]byte(`{invalid json}`)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"reason":"Invalid JSON format for the expected type"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
			}),
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestReadSchemaFile(t *testing.T) {
	tempDir := t.TempDir()
	path := tempDir + "/schema.json"
	require.NoError(t, os.WriteFile(path, []byte(`{"type":"object"}`), 0o600))

	content, apiErr := readSchemaFile(path)
	require.Nil(t, apiErr)
	require.Contains(t, content, `"type":"object"`)

	_, apiErr = readSchemaFile(tempDir + "/missing.json")
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "failed to read  JSON schema file", apiErr.Reason)
}

func TestCheckWellKnownEndpoints(t *testing.T) {
	ctx := context.Background()

	err := checkWellKnownEndpoints(ctx, "http://127.0.0.1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "neither .well-known")

	err = checkWellKnownEndpoints(ctx, "http://127.0.0.1/.well-known/openid-federation")
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not accessible")
}

func TestHandleCredentialIssuerStartCheckBadURL(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"::::://bad-url"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCredentialIssuerStartCheckSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgID, err := GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)

	issuerCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	issuerRecord := core.NewRecord(issuerCollection)
	issuerRecord.Set("url", "https://issuer.example.com")
	issuerRecord.Set("name", "Existing Issuer")
	issuerRecord.Set("owner", orgID)
	issuerRecord.Set("imported", true)
	require.NoError(t, app.Save(issuerRecord))

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForPartialResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForPartialResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-issuer",
			WorkflowRunID: "run-issuer",
		}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForPartialResult = func(
		client.Client,
		string,
		string,
		string,
		time.Duration,
		time.Duration,
	) (map[string]any, error) {
		return map[string]any{
			"issuerName":        "Issuer Name",
			"logo":              "https://logo.example.com/logo.png",
			"credentialsNumber": 2.0,
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.Equal(t, float64(2), payload["credentialsNumber"])

	updated, err := app.FindRecordById("credential_issuers", issuerRecord.Id)
	require.NoError(t, err)
	require.NotEmpty(t, updated.GetString("workflow_url"))
}

func TestHandleCredentialIssuerStartCheckReadSchemaErrorAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) {
		return "", apierror.New(http.StatusBadRequest, "schema", "bad", "bad")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCredentialIssuerStartCheckTemporalClientErrorAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf", WorkflowRunID: "run"}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.client.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleCredentialIssuerStartCheckUsesHostnameFallbackAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForPartialResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForPartialResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf", WorkflowRunID: "run"}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForPartialResult = func(
		client.Client,
		string,
		string,
		string,
		time.Duration,
		time.Duration,
	) (map[string]any, error) {
		return map[string]any{
			"issuerName":        "",
			"logo":              "",
			"credentialsNumber": 1.0,
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.fallback.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	orgID, err := GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	records, err := app.FindRecordsByFilter(
		"credential_issuers",
		"url = {:url} && owner = {:owner}",
		"",
		1,
		0,
		map[string]any{"url": "https://issuer.fallback.example.com", "owner": orgID},
	)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "issuer.fallback.example.com", records[0].GetString("name"))
}

func TestHandleCredentialIssuerStartCheckExistingRecordStartErrorAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgID, err := GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)

	issuerCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	record := core.NewRecord(issuerCollection)
	record.Set("url", "https://issuer.existing.example.com")
	record.Set("name", "Existing")
	record.Set("owner", orgID)
	record.Set("imported", true)
	require.NoError(t, app.Save(record))

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("boom")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.existing.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	_, err = app.FindRecordById("credential_issuers", record.Id)
	require.NoError(t, err)
}

func TestHandleCredentialIssuerStoreOrUpdateInvalidJSONAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/store-or-update-extracted-credentials",
		bytes.NewBufferString("{bad-json"),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStoreOrUpdateExtractedCredentials()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
}

func TestHandleCredentialIssuerStoreOrUpdateInvalidSavedJSONAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	issuer := core.NewRecord(issuerColl)
	issuer.Set("name", "Issuer")
	issuer.Set("owner", orgID)
	issuer.Set("url", "https://issuer.example.com")
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	credColl, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	cred := core.NewRecord(credColl)
	cred.Set("credential_issuer", issuer.Id)
	cred.Set("name", "cred-3")
	cred.Set("display_name", "Old Name")
	cred.Set("logo_url", "https://old.logo")
	cred.Set("json", `not-json`)
	cred.Set("owner", orgID)
	require.NoError(t, app.Save(cred))

	body := map[string]any{
		"issuerID": issuer.Id,
		"credKey":  "cred-3",
		"credential": map[string]any{
			"display": []any{
				map[string]any{
					"name": "New Name",
				},
			},
		},
		"conformant": true,
		"orgID":      orgID,
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/store-or-update-extracted-credentials",
		bytes.NewBuffer(payload),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStoreOrUpdateExtractedCredentials()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleCredentialIssuerCleanupCredentialsInvalidJSONAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/cleanup-credentials",
		bytes.NewBufferString("{bad-json"),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerCleanupCredentials()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
}

func TestHandleCredentialIssuerStartCheckWorkflowErrorDeletesNewRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, fmt.Errorf("boom")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.new.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	orgID, err := GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	issuers, err := app.FindRecordsByFilter(
		"credential_issuers",
		"url = {:url} && owner = {:owner}",
		"",
		0,
		0,
		map[string]any{"url": "https://issuer.new.example.com", "owner": orgID},
	)
	require.NoError(t, err)
	require.Empty(t, issuers)
}

func TestHandleCredentialIssuerStartCheckWaitErrorDeletesNewRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForPartialResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForPartialResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-issuer",
			WorkflowRunID: "run-issuer",
		}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForPartialResult = func(
		client.Client,
		string,
		string,
		string,
		time.Duration,
		time.Duration,
	) (map[string]any, error) {
		return nil, fmt.Errorf("wait failed")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.wait.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	orgID, err := GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	issuers, err := app.FindRecordsByFilter(
		"credential_issuers",
		"url = {:url} && owner = {:owner}",
		"",
		0,
		0,
		map[string]any{"url": "https://issuer.wait.example.com", "owner": orgID},
	)
	require.NoError(t, err)
	require.Empty(t, issuers)
}

func TestHandleCredentialIssuerStartCheckInvalidCredentialsNumber(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origCheck := credentialIssuerCheckWellKnownEndpoints
	origRead := credentialIssuerReadSchemaFile
	origStart := credentialIssuerStartWorkflow
	origClient := credentialIssuerTemporalClient
	origWait := credentialIssuerWaitForPartialResult
	t.Cleanup(func() {
		credentialIssuerCheckWellKnownEndpoints = origCheck
		credentialIssuerReadSchemaFile = origRead
		credentialIssuerStartWorkflow = origStart
		credentialIssuerTemporalClient = origClient
		credentialIssuerWaitForPartialResult = origWait
	})

	credentialIssuerCheckWellKnownEndpoints = func(context.Context, string) error { return nil }
	credentialIssuerReadSchemaFile = func(string) (string, *apierror.APIError) { return "schema", nil }
	credentialIssuerStartWorkflow = func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-issuer",
			WorkflowRunID: "run-issuer",
		}, nil
	}
	credentialIssuerTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	credentialIssuerWaitForPartialResult = func(
		client.Client,
		string,
		string,
		string,
		time.Duration,
		time.Duration,
	) (map[string]any, error) {
		return map[string]any{"issuerName": "Issuer", "credentialsNumber": "bad"}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/start-check",
		bytes.NewBufferString(`{"credentialIssuerUrl":"https://issuer.bad.example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleCredentialIssuerStoreOrUpdateExtractedCredentials(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	issuer := core.NewRecord(issuerColl)
	issuer.Set("name", "Issuer")
	issuer.Set("owner", orgID)
	issuer.Set("url", "https://issuer.example.com")
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	body := map[string]any{
		"issuerID": issuer.Id,
		"credKey":  "cred-1",
		"credential": map[string]any{
			"format": "jwt",
			"display": []any{
				map[string]any{
					"name":        "Credential One",
					"description": "desc",
					"logo": map[string]any{
						"uri": "https://logo.example.com/logo.png",
					},
				},
			},
		},
		"conformant": true,
		"orgID":      orgID,
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/store-or-update-extracted-credentials",
		bytes.NewBuffer(payload),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStoreOrUpdateExtractedCredentials()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	record, err := app.FindFirstRecordByFilter(
		"credentials",
		"name = {:key} && credential_issuer = {:issuerID}",
		map[string]any{"key": "cred-1", "issuerID": issuer.Id},
	)
	require.NoError(t, err)
	require.Equal(t, "Credential One", record.GetString("display_name"))
	require.Equal(t, "jwt", record.GetString("format"))
}

func TestHandleCredentialIssuerStoreOrUpdateUpdatesExistingRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	issuer := core.NewRecord(issuerColl)
	issuer.Set("name", "Issuer")
	issuer.Set("owner", orgID)
	issuer.Set("url", "https://issuer.example.com")
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	credColl, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	cred := core.NewRecord(credColl)
	cred.Set("credential_issuer", issuer.Id)
	cred.Set("name", "cred-2")
	cred.Set("display_name", "Old Name")
	cred.Set("logo_url", "https://old.logo")
	cred.Set("owner", orgID)
	cred.Set("json", `{"display":[{"name":"Old Name","logo":{"uri":"https://old.logo"}}]}`)
	require.NoError(t, app.Save(cred))

	body := map[string]any{
		"issuerID": issuer.Id,
		"credKey":  "cred-2",
		"credential": map[string]any{
			"display": []any{
				map[string]any{
					"name": "New Name",
					"logo": map[string]any{"uri": "https://new.logo"},
				},
			},
		},
		"conformant": false,
		"orgID":      orgID,
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/store-or-update-extracted-credentials",
		bytes.NewBuffer(payload),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerStoreOrUpdateExtractedCredentials()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	updated, err := app.FindRecordById("credentials", cred.Id)
	require.NoError(t, err)
	require.Equal(t, "New Name", updated.GetString("display_name"))
	require.Equal(t, "https://new.logo", updated.GetString("logo_url"))
}

func TestHandleCredentialIssuerCleanupCredentials(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	issuer := core.NewRecord(issuerColl)
	issuer.Set("name", "Issuer")
	issuer.Set("owner", orgID)
	issuer.Set("url", "https://issuer.example.com")
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	credColl, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	credRecord := core.NewRecord(credColl)
	credRecord.Set("name", "delete")
	credRecord.Set("credential_issuer", issuer.Id)
	credRecord.Set("owner", orgID)
	require.NoError(t, app.Save(credRecord))

	body := fmt.Sprintf(`{"issuerID":"%s","validKeys":[]}`, issuer.Id)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/credentials_issuers/cleanup-credentials",
		bytes.NewBufferString(body),
	)
	rec := httptest.NewRecorder()

	err = HandleCredentialIssuerCleanupCredentials()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	_, err = app.FindRecordById("credentials", credRecord.Id)
	require.Error(t, err)
}

func TestIsPrivateIP(t *testing.T) {
	require.True(t, isPrivateIP(net.IPv4(10, 0, 0, 1)))
	require.True(t, isPrivateIP(net.IPv4(192, 168, 1, 1)))
	require.True(t, isPrivateIP(net.ParseIP("::1")))
	require.False(t, isPrivateIP(net.IPv4(8, 8, 8, 8)))
}

func TestCheckEndpointExistsInvalidURL(t *testing.T) {
	err := checkEndpointExists(context.Background(), "://bad")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

func TestCheckEndpointExistsUnsupportedScheme(t *testing.T) {
	err := checkEndpointExists(context.Background(), "ftp://example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported URL scheme")
}

func TestParseCredentialDisplay(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]any
		wantName   string
		wantLocale string
		wantLogo   string
		wantDesc   string
	}{
		{
			name: "full display with logo",
			input: map[string]any{
				"display": []any{
					map[string]any{
						"name":        "University Degree",
						"locale":      "en-US",
						"description": "A degree credential",
						"logo": map[string]any{
							"uri": "https://example.com/logo.png",
						},
					},
				},
			},
			wantName:   "University Degree",
			wantLocale: "en-US",
			wantLogo:   "https://example.com/logo.png",
			wantDesc:   "A degree credential",
		},
		{
			name: "display without logo",
			input: map[string]any{
				"display": []any{
					map[string]any{
						"name":        "Simple Credential",
						"locale":      "en-GB",
						"description": "No logo provided",
					},
				},
			},
			wantName:   "Simple Credential",
			wantLocale: "en-GB",
			wantLogo:   "",
			wantDesc:   "No logo provided",
		},
		{
			name: "display without description",
			input: map[string]any{
				"display": []any{
					map[string]any{
						"name":   "Name Only",
						"locale": "fr-FR",
					},
				},
			},
			wantName:   "Name Only",
			wantLocale: "fr-FR",
			wantLogo:   "",
			wantDesc:   "",
		},
		{
			name: "empty display list",
			input: map[string]any{
				"display": []any{},
			},
			wantName:   "",
			wantLocale: "",
			wantLogo:   "",
			wantDesc:   "",
		},
		{
			name: "missing display field",
			input: map[string]any{
				"format": "jwt_vc_json",
			},
			wantName:   "",
			wantLocale: "",
			wantLogo:   "",
			wantDesc:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotLocale, gotLogo, gotDesc := parseCredentialDisplay(tt.input)

			require.Equal(t, tt.wantName, gotName, "name mismatch")
			require.Equal(t, tt.wantLocale, gotLocale, "locale mismatch")
			require.Equal(t, tt.wantLogo, gotLogo, "logo mismatch")
			require.Equal(t, tt.wantDesc, gotDesc, "description mismatch")
		})
	}
}

func TestHookUpdateCredentialsIssuersCreatesSchedule(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	origDial := credentialsIssuerTemporalDial
	t.Cleanup(func() {
		credentialsIssuerTemporalDial = origDial
	})

	mockClient := &temporalmocks.Client{}
	mockScheduleClient := &temporalmocks.ScheduleClient{}
	mockHandle := &temporalmocks.ScheduleHandle{}

	var capturedOptions client.ScheduleOptions
	mockScheduleClient.
		On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedOptions = args.Get(1).(client.ScheduleOptions)
		}).
		Return(mockHandle, nil).
		Once()
	mockClient.On("ScheduleClient").Return(mockScheduleClient).Once()
	mockClient.On("Close").Return(nil).Once()
	mockHandle.On("Describe", mock.Anything).Return(&client.ScheduleDescription{}, nil).Once()

	credentialsIssuerTemporalDial = func(_ client.Options) (client.Client, error) {
		return mockClient, nil
	}

	collection, err := app.FindCollectionByNameOrId("features")
	require.NoError(t, err)

	record := core.NewRecord(collection)
	record.Set("name", "updateIssuers")
	record.Set("active", true)
	record.Set("envVariables", types.JSONRaw(`{"interval":"daily"}`))

	event := &core.RecordEvent{}
	event.Record = record
	err = handleUpdateCredentialsIssuers(event)
	require.NoError(t, err)

	require.NotEmpty(t, capturedOptions.ID)
	require.Len(t, capturedOptions.Spec.Intervals, 1)
	require.Equal(t, 24*time.Hour, capturedOptions.Spec.Intervals[0].Every)
	action, ok := capturedOptions.Action.(*client.ScheduleWorkflowAction)
	require.True(t, ok)
	require.NotNil(t, action.Workflow)
	require.Equal(t, credential_workflow.FetchIssuersTaskQueue, action.TaskQueue)
}
