// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
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

// AppURL returns the public application URL from PocketBase Settings.
func AppURL(app core.App) string {
	return app.Settings().Meta.AppURL
}

// appConfig returns the server-owned app config keys derived from PocketBase
// Settings.
func appConfig(app core.App) map[string]any {
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
	for key, value := range appConfig(app) {
		config[key] = value
	}
	return config
}

// IsServerOwnedConfigKey reports whether key may be set in workflow config only
// by the server. User pipeline YAML, step config, and rerun bodies must never
// set these keys.
func IsServerOwnedConfigKey(key string) bool {
	switch key {
	case AppURLConfigKey, AppNameConfigKey, AppLogoConfigKey,
		// The run's organization: records are resolved on its behalf.
		"namespace",
		// Temporary records the run deletes when it ends.
		"temp_wallet_version", "temp_credentials", "temp_use_case_verifications",
		// Where the run reports to GitHub and to its device semaphore.
		"github_pr_comment",
		"mobile_device_semaphore_ticket_id",
		"mobile_device_semaphore_device_ids",
		"mobile_device_semaphore_leader_device_id",
		"mobile_device_semaphore_owner_namespace":
		return true
	default:
		return false
	}
}
