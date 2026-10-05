// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const (
	// maxLogoSize matches the maxSize of the wallets.logo file field.
	maxLogoSize      = 2 << 20
	logoFetchTimeout = 10 * time.Second
	maxLogoRedirects = 5
)

var errBlockedDestination = errors.New("logo destination is not a public address")

// allowedLogoContentTypes are the sniffed types the logo file fields accept.
var allowedLogoContentTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

func LogoHooks(app core.App) {
	bindLogoHooks(app, newLogoHTTPClient(isPublicIP))
}

func bindLogoHooks(app core.App, client *http.Client) {
	handle := func(e *core.RecordEvent) error {
		return handleLogo(e, client)
	}
	app.OnRecordCreate().BindFunc(handle)
	app.OnRecordUpdate().BindFunc(handle)
}

func handleLogo(e *core.RecordEvent, client *http.Client) error {
	logos := e.Record.GetUnsavedFiles("logo")
	if len(logos) > 0 {
		return e.Next()
	}

	logoURL := e.Record.GetString("logo_url")
	if logoURL == "" {
		return e.Next()
	}

	originalLogoURL := e.Record.Original().GetString("logo_url")
	if originalLogoURL == logoURL {
		return e.Next()
	}

	file, err := downloadImage(e.Context, client, logoURL)
	if err != nil {
		log.Printf("ERROR download: %v", err)
		return e.Next()
	}

	e.Record.Set("logo", []*filesystem.File{file})
	return e.Next()
}

// DownloadImage fetches a logo from a public http(s) host and returns it only
// when it is a bounded image.
func DownloadImage(ctx context.Context, imageURL string) (*filesystem.File, error) {
	return downloadImage(ctx, newLogoHTTPClient(isPublicIP), imageURL)
}

func downloadImage(
	ctx context.Context,
	client *http.Client,
	imageURL string,
) (*filesystem.File, error) {
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if err := checkLogoURL(parsed); err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxLogoSize+1))
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty image")
	}
	if len(data) > maxLogoSize {
		return nil, fmt.Errorf("image exceeds %d bytes", maxLogoSize)
	}
	if contentType := http.DetectContentType(data); !allowedLogoContentTypes[contentType] {
		return nil, fmt.Errorf("unsupported image content type %q", contentType)
	}

	filename := extractFilenameFromURL(imageURL)
	return filesystem.NewFileFromBytes(data, filename)
}

func checkLogoURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported protocol scheme %q", u.Scheme)
	}
	if u.Hostname() == "" {
		return errors.New("missing host")
	}
	return nil
}

// newLogoHTTPClient checks every dialed address after DNS resolution, so
// redirects and rebinding hostnames cannot reach an address allow rejects.
func newLogoHTTPClient(allow func(net.IP) bool) *http.Client {
	dialer := &net.Dialer{
		Timeout: logoFetchTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("%w: %s", errBlockedDestination, address)
			}
			if ip := net.ParseIP(host); ip == nil || !allow(ip) {
				return fmt.Errorf("%w: %s", errBlockedDestination, host)
			}
			return nil
		},
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A proxy would be the dialed address, hiding the real destination.
	transport.Proxy = nil
	transport.DialContext = dialer.DialContext
	transport.DisableKeepAlives = true

	return &http.Client{
		Timeout:   logoFetchTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxLogoRedirects {
				return fmt.Errorf("stopped after %d redirects", maxLogoRedirects)
			}
			return checkLogoURL(req.URL)
		},
	}
}

func isPublicIP(ip net.IP) bool {
	return !ip.IsLoopback() &&
		!ip.IsPrivate() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsMulticast() &&
		!ip.IsUnspecified()
}

func extractFilenameFromURL(imageURL string) string {
	parts := strings.Split(imageURL, "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		cleanURL := strings.ReplaceAll(imageURL, "://", "_")
		cleanURL = strings.ReplaceAll(cleanURL, "/", "_")
		cleanURL = strings.ReplaceAll(cleanURL, "?", "_")
		return cleanURL + ".jpg"
	}

	lastPart := parts[len(parts)-1]
	if idx := strings.Index(lastPart, "?"); idx != -1 {
		lastPart = lastPart[:idx]
	}

	if idx := strings.Index(lastPart, "#"); idx != -1 {
		lastPart = lastPart[:idx]
	}

	if !strings.Contains(lastPart, ".") {
		lastPart += ".jpg"
	}

	return lastPart
}
