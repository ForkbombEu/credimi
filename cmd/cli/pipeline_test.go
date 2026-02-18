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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const minimalPipelineYAML = `
name: Test Pipeline
steps:
  - id: step1
    use: rest
`

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = orig

	output, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(output)
}

func TestParsePipelineName(t *testing.T) {
	name, err := parsePipelineName([]byte("name: demo\n"))
	require.NoError(t, err)
	require.Equal(t, "demo", name)

	_, err = parsePipelineName([]byte(":bad"))
	require.Error(t, err)
}

func TestDecodeJSONPayload(t *testing.T) {
	payload := decodeJSONPayload([]byte(`{"ok":true}`))
	require.Equal(t, map[string]any{"ok": true}, payload)

	payload = decodeJSONPayload([]byte("not-json"))
	raw := payload.(map[string]any)["raw"].(string)
	require.Contains(t, raw, "not-json")
}

func TestAuthenticateAndGetOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/apikey/authenticate":
			if r.Header.Get("X-Api-Key") != "key" {
				http.Error(w, "bad key", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
		case "/api/organizations/my":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer tok" {
				http.Error(w, "bad token", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":              "org-1",
				"canonified_name": "acme",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origURL := instanceURL
	origKey := apiKey
	t.Cleanup(func() {
		instanceURL = origURL
		apiKey = origKey
	})

	instanceURL = server.URL
	apiKey = "key"

	token, err := authenticate(context.Background())
	require.NoError(t, err)
	require.Equal(t, "tok", token)

	orgID, canon, err := getMyOrganization(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "org-1", orgID)
	require.Equal(t, "acme", canon)
}

func TestFindOrCreatePipeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/collections/pipelines/records":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "pipe-1", "canonified_name": "pipe-one"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/collections/pipelines/records":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "pipe-2", "canonified_name": "pipe-two"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	origURL := instanceURL
	t.Cleanup(func() {
		instanceURL = origURL
	})
	instanceURL = server.URL

	input := &PipelineCLIInput{Name: "pipe", YAML: minimalPipelineYAML}
	existing, err := findOrCreatePipeline(context.Background(), "token", "org-1", input)
	require.NoError(t, err)
	require.Equal(t, "pipe-1", existing["id"])

	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/collections/pipelines/records":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/collections/pipelines/records":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "pipe-2", "canonified_name": "pipe-two"})
		default:
			http.NotFound(w, r)
		}
	})

	created, err := findOrCreatePipeline(context.Background(), "token", "org-1", input)
	require.NoError(t, err)
	require.Equal(t, "pipe-2", created["id"])
}

func TestCreatePipeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/collections/pipelines/records" {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		require.Equal(t, "Pipeline stored from CLI", payload["description"])
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "pipe-3"})
	}))
	defer server.Close()

	origURL := instanceURL
	t.Cleanup(func() {
		instanceURL = origURL
	})
	instanceURL = server.URL

	input := &PipelineCLIInput{Name: "pipe", YAML: minimalPipelineYAML}
	created, err := createPipeline(context.Background(), "token", "org-1", input)
	require.NoError(t, err)
	require.Equal(t, "pipe-3", created["id"])
}

func TestReadPipelineInputFromFile(t *testing.T) {
	temp := t.TempDir()
	filePath := filepath.Join(temp, "pipeline.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte("name: demo\n"), 0600))

	origPath := yamlPath
	t.Cleanup(func() {
		yamlPath = origPath
	})
	yamlPath = filePath

	input, err := readPipelineInput()
	require.NoError(t, err)
	require.Equal(t, "demo", input.Name)
}

func TestStartPipelineQueued(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/pipeline/queue" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(pipelineQueueResponse{
			Mode:      "queued",
			TicketID:  "ticket-1",
			RunnerIDs: []string{"runner-1"},
			Position:  0,
			LineLen:   1,
		})
	}))
	defer server.Close()

	origURL := instanceURL
	t.Cleanup(func() {
		instanceURL = origURL
	})
	instanceURL = server.URL

	rec := map[string]any{
		"yaml":            minimalPipelineYAML,
		"canonified_name": "pipe-one",
	}

	err := startPipeline(context.Background(), "token", "acme", rec)
	require.NoError(t, err)
}

func TestStartPipelineStarted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/pipeline/queue" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(pipelineQueueResponse{
			Mode:              "started",
			WorkflowID:        "wf-1",
			RunID:             "run-1",
			WorkflowNamespace: "ns-1",
		})
	}))
	defer server.Close()

	origURL := instanceURL
	t.Cleanup(func() {
		instanceURL = origURL
	})
	instanceURL = server.URL

	rec := map[string]any{
		"yaml":            minimalPipelineYAML,
		"canonified_name": "pipe-one",
	}

	err := startPipeline(context.Background(), "token", "acme", rec)
	require.NoError(t, err)
}
