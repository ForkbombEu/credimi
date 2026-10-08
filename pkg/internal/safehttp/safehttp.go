// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package safehttp builds HTTP clients for fetching user-chosen URLs. Every
// connection they open, including the ones that follow redirects, must target
// a public unicast address.
package safehttp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// ErrBlockedDestination reports a connection refused because its address is
// not public.
var ErrBlockedDestination = errors.New("destination is not a public address")

// sharedAddressSpace is the carrier-grade NAT range (RFC 6598).
var sharedAddressSpace = &net.IPNet{
	IP:   net.IPv4(100, 64, 0, 0).To4(),
	Mask: net.CIDRMask(10, 32),
}

// IsPublicIP reports whether ip is a public unicast address. It rejects
// loopback, private and ULA (fc00::/7), unspecified, link-local (which holds
// the 169.254.169.254 cloud metadata address), multicast and CGNAT addresses.
func IsPublicIP(ip net.IP) bool {
	return !ip.IsLoopback() &&
		!ip.IsPrivate() &&
		!ip.IsUnspecified() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsMulticast() &&
		!sharedAddressSpace.Contains(ip)
}

// Config sets the limits of a client returned by NewClient.
type Config struct {
	// Timeout bounds the dial and the whole request; zero means no limit.
	Timeout time.Duration
	// MaxRedirects is the number of redirects followed before failing.
	MaxRedirects int
	// Allow decides which dialed addresses are accepted. Nil means IsPublicIP.
	// Only tests that talk to loopback servers set it.
	Allow func(net.IP) bool
}

// DialControl returns a net.Dialer Control function that refuses, after DNS
// resolution, every address allow rejects. Nil allow means IsPublicIP.
// Rejections wrap ErrBlockedDestination.
func DialControl(allow func(net.IP) bool) func(network, address string, c syscall.RawConn) error {
	if allow == nil {
		allow = IsPublicIP
	}

	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrBlockedDestination, address)
		}
		if ip := net.ParseIP(host); ip == nil || !allow(ip) {
			return fmt.Errorf("%w: %s", ErrBlockedDestination, host)
		}
		return nil
	}
}

// NewClient returns a client that checks every dialed address after DNS
// resolution, so redirects and rebinding hostnames cannot reach an address
// Allow rejects. Redirects may only use http or https.
func NewClient(cfg Config) *http.Client {
	dialer := &net.Dialer{
		Timeout: cfg.Timeout,
		Control: DialControl(cfg.Allow),
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A proxy would be the dialed address, hiding the real destination.
	transport.Proxy = nil
	transport.DialContext = dialer.DialContext
	transport.DisableKeepAlives = true

	maxRedirects := cfg.MaxRedirects
	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// via holds every earlier request, so it has one entry per redirect.
			if len(via) > maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("unsupported protocol scheme %q", req.URL.Scheme)
			}
			return nil
		},
	}
}
