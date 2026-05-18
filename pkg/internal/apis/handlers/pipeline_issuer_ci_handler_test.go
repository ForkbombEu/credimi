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
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const issuerCIStepCIYAML = `version: "1.1"
name: Captures
env:
  host: https://issuer.example/old
  body: credentialIds=pid
tests:
  example:
    steps: []
`

func TestRewriteStepCIHost(t *testing.T) {
	const original = `version: "1.1"
name: Captures
env:
  host: https://issuer.example/old
  body: credentialIds=pid
tests:
  example:
    steps: []
`

	rewritten, ok := rewriteStepCIHost(original, "https://issuer.example/temp")
	require.True(t, ok)
	require.Contains(t, rewritten, "host: https://issuer.example/temp")
	require.Contains(t, rewritten, "body: credentialIds=pid")
}

func TestRewriteStepCIHost_IgnoresMissingEnvHost(t *testing.T) {
	rewritten, ok := rewriteStepCIHost(
		"version: '1.1'\nenv:\n  body: x\n",
		"https://issuer.example/temp",
	)
	require.False(t, ok)
	require.Empty(t, rewritten)
}

func TestRewritePipelineRunIssuerYAML(t *testing.T) {
	const pipelineYAML = `name: test
steps:
  - id: offer
    use: credential-offer
    with:
      credential_id: org/issuer/pid
  - id: other
    use: credential-offer
    with:
      credential_id: org/issuer/ignored
`

	rewritten, apiErr := rewritePipelineRunIssuerYAML(
		pipelineYAML,
		map[string]string{"org/issuer/pid": "org/issuer/pid-temp"},
	)
	require.Nil(t, apiErr)
	require.Contains(t, rewritten, "credential_id: org/issuer/pid-temp")
	require.Contains(t, rewritten, "credential_id: org/issuer/ignored")
}

func TestPipelineRunIssuerAcceptsCredimiAPIKeyForOwnedPipeline(t *testing.T) {
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
		require.Contains(t, yaml, "credential_id: usera-s-organization/issuer-ci/pid-abc123")
		require.NotContains(t, yaml, "credential_id: usera-s-organization/issuer-ci/pid\n")
		require.Contains(t, config, issuerCITempCredentialsConfigKey)
		return workflowengine.WorkflowResult{
			WorkflowID:    "issuer-wf",
			WorkflowRunID: "issuer-run",
		}, nil
	}

	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	credentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer CI",
		"pid",
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAML(credentialID),
		false,
	)

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"running"`)
	require.Contains(t, rec.Body.String(), `"temp_credentials"`)
}

func TestPipelineRunIssuerCreatesGitHubPRCommentForDirectRun(t *testing.T) {
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
		require.Contains(t, config, "github_pr_comment")
		return workflowengine.WorkflowResult{
			WorkflowID:    "issuer-wf",
			WorkflowRunID: "issuer-run",
		}, nil
	}
	commenter := &walletAPKCommenterStub{}
	installWalletAPKCommenterStub(t, commenter)

	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"
	seedWalletAPKUserAPIKey(t, app)
	credentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer CI",
		"pid",
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAML(credentialID),
		false,
	)

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
		"metadata": map[string]any{
			"repository": "forkbombeu/issuer",
			"event": map[string]any{
				"number": 17,
			},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, commenter.updates, 1)
	require.Equal(t, "forkbombeu/issuer", commenter.updates[0].Repository)
	require.Equal(t, 17, commenter.updates[0].PullRequestNumber)
	require.Equal(t, "abc123", commenter.updates[0].CommitSHA)
	require.Equal(t, "running", commenter.updates[0].Status)
	require.Equal(t, "issuer-wf", commenter.updates[0].WorkflowID)
	require.Equal(t, "issuer-run", commenter.updates[0].RunID)
	require.Equal(t, activities.GitHubPRCommentSectionIssuer, commenter.updates[0].SectionTitle)
	require.Equal(
		t,
		"https://credimi.test/my/pipelines/usera-s-organization/issuer-ci-pipeline",
		commenter.updates[0].PipelineURL,
	)
}

func TestPipelineRunIssuerCredentialIDsRewriteOnlyRequestedCredentials(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	origStart := startPipelineWorkflow
	t.Cleanup(func() {
		startPipelineWorkflow = origStart
	})

	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	selectedCredentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer CI",
		"pid",
	)
	ignoredCredentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer Other",
		"other",
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAMLWithCredentials(selectedCredentialID, ignoredCredentialID),
		false,
	)

	startPipelineWorkflow = func(
		yaml string,
		config map[string]any,
		memo map[string]any,
		pipelineIdentifier string,
	) (workflowengine.WorkflowResult, error) {
		require.Contains(t, yaml, "credential_id: usera-s-organization/issuer-ci/pid-abc123")
		require.NotContains(t, yaml, "credential_id: "+selectedCredentialID+"\n")
		require.Contains(t, yaml, "credential_id: "+ignoredCredentialID)
		require.Contains(t, config, issuerCITempCredentialsConfigKey)
		return workflowengine.WorkflowResult{
			WorkflowID:    "issuer-wf",
			WorkflowRunID: "issuer-run",
		}, nil
	}

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
		"credential_ids":      []string{selectedCredentialID},
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"temp_credentials"`)
	require.Contains(
		t,
		rec.Body.String(),
		`"identifier":"usera-s-organization/issuer-ci/pid-abc123"`,
	)
	require.NotContains(t, rec.Body.String(), "issuer-other/other-abc123")
}

func TestPipelineRunIssuerRejectsCredentialIDsNotReferencedByPipeline(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	pipelineCredentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer CI",
		"pid",
	)
	unusedCredentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer Other",
		"other",
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAML(pipelineCredentialID),
		false,
	)

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
		"credential_ids":      []string{unusedCredentialID},
	})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(
		t,
		rec.Body.String(),
		"no credential-offer step references a requested credential_id",
	)
}

func TestPipelineRunIssuerRejectsRawCredentialRecordIDs(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	credentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer CI",
		"pid",
	)
	credentialRecord, err := canonify.Resolve(app, credentialID)
	require.NoError(t, err)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAML(credentialID),
		false,
	)

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
		"credential_ids":      []string{credentialRecord.Id},
	})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(
		t,
		rec.Body.String(),
		"credential_ids must be a canonical credential identifier",
	)
}

func TestPipelineRunIssuerRejectsPrivateForeignPipeline(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	otherOrg := createOtherWalletAPKOrganization(t, app)
	credentialID := createIssuerCICredential(
		t,
		app,
		orgID,
		"Issuer CI",
		"pid",
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		otherOrg.Id,
		"foreign-private-pipeline",
		issuerCIPipelineYAML(credentialID),
		false,
	)

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "other-org/foreign-private-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
	})

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(
		t,
		rec.Body.String(),
		"pipeline must belong to caller organization or be published",
	)
}

func setupPipelineIssuerCIApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	return app
}

func performPipelineIssuerCIRequest(
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

		req := httptest.NewRequest(http.MethodPost, "/api/pipeline/run-issuer", jsonBody(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", walletAPKUserAPIKey)
		mux.ServeHTTP(rec, req)
		return nil
	})
	require.NoError(t, serveErr)
	return rec
}

func createIssuerCICredential(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	issuerName string,
	credentialName string,
) string {
	t.Helper()

	issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	issuer := core.NewRecord(issuerColl)
	issuer.Set("owner", orgID)
	issuer.Set("name", issuerName)
	issuer.Set("url", "https://issuer.example")
	issuer.Set("published", false)
	require.NoError(t, app.Save(issuer))

	credentialColl, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	credential := core.NewRecord(credentialColl)
	credential.Set("owner", orgID)
	credential.Set("credential_issuer", issuer.Id)
	credential.Set("name", credentialName)
	credential.Set("yaml", issuerCIStepCIYAML)
	credential.Set("published", false)
	require.NoError(t, app.Save(credential))

	return "usera-s-organization/" + issuer.GetString("canonified_name") + "/" +
		credential.GetString("canonified_name")
}

func issuerCIPipelineYAML(credentialID string) string {
	return "name: test\nsteps:\n  - id: offer\n    use: credential-offer\n    with:\n      credential_id: " +
		credentialID + "\n"
}

func issuerCIPipelineYAMLWithCredentials(credentialIDs ...string) string {
	pipelineYAML := "name: test\nsteps:\n"
	for index, credentialID := range credentialIDs {
		pipelineYAML += "  - id: offer-" + string(rune('a'+index)) +
			"\n    use: credential-offer\n    with:\n      credential_id: " + credentialID + "\n"
	}
	return pipelineYAML
}
