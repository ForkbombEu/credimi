// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/safehttp"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestIsQuickTunnelURL(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"https://demo.trycloudflare.com", true},
		{"https://demo.trycloudflare.com/health", true},
		{"https://DEMO.TryCloudflare.com", true},
		{"https://trycloudflare.com", false},
		{"https://trycloudflare.com.attacker.example", false},
		{"https://eviltrycloudflare.com", false},
		{"https://runner.example:8050", false},
		{"", false},
	} {
		require.Equal(t, tc.want, IsQuickTunnelURL(tc.url), tc.url)
	}
}

func newTransportTestRunner(runnerURL string, adminManaged bool) *core.Record {
	record := core.NewRecord(core.NewBaseCollection("mobile_runners"))
	record.Set("ip", runnerURL)
	record.Set("admin_managed", adminManaged)
	return record
}

func TestTransportForAdminManagedRunners(t *testing.T) {
	plain := newTransportTestRunner("http://runner.example", true)
	require.Nil(t, Transport(plain))
	require.Same(t, http.DefaultClient, HTTPClient(plain))

	transport := Transport(newTransportTestRunner("https://demo.trycloudflare.com", true))
	require.NotNil(t, transport)
	require.NotSame(t, http.DefaultTransport, transport)
	// One transport is reused: a per-call transport would leak connection pools.
	require.Same(
		t,
		transport,
		Transport(newTransportTestRunner("https://other.trycloudflare.com", true)),
	)
}

func TestTransportForTenantRunners(t *testing.T) {
	transport := Transport(newTransportTestRunner("https://runner.example", false))
	require.NotNil(t, transport, "tenant runners never use the unchecked default transport")
	require.Equal(t, transport, Transport(newTransportTestRunner("https://other.example", false)))

	tunnel := Transport(newTransportTestRunner("https://demo.trycloudflare.com", false))
	require.NotNil(t, tunnel)
	require.NotEqual(t, transport, tunnel)
	require.NotEqual(
		t,
		Transport(newTransportTestRunner("https://demo.trycloudflare.com", true)),
		tunnel,
	)
}

// A tenant chooses its runner's address, so Credimi must never reach its own
// network through it; an admin-managed runner is operator infrastructure and
// may live there.
func TestRunnerDestinationPolicy(t *testing.T) {
	var hits atomic.Int32
	var conns atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	countConns := func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	plainServer := httptest.NewUnstartedServer(handler)
	plainServer.Config.ConnState = countConns
	plainServer.Start()
	t.Cleanup(plainServer.Close)
	tlsServer := httptest.NewUnstartedServer(handler)
	tlsServer.Config.ConnState = countConns
	tlsServer.StartTLS()
	t.Cleanup(tlsServer.Close)

	for _, tc := range []struct {
		name         string
		runnerURL    string
		adminManaged bool
		wantBlocked  bool
	}{
		{name: "tenant runner over http", runnerURL: plainServer.URL, wantBlocked: true},
		{name: "tenant runner at loopback", runnerURL: tlsServer.URL, wantBlocked: true},
		{name: "tenant hostname resolving to loopback", runnerURL: "https://localhost:1", wantBlocked: true},
		{name: "admin-managed runner at loopback", runnerURL: plainServer.URL, adminManaged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hits.Store(0)
			conns.Store(0)
			runner := newTransportTestRunner(tc.runnerURL, tc.adminManaged)
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				RunnerURL(runner)+"/health",
				nil,
			)
			require.NoError(t, err)

			resp, err := HTTPClient(runner).Do(req)
			if tc.wantBlocked {
				require.ErrorIs(t, err, safehttp.ErrBlockedDestination)
				require.Zero(t, conns.Load(), "the runner must never be contacted")
				require.Zero(t, hits.Load())
				return
			}
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.EqualValues(t, 1, hits.Load())
		})
	}
}

// The address check runs when the connection is dialed, through
// tenantDialAllow, so it sees the resolved address and not the hostname.
func TestTenantRunnerDialChecksResolvedAddress(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	original := tenantDialAllow
	t.Cleanup(func() { tenantDialAllow = original })
	var dialed atomic.Value
	tenantDialAllow = func(ip net.IP) bool {
		dialed.Store(ip.String())
		return false
	}

	runner := newTransportTestRunner(
		strings.Replace(server.URL, "127.0.0.1", "localhost", 1),
		false,
	)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		RunnerURL(runner),
		nil,
	)
	require.NoError(t, err)
	_, err = HTTPClient(runner).Do(req)
	require.ErrorIs(t, err, safehttp.ErrBlockedDestination)
	require.Contains(t, []string{"127.0.0.1", "::1"}, dialed.Load())
}

func TestURLUsable(t *testing.T) {
	require.True(t, URLUsable("https://runner.example"))
	require.True(t, URLUsable("http://192.168.1.10:8050"))
	require.False(t, URLUsable("192.168.1.10:8050"))
	require.False(t, URLUsable(""))
	require.False(t, URLUsable("ftp://runner.example"))
}
