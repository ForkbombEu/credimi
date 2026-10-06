// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"crypto/tls"
	"net/http"
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

func TestSetLocalURL(t *testing.T) {
	tests := []struct {
		name   string
		server *http.Server
		want   string
	}{
		{
			name:   "unspecified IPv4",
			server: &http.Server{Addr: "0.0.0.0:8090"},
			want:   "http://127.0.0.1:8090",
		},
		{
			name:   "unspecified IPv6",
			server: &http.Server{Addr: "[::]:9000"},
			want:   "http://127.0.0.1:9000",
		},
		{
			name:   "empty host",
			server: &http.Server{Addr: ":8092"},
			want:   "http://127.0.0.1:8092",
		},
		{
			name:   "loopback",
			server: &http.Server{Addr: "127.0.0.1:8091"},
			want:   "http://127.0.0.1:8091",
		},
		{
			name: "TLS server",
			server: &http.Server{
				Addr:      "0.0.0.0:443",
				TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
			want: "https://127.0.0.1:443",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newAppConfigTestApp(t)
			SetLocalURL(app, tc.server)
			require.Equal(t, tc.want, LocalURL(app))
		})
	}
}

func TestLocalURLUnset(t *testing.T) {
	app := newAppConfigTestApp(t)
	require.Empty(t, LocalURL(app))
}
