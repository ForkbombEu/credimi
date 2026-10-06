// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInternalAppURLFromConfig(t *testing.T) {
	t.Run("falls back to app_url when internal not set", func(t *testing.T) {
		require.Equal(
			t,
			"https://app.example",
			InternalAppURLFromConfig(map[string]any{"app_url": "https://app.example"}),
		)
	})

	t.Run("prefers internal_app_url when set", func(t *testing.T) {
		require.Equal(
			t,
			"http://127.0.0.1:8090",
			InternalAppURLFromConfig(map[string]any{
				"app_url":          "https://app.example",
				"internal_app_url": "http://127.0.0.1:8090",
			}),
		)
	})

	t.Run("blank internal_app_url falls back to app_url", func(t *testing.T) {
		require.Equal(
			t,
			"https://app.example",
			InternalAppURLFromConfig(map[string]any{
				"app_url":          "https://app.example",
				"internal_app_url": "   ",
			}),
		)
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		require.Equal(
			t,
			"http://credimi:8090",
			InternalAppURLFromConfig(map[string]any{
				"app_url":          "https://app.example",
				"internal_app_url": " http://credimi:8090 ",
			}),
		)
	})

	t.Run("non-string app_url yields empty", func(t *testing.T) {
		require.Equal(t, "", InternalAppURLFromConfig(map[string]any{"app_url": 42}))
	})

	t.Run("nil config yields empty", func(t *testing.T) {
		require.Equal(t, "", InternalAppURLFromConfig(nil))
	})

	t.Run("missing keys yield empty", func(t *testing.T) {
		require.Equal(t, "", InternalAppURLFromConfig(map[string]any{}))
	})
}

func TestWithInternalAppURL(t *testing.T) {
	t.Run("injects internal url when env set", func(t *testing.T) {
		t.Setenv(InternalAppURLConfigKeyEnv, "http://credimi:8090")
		config := WithInternalAppURL(map[string]any{"app_url": "https://app.example"})
		require.Equal(t, "http://credimi:8090", config[InternalAppURLConfigKey])
	})

	t.Run("leaves config unchanged when env unset", func(t *testing.T) {
		t.Setenv(InternalAppURLConfigKeyEnv, "")
		config := WithInternalAppURL(map[string]any{"app_url": "https://app.example"})
		require.Len(t, config, 1)
		require.NotContains(t, config, InternalAppURLConfigKey)
	})
}

func TestInternalAppURLOverride(t *testing.T) {
	t.Run("returns trimmed env value", func(t *testing.T) {
		t.Setenv(InternalAppURLConfigKeyEnv, " http://credimi:8090 ")
		require.Equal(t, "http://credimi:8090", InternalAppURLOverride())
	})

	t.Run("returns empty when unset", func(t *testing.T) {
		t.Setenv(InternalAppURLConfigKeyEnv, "")
		require.Equal(t, "", InternalAppURLOverride())
	})
}

func TestValidateInternalAppURLDestination(t *testing.T) {
	tests := []struct {
		name        string
		internalURL string
		appURL      string
		destination string
		wantErr     string
	}{
		{
			name:        "internal override origin",
			internalURL: "http://credimi:8090",
			appURL:      "https://app.example",
			destination: "http://credimi:8090/api/pipeline/get-yaml",
		},
		{
			name:        "app url origin with explicit default port and case",
			appURL:      "https://App.Example/",
			destination: "https://app.example:443/api/credential/get-credential-offer",
		},
		{
			name:        "foreign host",
			internalURL: "http://credimi:8090",
			appURL:      "https://app.example",
			destination: "https://attacker.example/api/credential/get-credential-offer",
			wantErr:     "is not a configured Credimi base URL",
		},
		{
			name:        "same host other port",
			appURL:      "https://app.example",
			destination: "https://app.example:8443/api",
			wantErr:     "is not a configured Credimi base URL",
		},
		{
			name:        "scheme downgrade",
			appURL:      "https://app.example",
			destination: "http://app.example/api",
			wantErr:     "is not a configured Credimi base URL",
		},
		{
			name:        "userinfo host confusion",
			appURL:      "https://app.example",
			destination: "https://app.example@attacker.example/api",
			wantErr:     "is not a configured Credimi base URL",
		},
		{
			name:        "relative destination",
			appURL:      "https://app.example",
			destination: "/api/pipeline/get-yaml",
			wantErr:     "invalid destination URL",
		},
		{
			name:        "nothing configured",
			destination: "https://app.example/api",
			wantErr:     "no Credimi base URL is configured",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(InternalAppURLConfigKeyEnv, tc.internalURL)
			SetServerAppURLSource(func() string { return tc.appURL })
			t.Cleanup(func() { SetServerAppURLSource(func() string { return "" }) })

			err := ValidateInternalAppURLDestination(tc.destination)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
