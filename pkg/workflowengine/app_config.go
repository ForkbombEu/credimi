// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"net"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/pocketbase/core"
)

// AppURLConfigKey holds the public, user-facing base URL of the Credimi
// deployment (PocketBase Settings → App URL). Workflow code reads it only to
// build links shown to users (test runs, emails, PR comments, screenshots).
const AppURLConfigKey = "app_url"

// AppNameConfigKey holds the application name from PocketBase Settings.
const AppNameConfigKey = "app_name"

// AppLogoConfigKey holds the URL of the application logo.
const AppLogoConfigKey = "app_logo"

const localURLStoreKey = "credimi.workflowengine.local_url"

// AppURL returns the public application URL from PocketBase Settings.
func AppURL(app core.App) string {
	return app.Settings().Meta.AppURL
}

// AppConfig returns the server-owned app config keys derived from PocketBase
// Settings.
func AppConfig(app core.App) map[string]any {
	appURL := AppURL(app)
	appName := app.Settings().Meta.AppName
	return map[string]any{
		AppURLConfigKey:  appURL,
		AppNameConfigKey: appName,
		AppLogoConfigKey: utils.JoinURL(
			appURL,
			"logos",
			strings.ToLower(appName)+"_logo-transp_emblem.png",
		),
	}
}

// WithAppConfig writes the server-owned app config keys into config, replacing
// any existing values, and returns it. A nil config is allocated.
func WithAppConfig(app core.App, config map[string]any) map[string]any {
	if config == nil {
		config = make(map[string]any, 3)
	}
	for key, value := range AppConfig(app) {
		config[key] = value
	}
	return config
}

// IsServerOwnedConfigKey reports whether key may be set in workflow config only
// by the server. User pipeline YAML, step config, and rerun bodies must never
// set these keys.
func IsServerOwnedConfigKey(key string) bool {
	switch key {
	case AppURLConfigKey, AppNameConfigKey, AppLogoConfigKey:
		return true
	default:
		return false
	}
}

// SetLocalURL stores the loopback base URL of the HTTP server the app is
// serving on, so in-process workers can reach it.
func SetLocalURL(app core.App, server *http.Server) {
	host, port, err := net.SplitHostPort(server.Addr)
	if err != nil {
		return
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	scheme := "http"
	if server.TLSConfig != nil {
		scheme = "https"
	}
	app.Store().Set(localURLStoreKey, scheme+"://"+net.JoinHostPort(host, port))
}

// LocalURL returns the URL stored by SetLocalURL, or "" when unset.
func LocalURL(app core.App) string {
	value, _ := app.Store().Get(localURLStoreKey).(string)
	return value
}
