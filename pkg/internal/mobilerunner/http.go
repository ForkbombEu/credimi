// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/forkbombeu/credimi/pkg/internal/safehttp"
	"github.com/pocketbase/pocketbase/core"
)

// A quick-tunnel hostname is created when cloudflared connects and needs a few
// seconds to propagate. A resolver queried inside that window caches the
// NXDOMAIN for the zone's negative TTL (30 minutes for trycloudflare.com), and
// every later lookup through that resolver keeps failing while the tunnel
// serves traffic normally. Cloudflare's own resolver sits next to the
// authoritative servers, so runner HTTP for those hosts resolves there.
const trycloudflareSuffix = ".trycloudflare.com"

var cloudflareDNSServers = []string{"1.1.1.1:53", "1.0.0.1:53"}

// tenantDialAllow decides which addresses a tenant runner may be dialed at.
// Only tests that serve a tenant runner on loopback replace it.
var tenantDialAllow = safehttp.IsPublicIP

var (
	quickTunnelTransportOnce sync.Once
	quickTunnelTransport     http.RoundTripper

	tenantTransportOnce        sync.Once
	tenantTransport            http.RoundTripper
	tenantQuickTunnelTransport http.RoundTripper
)

// HTTPClient returns the client every Credimi call to runner must use; see
// Transport.
func HTTPClient(runner *core.Record) *http.Client {
	transport := Transport(runner)
	if transport == nil {
		return http.DefaultClient
	}

	return &http.Client{Transport: transport}
}

// Transport returns the round tripper every Credimi call to runner must go
// through, or nil when the default transport is correct. Callers that own
// their own client (an activity setting its own timeout) use this instead of
// HTTPClient.
//
// Admin-managed runners are operator infrastructure: they get the default
// transport, or the Cloudflare-resolving one for quick tunnels. A tenant
// chooses its runner's address, so tenant runners are reached only over https,
// without a proxy, and only at public addresses checked on every dial, which
// also covers redirects and rebinding hostnames. A rejected destination fails
// with an error wrapping safehttp.ErrBlockedDestination.
func Transport(runner *core.Record) http.RoundTripper {
	quickTunnel := IsQuickTunnelURL(RunnerURL(runner))
	if runner.GetBool("admin_managed") {
		if !quickTunnel {
			return nil
		}
		quickTunnelTransportOnce.Do(func() {
			quickTunnelTransport = newRunnerTransport(
				newCloudflareResolver(cloudflareDNSServers),
				nil,
			)
		})
		return quickTunnelTransport
	}

	tenantTransportOnce.Do(func() {
		control := safehttp.DialControl(func(ip net.IP) bool { return tenantDialAllow(ip) })
		tenantTransport = httpsOnlyTransport{next: newRunnerTransport(nil, control)}
		tenantQuickTunnelTransport = httpsOnlyTransport{
			next: newRunnerTransport(newCloudflareResolver(cloudflareDNSServers), control),
		}
	})
	if quickTunnel {
		return tenantQuickTunnelTransport
	}
	return tenantTransport
}

// httpsOnlyTransport refuses every request, redirects included, that is not
// https.
type httpsOnlyTransport struct {
	next http.RoundTripper
}

func (t httpsOnlyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, fmt.Errorf(
			"%w: tenant runners must be reached over https, got %s",
			safehttp.ErrBlockedDestination,
			req.URL.Redacted(),
		)
	}

	return t.next.RoundTrip(req)
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

func newCloudflareResolver(servers []string) *net.Resolver {
	var next atomic.Uint32
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			server := servers[(next.Add(1)-1)%uint32(len(servers))]
			if network == "" {
				network = "udp"
			}

			return (&net.Dialer{}).DialContext(ctx, network, server)
		},
	}
}

// newRunnerTransport clones the default transport, resolving through resolver
// when it is set. A non-nil control checks every dialed address, and then no
// proxy is used: the proxy would be the checked address instead of the runner.
func newRunnerTransport(
	resolver *net.Resolver,
	control func(network, address string, c syscall.RawConn) error,
) *http.Transport {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if ok {
		transport = transport.Clone()
	} else {
		transport = &http.Transport{}
	}
	transport.DialContext = (&net.Dialer{Resolver: resolver, Control: control}).DialContext
	if control != nil {
		transport.Proxy = nil
	}

	return transport
}
