// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

// AppURLConfigKey holds the public, user-facing base URL of the Credimi
// deployment (PocketBase Settings → App URL). It is persisted in workflow
// inputs, queue tickets, schedule configs, and memos, and is used to build
// links shown to users (test runs, emails, PR comments, screenshots).
const AppURLConfigKey = "app_url"

// InternalAppURLConfigKey optionally holds a deployment-local base URL for
// server-to-server HTTP calls from Temporal workflows and activities back to
// the Credimi API. When the app URL is proxied by a Cloudflare WAF that
// challenges non-browser traffic, set CREDIMI_INTERNAL_APP_URL so callbacks
// bypass the proxy. Handlers inject it into workflow config at start time.
const InternalAppURLConfigKey = "internal_app_url"

// InternalAppURLConfigKeyEnv is the environment variable handlers read to
// populate InternalAppURLConfigKey in workflow input configs.
const InternalAppURLConfigKeyEnv = "CREDIMI_INTERNAL_APP_URL"

// InternalAppURLOverride returns the deployment-local base URL from the
// environment, trimmed and empty when unset.
func InternalAppURLOverride() string {
	return strings.TrimSpace(os.Getenv(InternalAppURLConfigKeyEnv))
}

// IsServerOwnedConfigKey reports whether key holds a Credimi base URL that only
// the server may set in workflow config. InternalAppURLFromConfig trusts these
// keys as the destination of calls that carry the internal admin key, so user
// pipeline YAML, step config, and rerun bodies must never set them.
func IsServerOwnedConfigKey(key string) bool {
	return key == AppURLConfigKey || key == InternalAppURLConfigKey
}

var serverAppURLSource atomic.Pointer[func() string]

// SetServerAppURLSource registers the server's App URL setting (PocketBase
// Settings → App URL). Workers call ValidateInternalAppURLDestination, which
// accepts that origin and the CREDIMI_INTERNAL_APP_URL origin only.
func SetServerAppURLSource(source func() string) {
	serverAppURLSource.Store(&source)
}

// ValidateInternalAppURLDestination returns an error unless rawURL has the
// origin of a server-configured Credimi base URL: CREDIMI_INTERNAL_APP_URL or
// the App URL registered through SetServerAppURLSource. Callers check it before
// sending the internal admin key.
func ValidateInternalAppURLDestination(rawURL string) error {
	allowed := make([]string, 0, 2)
	if internalURL := InternalAppURLOverride(); internalURL != "" {
		allowed = append(allowed, internalURL)
	}
	if source := serverAppURLSource.Load(); source != nil {
		if appURL := strings.TrimSpace((*source)()); appURL != "" {
			allowed = append(allowed, appURL)
		}
	}
	if len(allowed) == 0 {
		return fmt.Errorf(
			"no Credimi base URL is configured: set %s or the App URL",
			InternalAppURLConfigKeyEnv,
		)
	}
	destination, ok := urlOrigin(rawURL)
	if !ok {
		return fmt.Errorf("invalid destination URL %q", rawURL)
	}
	for _, base := range allowed {
		if origin, ok := urlOrigin(base); ok && origin == destination {
			return nil
		}
	}
	return fmt.Errorf("destination %q is not a configured Credimi base URL", destination)
}

// urlOrigin returns scheme://host:port for an absolute http(s) URL, with the
// scheme and host lowercased and the default port made explicit.
func urlOrigin(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Hostname() == "" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	port := parsed.Port()
	switch {
	case scheme == "http" && port == "":
		port = "80"
	case scheme == "https" && port == "":
		port = "443"
	case scheme != "http" && scheme != "https":
		return "", false
	}
	return scheme + "://" + strings.ToLower(parsed.Hostname()) + ":" + port, true
}

// InternalAppURLFromConfig returns the base URL that workflows and activities
// must use for HTTP callbacks to the Credimi API. It prefers
// InternalAppURLConfigKey and falls back to AppURLConfigKey, so behavior is
// unchanged in deployments that do not configure an internal override.
func InternalAppURLFromConfig(config map[string]any) string {
	if config != nil {
		if v, ok := config[InternalAppURLConfigKey].(string); ok {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	v, _ := config[AppURLConfigKey].(string)
	return strings.TrimSpace(v)
}

// WithInternalAppURL returns config unchanged when no deployment-local
// override is configured, and otherwise injects InternalAppURLConfigKey describing
// it. Handlers that build workflow input config use this so worker-side HTTP
// callbacks can bypass a Cloudflare WAF while user-facing links keep the
// public app_url.
func WithInternalAppURL(config map[string]any) map[string]any {
	if internalURL := InternalAppURLOverride(); internalURL != "" {
		config[InternalAppURLConfigKey] = internalURL
	}
	return config
}
