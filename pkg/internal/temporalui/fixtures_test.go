// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixtureStoreMatchAndNamespaceRewrite(t *testing.T) {
	dir := t.TempDir()
	bodies := filepath.Join(dir, "bodies")
	require.NoError(t, os.Mkdir(bodies, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(bodies, "settings.json"),
		[]byte(`{"DisableWriteActions":true,"ns":"fcaf-1"}`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(bodies, "workflow.json"),
		[]byte(`{"workflowExecutionInfo":{"execution":{"workflowId":"wf-1"}}}`),
		0o644,
	))
	manifest := fixtureManifest{
		RecordedNamespace: "fcaf-1",
		Entries: []fixtureEntry{
			{
				Method:      http.MethodGet,
				Path:        "/temporal-ui/api/v1/settings",
				Status:      200,
				ContentType: "application/json",
				Body:        "bodies/settings.json",
			},
			{
				Method: http.MethodGet,
				Path:   "/temporal-ui/api/v1/namespaces/{namespace}/workflows/wf-1",
				Query: map[string]string{
					"execution.runId": "run-1",
				},
				Status:      200,
				ContentType: "application/json",
				Body:        "bodies/workflow.json",
			},
		},
	}
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644))

	store, err := openFixtureStore(dir)
	require.NoError(t, err)

	t.Setenv(FixturesEnv, dir)
	fixtureMu.Lock()
	delete(fixtureCache, dir)
	fixtureMu.Unlock()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"http://credimi.test/temporal-ui/api/v1/settings",
		nil,
	)
	ok, err := tryServeFixture(rec, req, "acme")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"ns":"acme"`)
	require.NotContains(t, rec.Body.String(), "fcaf-1")

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodGet,
		"http://credimi.test/temporal-ui/api/v1/namespaces/acme/workflows/wf-1"+
			"?execution.runId=run-1&waitNewEvent=true",
		nil,
	)
	ok, err = tryServeFixture(rec, req, "acme")
	require.NoError(t, err)
	require.True(t, ok)
	require.Contains(t, rec.Body.String(), "wf-1")

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodGet,
		"http://credimi.test/temporal-ui/api/v1/namespaces/acme/workflows/wf-1"+
			"?execution.runId=other",
		nil,
	)
	ok, err = tryServeFixture(rec, req, "acme")
	require.NoError(t, err)
	require.False(t, ok)

	_, found := store.match(
		http.MethodGet,
		"/temporal-ui/api/v1/namespaces/acme/workflows/wf-1",
		url.Values{"execution.runId": []string{"run-1"}},
		"acme",
	)
	require.True(t, found)
}

func TestLookupMyWorkflowRunFixture(t *testing.T) {
	dir := t.TempDir()
	bodies := filepath.Join(dir, "bodies")
	require.NoError(t, os.Mkdir(bodies, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(bodies, "run.json"),
		[]byte(`{"workflowExecutionInfo":{"status":"WORKFLOW_EXECUTION_STATUS_FAILED"}}`),
		0o644,
	))
	manifest := map[string]any{
		"recordedNamespace": "fcaf-1",
		"workflowId":        "wf-har",
		"runId":             "run-har",
		"entries":           []any{},
		"credimi": map[string]any{
			"getMyWorkflowRun": map[string]any{
				"status": 200,
				"body":   "bodies/run.json",
			},
		},
	}
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644))

	t.Setenv(FixturesEnv, dir)
	fixtureMu.Lock()
	delete(fixtureCache, dir)
	fixtureMu.Unlock()

	body, status, ok, err := LookupMyWorkflowRunFixture("wf-har", "run-har")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 200, status)
	require.Contains(t, string(body), "WORKFLOW_EXECUTION_STATUS_FAILED")

	_, _, ok, err = LookupMyWorkflowRunFixture("other", "run-har")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestFixtureStoreDisabled(t *testing.T) {
	t.Setenv(FixturesEnv, "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://credimi.test/temporal-ui/api/v1/settings", nil)
	ok, err := tryServeFixture(rec, req, "acme")
	require.NoError(t, err)
	require.False(t, ok)
}
