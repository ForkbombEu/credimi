// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestInternalHTTPActivityMissingEnv(t *testing.T) {
	t.Setenv("INTERNAL_ADMIN_API_KEY", "")
	activity := NewInternalHTTPActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: InternalHTTPActivityPayload{Method: http.MethodGet, URL: "https://example.com"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "INTERNAL_ADMIN_API_KEY is required")
}

func TestInternalHTTPActivityInjectsAPIKey(t *testing.T) {
	t.Setenv("INTERNAL_ADMIN_API_KEY", "secret-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "secret-key", r.Header.Get("X-Api-Key"))
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

func TestRedactHeaderMap(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer token")
	headers.Set("X-Api-Key", "abc")
	headers.Set("Cookie", "secret")
	headers.Set("X-Test", "value")

	redacted := redactHeaderMap(headers)
	require.Equal(t, []string{"[REDACTED]"}, redacted["Authorization"])
	require.Equal(t, []string{"[REDACTED]"}, redacted["X-Api-Key"])
	require.Equal(t, []string{"[REDACTED]"}, redacted["Cookie"])
	require.Equal(t, []string{"value"}, redacted["X-Test"])
}
