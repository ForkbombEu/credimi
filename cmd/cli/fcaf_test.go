// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetryableFCAFQueueStatus(t *testing.T) {
	t.Parallel()

	require.False(t, retryableFCAFQueueStatus(http.StatusOK))
	require.False(t, retryableFCAFQueueStatus(http.StatusBadRequest))
	require.False(t, retryableFCAFQueueStatus(http.StatusUnauthorized))
	require.True(t, retryableFCAFQueueStatus(http.StatusRequestTimeout))
	require.True(t, retryableFCAFQueueStatus(http.StatusTooManyRequests))
	require.True(t, retryableFCAFQueueStatus(http.StatusInternalServerError))
	require.True(t, retryableFCAFQueueStatus(http.StatusServiceUnavailable))
}

func TestRewriteFCAFOrganization(t *testing.T) {
	t.Parallel()

	input := []byte(
		"global_device_id: fcaf-1/usb/device\n" +
			"action_id: fcaf-1/eudiw-beta-wallet/onboarding-1\n",
	)

	t.Run("rewrites to target org", func(t *testing.T) {
		got := string(rewriteFCAFOrganization(input, "acme-org"))
		require.Contains(t, got, "global_device_id: acme-org/usb/device")
		require.Contains(t, got, "action_id: acme-org/eudiw-beta-wallet/onboarding-1")
		require.NotContains(t, got, "fcaf-1/")
	})

	t.Run("keeps source org unchanged", func(t *testing.T) {
		require.Equal(t, string(input), string(rewriteFCAFOrganization(input, fcafSourceOrg)))
	})

	t.Run("empty target org is a no-op", func(t *testing.T) {
		require.Equal(t, string(input), string(rewriteFCAFOrganization(input, "")))
	})
}

func TestWalkFCAFImports(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	write("bundle/credentials/issuer/cred.yaml", "yaml: x\n")
	write("bundle/wallet/action.yaml", "yaml: y\n")
	write("bundle/credential-issuers/issuer.yaml", "yaml: z\n")

	paths, err := walkFCAFImports(root, "/credentials/")
	require.NoError(t, err)
	require.Len(t, paths, 1)
	require.Contains(t, paths[0], "credentials")
}

func TestSyncFCAFWalletActionSyncsTags(t *testing.T) {
	tests := []struct {
		name    string
		records []map[string]any
		method  string
		path    string
	}{
		{
			name:    "create",
			records: []map[string]any{},
			method:  http.MethodPost,
			path:    "/api/collections/wallet_actions/records",
		},
		{
			name:    "update",
			records: []map[string]any{{"id": "action-id"}},
			method:  http.MethodPatch,
			path:    "/api/collections/wallet_actions/records/action-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var payload map[string]any
			server := newTestServer(
				t,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet {
						require.Equal(
							t,
							`owner="org-id" && wallet="wallet-id" && name="action"`,
							r.URL.Query().Get("filter"),
						)
						require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
							"items": test.records,
						}))
						return
					}
					require.Equal(t, test.method, r.Method)
					require.Equal(t, test.path, r.URL.Path)
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				}),
			)
			defer server.Close()
			restoreDefaults := overrideHTTPDefaults(server)
			defer restoreDefaults()

			err := syncFCAFWalletAction(
				context.Background(),
				"token",
				"org-id",
				"wallet-id",
				fcafWalletActionRef{
					Name:     "action",
					Category: "other",
					Tags:     "fcaf, ws_rp, helper",
				},
				"code",
			)
			require.NoError(t, err)
			require.Equal(t, "fcaf, ws_rp, helper", payload["tags"])
		})
	}
}

func TestRunWalletFCAFBuildsThePipelineOnCredimi(t *testing.T) {
	var payload map[string]any
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/action-author/fcaf-pipeline", r.URL.Path)
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"pipeline":   "acme/fcaf-relying-party-validation-wallet-1-scenarios",
			"ticket_id":  "ticket",
			"device_id":  "acme/runner/pixel",
			"status":     "completed",
			"run_url":    "https://credimi.test/my/tests/runs/wf/run",
			"skipped":    []any{map[string]any{"scenario": "s.yaml", "missing_roles": []any{"r"}}},
			"test_count": 3,
		}))
	}))
	defer server.Close()
	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()
	wallet, device, scenarios := fcafWallet, fcafDeviceID, fcafScenarios
	t.Cleanup(func() { fcafWallet, fcafDeviceID, fcafScenarios = wallet, device, scenarios })
	fcafWallet = "acme/wallet/1-0"
	fcafDeviceID = "acme/runner/pixel"
	fcafScenarios = []string{"s.yaml"}

	run, err := runWalletFCAF(context.Background(), "token")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"wallet_version_identifier": "acme/wallet/1-0",
		"device_id":                 "acme/runner/pixel",
		"scenarios":                 []any{"s.yaml"},
	}, payload)
	require.Equal(t, "completed", run.Status)
	require.Equal(t, "fcaf-relying-party-validation-wallet-1-scenarios", run.Name)
}

// TestRunWalletFCAFReportFailsWhenTheEndpointFails pins that --wallet exits
// non-zero with the endpoint's answer when Credimi refuses the build.
func TestRunWalletFCAFReportFailsWhenTheEndpointFails(t *testing.T) {
	const refusal = `{"message":"only an owner of the organization can build and run ` +
		`its FCAF pipelines"}`
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(refusal))
	}))
	defer server.Close()
	restoreDefaults := overrideHTTPDefaults(server)
	defer restoreDefaults()
	wallet, device, output := fcafWallet, fcafDeviceID, fcafOutput
	t.Cleanup(func() { fcafWallet, fcafDeviceID, fcafOutput = wallet, device, output })
	fcafWallet, fcafDeviceID, fcafOutput = "acme/wallet/1-0", "acme/runner/pixel", ""

	err := runWalletFCAFReport(context.Background(), "token")
	require.ErrorContains(t, err, "HTTP 403")
	require.ErrorContains(t, err, "only an owner of the organization")
}

func TestApplyFCAFWalletFlags(t *testing.T) {
	wallet, scenarios, timeout := fcafWallet, fcafScenarios, fcafTimeout
	t.Cleanup(func() { fcafWallet, fcafScenarios, fcafTimeout = wallet, scenarios, timeout })

	fcafWallet, fcafScenarios = "", []string{"s.yaml"}
	require.ErrorContains(t, applyFCAFWalletFlags(false), "--scenarios needs --wallet")

	fcafWallet, fcafTimeout = "acme/wallet/1-0", time.Minute
	require.NoError(t, applyFCAFWalletFlags(true))
	require.Equal(t, time.Minute, fcafTimeout, "an explicit --timeout stays")
	require.NoError(t, applyFCAFWalletFlags(false))
	require.Equal(t, fcafWalletTimeout, fcafTimeout)
}
