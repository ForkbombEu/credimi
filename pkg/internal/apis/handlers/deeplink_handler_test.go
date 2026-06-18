// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestHandleGetDeeplinkInvalidJSON(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
}

func TestHandleGetDeeplinkWaitError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	restore := installDeeplinkSeams(t)
	defer restore()

	deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	deeplinkTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	deeplinkWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("wait failed")
	}

	body, _ := json.Marshal(CredentialDeeplinkRequest{Yaml: "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "failed to get workflow result")
}

func TestHandleGetDeeplinkMalformedOutput(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	restore := installDeeplinkSeams(t)
	defer restore()

	deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	deeplinkTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	deeplinkWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{Output: "invalid"}, nil
	}

	body, _ := json.Marshal(CredentialDeeplinkRequest{Yaml: "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "output is not an array")
}

func TestHandleGetDeeplinkSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	restore := installDeeplinkSeams(t)
	defer restore()

	var capturedInput workflowengine.WorkflowInput
	deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		capturedInput = input
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	deeplinkTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	deeplinkWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			Output: []any{
				map[string]any{
					"steps": []any{
						map[string]any{
							"captures": map[string]any{
								"deeplink": "credimi://link",
							},
						},
					},
				},
			},
		}, nil
	}

	body, _ := json.Marshal(CredentialDeeplinkRequest{
		Yaml:    "test",
		Secrets: "secret1: value1\nsecret2: value2\n",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "credimi://link")
	require.Equal(t, map[string]any{
		"secret1": "value1",
		"secret2": "value2",
	}, capturedInput.Secrets)
}

func setupDeeplinkApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(app)
		DeepLinkRoutes.Add(app)

		issuerColl, _ := app.FindCollectionByNameOrId("credential_issuers")
		issuerRecord := core.NewRecord(issuerColl)
		issuerRecord.Set("id", "issuer123456789")
		issuerRecord.Set("name", "test issuer")
		issuerRecord.Set("url", "https://test-issuer.example.com")
		issuerRecord.Set("owner", orgID)
		require.NoError(t, app.Save(issuerRecord))

		credColl, _ := app.FindCollectionByNameOrId("credentials")
		credColl, _ = app.FindCollectionByNameOrId("credentials")
		credential := core.NewRecord(credColl)
		credential.Set("owner", orgID)
		credential.Set("name", "test credential")
		credential.Set("credential_issuer", issuerRecord.Id)
		credential.Set("deeplink", "openid-credential-offer://...")
		require.NoError(t, app.Save(credential))

		verifierColl, _ := app.FindCollectionByNameOrId("verifiers")
		verifierRecord := core.NewRecord(verifierColl)
		verifierRecord.Set("id", "verify123456789")
		verifierRecord.Set("name", "test verifier")
		verifierRecord.Set("url", "https://test-verifier.example.com")
		verifierRecord.Set("standard_and_version", "openid4vci_wallet/draft-15")
		verifierRecord.Set("owner", orgID)
		verifierRecord.Set("format", "mDOC")
		verifierRecord.Set("signing_algorithms", "EdDSA")
		verifierRecord.Set("cryptographic_binding_methods", "jwk")
		verifierRecord.Set("description", "des")
		require.NoError(t, app.Save(verifierRecord))

		useCaseColl, _ := app.FindCollectionByNameOrId("use_cases_verifications")
		useCaseColl, _ = app.FindCollectionByNameOrId("use_cases_verifications")
		useCase := core.NewRecord(useCaseColl)
		useCase.Set("owner", orgID)
		useCase.Set("name", "test use cases")
		useCase.Set("verifier", verifierRecord.Id)
		useCase.Set("yaml", "my yaml")
		useCase.Set("deeplink", "openid-verifier://...")
		require.NoError(t, app.Save(useCase))

		return app
	}
}


func TestGetCredentialDeeplink(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	var capturedInput workflowengine.WorkflowInput
	installSuccessfulDeeplinkWorkflow(t, &capturedInput)

	scenarios := []tests.ApiScenario{
		{
			Name:           "get credential deeplink-success",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential",
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"openid-credential-offer://",
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get credential deeplink - missing id",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink",
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"error":"request"`,
				`"reason":"missing credential id"`,
				`"message":"id parameter is required"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get credential deeplink - invalid credential path",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink?id=usera-s-organization/test-issuer-2/test-credential",
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"error":"resolve"`,
				`"reason":"failed to resolve credential path"`,
				`"message":"sql: no rows in result set"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get credential deeplink - rejects non-credential record",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink?id=usera-s-organization/test-verifier/test-use-cases",
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"error":"record"`,
				`"reason":"invalid record type"`,
				`"message":"id must resolve to a credentials record"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get credential deeplink - redirect",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential&redirect=true",
			ExpectedStatus: http.StatusMovedPermanently,
			TestAppFactory: setupDeeplinkApp(orgID),
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				require.Equal(t.(*testing.T), "openid-credential-offer://...", res.Header.Get("Location"))
			},
		},
		{
			Name:           "get credential deeplink - empty deeplink",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential",
			ExpectedStatus: http.StatusInternalServerError,
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
			Name:           "get credential deeplink with yaml - success",
			Method:         http.MethodGet,
			URL:            "/api/credential/deeplink?id=usera-s-organization/test-issuer-1/test-credential",
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`mock-deeplink-from-yaml`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)
				coll, _ := app.FindCollectionByNameOrId("credentials")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test credential"`)
				r.Set("deeplink", "")
				r.Set("yaml", "test: yaml content")
				r.Set("secrets", "token: credential-secret\n")
				require.NoError(t, app.Save(r))

				return app
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				require.Equal(t.(*testing.T), map[string]any{
					"token": "credential-secret",
				}, capturedInput.Secrets)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestGetVerificationDeeplink(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	var capturedInput workflowengine.WorkflowInput
	installSuccessfulDeeplinkWorkflow(t, &capturedInput)

	scenarios := []tests.ApiScenario{
		{
			Name:           "get verification deeplink - missing id",
			Method:         http.MethodGet,
			URL:            "/api/verification/deeplink",
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"error":"request"`,
				`"reason":"missing record id"`,
				`"message":"id parameter is required"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get verification deeplink - invalid verification path",
			Method:         http.MethodGet,
			URL:            "/api/verification/deeplink?id=usera-s-organization/test-verifier-2/test-use-cases",
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"error":"resolve"`,
				`"reason":"failed to resolve verification path"`,
				`"message":"sql: no rows in result set"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get verification deeplink - rejects non-use-case-verification record",
			Method:         http.MethodGet,
			URL:            "/api/verification/deeplink?id=usera-s-organization/test-issuer-1/test-credential",
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"error":"record"`,
				`"reason":"invalid record type"`,
				`"message":"id must resolve to a use_cases_verifications record"`,
			},
			TestAppFactory: setupDeeplinkApp(orgID),
		},
		{
			Name:           "get verification deeplink with yaml - success",
			Method:         http.MethodGet,
			URL:            "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases",
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`mock-deeplink-from-yaml`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupDeeplinkApp(orgID)(t)
				coll, _ := app.FindCollectionByNameOrId("use_cases_verifications")
				r, _ := app.FindFirstRecordByFilter(coll.Name, `name="test use cases"`)
				r.Set("secrets", "pin: '1234'\n")
				require.NoError(t, app.Save(r))

				return app
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				require.Equal(t.(*testing.T), map[string]any{
					"pin": "1234",
				}, capturedInput.Secrets)
			},
		},
		{
			Name:           "get verification deeplink - redirect",
			Method:         http.MethodGet,
			URL:            "/api/verification/deeplink?id=usera-s-organization/test-verifier/test-use-cases&redirect=true",
			ExpectedStatus: http.StatusMovedPermanently,
			TestAppFactory: setupDeeplinkApp(orgID),
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				require.Equal(t.(*testing.T), "mock-deeplink-from-yaml", res.Header.Get("Location"))
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func installSuccessfulDeeplinkWorkflow(t testing.TB, capturedInput *workflowengine.WorkflowInput) {
	t.Helper()

	restore := installDeeplinkSeams(t)
	t.Cleanup(restore)

	deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		if capturedInput != nil {
			*capturedInput = input
		}
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	deeplinkTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	deeplinkWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			Output: []any{
				map[string]any{
					"steps": []any{
						map[string]any{
							"captures": map[string]any{
								"deeplink": "mock-deeplink-from-yaml",
							},
						},
					},
				},
			},
		}, nil
	}
}

func installDeeplinkSeams(t testing.TB) func() {
	t.Helper()

	origStart := deeplinkStartWorkflow
	origClient := deeplinkTemporalClient
	origWait := deeplinkWaitForWorkflowResult

	return func() {
		deeplinkStartWorkflow = origStart
		deeplinkTemporalClient = origClient
		deeplinkWaitForWorkflowResult = origWait
	}
}
