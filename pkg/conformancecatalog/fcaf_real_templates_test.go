// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadFromDirIndexesRealFCAFTests(t *testing.T) {
	root := realTemplatesDir(t)

	checks, err := LoadFromDir(root)
	require.NoError(t, err)

	var fcaf []Check
	for _, ch := range checks {
		if ch.Standard == "fcaf" {
			fcaf = append(fcaf, ch)
		}
	}
	require.Greater(t, len(fcaf), 100, "expected hundreds of FCAF tests from config_templates")
	for _, ch := range fcaf {
		require.True(t, strings.HasPrefix(ch.Path, "fcaf/"), ch.Path)
		parts := strings.Split(ch.Path, "/")
		require.Len(t, parts, 4, "path identity fcaf/<version>/<suite>/<test_id>: %s", ch.Path)
		require.NotEmpty(t, parts[3], ch.Path)
		require.True(
			t,
			strings.HasSuffix(ch.File, ".yaml") || strings.HasSuffix(ch.File, ".yml"),
			ch.File,
		)
		require.NotEmpty(t, ch.Title)
		require.Contains(t, ch.VisibleIn, SurfacePipeline)
		require.Equal(t, "wallet_solution", ch.SUT, ch.Path)
		require.Equal(t, "relying_party", ch.Role, ch.Path)
		require.Equal(t, "fcaf", ch.Provider, ch.Path)
		require.Empty(t, ch.Protocol, ch.Path)
		require.Equal(t, "FCAF Functional Conformance Assessment", ch.SuiteName, ch.Path)
		require.NotEmpty(t, ch.SuiteLogo, ch.Path)
		require.Equal(t, "openid4vp", ch.NormStandard, ch.Path)
		require.Equal(t, "wallet", ch.Component, ch.Path)
		require.Empty(t, ch.NormVersion, ch.Path)
	}
}

// TestLoadFromDirClassicSuiteFacetsFromMetadata asserts classic OpenID/vLEI rows
// get protocol/role/provider from suite metadata.yaml (not the UID map), and that
// sut stays empty (product-class axis is N/A for classic suites).
func TestLoadFromDirClassicSuiteFacetsFromMetadata(t *testing.T) {
	root := realTemplatesDir(t)

	checks, err := LoadFromDir(root)
	require.NoError(t, err)

	wantByStandard := map[string]struct {
		Protocol string
		Role     string
	}{
		"openid4vp_wallet":   {Protocol: "openid4vp", Role: "wallet"},
		"openid4vp_verifier": {Protocol: "openid4vp", Role: "verifier"},
		"openid4vci_wallet":  {Protocol: "openid4vci", Role: "wallet"},
		"openid4vci_issuer":  {Protocol: "openid4vci", Role: "issuer"},
		"vlei":               {Protocol: "vlei"},
	}

	var classic int
	for _, ch := range checks {
		if ch.Standard == "fcaf" {
			continue
		}
		classic++
		want, ok := wantByStandard[ch.Standard]
		require.True(t, ok, "unexpected standard %q path=%s", ch.Standard, ch.Path)
		require.Equal(t, want.Protocol, ch.Protocol, ch.Path)
		require.Equal(t, want.Role, ch.Role, ch.Path)
		require.Empty(t, ch.SUT, "classic sut must stay empty: %s", ch.Path)
		require.Equal(t, ch.Suite, ch.Provider, "provider should match suite uid: %s", ch.Path)
		require.NotEmpty(t, ch.SuiteName, "authored suite name expected: %s", ch.Path)

		id := NormalizePathIdentity(ch.Standard, ch.Version, ch.Suite)
		require.Equal(t, id.Standard, ch.NormStandard, ch.Path)
		require.Equal(t, id.Component, ch.Component, ch.Path)
		require.Equal(t, id.Version, ch.NormVersion, ch.Path)
		if want.Role != "" {
			require.Equal(t, want.Role, ch.Component, "component should match classic role: %s", ch.Path)
		}
	}
	require.Greater(t, classic, 0, "expected classic suite checks from config_templates")
}

func realTemplatesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "config_templates"))
}
