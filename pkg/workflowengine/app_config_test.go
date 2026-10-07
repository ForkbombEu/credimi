// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func newAppConfigTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp("../../test_pb_data")
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	app.Settings().Meta.AppURL = "https://x.test"
	app.Settings().Meta.AppName = "Credimi"
	return app
}

func TestAppConfig(t *testing.T) {
	app := newAppConfigTestApp(t)

	require.Equal(t, "https://x.test", AppURL(app))
	require.Equal(t, map[string]any{
		AppURLConfigKey:  "https://x.test",
		AppNameConfigKey: "Credimi",
		AppLogoConfigKey: "https://x.test/logos/credimi_logo-transp_emblem.png",
	}, AppConfig(app))
}

func TestWithAppConfig(t *testing.T) {
	app := newAppConfigTestApp(t)

	tests := []struct {
		name   string
		config map[string]any
		want   map[string]any
	}{
		{
			name:   "allocates nil map",
			config: nil,
			want:   AppConfig(app),
		},
		{
			name: "overwrites user values and keeps other keys",
			config: map[string]any{
				AppURLConfigKey:  "https://stale.example",
				AppNameConfigKey: "other",
				AppLogoConfigKey: "https://stale.example/logo.png",
				"namespace":      "acme",
			},
			want: map[string]any{
				AppURLConfigKey:  "https://x.test",
				AppNameConfigKey: "Credimi",
				AppLogoConfigKey: "https://x.test/logos/credimi_logo-transp_emblem.png",
				"namespace":      "acme",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, WithAppConfig(app, tc.config))
		})
	}
}

func TestIsServerOwnedConfigKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{key: AppURLConfigKey, want: true},
		{key: AppNameConfigKey, want: true},
		{key: AppLogoConfigKey, want: true},
		{key: "namespace", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			require.Equal(t, tc.want, IsServerOwnedConfigKey(tc.key))
		})
	}
}
