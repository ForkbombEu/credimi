// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package webpush

import (
	"context"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func setupWebPushApp(t testing.TB) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	subscriptions := core.NewCollection(core.CollectionTypeBase, pushSubscriptionsCollection)
	subscriptions.Fields.Add(
		&core.RelationField{
			Name:          "user",
			CollectionId:  "_pb_users_auth_",
			MaxSelect:     1,
			CascadeDelete: true,
			Required:      true,
		},
		&core.TextField{Name: "endpoint", Required: true},
		&core.JSONField{Name: "keys", Required: true},
	)
	require.NoError(t, app.Save(subscriptions))

	settings := core.NewCollection(core.CollectionTypeBase, webPushSettingsCollection)
	settings.Fields.Add(
		&core.TextField{Name: "vapid_public_key", Required: true},
		&core.TextField{Name: "vapid_private_key", Required: true},
	)
	require.NoError(t, app.Save(settings))

	return app
}
func firstOrgMember(t testing.TB, app *tests.TestApp) (orgID string, userID string) {
	t.Helper()

	authorization, err := app.FindFirstRecordByFilter(orgAuthorizationsCollection, "")
	require.NoError(t, err)

	return authorization.GetString("organization"), authorization.GetString("user")
}

func firstNonMemberUser(t testing.TB, app *tests.TestApp, memberUserID string) string {
	t.Helper()

	user, err := app.FindFirstRecordByFilter(
		"users",
		"id != {:id}",
		dbx.Params{"id": memberUserID},
	)
	require.NoError(t, err)

	return user.Id
}

func createPushSubscription(
	t testing.TB,
	app *tests.TestApp,
	userID string,
	endpoint string,
) *core.Record {
	t.Helper()

	private, x, y, err := elliptic.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	require.NotEmpty(t, private)

	collection, err := app.FindCollectionByNameOrId(pushSubscriptionsCollection)
	require.NoError(t, err)

	record := core.NewRecord(collection)
	record.Set("user", userID)
	record.Set("endpoint", endpoint)
	record.Set("keys", map[string]any{
		"p256dh": base64.RawURLEncoding.EncodeToString(elliptic.Marshal(elliptic.P256(), x, y)),
		"auth":   base64.RawURLEncoding.EncodeToString([]byte("test-auth-16byte")),
	})
	require.NoError(t, app.Save(record))
	return record
}

func TestGetVAPIDKeyPairGeneratesAndPersists(t *testing.T) {
	app := setupWebPushApp(t)
	defer app.Cleanup()

	t.Setenv(EnvVAPIDPublicKey, "")
	t.Setenv(EnvVAPIDPrivateKey, "")

	publicKey, privateKey, err := GetVAPIDKeyPair(app)
	require.NoError(t, err)
	require.NotEmpty(t, publicKey)
	require.NotEmpty(t, privateKey)

	secondPublicKey, secondPrivateKey, err := GetVAPIDKeyPair(app)
	require.NoError(t, err)
	require.Equal(t, publicKey, secondPublicKey)
	require.Equal(t, privateKey, secondPrivateKey)

	total, err := app.CountRecords(webPushSettingsCollection)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
}

func TestGetVAPIDKeyPairEnvOverride(t *testing.T) {
	app := setupWebPushApp(t)
	defer app.Cleanup()

	t.Setenv(EnvVAPIDPublicKey, "env-public-key")
	t.Setenv(EnvVAPIDPrivateKey, "env-private-key")

	publicKey, privateKey, err := GetVAPIDKeyPair(app)
	require.NoError(t, err)
	require.Equal(t, "env-public-key", publicKey)
	require.Equal(t, "env-private-key", privateKey)

	total, err := app.CountRecords(webPushSettingsCollection)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
}

func TestGetVAPIDKeyPairEnvPartialOverrideError(t *testing.T) {
	app := setupWebPushApp(t)
	defer app.Cleanup()

	t.Setenv(EnvVAPIDPublicKey, "env-public-key")
	t.Setenv(EnvVAPIDPrivateKey, "")

	_, _, err := GetVAPIDKeyPair(app)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must both be set")
}

func TestBuildPipelineRunURL(t *testing.T) {
	require.Equal(
		t,
		"https://credimi.test/my/tests/runs/wf-1/run-1",
		buildPipelineRunURL("https://credimi.test", "wf-1", "run-1"),
	)
}

func TestNotifyPipelineRunCompletionSendsToOrgMembersOnly(t *testing.T) {
	app := setupWebPushApp(t)
	defer app.Cleanup()

	t.Setenv(EnvVAPIDPublicKey, "")
	t.Setenv(EnvVAPIDPrivateKey, "")

	orgID, memberUserID := firstOrgMember(t, app)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// The non-member user has no authorization for the organization.
	createPushSubscription(t, app, memberUserID, server.URL)
	createPushSubscription(t, app, firstNonMemberUser(t, app, memberUserID), server.URL)

	sent, err := NotifyPipelineRunCompletion(context.Background(), app, CompletionRequest{
		OrgID:        orgID,
		PipelineName: "my-pipeline",
		WorkflowID:   "wf-1",
		RunID:        "run-1",
		Result:       "success",
		AppURL:       "https://credimi.test",
	})
	require.NoError(t, err)
	require.Equal(t, 1, sent)
	require.Equal(t, int32(1), requests.Load())
}

func TestNotifyPipelineRunCompletionPrunesDeadSubscriptions(t *testing.T) {
	app := setupWebPushApp(t)
	defer app.Cleanup()

	t.Setenv(EnvVAPIDPublicKey, "")
	t.Setenv(EnvVAPIDPrivateKey, "")

	orgID, userID := firstOrgMember(t, app)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer server.Close()

	subscription := createPushSubscription(t, app, userID, server.URL)

	sent, err := NotifyPipelineRunCompletion(context.Background(), app, CompletionRequest{
		OrgID:        orgID,
		PipelineName: "my-pipeline",
		WorkflowID:   "wf-1",
		RunID:        "run-1",
		Result:       "failed",
		AppURL:       "https://credimi.test",
	})
	require.NoError(t, err)
	require.Equal(t, 0, sent)

	_, err = app.FindRecordById(pushSubscriptionsCollection, subscription.Id)
	require.Error(t, err)
}

func TestNotifyPipelineRunCompletionCollectsFailures(t *testing.T) {
	app := setupWebPushApp(t)
	defer app.Cleanup()

	t.Setenv(EnvVAPIDPublicKey, "")
	t.Setenv(EnvVAPIDPrivateKey, "")

	orgID, userID := firstOrgMember(t, app)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	subscription := createPushSubscription(t, app, userID, server.URL)

	sent, err := NotifyPipelineRunCompletion(context.Background(), app, CompletionRequest{
		OrgID:        orgID,
		PipelineName: "my-pipeline",
		WorkflowID:   "wf-1",
		RunID:        "run-1",
		Result:       "failed",
		AppURL:       "https://credimi.test",
	})
	require.Error(t, err)
	require.Equal(t, 0, sent)
	require.Contains(t, err.Error(), "push service returned status: 500")

	_, err = app.FindRecordById(pushSubscriptionsCollection, subscription.Id)
	require.NoError(t, err)
}
