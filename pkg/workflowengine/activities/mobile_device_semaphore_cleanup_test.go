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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/require"
)

func TestCleanupMobileDeviceSemaphoreResourcesActivitySuccess(t *testing.T) {
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

	act := NewCleanupMobileDeviceSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileDeviceSemaphoreResourcesActivityInput{
			AppURL: server.URL,
			Cleanup: &mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata{
				TempWalletVersionID: "wallet-1",
				TempCredentials: []mobiledevicesemaphore.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
					{
						RecordID: "cred-1",
					},
				},
				TempUseCaseVerifications: []mobiledevicesemaphore.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
					{
						RecordID: "usecase-1",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileDeviceSemaphoreResourcesActivityOutput)
	require.Empty(t, output.CleanupFailures)
}

func TestCleanupMobileDeviceSemaphoreResourcesActivityMissingAppURL(t *testing.T) {
	act := NewCleanupMobileDeviceSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileDeviceSemaphoreResourcesActivityInput{
			Cleanup: &mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata{
				TempWalletVersionID: "wallet-1",
			},
		},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileDeviceSemaphoreResourcesActivityOutput)
	require.Equal(
		t,
		[]string{"app_url missing for queued resource cleanup"},
		output.CleanupFailures,
	)
}

func TestCleanupMobileDeviceSemaphoreResourcesActivityFailureStatus(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "test-internal-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"owner mismatch"}`))
	}))
	defer server.Close()

	act := NewCleanupMobileDeviceSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileDeviceSemaphoreResourcesActivityInput{
			AppURL: server.URL,
			Cleanup: &mobiledevicesemaphore.MobileDeviceSemaphoreCleanupMetadata{
				TempWalletVersionID: "wallet-1",
			},
		},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileDeviceSemaphoreResourcesActivityOutput)
	require.Len(t, output.CleanupFailures, 1)
	require.Contains(t, output.CleanupFailures[0], "status 403")
}

func TestCleanupMobileDeviceSemaphoreResourcesActivityNilCleanup(t *testing.T) {
	act := NewCleanupMobileDeviceSemaphoreResourcesActivity()
	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CleanupMobileDeviceSemaphoreResourcesActivityInput{},
	})
	require.NoError(t, err)

	output := result.Output.(CleanupMobileDeviceSemaphoreResourcesActivityOutput)
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
