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

	loaded, err := LoadFromDir(root)
	require.NoError(t, err)

	var fcaf []Check
	for _, ch := range loaded.Checks {
		if ch.FSStandard == "fcaf" {
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
		require.Equal(t, "openid4vp", ch.Standard, ch.Path)
		require.Equal(t, "wallet", ch.Component, ch.Path)
		require.Empty(t, ch.Version, ch.Path)
	}

	disp := loaded.SuiteDisplay["fcaf/wallet_solution/relying_party"]
	require.Equal(t, "FCAF Functional Conformance Assessment", disp.Name)
	require.NotEmpty(t, disp.Logo)

	suites := ProjectSuites(fcaf, loaded.SuiteDisplay)
	require.NotEmpty(t, suites)
	for _, s := range suites {
		require.Equal(t, "FCAF Functional Conformance Assessment", s.SuiteName, s.PathPrefix)
		require.NotEmpty(t, s.SuiteLogo, s.PathPrefix)
	}
}

// TestLoadFromDirClassicSuiteFacetsFromMetadata asserts classic OpenID/vLEI rows
// get protocol/role/provider from suite metadata.yaml (not the UID map), and that
// sut stays empty (product-class axis is N/A for classic suites).
func TestLoadFromDirClassicSuiteFacetsFromMetadata(t *testing.T) {
	root := realTemplatesDir(t)

	loaded, err := LoadFromDir(root)
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

	var classic []Check
	for _, ch := range loaded.Checks {
		if ch.FSStandard == "fcaf" {
			continue
		}
		classic = append(classic, ch)
		want, ok := wantByStandard[ch.FSStandard]
		require.True(t, ok, "unexpected standard %q path=%s", ch.FSStandard, ch.Path)
		require.Equal(t, want.Protocol, ch.Protocol, ch.Path)
		require.Equal(t, want.Role, ch.Role, ch.Path)
		require.Empty(t, ch.SUT, "classic sut must stay empty: %s", ch.Path)
		require.Equal(t, ch.Suite, ch.Provider, "provider should match suite uid: %s", ch.Path)

		id := NormalizePathIdentity(ch.FSStandard, ch.FSVersion, ch.Suite)
		require.Equal(t, id.Standard, ch.Standard, ch.Path)
		require.Equal(t, id.Component, ch.Component, ch.Path)
		require.Equal(t, id.Version, ch.Version, ch.Path)
		if want.Role != "" {
			require.Equal(t, want.Role, ch.Component, "component should match classic role: %s", ch.Path)
		}
	}
	require.Greater(t, len(classic), 0, "expected classic suite checks from config_templates")

	suites := ProjectSuites(classic, loaded.SuiteDisplay)
	require.NotEmpty(t, suites)
	for _, s := range suites {
		require.NotEmpty(t, s.SuiteName, "authored suite name expected: %s", s.PathPrefix)
	}
}

func realTemplatesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "config_templates"))
}
