// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package safehttp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		ip     string
		public bool
	}{
		{"127.0.0.1", false},
		{"10.1.2.3", false},
		{"172.16.0.1", false},
		{"192.168.1.1", false},
		{"169.254.169.254", false},
		{"0.0.0.0", false},
		{"224.0.0.1", false},
		{"100.64.0.1", false},
		{"100.127.255.254", false},
		{"::1", false},
		{"::", false},
		{"fe80::1", false},
		{"fc00::1", false},
		{"fd00::1", false},
		{"ff02::1", false},
		{"::ffff:127.0.0.1", false},
		{"::ffff:169.254.169.254", false},
		{"::ffff:100.64.0.1", false},
		{"100.63.255.255", true},
		{"100.128.0.0", true},
		{"8.8.8.8", true},
		{"93.184.216.34", true},
		{"2606:4700:4700::1111", true},
	}
	for _, tc := range tests {
		t.Run(tc.ip, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			require.NotNil(t, ip)
			require.Equal(t, tc.public, IsPublicIP(ip))
		})
	}
}

func countingServer(t *testing.T, addr string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusOK)
		}),
	)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Skipf("%s not bindable: %v", addr, err)
	}
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, &hits
}

func get(t *testing.T, client *http.Client, rawURL string) error {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func TestNewClientRejectsInternalDestinations(t *testing.T) {
	srv, hits := countingServer(t, "127.0.0.1:0")
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)

	client := NewClient(Config{Timeout: 5 * time.Second, MaxRedirects: 5})
	for _, rawURL := range []string{
		srv.URL,
		"http://localhost:" + port + "/",
		// 0.0.0.0 dials the local host.
		"http://0.0.0.0:" + port + "/",
		"http://[::ffff:127.0.0.1]:" + port + "/",
	} {
		t.Run(rawURL, func(t *testing.T) {
			require.ErrorIs(t, get(t, client, rawURL), ErrBlockedDestination)
		})
	}
	require.Zero(t, hits.Load(), "the internal server must never be contacted")
}

func TestNewClientRejectsRedirectToBlockedAddress(t *testing.T) {
	internal, internalHits := countingServer(t, "127.0.0.2:0")
	public := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL+"/latest/meta-data", http.StatusFound)
	}))
	t.Cleanup(public.Close)

	onlyPublicServer := func(ip net.IP) bool { return ip.Equal(net.IPv4(127, 0, 0, 1)) }
	client := NewClient(Config{Timeout: 5 * time.Second, MaxRedirects: 5, Allow: onlyPublicServer})

	require.ErrorIs(t, get(t, client, public.URL), ErrBlockedDestination)
	require.Zero(t, internalHits.Load())
}

func TestNewClientRedirectLimits(t *testing.T) {
	var hops atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops.Add(1)
		switch r.URL.Path {
		case "/ftp":
			http.Redirect(w, r, "ftp://example.com/x", http.StatusFound)
		case "/final":
			w.WriteHeader(http.StatusOK)
		default:
			http.Redirect(w, r, "/final", http.StatusFound)
		}
	}))
	t.Cleanup(srv.Close)
	allowAll := func(net.IP) bool { return true }

	t.Run("follows redirects up to the limit", func(t *testing.T) {
		client := NewClient(Config{Timeout: 5 * time.Second, MaxRedirects: 1, Allow: allowAll})
		require.NoError(t, get(t, client, srv.URL+"/start"))
	})
	t.Run("zero refuses redirects", func(t *testing.T) {
		client := NewClient(Config{Timeout: 5 * time.Second, Allow: allowAll})
		require.ErrorContains(t, get(t, client, srv.URL+"/start"), "stopped after 0 redirects")
	})
	t.Run("rejects non http scheme", func(t *testing.T) {
		client := NewClient(Config{Timeout: 5 * time.Second, MaxRedirects: 5, Allow: allowAll})
		require.ErrorContains(t, get(t, client, srv.URL+"/ftp"), "unsupported protocol scheme")
	})
}
