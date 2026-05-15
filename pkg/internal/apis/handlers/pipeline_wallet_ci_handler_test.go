// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const walletAPKUserAPIKey = "wallet-apk-user-api-key"

func walletAPKMetadata() map[string]any {
	return map[string]any{"sha": "abc123"}
}

func walletAPKMetadataField(sha string) string {
	return `{"sha":"` + sha + `"}`
}

func setupPipelineWalletAPKApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	orgRecord, err := app.FindFirstRecordByFilter(
		"organizations",
		`name="userA's organization"`,
	)
	require.NoError(t, err)
	createWalletAPKMobileRunner(t, app, orgRecord.Id, "runner-1", "ios_simulator", false)
	createWalletAPKMobileRunner(t, app, orgRecord.Id, "runner-global", "ios_simulator", false)

	return app
}

func seedWalletAPKUserAPIKey(t testing.TB, app *tests.TestApp) {
	t.Helper()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	coll, err := app.FindCollectionByNameOrId("api_keys")
	require.NoError(t, err)
	hash, err := bcrypt.GenerateFromPassword([]byte(walletAPKUserAPIKey), bcrypt.MinCost)
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("name", "wallet-apk-user-key")
	record.Set("key", string(hash))
	record.Set("user", user.Id)
	record.Set("superuser", "")
	record.Set("key_type", "user")
	record.Set("revoked", false)
	require.NoError(t, app.Save(record))
}

func createWalletAPITestPipeline(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	yaml string,
) *core.Record {
	return createWalletAPITestPipelineNamed(t, app, orgID, "pipeline123", yaml, false)
}

func createWalletAPITestPipelineNamed(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
	yaml string,
	published bool,
) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("description", "test pipeline")
	record.Set("steps", map[string]any{"rest-chain": map[string]any{"yaml": yaml}})
	record.Set("yaml", yaml)
	record.Set("published", published)
	require.NoError(t, app.Save(record))

	return record
}

func blankWalletAPITestPipelineYAML(t testing.TB, app *tests.TestApp, pipelineID string) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipelines SET yaml = '' WHERE id = {:id}`,
	).Bind(dbx.Params{"id": pipelineID}).Execute()
	require.NoError(t, err)
}

func createWalletAPKMobileRunner(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
	runnerType string,
	published bool,
) {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("ip", "https://"+strings.ReplaceAll(strings.ToLower(name), " ", "-")+".example.test")
	record.Set("type", runnerType)
	record.Set("published", published)
	require.NoError(t, app.Save(record))
}

func createWalletAPKWallet(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	walletName string,
) *core.Record {
	t.Helper()

	walletColl, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	walletRecord, err := app.FindFirstRecordByFilter(
		walletColl,
		"owner = {:owner} && canonified_name = {:name}",
		dbx.Params{
			"owner": orgID,
			"name":  walletName,
		},
	)
	if err != nil {
		walletRecord = core.NewRecord(walletColl)
		walletRecord.Set("name", walletName)
		walletRecord.Set("owner", orgID)
		require.NoError(t, app.Save(walletRecord))
	}

	return walletRecord
}

func createOtherWalletAPKOrganization(t testing.TB, app *tests.TestApp) *core.Record {
	t.Helper()

	orgColl, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	orgRecord := core.NewRecord(orgColl)
	orgRecord.Set("name", "Other Org")
	orgRecord.Set("canonified_name", "other-org")
	require.NoError(t, app.Save(orgRecord))
	return orgRecord
}

func createWalletAPKVersion(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	walletName string,
	tag string,
) string {
	t.Helper()

	walletRecord := createWalletAPKWallet(t, app, orgID, walletName)

	versionColl, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	versionRecord := core.NewRecord(versionColl)
	versionRecord.Set("wallet", walletRecord.Id)
	versionRecord.Set("tag", tag)
	versionRecord.Set("owner", orgID)
	versionRecord.Set(
		"android_installer",
		[]*filesystem.File{NewTestFile("app.apk", []byte("dummy apk content"))},
	)
	require.NoError(t, app.Save(versionRecord))

	return "usera-s-organization/" + walletRecord.GetString("canonified_name") + "/" +
		versionRecord.GetString("canonified_tag")
}

func walletAPKMultipartBody(
	t testing.TB,
	fields map[string]string,
	includeFile bool,
) (*bytes.Reader, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	if includeFile {
		fileWriter, err := writer.CreateFormFile(walletAPKFormFileField, "wallet.apk")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("apk"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	return bytes.NewReader(body.Bytes()), writer.FormDataContentType()
}

func installWalletAPKURLDownloaderStub(t testing.TB) {
	t.Helper()

	original := walletAPKURLDownloader
	t.Cleanup(func() {
		walletAPKURLDownloader = original
	})
	walletAPKURLDownloader = func(ctx context.Context, apkURL string, filename string) (*filesystem.File, error) {
		return filesystem.NewFileFromBytes([]byte("downloaded apk"), filename)
	}
}

type walletAPKCommenterStub struct {
	updates []activities.UpdateGitHubPRCommentInput
	err     error
}

func (s *walletAPKCommenterStub) Signal(
	ctx context.Context,
	update activities.UpdateGitHubPRCommentInput,
) error {
	s.updates = append(s.updates, update)
	return s.err
}

func installWalletAPKCommenterStub(t testing.TB, stub *walletAPKCommenterStub) {
	t.Helper()

	original := signalGitHubPRCommentUpdate
	t.Cleanup(func() {
		signalGitHubPRCommentUpdate = original
	})
	signalGitHubPRCommentUpdate = func(ctx context.Context, update activities.UpdateGitHubPRCommentInput) error {
		return stub.Signal(ctx, update)
	}
}

func TestPipelineRunWalletAPKRequestContract(t *testing.T) {
	installWalletAPKURLDownloaderStub(t)
	queueStub := &queueStub{}
	installQueueStubs(t, queueStub)

	validJSON := func() *bytes.Reader {
		return jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"metadata":            walletAPKMetadata(),
			"apk_url":             "http://ci.example.test/wallet.apk",
		})
	}
	userKeyHeaders := map[string]string{
		"Content-Type":    "application/json",
		"Credimi-Api-Key": walletAPKUserAPIKey,
	}

	bothBody, bothContentType := walletAPKMultipartBody(t, map[string]string{
		"pipeline_identifier": "usera-s-organization/pipeline123",
		"metadata":            walletAPKMetadataField("abc123"),
		"apk_url":             "http://ci.example.test/wallet.apk",
	}, true)

	scenarios := []tests.ApiScenario{
		{
			Name:   "requires auth",
			Method: http.MethodPost,
			URL:    "/api/pipeline/run-wallet-apk",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:           validJSON(),
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"authentication_required",
			},
			TestAppFactory: setupPipelineWalletAPKApp,
		},
		{
			Name:    "requires pipeline identifier",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"metadata": walletAPKMetadata(),
				"apk_url":  "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"pipeline_identifier",
				"missing pipeline_identifier",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				return app
			},
		},
		{
			Name:    "requires metadata sha",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"metadata.sha",
				"missing metadata.sha",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				return app
			},
		},
		{
			Name:    "requires one apk source",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"metadata":            walletAPKMetadata(),
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"apk_file or apk_url is required",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				return app
			},
		},
		{
			Name:   "rejects both apk sources",
			Method: http.MethodPost,
			URL:    "/api/pipeline/run-wallet-apk",
			Headers: map[string]string{
				"Content-Type":    bothContentType,
				"Credimi-Api-Key": walletAPKUserAPIKey,
			},
			Body:           bothBody,
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"provide either apk_file or apk_url",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				return app
			},
		},
		{
			Name:           "accepts user api key auth",
			Method:         http.MethodPost,
			URL:            "/api/pipeline/run-wallet-apk",
			Headers:        userKeyHeaders,
			Body:           validJSON(),
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"queued"`,
				`"temp_wallet_version_identifier"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				versionID := createWalletAPKVersion(t, app, orgID, "wallet123", "1.0.0")
				createWalletAPITestPipeline(t, app, orgID, walletAPKPipelineYAML(versionID))
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPipelineRunWalletAPKContextResolution(t *testing.T) {
	installWalletAPKURLDownloaderStub(t)
	queueStub := &queueStub{}
	installQueueStubs(t, queueStub)

	userKeyHeaders := map[string]string{
		"Content-Type":    "application/json",
		"Credimi-Api-Key": walletAPKUserAPIKey,
	}

	scenarios := []tests.ApiScenario{
		{
			Name:    "unknown pipeline",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/missing",
				"metadata":            walletAPKMetadata(),
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				"pipeline not found",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				return app
			},
		},
		{
			Name:    "pipeline yaml is required",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"metadata":            walletAPKMetadata(),
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"pipeline yaml is required",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				pipeline := createWalletAPITestPipeline(t, app, orgID, "name: test\nsteps: []\n")
				blankWalletAPITestPipelineYAML(t, app, pipeline.Id)
				return app
			},
		},
		{
			Name:    "resolves namespace from user api key",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"metadata":            walletAPKMetadata(),
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"queued"`,
				`"runner_ids":["usera-s-organization/runner-1"]`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				versionID := createWalletAPKVersion(t, app, orgID, "wallet123", "1.0.0")
				createWalletAPITestPipeline(t, app, orgID, walletAPKPipelineYAML(versionID))
				return app
			},
		},
		{
			Name:    "rejects raw wallet version record id",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"metadata":            walletAPKMetadata(),
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"wallet_version must be a canonical wallet version identifier",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				versionID := createWalletAPKVersion(t, app, orgID, "wallet123", "1.0.0")
				versionRecord, err := canonify.Resolve(app, versionID)
				require.NoError(t, err)
				createWalletAPITestPipeline(t, app, orgID, walletAPKPipelineYAML(versionRecord.Id))
				return app
			},
		},
		{
			Name:    "rejects private pipeline owned by another organization",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "other-org/private-pipeline",
				"metadata":            walletAPKMetadata(),
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"pipeline is not owned by caller or published",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				otherOrg := createOtherWalletAPKOrganization(t, app)
				versionID := createWalletAPKVersion(
					t,
					app,
					orgID,
					"wallet-private-pipeline",
					"1.0.0",
				)
				createWalletAPITestPipelineNamed(
					t,
					app,
					otherOrg.Id,
					"private-pipeline",
					walletAPKPipelineYAML(versionID),
					false,
				)
				return app
			},
		},
		{
			Name:    "allows published pipeline owned by another organization",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "other-org/published-pipeline",
				"metadata":            walletAPKMetadata(),
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"queued"`,
				`"runner_ids":["usera-s-organization/runner-1"]`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedWalletAPKUserAPIKey(t, app)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				otherOrg := createOtherWalletAPKOrganization(t, app)
				versionID := createWalletAPKVersion(
					t,
					app,
					orgID,
					"wallet-published-pipeline",
					"1.0.0",
				)
				createWalletAPITestPipelineNamed(
					t,
					app,
					otherOrg.Id,
					"published-pipeline",
					walletAPKPipelineYAML(versionID),
					true,
				)
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPipelineRunWalletAPKEnqueuesManipulatedYAML(t *testing.T) {
	installWalletAPKURLDownloaderStub(t)
	queueStub := &queueStub{}
	installQueueStubs(t, queueStub)

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	metadata := map[string]any{
		"sha":        "abc123",
		"repository": "octocat/hello-world",
		"run_id":     "1536140711",
	}

	scenario := tests.ApiScenario{
		Name:   "enqueues rewritten yaml with cleanup config",
		Method: http.MethodPost,
		URL:    "/api/pipeline/run-wallet-apk",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"Credimi-Api-Key": walletAPKUserAPIKey,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"metadata":            metadata,
			"apk_url":             "http://ci.example.test/wallet.apk",
		}),
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"status":"queued"`,
			`"temp_wallet_version_id"`,
			`"temp_wallet_version_identifier":"usera-s-organization/wallet-enqueue/abc123"`,
			`"pipeline_url":"https://credimi.test/my/pipelines/usera-s-organization/pipeline123"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupPipelineWalletAPKApp(t)
			app.Settings().Meta.AppURL = "https://credimi.test"
			seedWalletAPKUserAPIKey(t, app)
			versionID := createWalletAPKVersion(t, app, orgID, "wallet-enqueue", "1.0.0")
			createWalletAPITestPipeline(t, app, orgID, walletAPKPipelineYAML(versionID))
			return app
		},
	}
	scenario.Test(t)

	require.Len(t, queueStub.enqueueRequests, 1)
	require.Equal(t, metadata, queueStub.enqueueRequests[0].Memo["metadata"])
	require.Equal(
		t,
		pipelineinternal.RunTypeCI,
		queueStub.enqueueRequests[0].Memo[pipelineinternal.RunTypeMemoKey],
	)
	workflow, err := pipelineinternal.ParseWorkflow(queueStub.enqueueRequests[0].YAML)
	require.NoError(t, err)
	require.Equal(
		t,
		"usera-s-organization/wallet-enqueue/abc123",
		workflow.Steps[0].With.Payload["version_id"],
	)

	require.NotContains(t, workflow.Config, walletAPKCleanupConfigKey)

	cleanup, ok := queueStub.enqueueRequests[0].PipelineConfig[walletAPKCleanupConfigKey].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, cleanup["record_id"])
	require.Equal(t, "usera-s-organization/wallet-enqueue/abc123", cleanup["identifier"])
	require.Equal(t, orgID, cleanup["owner_id"])
	require.Equal(t, true, cleanup["cleanup"])
	require.NotNil(t, queueStub.enqueueRequests[0].Cleanup)
	require.NotEmpty(t, queueStub.enqueueRequests[0].Cleanup.TempWalletVersionID)
	require.Equal(t, orgID, queueStub.enqueueRequests[0].Cleanup.TempWalletVersionOwnerID)
	require.Equal(
		t,
		"usera-s-organization/wallet-enqueue/abc123",
		queueStub.enqueueRequests[0].Cleanup.TempWalletVersionIdentifier,
	)
}

func TestPipelineRunWalletAPKCreatesGitHubPRQueuedComment(t *testing.T) {
	installWalletAPKURLDownloaderStub(t)
	queueStub := &queueStub{}
	installQueueStubs(t, queueStub)
	commenter := &walletAPKCommenterStub{}
	installWalletAPKCommenterStub(t, commenter)

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenario := tests.ApiScenario{
		Name:   "creates github pr comment for pr metadata",
		Method: http.MethodPost,
		URL:    "/api/pipeline/run-wallet-apk",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"Credimi-Api-Key": walletAPKUserAPIKey,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"metadata": map[string]any{
				"sha":        "abc123",
				"repository": "forkbombeu/wallet",
				"event": map[string]any{
					"number": 17,
					"pull_request": map[string]any{
						"head": map[string]any{
							"sha": "abcdef1234567890",
						},
					},
				},
			},
			"apk_url": "http://ci.example.test/wallet.apk",
		}),
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"status":"queued"`,
			`"position":1`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupPipelineWalletAPKApp(t)
			app.Settings().Meta.AppURL = "https://credimi.test"
			seedWalletAPKUserAPIKey(t, app)
			versionID := createWalletAPKVersion(t, app, orgID, "wallet-github-pr", "1.0.0")
			createWalletAPITestPipeline(t, app, orgID, walletAPKPipelineYAML(versionID))
			return app
		},
	}
	scenario.Test(t)

	require.Len(t, commenter.updates, 1)
	require.Equal(t, "forkbombeu/wallet", commenter.updates[0].Repository)
	require.Equal(t, 17, commenter.updates[0].PullRequestNumber)
	require.Equal(t, "abcdef1234567890", commenter.updates[0].CommitSHA)
	require.Equal(t, "queued", commenter.updates[0].Status)
	require.Equal(t, activities.GitHubPRCommentSectionWalletAPK, commenter.updates[0].SectionTitle)
	require.NotNil(t, commenter.updates[0].Position)
	require.Equal(t, 1, *commenter.updates[0].Position)
	require.Equal(
		t,
		"https://credimi.test/my/pipelines/usera-s-organization/pipeline123",
		commenter.updates[0].PipelineURL,
	)
	require.Len(t, queueStub.enqueueRequests, 1)
	require.NotNil(t, queueStub.enqueueRequests[0].Notification)
	require.NotNil(t, queueStub.enqueueRequests[0].Notification.GitHubPR)
	require.Equal(
		t,
		"forkbombeu/wallet",
		queueStub.enqueueRequests[0].Notification.GitHubPR.Repository,
	)
	require.Equal(
		t,
		17,
		queueStub.enqueueRequests[0].Notification.GitHubPR.PullRequestNumber,
	)
	require.Equal(
		t,
		"abcdef1234567890",
		queueStub.enqueueRequests[0].Notification.GitHubPR.CommitSHA,
	)
}

func TestPipelineRunWalletAPKInjectsGlobalRunnerID(t *testing.T) {
	installWalletAPKURLDownloaderStub(t)
	queueStub := &queueStub{}
	installQueueStubs(t, queueStub)

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenario := tests.ApiScenario{
		Name:   "injects runner_id as global_runner_id",
		Method: http.MethodPost,
		URL:    "/api/pipeline/run-wallet-apk",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"Credimi-Api-Key": walletAPKUserAPIKey,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"metadata":            walletAPKMetadata(),
			"runner_id":           "usera-s-organization/runner-global",
			"apk_url":             "http://ci.example.test/wallet.apk",
		}),
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"status":"queued"`,
			`"runner_ids":["usera-s-organization/runner-global"]`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupPipelineWalletAPKApp(t)
			seedWalletAPKUserAPIKey(t, app)
			versionID := createWalletAPKVersion(t, app, orgID, "wallet-global-runner", "1.0.0")
			createWalletAPITestPipeline(
				t,
				app,
				orgID,
				walletAPKPipelineYAMLWithoutRunner(versionID),
			)
			return app
		},
	}
	scenario.Test(t)

	require.Len(t, queueStub.enqueueRequests, 1)
	workflow, err := pipelineinternal.ParseWorkflow(queueStub.enqueueRequests[0].YAML)
	require.NoError(t, err)
	require.Equal(t, "usera-s-organization/runner-global", workflow.Runtime.GlobalRunnerID)
	require.Equal(
		t,
		"usera-s-organization/wallet-global-runner/abc123",
		workflow.Steps[0].With.Payload["version_id"],
	)
}

func TestPipelineRunWalletAPKSelectsRunnerByType(t *testing.T) {
	installWalletAPKURLDownloaderStub(t)
	queueStub := &queueStub{}
	installQueueStubs(t, queueStub)

	origQueueState := queryMobileRunnerSemaphoreState
	t.Cleanup(func() { queryMobileRunnerSemaphoreState = origQueueState })
	origHealthCheck := pipelineCIRunnerHealthCheck
	t.Cleanup(func() { pipelineCIRunnerHealthCheck = origHealthCheck })
	pipelineCIRunnerHealthCheck = func(_ context.Context, runnerURL string) (bool, error) {
		return !strings.Contains(runnerURL, "offline-runner"), nil
	}
	queryMobileRunnerSemaphoreState = func(
		_ context.Context,
		runnerID string,
	) (workflows.MobileRunnerSemaphoreStateView, error) {
		switch runnerID {
		case "usera-s-organization/busy-runner":
			return workflows.MobileRunnerSemaphoreStateView{
				RunnerID:  runnerID,
				QueueLen:  2,
				SlotsUsed: 1,
			}, nil
		case "usera-s-organization/free-runner":
			return workflows.MobileRunnerSemaphoreStateView{
				RunnerID: runnerID,
				QueueLen: 1,
			}, nil
		case "usera-s-organization/hidden-runner":
			return workflows.MobileRunnerSemaphoreStateView{
				RunnerID: runnerID,
				QueueLen: 10,
			}, nil
		default:
			return workflows.MobileRunnerSemaphoreStateView{}, errSemaphoreNotFound
		}
	}

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenario := tests.ApiScenario{
		Name:   "selects owned private runner before published runner for runner_type",
		Method: http.MethodPost,
		URL:    "/api/pipeline/run-wallet-apk",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"Credimi-Api-Key": walletAPKUserAPIKey,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"metadata":            walletAPKMetadata(),
			"runner_type":         "android_phone",
			"apk_url":             "http://ci.example.test/wallet.apk",
		}),
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"status":"queued"`,
			`"runner_ids":["usera-s-organization/hidden-runner"]`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupPipelineWalletAPKApp(t)
			seedWalletAPKUserAPIKey(t, app)
			versionID := createWalletAPKVersion(t, app, orgID, "wallet-runner-type", "1.0.0")
			createWalletAPITestPipeline(
				t,
				app,
				orgID,
				walletAPKPipelineYAMLWithoutRunner(versionID),
			)
			createWalletAPKMobileRunner(t, app, orgID, "Busy Runner", "android_phone", true)
			createWalletAPKMobileRunner(t, app, orgID, "Free Runner", "android_phone", true)
			createWalletAPKMobileRunner(t, app, orgID, "Offline Runner", "android_phone", true)
			createWalletAPKMobileRunner(t, app, orgID, "Hidden Runner", "android_phone", false)
			createWalletAPKMobileRunner(t, app, orgID, "Other Type", "ios_phone", true)
			return app
		},
	}
	scenario.Test(t)

	require.Len(t, queueStub.enqueueRequests, 1)
	workflow, err := pipelineinternal.ParseWorkflow(queueStub.enqueueRequests[0].YAML)
	require.NoError(t, err)
	require.Equal(t, "usera-s-organization/hidden-runner", workflow.Runtime.GlobalRunnerID)
}

func TestSelectPipelineRunWalletAPKRunnerByTypeRequiresOnlineRunner(t *testing.T) {
	origHealthCheck := pipelineCIRunnerHealthCheck
	t.Cleanup(func() { pipelineCIRunnerHealthCheck = origHealthCheck })
	pipelineCIRunnerHealthCheck = func(_ context.Context, _ string) (bool, error) {
		return false, nil
	}

	app := setupPipelineWalletAPKApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	createWalletAPKMobileRunner(t, app, orgID, "Offline One", "android_phone", true)
	createWalletAPKMobileRunner(t, app, orgID, "Offline Two", "android_phone", true)

	runnerID, apiErr := selectPipelineCIRunnerByType(
		context.Background(),
		app,
		orgID,
		"android_phone",
	)

	require.Empty(t, runnerID)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusServiceUnavailable, apiErr.Code)
	require.Equal(t, "no online published runner found for runner_type", apiErr.Reason)
}

func TestInjectPipelineRunWalletAPKGlobalRunnerID(t *testing.T) {
	t.Run("rejects step runner ids", func(t *testing.T) {
		workflowDefinition, apiErr := parsePipelineCIWorkflow(
			"name: test\nsteps:\n  - id: step-1\n    use: mobile-automation\n    with:\n      runner_id: usera-s-organization/runner-1\n",
		)
		require.Nil(t, apiErr)
		hasStepRunner, needsGlobalRunner := pipelineCIMobileRunnerSelectionState(workflowDefinition)

		_, apiErr = injectPipelineCIGlobalRunnerID(
			"name: test\nsteps:\n  - id: step-1\n    use: mobile-automation\n    with:\n      runner_id: usera-s-organization/runner-1\n",
			workflowDefinition,
			"runner-global",
			hasStepRunner,
			needsGlobalRunner,
		)

		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.Code)
		require.Equal(t, "runner_id cannot be combined with step runner_id", apiErr.Reason)
	})

	t.Run("leaves yaml unchanged when runner id is empty", func(t *testing.T) {
		inputYAML := "name: test\nsteps: []\n"
		workflowDefinition, apiErr := parsePipelineCIWorkflow(inputYAML)
		require.Nil(t, apiErr)
		hasStepRunner, needsGlobalRunner := pipelineCIMobileRunnerSelectionState(workflowDefinition)

		got, apiErr := injectPipelineCIGlobalRunnerID(
			inputYAML,
			workflowDefinition,
			"",
			hasStepRunner,
			needsGlobalRunner,
		)

		require.Nil(t, apiErr)
		require.Equal(t, inputYAML, got)
	})
}

func TestValidatePipelineCIGlobalRunnerRequest(t *testing.T) {
	apiErr := validatePipelineCIGlobalRunnerRequest("", false, true)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "runner_id", apiErr.Domain)
	require.Equal(t, "runner_id or runner_type is required", apiErr.Reason)

	apiErr = validatePipelineCIGlobalRunnerRequest("", true, true)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "mobile-automation runner_id configuration is incomplete", apiErr.Reason)

	require.Nil(t, validatePipelineCIGlobalRunnerRequest("", false, false))
	require.Nil(t, validatePipelineCIGlobalRunnerRequest("runner-1", false, true))
}

func TestPipelineCIIgnoresRunnerHintsWithoutMobileAutomation(t *testing.T) {
	workflowDefinition, apiErr := parsePipelineCIWorkflow("name: test\nsteps: []\n")
	require.Nil(t, apiErr)

	runnerID, hasStepRunner, needsGlobalRunner, apiErr := resolvePipelineRunWalletAPKRunnerID(
		context.Background(),
		nil,
		"",
		workflowDefinition,
		pipelineRunWalletAPKRequest{
			RunnerID:   "runner-1",
			RunnerType: "android_phone",
		},
	)
	require.Nil(t, apiErr)
	require.Empty(t, runnerID)
	require.False(t, hasStepRunner)
	require.False(t, needsGlobalRunner)
	require.Equal(
		t,
		"runner_id and runner_type are ignored because pipeline has no mobile-automation steps",
		pipelineCIIgnoredRunnerWarning(
			"runner-1",
			"android_phone",
			hasStepRunner,
			needsGlobalRunner,
		),
	)

	runnerID, hasStepRunner, needsGlobalRunner, apiErr = resolvePipelineCIRunnerID(
		context.Background(),
		nil,
		"",
		workflowDefinition,
		pipelineCIBaseRequest{
			RunnerID:   "runner-1",
			RunnerType: "android_phone",
		},
	)
	require.Nil(t, apiErr)
	require.Empty(t, runnerID)
	require.False(t, hasStepRunner)
	require.False(t, needsGlobalRunner)
}

func TestValidatePipelineRunWalletAPKRequestRejectsInvalidRunnerType(t *testing.T) {
	apiErr := validatePipelineRunWalletAPKRequest(pipelineRunWalletAPKRequest{
		PipelineIdentifier: "usera-s-organization/pipeline123",
		CommitSHA:          "abc123",
		RunnerType:         "desktop",
		APKURL:             "http://ci.example.test/wallet.apk",
	})

	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "runner_type is invalid", apiErr.Reason)
}

func TestPipelineRunWalletAPKRollsBackTempVersionOnQueueFailure(t *testing.T) {
	installWalletAPKURLDownloaderStub(t)
	queueStub := &queueStub{}
	installQueueStubs(t, queueStub)
	enqueueRunTicket = func(
		ctx context.Context,
		runnerID string,
		req workflows.MobileRunnerSemaphoreEnqueueRunRequest,
	) (workflows.MobileRunnerSemaphoreEnqueueRunResponse, error) {
		return workflows.MobileRunnerSemaphoreEnqueueRunResponse{}, errors.New("queue unavailable")
	}

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	app := setupPipelineWalletAPKApp(t)
	defer app.Cleanup()
	seedWalletAPKUserAPIKey(t, app)

	versionID := createWalletAPKVersion(t, app, orgID, "wallet-rollback", "1.0.0")
	createWalletAPITestPipeline(t, app, orgID, walletAPKPipelineYAML(versionID))
	walletRecord := createWalletAPKWallet(t, app, orgID, "wallet-rollback")

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/run-wallet-apk",
			jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"metadata":            walletAPKMetadata(),
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
		)
		req.Header.Set("Credimi-Api-Key", walletAPKUserAPIKey)
		req.Header.Set("content-type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Contains(t, rec.Body.String(), "failed to enqueue pipeline run")
		return nil
	})
	require.NoError(t, serveErr)

	records, err := app.FindRecordsByFilter(
		"wallet_versions",
		"wallet={:wallet} && owner={:owner} && canonified_tag={:tag}",
		"",
		-1,
		0,
		dbx.Params{
			"wallet": walletRecord.Id,
			"owner":  orgID,
			"tag":    "abc123",
		},
	)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestResolvePipelineRunWalletAPKFile(t *testing.T) {
	t.Run("accepts multipart upload", func(t *testing.T) {
		body, contentType := walletAPKMultipartBody(t, map[string]string{}, true)
		req, err := http.NewRequest(http.MethodPost, "/", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)
		require.NoError(t, req.ParseMultipartForm(1000<<20))

		fileHeader := req.MultipartForm.File[walletAPKFormFileField][0]
		file, apiErr := resolvePipelineRunWalletAPKFile(
			context.Background(),
			pipelineRunWalletAPKRequest{
				CommitSHA: "ABC-123",
				APKFile:   fileHeader,
			},
		)

		require.Nil(t, apiErr)
		require.Equal(t, "abc-123.apk", file.OriginalName)
		require.Equal(t, int64(3), file.Size)
	})

	t.Run("rejects unsupported url scheme", func(t *testing.T) {
		_, apiErr := resolvePipelineRunWalletAPKFile(
			context.Background(),
			pipelineRunWalletAPKRequest{
				CommitSHA: "abc123",
				APKURL:    "file:///tmp/wallet.apk",
			},
		)

		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.Code)
		require.Equal(t, "invalid apk_url", apiErr.Reason)
	})

	t.Run("downloads http url", func(t *testing.T) {
		installWalletAPKURLDownloaderStub(t)

		file, apiErr := resolvePipelineRunWalletAPKFile(
			context.Background(),
			pipelineRunWalletAPKRequest{
				CommitSHA: "ABC123",
				APKURL:    "http://ci.example.test/wallet.apk",
			},
		)

		require.Nil(t, apiErr)
		require.Equal(t, "abc123.apk", file.OriginalName)
		require.Equal(t, int64(len("downloaded apk")), file.Size)
	})
}

func TestDownloadWalletAPKFromURL(t *testing.T) {
	t.Run("downloads apk bytes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "9")
			_, err := w.Write([]byte("apk bytes"))
			require.NoError(t, err)
		}))
		defer server.Close()

		file, err := downloadWalletAPKFromURL(context.Background(), server.URL, "wallet.apk")
		require.NoError(t, err)
		require.Equal(t, "wallet.apk", file.OriginalName)
	})

	t.Run("rejects failed status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "no apk", http.StatusNotFound)
		}))
		defer server.Close()

		file, err := downloadWalletAPKFromURL(context.Background(), server.URL, "wallet.apk")
		require.Error(t, err)
		require.Nil(t, file)
		require.Contains(t, err.Error(), "unexpected status")
	})
}

func TestCreatePipelineRunWalletAPKTempVersion(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	newRunContext := func(
		t testing.TB,
		app *tests.TestApp,
		wallet *core.Record,
		commitSHA string,
	) pipelineRunWalletAPKContext {
		t.Helper()

		orgRecord, err := app.FindRecordById("organizations", orgID)
		require.NoError(t, err)

		return pipelineRunWalletAPKContext{
			input: pipelineRunWalletAPKRequest{
				CommitSHA: commitSHA,
			},
			organizationRecord: orgRecord,
			namespace:          orgRecord.GetString("canonified_name"),
			walletRecord:       wallet,
			apkFile:            NewTestFile("wallet.apk", []byte("apk")),
		}
	}

	t.Run("creates caller-owned temporary wallet version", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		wallet := createWalletAPKWallet(t, app, orgID, "wallet-temp-create")
		tempVersion, apiErr := createPipelineRunWalletAPKTempVersion(
			app,
			newRunContext(t, app, wallet, "ABC-123"),
		)

		require.Nil(t, apiErr)
		require.NotEmpty(t, tempVersion.Record.Id)
		require.Equal(t, "abc-123", tempVersion.Record.GetString("tag"))
		require.Equal(t, "abc-123", tempVersion.Record.GetString("canonified_tag"))
		require.Equal(t, orgID, tempVersion.Record.GetString("owner"))
		require.Equal(t, wallet.Id, tempVersion.Record.GetString("wallet"))
		require.Equal(t, "usera-s-organization/wallet-temp-create/abc-123", tempVersion.Identifier)
		require.Len(t, tempVersion.Record.GetStringSlice("android_installer"), 1)
	})

	t.Run(
		"creates temporary version for published wallet owned by another organization",
		func(t *testing.T) {
			app := setupPipelineWalletAPKApp(t)
			defer app.Cleanup()

			otherOrg := createOtherWalletAPKOrganization(t, app)
			wallet := createWalletAPKWallet(t, app, otherOrg.Id, "wallet-published")
			wallet.Set("published", true)
			require.NoError(t, app.Save(wallet))

			tempVersion, apiErr := createPipelineRunWalletAPKTempVersion(
				app,
				newRunContext(t, app, wallet, "abc123"),
			)

			require.Nil(t, apiErr)
			require.NotEmpty(t, tempVersion.Record.Id)
			require.Equal(t, orgID, tempVersion.Record.GetString("owner"))
			require.Equal(t, wallet.Id, tempVersion.Record.GetString("wallet"))
			require.Equal(t, "other-org/wallet-published/abc123", tempVersion.Identifier)
		},
	)

	t.Run("rejects private wallet owned by another organization", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		otherOrg := createOtherWalletAPKOrganization(t, app)
		wallet := createWalletAPKWallet(t, app, otherOrg.Id, "wallet-private")

		_, apiErr := createPipelineRunWalletAPKTempVersion(
			app,
			newRunContext(t, app, wallet, "abc123"),
		)

		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusForbidden, apiErr.Code)
		require.Equal(t, "wallet is not owned by caller or published", apiErr.Reason)
	})

	t.Run("allows duplicate commit sha through canonified tag suffix", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		wallet := createWalletAPKWallet(t, app, orgID, "wallet-temp-duplicate")
		first, apiErr := createPipelineRunWalletAPKTempVersion(
			app,
			newRunContext(t, app, wallet, "ABC-123"),
		)
		require.Nil(t, apiErr)
		require.Equal(t, "abc-123", first.Record.GetString("canonified_tag"))

		second, apiErr := createPipelineRunWalletAPKTempVersion(
			app,
			newRunContext(t, app, wallet, "abc-123"),
		)

		require.Nil(t, apiErr)
		require.Equal(t, "abc-123-1", second.Record.GetString("canonified_tag"))
		require.Equal(t, "usera-s-organization/wallet-temp-duplicate/abc-123-1", second.Identifier)
	})
}

func TestRewritePipelineRunWalletAPKYAML(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	t.Run("rewrites all wallet version references", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		versionA := createWalletAPKVersion(t, app, orgID, "wallet-rewrite", "1.0.0")
		versionB := createWalletAPKVersion(t, app, orgID, "wallet-rewrite", "2.0.0")
		pipelineYAML := "name: test\nsteps:\n" +
			"  - id: install-a\n    use: mobile-automation\n    with:\n      version_id: " + versionA + "\n      action_id: action-a\n" +
			"  - id: parse\n    use: json-parse\n    with:\n      raw_json: '{}'\n" +
			"  - id: install-b\n    use: mobile-automation\n    with:\n      payload:\n        version_id: " + versionB + "\n        action_id: action-b\n"

		_, refs, apiErr := resolvePipelineRunWalletAPKWallet(app, pipelineYAML)
		require.Nil(t, apiErr)

		rewritten, apiErr := rewritePipelineRunWalletAPKYAML(
			pipelineYAML,
			refs,
			"usera-s-organization/wallet-rewrite/abc123",
		)

		require.Nil(t, apiErr)
		workflow, err := pipelineinternal.ParseWorkflow(rewritten)
		require.NoError(t, err)
		require.Equal(
			t,
			"usera-s-organization/wallet-rewrite/abc123",
			workflow.Steps[0].With.Payload["version_id"],
		)
		require.Equal(t, "{}", workflow.Steps[1].With.Payload["raw_json"])
		require.Equal(
			t,
			"usera-s-organization/wallet-rewrite/abc123",
			workflow.Steps[2].With.Payload["version_id"],
		)
	})

	t.Run("rewrites nested success and error references", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		versionID := createWalletAPKVersion(t, app, orgID, "wallet-rewrite-nested", "1.0.0")
		pipelineYAML := "name: test\nsteps:\n" +
			"  - id: first\n    use: json-parse\n    with:\n      raw_json: '{}'\n    on_success:\n      - id: nested-success\n        use: mobile-automation\n        with:\n          version_id: " + versionID + "\n          action_id: action-a\n    on_error:\n      - id: nested-error\n        use: mobile-automation\n        with:\n          version_id: " + versionID + "\n          action_id: action-b\n"

		_, refs, apiErr := resolvePipelineRunWalletAPKWallet(app, pipelineYAML)
		require.Nil(t, apiErr)

		rewritten, apiErr := rewritePipelineRunWalletAPKYAML(
			pipelineYAML,
			refs,
			"usera-s-organization/wallet-rewrite-nested/abc123",
		)

		require.Nil(t, apiErr)
		workflow, err := pipelineinternal.ParseWorkflow(rewritten)
		require.NoError(t, err)
		require.Equal(
			t,
			"usera-s-organization/wallet-rewrite-nested/abc123",
			workflow.Steps[0].OnError[0].With.Payload["version_id"],
		)
		require.Equal(
			t,
			"usera-s-organization/wallet-rewrite-nested/abc123",
			workflow.Steps[0].OnSuccess[0].With.Payload["version_id"],
		)
	})

	t.Run("does not mutate stored pipeline yaml", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		versionID := createWalletAPKVersion(t, app, orgID, "wallet-rewrite-stored", "1.0.0")
		pipelineYAML := walletAPKPipelineYAML(versionID)
		pipelineRecord := createWalletAPITestPipeline(t, app, orgID, pipelineYAML)

		_, refs, apiErr := resolvePipelineRunWalletAPKWallet(app, pipelineYAML)
		require.Nil(t, apiErr)
		_, apiErr = rewritePipelineRunWalletAPKYAML(
			pipelineYAML,
			refs,
			"usera-s-organization/wallet-rewrite-stored/abc123",
		)
		require.Nil(t, apiErr)

		reloaded, err := app.FindRecordById("pipelines", pipelineRecord.Id)
		require.NoError(t, err)
		require.Equal(t, pipelineYAML, reloaded.GetString("yaml"))
	})
}

func walletAPKPipelineYAML(versionID string) string {
	return "name: test\nsteps:\n  - id: install-wallet\n    use: mobile-automation\n    with:\n      action_id: usera-s-organization/wallet123/install\n      version_id: " + versionID + "\n      runner_id: usera-s-organization/runner-1\n"
}

func walletAPKPipelineYAMLWithoutRunner(versionID string) string {
	return "name: test\nsteps:\n  - id: install-wallet\n    use: mobile-automation\n    with:\n      action_id: usera-s-organization/wallet123/install\n      version_id: " + versionID + "\n"
}

func TestResolvePipelineRunWalletAPKWallet(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	t.Run("rejects zero wallet versions", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		_, _, apiErr := resolvePipelineRunWalletAPKWallet(app, "name: test\nsteps: []\n")
		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.Code)
		require.Contains(t, apiErr.Reason, "exactly one wallet")
	})

	t.Run("accepts multiple versions from one wallet", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		versionA := createWalletAPKVersion(t, app, orgID, "wallet-one", "1.0.0")
		versionB := createWalletAPKVersion(t, app, orgID, "wallet-one", "2.0.0")
		yaml := "name: test\nsteps:\n" +
			"  - id: install-a\n    use: mobile-automation\n    with:\n      version_id: " + versionA + "\n      action_id: action-a\n" +
			"  - id: install-b\n    use: mobile-automation\n    with:\n      version_id: " + versionB + "\n      action_id: action-b\n"

		wallet, refs, apiErr := resolvePipelineRunWalletAPKWallet(app, yaml)
		require.Nil(t, apiErr)
		require.Equal(t, "wallet-one", wallet.GetString("canonified_name"))
		require.Len(t, refs, 2)
	})

	t.Run("rejects multiple wallets", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		versionA := createWalletAPKVersion(t, app, orgID, "wallet-a", "1.0.0")
		versionB := createWalletAPKVersion(t, app, orgID, "wallet-b", "1.0.0")
		yaml := "name: test\nsteps:\n" +
			"  - id: install-a\n    use: mobile-automation\n    with:\n      version_id: " + versionA + "\n      action_id: action-a\n" +
			"  - id: install-b\n    use: mobile-automation\n    with:\n      version_id: " + versionB + "\n      action_id: action-b\n"

		_, _, apiErr := resolvePipelineRunWalletAPKWallet(app, yaml)
		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.Code)
		require.Contains(t, apiErr.Reason, "exactly one wallet")
	})

	t.Run("collects nested success and error steps", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		versionID := createWalletAPKVersion(t, app, orgID, "wallet-nested", "1.0.0")
		yaml := "name: test\nsteps:\n" +
			"  - id: first\n    use: json-parse\n    with:\n      raw_json: '{}'\n    on_success:\n      - id: nested-success\n        use: mobile-automation\n        with:\n          version_id: " + versionID + "\n          action_id: action-a\n    on_error:\n      - id: nested-error\n        use: mobile-automation\n        with:\n          version_id: " + versionID + "\n          action_id: action-b\n"

		_, refs, apiErr := resolvePipelineRunWalletAPKWallet(app, yaml)
		require.Nil(t, apiErr)
		require.Len(t, refs, 2)
		require.Equal(t, versionID, refs[0].VersionID)
		require.Equal(t, versionID, refs[1].VersionID)
	})

	t.Run("rejects external source version", func(t *testing.T) {
		app := setupPipelineWalletAPKApp(t)
		defer app.Cleanup()

		yaml := walletAPKPipelineYAML(walletAPKExternalSourceVersionID)
		_, _, apiErr := resolvePipelineRunWalletAPKWallet(app, yaml)
		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.Code)
		require.Contains(t, apiErr.Reason, "external source")
	})
}

func TestBuildPipelineRunWalletAPKResponse(t *testing.T) {
	enqueuedAt := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	queuePosition := 2
	apiPosition := 3
	lineLen := 5

	t.Run("queued response keeps queue fields and returns one based position", func(t *testing.T) {
		response := buildPipelineRunWalletAPKResponse(
			PipelineQueueResponse{
				Status:     workflowengine.MobileRunnerSemaphoreRunQueued,
				TicketID:   "ticket-1",
				RunnerIDs:  []string{"runner-1"},
				EnqueuedAt: &enqueuedAt,
				Position:   &queuePosition,
				LineLen:    &lineLen,
			},
			"version-record-1",
			"usera-s-organization/wallet/abc123",
			"usera-s-organization/pipeline123",
			"runner_id and runner_type are ignored because pipeline has no mobile-automation steps",
		)

		require.Equal(t, workflowengine.MobileRunnerSemaphoreRunQueued, response.Status)
		require.Equal(t, "ticket-1", response.TicketID)
		require.Equal(t, []string{"runner-1"}, response.RunnerIDs)
		require.Equal(t, &apiPosition, response.Position)
		require.Equal(t, &lineLen, response.LineLen)
		require.Equal(t, "version-record-1", response.TempWalletVersionID)
		require.Equal(t, "usera-s-organization/wallet/abc123", response.TempWalletVersionIdentifier)
		require.Equal(t, "usera-s-organization/pipeline123", response.PipelineIdentifier)
		require.Equal(
			t,
			"runner_id and runner_type are ignored because pipeline has no mobile-automation steps",
			response.Warning,
		)
	})

	t.Run("running response keeps workflow identifiers", func(t *testing.T) {
		response := buildPipelineRunWalletAPKResponse(
			PipelineQueueResponse{
				Status:     workflowengine.MobileRunnerSemaphoreRunRunning,
				WorkflowID: "workflow-1",
				RunID:      "run-1",
			},
			"version-record-1",
			"usera-s-organization/wallet/abc123",
			"usera-s-organization/pipeline123",
			"",
		)

		require.Equal(t, workflowengine.MobileRunnerSemaphoreRunRunning, response.Status)
		require.Equal(t, "workflow-1", response.WorkflowID)
		require.Equal(t, "run-1", response.RunID)
		require.Equal(t, "version-record-1", response.TempWalletVersionID)
	})
}
