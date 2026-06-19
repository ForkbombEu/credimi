// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestInternalHTTPActivityMissingEnv(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "")
	activity := NewInternalHTTPActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: InternalHTTPActivityPayload{Method: http.MethodGet, URL: "https://example.com"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CREDIMI_INTERNAL_ADMIN_KEY is required")
}

func TestInternalHTTPActivityInjectsAPIKey(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "secret-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "secret-key", r.Header.Get("Credimi-Api-Key"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	activity := NewInternalHTTPActivity()
	res, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: InternalHTTPActivityPayload{
			Method:         http.MethodGet,
			URL:            server.URL,
			ExpectedStatus: http.StatusOK,
		},
	})
	require.NoError(t, err)
	output, ok := res.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.StatusOK, output["status"])
}

func TestInternalHTTPActivityOmittedExpectedStatusDoesNotExpectZero(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "secret-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	activity := NewInternalHTTPActivity()
	res, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: InternalHTTPActivityPayload{
			Method: http.MethodGet,
			URL:    server.URL,
		},
	})
	require.NoError(t, err)
	output, ok := res.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.StatusOK, output["status"])
}

func TestRedactHeaderMap(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer token")
	headers.Set("Credimi-Api-Key", "abc")
	headers.Set("Cookie", "secret")
	headers.Set("X-Test", "value")

	redacted := redactHeaderMap(headers)
	require.Equal(t, []string{"[REDACTED]"}, redacted["Authorization"])
	require.Equal(t, []string{"[REDACTED]"}, redacted["Credimi-Api-Key"])
	require.Equal(t, []string{"[REDACTED]"}, redacted["Cookie"])
	require.Equal(t, []string{"value"}, redacted["X-Test"])
}

func TestSplitSecretsFromOutput(t *testing.T) {
	output, secrets, err := splitSecretsFromOutput(map[string]any{
		"code":    "steps: []",
		"secrets": map[string]any{"token": "secret-value"},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"code": "steps: []"}, output)
	require.Equal(t, map[string]any{"token": "secret-value"}, secrets)

	dc := temporalcrypto.NewDataConverter(bytes.Repeat([]byte{1}, 32))
	payload, err := dc.ToPayload(workflowengine.ActivityResult{
		Output:  output,
		Secrets: secrets,
	})
	require.NoError(t, err)
	require.Contains(t, string(payload.GetData()), `"code":"steps: []"`)
	require.NotContains(t, string(payload.GetData()), "secret-value")
}
