// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

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
	quickTunnelClientOnce sync.Once
	quickTunnelClient     *http.Client
)

// mobileRunnerHTTPClient returns the client to use for a runner URL: the
// Cloudflare-resolving client for quick tunnels, the default client otherwise.
func mobileRunnerHTTPClient(runnerURL string) *http.Client {
	if !isQuickTunnelURL(runnerURL) {
		return http.DefaultClient
	}
	quickTunnelClientOnce.Do(func() {
		quickTunnelClient = newQuickTunnelHTTPClient(cloudflareDNSServers)
	})

	return quickTunnelClient
}

func isQuickTunnelURL(runnerURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(runnerURL))
	if err != nil {
		return false
	}
	hostname := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")

	return strings.HasSuffix(hostname, trycloudflareSuffix) &&
		len(hostname) > len(trycloudflareSuffix)
}

// mobileRunnerURLUsable reports whether a runner URL can be called at all.
// A catalog surface that trusts heartbeats still has to exclude a runner whose
// stored URL is malformed: it reports healthy while being uncallable.
func mobileRunnerURLUsable(runnerURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(runnerURL))
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func newQuickTunnelHTTPClient(servers []string) *http.Client {
	if len(servers) == 0 {
		return http.DefaultClient
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
	dialer := &net.Dialer{Resolver: resolver}
	transport.DialContext = dialer.DialContext

	return &http.Client{Transport: transport}
}
