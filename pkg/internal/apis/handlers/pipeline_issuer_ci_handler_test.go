// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/dbx"
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

func requireNoTempCredential(t testing.TB, app *tests.TestApp, name string) {
	t.Helper()

	records, err := app.FindRecordsByFilter(
		"credentials",
		"name = {:name}",
		"",
		0,
		0,
		dbx.Params{"name": name},
	)
	require.NoError(t, err)
	require.Empty(t, records, "temporary credential %q must not survive a failed run", name)
}

func TestPipelineRunIssuerRequestValidation(t *testing.T) {
	type requestCase struct {
		name        string
		contentType string
		body        string
		jsonBody    map[string]any
		wantStatus  int
		wantReason  string
	}
	base := func(overrides map[string]any) map[string]any {
		body := map[string]any{
			"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
			"commit_sha":          "abc123",
			"issuer_url":          "https://issuer.example/temp",
		}
		for key, value := range overrides {
			if value == nil {
				delete(body, key)
				continue
			}
			body[key] = value
		}
		return body
	}
	cases := []requestCase{
		{
			name:        "malformed JSON",
			contentType: "application/json",
			body:        `{"pipeline_identifier":`,
			wantStatus:  http.StatusBadRequest,
			wantReason:  "invalid JSON input",
		},
		{
			name:       "metadata that is not an object",
			jsonBody:   base(map[string]any{"metadata": "sha=abc"}),
			wantStatus: http.StatusBadRequest,
			wantReason: "metadata must be valid JSON",
		},
		{
			name:        "form metadata that is not JSON",
			contentType: "application/x-www-form-urlencoded",
			body: url.Values{
				"pipeline_identifier": {"usera-s-organization/issuer-ci-pipeline"},
				"issuer_url":          {"https://issuer.example/temp"},
				"metadata":            {"{not json"},
			}.Encode(),
			wantStatus: http.StatusBadRequest,
			wantReason: "metadata must be valid JSON",
		},
		{
			name:       "missing pipeline identifier",
			jsonBody:   base(map[string]any{"pipeline_identifier": " "}),
			wantStatus: http.StatusBadRequest,
			wantReason: "pipeline_identifier is required",
		},
		{
			name:       "missing commit sha and metadata sha",
			jsonBody:   base(map[string]any{"commit_sha": nil, "metadata": map[string]any{}}),
			wantStatus: http.StatusBadRequest,
			wantReason: "commit_sha or metadata.sha is required",
		},
		{
			name:       "non http issuer url",
			jsonBody:   base(map[string]any{"issuer_url": "ftp://issuer.example"}),
			wantStatus: http.StatusBadRequest,
			wantReason: "issuer_url is invalid",
		},
		{
			name:       "issuer url without host",
			jsonBody:   base(map[string]any{"issuer_url": "https://"}),
			wantStatus: http.StatusBadRequest,
			wantReason: "issuer_url is invalid",
		},
		{
			name:       "unknown device type",
			jsonBody:   base(map[string]any{"device_type": "desktop"}),
			wantStatus: http.StatusBadRequest,
			wantReason: "device_type is invalid",
		},
		{
			name: "unknown pipeline",
			jsonBody: base(map[string]any{
				"pipeline_identifier": "usera-s-organization/missing-pipeline",
			}),
			wantStatus: http.StatusNotFound,
			wantReason: "pipeline not found",
		},
		{
			name: "unknown requested credential",
			jsonBody: base(map[string]any{
				"credential_ids": []string{"usera-s-organization/issuer-ci/missing"},
			}),
			wantStatus: http.StatusBadRequest,
			wantReason: "credential_ids is invalid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orgID, err := getOrgIDfromName("userA's organization")
			require.NoError(t, err)
			app := setupPipelineIssuerCIApp(t)
			defer app.Cleanup()
			seedWalletAPKUserAPIKey(t, app)

			origStart := startPipelineWorkflow
			t.Cleanup(func() { startPipelineWorkflow = origStart })
			startPipelineWorkflow = func(
				string,
				map[string]any,
				map[string]any,
				string,
			) (workflowengine.WorkflowResult, error) {
				t.Fatal("an invalid request must not start a pipeline")
				return workflowengine.WorkflowResult{}, nil
			}

			credentialID := createIssuerCICredential(t, app, orgID, "Issuer CI", "pid")
			createWalletAPITestPipelineNamed(
				t,
				app,
				orgID,
				"issuer-ci-pipeline",
				issuerCIPipelineYAML(credentialID),
				false,
			)

			var rec *httptest.ResponseRecorder
			if tc.jsonBody != nil {
				rec = performPipelineIssuerCIRequest(t, app, tc.jsonBody)
			} else {
				rec = performPipelineIssuerCIRawRequest(t, app, tc.contentType, tc.body)
			}

			require.Equal(t, tc.wantStatus, rec.Code, rec.Body.String())
			require.Equal(t, tc.wantReason, decodeHandlerErrorResponse(t, rec).Error.Reason)
		})
	}
}

func TestPipelineRunIssuerRejectsUnusablePipelines(t *testing.T) {
	cases := []struct {
		name             string
		yaml             func(credentialID string) string
		blankYAML        bool
		afterTempRecords bool
		wantStatus       int
		wantReason       string
	}{
		{
			name:       "pipeline without yaml",
			yaml:       issuerCIPipelineYAML,
			blankYAML:  true,
			wantStatus: http.StatusBadRequest,
			wantReason: "pipeline yaml is required",
		},
		{
			name: "pipeline without credential-offer steps",
			yaml: func(string) string {
				return "name: test\nsteps:\n  - id: parse\n    use: json-parse\n    with:\n      raw_json: '{}'\n"
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "pipeline must reference at least one credential-offer credential",
		},
		{
			name: "credential-offer step referencing an unknown credential",
			yaml: func(string) string {
				return issuerCIPipelineYAML("usera-s-organization/issuer-ci/missing")
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "credential_id is invalid",
		},
		{
			name: "mobile-automation step without a device and no device selection",
			yaml: func(credentialID string) string {
				return issuerCIPipelineYAML(credentialID) +
					"  - id: wallet\n    use: mobile-automation\n    with:\n      action_id: action-a\n"
			},
			afterTempRecords: true,
			wantStatus:       http.StatusBadRequest,
			wantReason:       "device_id or device_type is required",
		},
		{
			name: "mobile-automation steps mixing step devices with missing ones",
			yaml: func(credentialID string) string {
				return issuerCIPipelineYAML(credentialID) +
					"  - id: wallet-a\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-1/device-1\n" +
					"  - id: wallet-b\n    use: mobile-automation\n    with:\n      action_id: action-b\n"
			},
			afterTempRecords: true,
			wantStatus:       http.StatusBadRequest,
			wantReason:       "mobile-automation device_id configuration is incomplete",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orgID, err := getOrgIDfromName("userA's organization")
			require.NoError(t, err)
			app := setupPipelineIssuerCIApp(t)
			defer app.Cleanup()
			seedWalletAPKUserAPIKey(t, app)

			origStart := startPipelineWorkflow
			t.Cleanup(func() { startPipelineWorkflow = origStart })
			startPipelineWorkflow = func(
				string,
				map[string]any,
				map[string]any,
				string,
			) (workflowengine.WorkflowResult, error) {
				t.Fatal("an unusable pipeline must not be started")
				return workflowengine.WorkflowResult{}, nil
			}

			credentialID := createIssuerCICredential(t, app, orgID, "Issuer CI", "pid")
			pipeline := createWalletAPITestPipelineNamed(
				t,
				app,
				orgID,
				"issuer-ci-pipeline",
				tc.yaml(credentialID),
				false,
			)
			if tc.blankYAML {
				blankWalletAPITestPipelineYAML(t, app, pipeline.Id)
			}

			rec := performPipelineIssuerCIRequest(t, app, map[string]any{
				"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
				"commit_sha":          "abc123",
				"issuer_url":          "https://issuer.example/temp",
			})

			require.Equal(t, tc.wantStatus, rec.Code, rec.Body.String())
			require.Equal(t, tc.wantReason, decodeHandlerErrorResponse(t, rec).Error.Reason)
			if tc.afterTempRecords {
				// Temporary credentials created before the failure are rolled back.
				requireNoTempCredential(t, app, "pid-abc123")
			}
		})
	}
}

func TestPipelineRunIssuerRejectsForeignPrivateCredentialAndRollsBack(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	otherOrg := createOtherWalletAPKOrganization(t, app)

	ownCredentialID := createIssuerCICredential(t, app, orgID, "Issuer CI", "pid")
	foreignCredentialID := strings.Replace(
		createIssuerCICredential(t, app, otherOrg.Id, "Foreign Issuer", "secret"),
		"usera-s-organization/",
		"other-org/",
		1,
	)
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAMLWithCredentials(ownCredentialID, foreignCredentialID),
		false,
	)

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
	})

	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	require.Equal(
		t,
		"credential is not owned by caller or published",
		decodeHandlerErrorResponse(t, rec).Error.Reason,
	)
	// The own credential was cloned first and must be rolled back.
	requireNoTempCredential(t, app, "pid-abc123")
	requireNoTempCredential(t, app, "secret-abc123")
}

func TestPipelineRunIssuerFormRequestUsesMetadataSHA(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	credentialID := createIssuerCICredential(t, app, orgID, "Issuer CI", "pid")
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAML(credentialID),
		false,
	)

	origStart := startPipelineWorkflow
	t.Cleanup(func() { startPipelineWorkflow = origStart })
	var startedYAML string
	startPipelineWorkflow = func(
		yaml string,
		_ map[string]any,
		_ map[string]any,
		_ string,
	) (workflowengine.WorkflowResult, error) {
		startedYAML = yaml
		return workflowengine.WorkflowResult{WorkflowID: "issuer-wf", WorkflowRunID: "run"}, nil
	}

	rec := performPipelineIssuerCIRawRequest(
		t,
		app,
		"application/x-www-form-urlencoded",
		url.Values{
			"pipeline_identifier": {" usera-s-organization/issuer-ci-pipeline "},
			"issuer_url":          {"https://issuer.example/temp"},
			"metadata":            {`{"sha":"Def456"}`},
			// duplicate and comma separated ids collapse to the single referenced credential
			"credential_ids": {"/" + credentialID + "," + credentialID},
		}.Encode(),
	)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body PipelineRunIssuerResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.TempCredentials, 1)
	require.Equal(
		t,
		"usera-s-organization/issuer-ci/pid-def456",
		body.TempCredentials[0].Identifier,
	)
	require.Contains(t, startedYAML, "credential_id: usera-s-organization/issuer-ci/pid-def456")

	temp, err := app.FindRecordById("credentials", body.TempCredentials[0].ID)
	require.NoError(t, err)
	require.False(t, temp.GetBool("published"))
	require.Equal(t, orgID, temp.GetString("owner"))
	require.Contains(t, temp.GetString("yaml"), "host: https://issuer.example/temp")
}

func TestPipelineRunIssuerRepeatedCommitGetsUniqueTempCredential(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	credentialID := createIssuerCICredential(t, app, orgID, "Issuer CI", "pid")
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAML(credentialID),
		false,
	)

	origStart := startPipelineWorkflow
	t.Cleanup(func() { startPipelineWorkflow = origStart })
	startPipelineWorkflow = func(
		string,
		map[string]any,
		map[string]any,
		string,
	) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "issuer-wf", WorkflowRunID: "run"}, nil
	}

	requestBody, err := json.Marshal(map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		// a single string is accepted for credential_ids
		"credential_ids": credentialID,
		"metadata":       map[string]any{"sha": "abc123"},
		"issuer_url":     "https://issuer.example/temp",
	})
	require.NoError(t, err)
	recs := performPipelineIssuerCIRawRequests(
		t,
		app,
		"application/json",
		string(requestBody),
		string(requestBody),
	)

	identifiers := make([]string, 0, len(recs))
	for _, rec := range recs {
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body PipelineRunIssuerResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.TempCredentials, 1)
		identifiers = append(identifiers, body.TempCredentials[0].Identifier)
	}
	require.Equal(t, []string{
		"usera-s-organization/issuer-ci/pid-abc123",
		"usera-s-organization/issuer-ci/pid-abc123-1",
	}, identifiers)
}

func TestPipelineRunIssuerRejectsRequestedCredentialsNotAllReferenced(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)
	usedCredentialID := createIssuerCICredential(t, app, orgID, "Issuer CI", "pid")
	unusedCredentialID := createIssuerCICredential(t, app, orgID, "Issuer Other", "other")
	createWalletAPITestPipelineNamed(
		t,
		app,
		orgID,
		"issuer-ci-pipeline",
		issuerCIPipelineYAML(usedCredentialID),
		false,
	)

	rec := performPipelineIssuerCIRequest(t, app, map[string]any{
		"pipeline_identifier": "usera-s-organization/issuer-ci-pipeline",
		"commit_sha":          "abc123",
		"issuer_url":          "https://issuer.example/temp",
		"credential_ids":      []string{usedCredentialID, unusedCredentialID},
	})

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Equal(
		t,
		"credential_ids must be referenced by the pipeline",
		decodeHandlerErrorResponse(t, rec).Error.Reason,
	)
	requireNoTempCredential(t, app, "pid-abc123")
}

func TestBuildPipelineRunIssuerResponse(t *testing.T) {
	coll := core.NewBaseCollection("credentials")
	record := core.NewRecord(coll)
	record.Id = "temp-1"
	position := 0

	response := buildPipelineRunIssuerResponse(
		PipelineQueueResponse{Position: &position},
		[]tempCredential{
			{Record: nil, Identifier: "skipped"},
			{Record: record, Identifier: "org/issuer/pid-abc"},
		},
		"org/pipeline",
		"warning",
	)

	require.NotNil(t, response.Position)
	require.Equal(t, 1, *response.Position, "queue positions are reported 1-based")
	require.Equal(t, 0, position, "the caller's position must not be mutated")
	require.Equal(
		t,
		[]TempCredentialResponse{{ID: "temp-1", Identifier: "org/issuer/pid-abc"}},
		response.TempCredentials,
	)
	require.Equal(t, "org/pipeline", response.PipelineIdentifier)
	require.Equal(t, "warning", response.Warning)
}

func TestDeleteTempCredentialForOwner(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	app := setupPipelineIssuerCIApp(t)
	defer app.Cleanup()
	credentialID := createIssuerCICredential(t, app, orgID, "Issuer CI", "pid")
	credential, err := canonify.Resolve(app, credentialID)
	require.NoError(t, err)

	t.Run("already deleted credential is a no-op", func(t *testing.T) {
		require.Nil(t, deleteTempCredentialForOwner(app, "missingrecord01", orgID))
	})

	t.Run("credential of another organization is kept", func(t *testing.T) {
		apiErr := deleteTempCredentialForOwner(app, credential.Id, "otherorg000001")
		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusForbidden, apiErr.Code)
		require.Equal(t, "temporary credential owner mismatch", apiErr.Reason)
		_, err := app.FindRecordById("credentials", credential.Id)
		require.NoError(t, err)
	})

	t.Run("owner deletes the credential", func(t *testing.T) {
		require.Nil(t, deleteTempCredentialForOwner(app, " "+credential.Id+" ", orgID))
		_, err := app.FindRecordById("credentials", credential.Id)
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
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

	raw, err := io.ReadAll(jsonBody(body))
	require.NoError(t, err)
	return performPipelineIssuerCIRawRequest(t, app, "application/json", string(raw))
}

func performPipelineIssuerCIRawRequest(
	t testing.TB,
	app *tests.TestApp,
	contentType string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	return performPipelineIssuerCIRawRequests(t, app, contentType, body)[0]
}

// performPipelineIssuerCIRawRequests serves every body through one router, as
// routes can be registered on an app only once.
func performPipelineIssuerCIRawRequests(
	t testing.TB,
	app *tests.TestApp,
	contentType string,
	bodies ...string,
) []*httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	recs := make([]*httptest.ResponseRecorder, 0, len(bodies))
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		for _, body := range bodies {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/pipeline/run-issuer",
				strings.NewReader(body),
			)
			req.Header.Set("Content-Type", contentType)
			req.Header.Set("Credimi-Api-Key", walletAPKUserAPIKey)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			recs = append(recs, rec)
		}
		return nil
	})
	require.NoError(t, serveErr)
	return recs
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
