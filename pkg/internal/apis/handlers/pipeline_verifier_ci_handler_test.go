// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const verifierCIStepCIYAML = `version: "1.1"
name: Verifier Capture
env:
  host: https://verifier.example/old
  body: useCaseId=pid
tests:
  example:
    steps: []
`

func TestPipelineRunVerifierAcceptsCredimiAPIKeyForOwnedPipeline(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	origStart := startPipelineWorkflow
	t.Cleanup(func() {
		startPipelineWorkflow = origStart
	})
	startPipelineWorkflow = func(
		yaml string,
		config map[string]any,
		memo map[string]any,
		pipelineIdentifier string,
	) (workflowengine.WorkflowResult, error) {
		require.Contains(
			t,
			yaml,
			"use_case_id: usera-s-organization/verifier-ci/pid-abc123",
		)
		require.NotContains(
			t,
			yaml,
			"use_case_id: usera-s-organization/verifier-ci/pid\n",
		)
		require.Contains(t, config, verifierCITempUseCasesConfigKey)
		return workflowengine.WorkflowResult{
			WorkflowID:    "verifier-wf",
			WorkflowRunID: "verifier-run",
		}, nil
	}

	app := setupPipelineVerifierCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	useCaseID := createVerifierCIUseCase(
		t,
		app,
		orgID,
		"Verifier CI",
		"pid",
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"verifier-ci-pipeline",
		verifierCIPipelineYAML(useCaseID),
		false,
	)

	rec := performPipelineVerifierCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/verifier-ci-pipeline",
		"commit_sha":          "abc123",
		"verifier_url":        "https://verifier.example/temp",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"running"`)
	require.Contains(t, rec.Body.String(), `"temp_use_cases"`)
}

func TestPipelineRunVerifierUseCaseIDsRewriteOnlyRequestedUseCases(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	origStart := startPipelineWorkflow
	t.Cleanup(func() {
		startPipelineWorkflow = origStart
	})

	app := setupPipelineVerifierCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	selectedUseCaseID := createVerifierCIUseCase(
		t,
		app,
		orgID,
		"Verifier CI",
		"pid",
	)
	ignoredUseCaseID := createVerifierCIUseCase(
		t,
		app,
		orgID,
		"Verifier Other",
		"other",
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"verifier-ci-pipeline",
		verifierCIPipelineYAMLWithUseCases(selectedUseCaseID, ignoredUseCaseID),
		false,
	)

	startPipelineWorkflow = func(
		yaml string,
		config map[string]any,
		memo map[string]any,
		pipelineIdentifier string,
	) (workflowengine.WorkflowResult, error) {
		require.Contains(
			t,
			yaml,
			"use_case_id: usera-s-organization/verifier-ci/pid-abc123",
		)
		require.NotContains(t, yaml, "use_case_id: "+selectedUseCaseID+"\n")
		require.Contains(t, yaml, "use_case_id: "+ignoredUseCaseID)
		require.Contains(t, config, verifierCITempUseCasesConfigKey)
		return workflowengine.WorkflowResult{
			WorkflowID:    "verifier-wf",
			WorkflowRunID: "verifier-run",
		}, nil
	}

	rec := performPipelineVerifierCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/verifier-ci-pipeline",
		"commit_sha":          "abc123",
		"verifier_url":        "https://verifier.example/temp",
		"use_case_ids":        []string{selectedUseCaseID},
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"temp_use_cases"`)
	require.Contains(
		t,
		rec.Body.String(),
		`"identifier":"usera-s-organization/verifier-ci/pid-abc123"`,
	)
	require.NotContains(t, rec.Body.String(), "verifier-other/other-abc123")
}

func TestPipelineRunVerifierRejectsRawUseCaseRecordIDs(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineVerifierCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	useCaseID := createVerifierCIUseCase(
		t,
		app,
		orgID,
		"Verifier CI",
		"pid",
	)
	useCaseRecord, err := canonify.Resolve(app, useCaseID)
	require.NoError(t, err)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"verifier-ci-pipeline",
		verifierCIPipelineYAML(useCaseID),
		false,
	)

	rec := performPipelineVerifierCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/verifier-ci-pipeline",
		"commit_sha":          "abc123",
		"verifier_url":        "https://verifier.example/temp",
		"use_case_ids":        []string{useCaseRecord.Id},
	})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(
		t,
		rec.Body.String(),
		"use_case_ids must be a canonical use case verification identifier",
	)
}

func setupPipelineVerifierCIApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	return app
}

func performPipelineVerifierCIRequest(
	t testing.TB,
	app *tests.TestApp,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/pipeline/run-verifier", jsonBody(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", walletAPKUserAPIKey)
		mux.ServeHTTP(rec, req)
		return nil
	})
	require.NoError(t, serveErr)
	return rec
}

func createVerifierCIUseCase(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	verifierName string,
	useCaseName string,
) string {
	t.Helper()

	verifierColl, err := app.FindCollectionByNameOrId("verifiers")
	require.NoError(t, err)
	verifier := core.NewRecord(verifierColl)
	verifier.Set("owner", orgID)
	verifier.Set("name", verifierName)
	verifier.Set("url", "https://verifier.example")
	verifier.Set("standard_and_version", "testsuite/draft-01")
	verifier.Set("format", []string{"SD-JWT"})
	verifier.Set("signing_algorithms", []string{"ES256"})
	verifier.Set("cryptographic_binding_methods", []string{"jwk"})
	verifier.Set("description", "example description")
	verifier.Set("published", false)
	require.NoError(t, app.Save(verifier))

	useCaseColl, err := app.FindCollectionByNameOrId("use_cases_verifications")
	require.NoError(t, err)
	useCase := core.NewRecord(useCaseColl)
	useCase.Set("owner", orgID)
	useCase.Set("verifier", verifier.Id)
	useCase.Set("name", useCaseName)
	useCase.Set("yaml", verifierCIStepCIYAML)
	useCase.Set("published", false)
	require.NoError(t, app.Save(useCase))

	return "usera-s-organization/" + verifier.GetString("canonified_name") + "/" +
		useCase.GetString("canonified_name")
}

func verifierCIPipelineYAML(useCaseID string) string {
	return "name: test\nsteps:\n  - id: verify\n    use: use-case-verification-deeplink\n    with:\n      use_case_id: " +
		useCaseID + "\n"
}

func verifierCIPipelineYAMLWithUseCases(useCaseIDs ...string) string {
	pipelineYAML := "name: test\nsteps:\n"
	for index, useCaseID := range useCaseIDs {
		pipelineYAML += "  - id: verify-" + string(rune('a'+index)) +
			"\n    use: use-case-verification-deeplink\n    with:\n      use_case_id: " +
			useCaseID + "\n"
	}
	return pipelineYAML
}
