// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectSuitesGroupsByNormalizedAxes(t *testing.T) {
	t.Parallel()

	checks := []Check{
		{
			Path:         "openid4vp_wallet/1.0/openid_conformance_suite/a",
			Title:        "A",
			Standard:     "openid4vp_wallet",
			Version:      "1.0",
			Suite:        "openid_conformance_suite",
			File:         "a.yaml",
			Provider:     "openid_conformance_suite",
			SuiteName:    "OpenID Foundation Conformance Suite",
			SuiteLogo:    "https://example.test/oidf.png",
			VisibleIn:    []string{"pipeline"},
			NormStandard: "openid4vp",
			Component:    "wallet",
			NormVersion:  "1.0",
		},
		{
			Path:         "openid4vp_wallet/1.0/openid_conformance_suite/b",
			Title:        "B",
			Standard:     "openid4vp_wallet",
			Version:      "1.0",
			Suite:        "openid_conformance_suite",
			File:         "b.yaml",
			Provider:     "openid_conformance_suite",
			SuiteName:    "OpenID Foundation Conformance Suite",
			VisibleIn:    []string{"pipeline", "manual"},
			NormStandard: "openid4vp",
			Component:    "wallet",
			NormVersion:  "1.0",
		},
		{
			Path:         "openid4vp_wallet/draft-24/openid_conformance_suite/c",
			Title:        "C",
			Standard:     "openid4vp_wallet",
			Version:      "draft-24",
			Suite:        "openid_conformance_suite",
			File:         "c.yaml",
			Provider:     "openid_conformance_suite",
			SuiteName:    "OpenID Foundation Conformance Suite",
			VisibleIn:    []string{"pipeline"},
			NormStandard: "openid4vp",
			Component:    "wallet",
			NormVersion:  "draft-24",
		},
		{
			Path:         "fcaf/wallet_solution/relying_party/FCAF-001",
			Title:        "FCAF 001",
			Standard:     "fcaf",
			Version:      "wallet_solution",
			Suite:        "relying_party",
			File:         "FCAF-001.yaml",
			Provider:     "fcaf",
			SuiteName:    "FCAF Functional Conformance Assessment",
			VisibleIn:    []string{"pipeline"},
			NormStandard: "openid4vp",
			Component:    "wallet",
			NormVersion:  "",
		},
	}

	suites := ProjectSuites(checks)
	require.Len(t, suites, 3)

	byPrefix := map[string]SuiteRecord{}
	for _, s := range suites {
		byPrefix[s.PathPrefix] = s
	}

	s10 := byPrefix["openid4vp_wallet/1.0/openid_conformance_suite"]
	require.Equal(t, "openid4vp", s10.Standard)
	require.Equal(t, "wallet", s10.Component)
	require.Equal(t, "1.0", s10.Version)
	require.Equal(t, 2, s10.CheckCount)
	require.Equal(t, []string{"openid4vp_wallet/1.0/openid_conformance_suite/a", "openid4vp_wallet/1.0/openid_conformance_suite/b"}, s10.CheckPaths)
	require.ElementsMatch(t, []string{"manual", "pipeline"}, s10.VisibleIn)
	require.Equal(t, "openid4vp_wallet", s10.FSStandard)
	require.Equal(t, PathID(s10.PathPrefix), s10.ID)

	s24 := byPrefix["openid4vp_wallet/draft-24/openid_conformance_suite"]
	require.Equal(t, "draft-24", s24.Version)
	require.Equal(t, 1, s24.CheckCount)

	fcaf := byPrefix["fcaf/wallet_solution/relying_party"]
	require.Equal(t, "openid4vp", fcaf.Standard)
	require.Equal(t, "wallet", fcaf.Component)
	require.Equal(t, 0, fcaf.ComponentRank)
	require.Empty(t, fcaf.Version)
	require.Equal(t, "fcaf", fcaf.Provider)
	require.Equal(t, 1, fcaf.CheckCount)
}

func TestComponentRankOrder(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0, ComponentRank("wallet"))
	require.Equal(t, 1, ComponentRank("issuer"))
	require.Equal(t, 2, ComponentRank("verifier"))
	require.Equal(t, 9, ComponentRank(""))
	require.Equal(t, 9, ComponentRank("unknown"))
}

func TestProjectSuitesSortsWalletIssuerVerifierThenStandardSuite(t *testing.T) {
	t.Parallel()

	checks := []Check{
		{
			Path: "openid4vp_verifier/1.0/webuild/a", Title: "A",
			Standard: "openid4vp_verifier", Version: "1.0", Suite: "webuild", File: "a.yaml",
			NormStandard: "openid4vp", Component: "verifier", NormVersion: "1.0",
			VisibleIn: []string{"pipeline"},
		},
		{
			Path: "openid4vci_issuer/1.0/webuild/a", Title: "A",
			Standard: "openid4vci_issuer", Version: "1.0", Suite: "webuild", File: "a.yaml",
			NormStandard: "openid4vci", Component: "issuer", NormVersion: "1.0",
			VisibleIn: []string{"pipeline"},
		},
		{
			Path: "openid4vci_wallet/1.0/ewc/a", Title: "A",
			Standard: "openid4vci_wallet", Version: "1.0", Suite: "ewc", File: "a.yaml",
			NormStandard: "openid4vci", Component: "wallet", NormVersion: "1.0",
			VisibleIn: []string{"pipeline"},
		},
		{
			Path: "openid4vp_wallet/1.0/ewc/a", Title: "A",
			Standard: "openid4vp_wallet", Version: "1.0", Suite: "ewc", File: "a.yaml",
			NormStandard: "openid4vp", Component: "wallet", NormVersion: "1.0",
			VisibleIn: []string{"pipeline"},
		},
	}

	suites := ProjectSuites(checks)
	require.Len(t, suites, 4)
	require.Equal(t, []string{"wallet", "wallet", "issuer", "verifier"}, []string{
		suites[0].Component, suites[1].Component, suites[2].Component, suites[3].Component,
	})
	require.Equal(t, "openid4vci", suites[0].Standard)
	require.Equal(t, "openid4vp", suites[1].Standard)
	require.Equal(t, "ewc", suites[0].Suite)
	require.Equal(t, "ewc", suites[1].Suite)
}
