// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"net/http"
	"testing"

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

func TestTransportOnlyOverridesQuickTunnels(t *testing.T) {
	require.Nil(t, Transport("https://runner.example"))
	require.Same(t, http.DefaultClient, HTTPClient("https://runner.example"))

	transport := Transport("https://demo.trycloudflare.com")
	require.NotNil(t, transport)
	require.NotSame(t, http.DefaultTransport, transport)
	// One transport is reused: a per-call transport would leak connection pools.
	require.Same(t, transport, Transport("https://other.trycloudflare.com"))
}

func TestURLUsable(t *testing.T) {
	require.True(t, URLUsable("https://runner.example"))
	require.True(t, URLUsable("http://192.168.1.10:8050"))
	require.False(t, URLUsable("192.168.1.10:8050"))
	require.False(t, URLUsable(""))
	require.False(t, URLUsable("ftp://runner.example"))
}
