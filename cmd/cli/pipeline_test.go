// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePipelineName(t *testing.T) {
	name, err := parsePipelineName([]byte("name: demo\n"))
	require.NoError(t, err)
	require.Equal(t, "demo", name)

	_, err = parsePipelineName([]byte("name: ["))
	require.Error(t, err)
}

func TestAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/apikey/authenticate", r.URL.Path)
		require.Equal(t, "key-123", r.Header.Get("X-Api-Key"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"token": "token-abc",
		}))
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	prevAPIKey := apiKey
	apiKey = "key-123"
	t.Cleanup(func() {
		apiKey = prevAPIKey
	})

	token, err := authenticate(context.Background())
	require.NoError(t, err)
	require.Equal(t, "token-abc", token)
}

func TestAuthenticateFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("nope"))
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	prevAPIKey := apiKey
	apiKey = "key-123"
	t.Cleanup(func() {
		apiKey = prevAPIKey
	})

	_, err := authenticate(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "auth failed")
}

func TestGetMyOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/organizations/my", r.URL.Path)
		require.Equal(t, "Bearer token-abc", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":              "org-1",
			"canonified_name": "org-canon",
		}))
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	orgID, canon, err := getMyOrganization(context.Background(), "token-abc")
	require.NoError(t, err)
	require.Equal(t, "org-1", orgID)
	require.Equal(t, "org-canon", canon)
}

func TestGetMyOrganizationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	_, _, err := getMyOrganization(context.Background(), "token-abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get organization")
}

func TestFindOrCreatePipelineReturnsExisting(t *testing.T) {
	input := &PipelineCLIInput{Name: "demo", YAML: "name: demo"}
	existing := map[string]any{"id": "rec_123", "yaml": input.YAML}

	hasPost := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/collections/pipelines/records", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)

		payload := map[string]any{"items": []map[string]any{existing}}
		require.NoError(t, json.NewEncoder(w).Encode(payload))
		if r.Method == http.MethodPost {
			hasPost = true
		}
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	result, err := findOrCreatePipeline(context.Background(), "token", "org", input)
	require.NoError(t, err)
	require.Equal(t, existing["id"], result["id"])
	require.False(t, hasPost)
}

func TestFindOrCreatePipelineCreatesWhenMissing(t *testing.T) {
	input := &PipelineCLIInput{Name: "demo", YAML: "name: demo"}
	created := map[string]any{"id": "rec_456", "yaml": input.YAML}

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/collections/pipelines/records", r.URL.Path)

		switch call {
		case 0:
			require.Equal(t, http.MethodGet, r.Method)
			payload := map[string]any{"items": []map[string]any{}}
			require.NoError(t, json.NewEncoder(w).Encode(payload))
		case 1:
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, input.Name, body["name"])
			require.Equal(t, input.YAML, body["yaml"])

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(created))
		default:
			require.Fail(t, "unexpected request")
		}
		call++
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	result, err := findOrCreatePipeline(context.Background(), "token", "org", input)
	require.NoError(t, err)
	require.Equal(t, created["id"], result["id"])
	require.Equal(t, 2, call)
}

// TestStartPipelineQueuesRunnerPipelines verifies queue output for runner pipelines.
func TestStartPipelineQueuesRunnerPipelines(t *testing.T) {
	rec := map[string]any{
		"yaml":             "name: demo",
		"canonified_name":  "pipeline123",
		"description":      "test",
		"workflow":         "test",
		"workflow_id":      "wf",
		"workflow_run_id":  "run",
		"workflow_run_ids": []string{},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/pipeline/queue":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "org/pipeline123", body["pipeline_identifier"])
			require.Equal(t, "name: demo", body["yaml"])

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"mode":       "queued",
				"ticket_id":  "ticket-1",
				"runner_ids": []string{"runner-1"},
				"position":   0,
				"line_len":   2,
			}))
		default:
			require.Fail(t, "unexpected path")
		}
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	output := captureStdout(t, func() {
		require.NoError(t, startPipeline(context.Background(), "token", "org", rec))
	})

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, "queued", got["mode"])
	require.Equal(t, "ticket-1", got["ticket_id"])
	require.Equal(t, float64(1), got["position_human"])
	require.Equal(t, float64(0), got["position"])
	require.Equal(t, float64(2), got["line_len"])
}

// TestStartPipelineHandlesStartedPipelines verifies started output for non-runner pipelines.
func TestStartPipelineHandlesStartedPipelines(t *testing.T) {
	rec := map[string]any{
		"yaml":            "name: demo",
		"canonified_name": "pipeline123",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/pipeline/queue", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"mode":               "started",
			"workflow_id":        "wf-123",
			"run_id":             "run-456",
			"workflow_namespace": "org",
		}))
	}))
	defer server.Close()

	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()

	output := captureStdout(t, func() {
		require.NoError(t, startPipeline(context.Background(), "token", "org", rec))
	})

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, "started", got["mode"])
	require.Equal(t, "wf-123", got["workflow_id"])
	require.Equal(t, "run-456", got["run_id"])
	require.Equal(t, "org", got["workflow_namespace"])
}

func overrideHTTPDefaults(server *httptest.Server) func() {
	prevURL := instanceURL
	prevClient := http.DefaultClient

	instanceURL = server.URL
	http.DefaultClient = server.Client()

	return func() {
		instanceURL = prevURL
		http.DefaultClient = prevClient
	}
}

// captureStdout collects output written to stdout during the provided function.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = writer
	fn()
	_ = writer.Close()
	os.Stdout = orig

	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(output)
}
