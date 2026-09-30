// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

// A quick-tunnel hostname is created when cloudflared connects and needs a few
// seconds to propagate. A resolver queried inside that window caches the
// NXDOMAIN for the zone's negative TTL (30 minutes for trycloudflare.com), and
// every later lookup through that resolver keeps failing while the tunnel
// serves traffic normally. Cloudflare's own resolver sits next to the
// authoritative servers, so runner HTTP for those hosts resolves there.
const trycloudflareSuffix = ".trycloudflare.com"

var cloudflareDNSServers = []string{"1.1.1.1:53", "1.0.0.1:53"}

var (
	quickTunnelTransportOnce sync.Once
	quickTunnelTransport     http.RoundTripper
)

// HTTPClient returns the client to use for a runner URL: the
// Cloudflare-resolving client for quick tunnels, the default client otherwise.
func HTTPClient(runnerURL string) *http.Client {
	transport := Transport(runnerURL)
	if transport == nil {
		return http.DefaultClient
	}

	return &http.Client{Transport: transport}
}

// Transport returns the round tripper a runner URL must be called through, or
// nil when the default transport is correct. Callers that own their own client
// (an activity setting its own timeout) use this instead of HTTPClient.
func Transport(runnerURL string) http.RoundTripper {
	if !IsQuickTunnelURL(runnerURL) {
		return nil
	}
	quickTunnelTransportOnce.Do(func() {
		quickTunnelTransport = newQuickTunnelTransport(cloudflareDNSServers)
	})

	return quickTunnelTransport
}

func IsQuickTunnelURL(runnerURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(runnerURL))
	if err != nil {
		return false
	}
	hostname := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")

	return strings.HasSuffix(hostname, trycloudflareSuffix) &&
		len(hostname) > len(trycloudflareSuffix)
}

// URLUsable reports whether a runner URL can be called at all. A catalog
// surface that trusts heartbeats still has to exclude a runner whose stored URL
// is malformed: it reports healthy while being uncallable.
func URLUsable(runnerURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(runnerURL))
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func newQuickTunnelTransport(servers []string) http.RoundTripper {
	if len(servers) == 0 {
		return nil
	}
	var next atomic.Uint32
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			server := servers[(next.Add(1)-1)%uint32(len(servers))]
			if network == "" {
				network = "udp"
			}

			return (&net.Dialer{}).DialContext(ctx, network, server)
		},
	}
	transport, ok := http.DefaultTransport.(*http.Transport)
	if ok {
		transport = transport.Clone()
	} else {
		transport = &http.Transport{}
	}
	transport.DialContext = (&net.Dialer{Resolver: resolver}).DialContext

	return transport
}
