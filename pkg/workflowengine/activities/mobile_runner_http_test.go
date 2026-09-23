// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

// A runner call carries the same internal credential as any other internal
// call; the separate activity exists for its name and its resolver, not for a
// different contract.
func TestMobileRunnerHTTPActivityInjectsAPIKey(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "secret-key")
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "secret-key", r.Header.Get("Credimi-Api-Key"))
		require.Equal(t, "/credimi/installer-action", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	activity := NewMobileRunnerHTTPActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: MobileRunnerHTTPActivityPayload{
			Method:         http.MethodPost,
			URL:            server.URL + "/credimi/installer-action",
			ExpectedStatus: http.StatusOK,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

func TestMobileRunnerHTTPActivityRequiresInternalKey(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "")
	activity := NewMobileRunnerHTTPActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: MobileRunnerHTTPActivityPayload{
			Method: http.MethodGet,
			URL:    "https://runner.example",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CREDIMI_INTERNAL_ADMIN_KEY is required")
}

// The whole point of the dedicated activity: a quick-tunnel destination must
// not be resolved through the host resolver, which caches the hostname's
// propagation NXDOMAIN for half an hour.
func TestMobileRunnerHTTPActivityUsesQuickTunnelTransport(t *testing.T) {
	require.NotNil(t, mobilerunner.Transport("https://demo.trycloudflare.com"))
	require.Nil(t, mobilerunner.Transport("https://runner.example"))
}

func TestMobileRunnerHTTPActivityNameIsDistinct(t *testing.T) {
	require.NotEqual(t, NewInternalHTTPActivity().Name(), NewMobileRunnerHTTPActivity().Name())
}
