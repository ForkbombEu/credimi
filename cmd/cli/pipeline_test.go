// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
