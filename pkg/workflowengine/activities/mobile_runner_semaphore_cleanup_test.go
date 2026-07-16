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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/require"
)

func TestCleanupMobileRunnerSemaphoreResourcesActivitySuccess(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "test-internal-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/wallet/temp-version/wallet-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/credential/temp/cred-1":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"missing"}`))
		case "/api/verifier/temp-use-case/usecase-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	act := NewCleanupMobileRunnerSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileRunnerSemaphoreResourcesActivityInput{
			AppURL: server.URL,
			Cleanup: &mobilerunnersemaphore.MobileRunnerSemaphoreCleanupMetadata{
				TempWalletVersionID: "wallet-1",
				TempCredentials: []mobilerunnersemaphore.MobileRunnerSemaphoreTempCredentialCleanupMetadata{
					{
						RecordID: "cred-1",
					},
				},
				TempUseCaseVerifications: []mobilerunnersemaphore.MobileRunnerSemaphoreTempCredentialCleanupMetadata{
					{
						RecordID: "usecase-1",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileRunnerSemaphoreResourcesActivityOutput)
	require.Empty(t, output.CleanupFailures)
}

func TestCleanupMobileRunnerSemaphoreResourcesActivityMissingAppURL(t *testing.T) {
	act := NewCleanupMobileRunnerSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileRunnerSemaphoreResourcesActivityInput{
			Cleanup: &mobilerunnersemaphore.MobileRunnerSemaphoreCleanupMetadata{
				TempWalletVersionID: "wallet-1",
			},
		},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileRunnerSemaphoreResourcesActivityOutput)
	require.Equal(
		t,
		[]string{"app_url missing for queued resource cleanup"},
		output.CleanupFailures,
	)
}

func TestCleanupMobileRunnerSemaphoreResourcesActivityFailureStatus(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "test-internal-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"owner mismatch"}`))
	}))
	defer server.Close()

	act := NewCleanupMobileRunnerSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileRunnerSemaphoreResourcesActivityInput{
			AppURL: server.URL,
			Cleanup: &mobilerunnersemaphore.MobileRunnerSemaphoreCleanupMetadata{
				TempWalletVersionID: "wallet-1",
			},
		},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileRunnerSemaphoreResourcesActivityOutput)
	require.Len(t, output.CleanupFailures, 1)
	require.Contains(t, output.CleanupFailures[0], "status 403")
}

func TestCleanupMobileRunnerSemaphoreResourcesActivityNilCleanup(t *testing.T) {
	act := NewCleanupMobileRunnerSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileRunnerSemaphoreResourcesActivityInput{},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileRunnerSemaphoreResourcesActivityOutput)
	require.Empty(t, output.CleanupFailures)
}

func TestDecodeInternalHTTPStatus(t *testing.T) {
	status, body := decodeInternalHTTPStatus(map[string]any{
		"status": float64(http.StatusAccepted),
		"body":   "ok",
	})
	require.Equal(t, http.StatusAccepted, status)
	require.Equal(t, "ok", body)

	status, body = decodeInternalHTTPStatus("unexpected")
	require.Equal(t, 0, status)
	require.Equal(t, "unexpected", body)
}
