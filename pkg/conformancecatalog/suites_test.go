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
			Path:       "openid4vp_wallet/1.0/openid_conformance_suite/a",
			Title:      "A",
			FSStandard: "openid4vp_wallet",
			FSVersion:  "1.0",
			Suite:      "openid_conformance_suite",
			File:       "a.yaml",
			Provider:   "openid_conformance_suite",
			VisibleIn:  []string{"pipeline"},
			Standard:   "openid4vp",
			Component:  "wallet",
			Version:    "1.0",
		},
		{
			Path:       "openid4vp_wallet/1.0/openid_conformance_suite/b",
			Title:      "B",
			FSStandard: "openid4vp_wallet",
			FSVersion:  "1.0",
			Suite:      "openid_conformance_suite",
			File:       "b.yaml",
			Provider:   "openid_conformance_suite",
			VisibleIn:  []string{"pipeline", "manual"},
			Standard:   "openid4vp",
			Component:  "wallet",
			Version:    "1.0",
		},
		{
			Path:       "openid4vp_wallet/draft-24/openid_conformance_suite/c",
			Title:      "C",
			FSStandard: "openid4vp_wallet",
			FSVersion:  "draft-24",
			Suite:      "openid_conformance_suite",
			File:       "c.yaml",
			Provider:   "openid_conformance_suite",
			VisibleIn:  []string{"pipeline"},
			Standard:   "openid4vp",
			Component:  "wallet",
			Version:    "draft-24",
		},
		{
			Path:       "fcaf/wallet_solution/relying_party/FCAF-001",
			Title:      "FCAF 001",
			FSStandard: "fcaf",
			FSVersion:  "wallet_solution",
			Suite:      "relying_party",
			File:       "FCAF-001.yaml",
			Provider:   "fcaf",
			VisibleIn:  []string{"pipeline"},
			Standard:   "openid4vp",
			Component:  "wallet",
			Version:    "",
		},
	}

	display := map[string]suiteDisplayFields{
		"openid4vp_wallet/1.0/openid_conformance_suite": {
			Name: "OpenID Foundation Conformance Suite",
			Logo: "https://example.test/oidf.png",
		},
		"openid4vp_wallet/draft-24/openid_conformance_suite": {
			Name: "OpenID Foundation Conformance Suite",
		},
		"fcaf/wallet_solution/relying_party": {
			Name: "FCAF Functional Conformance Assessment",
		},
	}

	suites := ProjectSuites(checks, display)
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
	require.Equal(t, []SuiteMember{
		{Path: "openid4vp_wallet/1.0/openid_conformance_suite/a", Title: "A", File: "a.yaml"},
		{Path: "openid4vp_wallet/1.0/openid_conformance_suite/b", Title: "B", File: "b.yaml"},
	}, s10.Members)
	require.ElementsMatch(t, []string{"manual", "pipeline"}, s10.VisibleIn)
	require.Equal(t, "openid4vp_wallet", s10.FSStandard)
	require.Equal(t, PathID(s10.PathPrefix), s10.ID)
	require.Equal(t, "OpenID Foundation Conformance Suite", s10.SuiteName)
	require.Equal(t, "https://example.test/oidf.png", s10.SuiteLogo)

	s24 := byPrefix["openid4vp_wallet/draft-24/openid_conformance_suite"]
	require.Equal(t, "draft-24", s24.Version)
	require.Equal(t, 1, s24.CheckCount)
	require.Equal(t, "OpenID Foundation Conformance Suite", s24.SuiteName)

	fcaf := byPrefix["fcaf/wallet_solution/relying_party"]
	require.Equal(t, "openid4vp", fcaf.Standard)
	require.Equal(t, "wallet", fcaf.Component)
	require.Equal(t, 0, fcaf.ComponentRank)
	require.Empty(t, fcaf.Version)
	require.Equal(t, "fcaf", fcaf.Provider)
	require.Equal(t, 1, fcaf.CheckCount)
	require.Equal(t, "FCAF Functional Conformance Assessment", fcaf.SuiteName)
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
			FSStandard: "openid4vp_verifier", FSVersion: "1.0", Suite: "webuild", File: "a.yaml",
			Standard: "openid4vp", Component: "verifier", Version: "1.0",
			VisibleIn: []string{"pipeline"},
		},
		{
			Path: "openid4vci_issuer/1.0/webuild/a", Title: "A",
			FSStandard: "openid4vci_issuer", FSVersion: "1.0", Suite: "webuild", File: "a.yaml",
			Standard: "openid4vci", Component: "issuer", Version: "1.0",
			VisibleIn: []string{"pipeline"},
		},
		{
			Path: "openid4vci_wallet/1.0/ewc/a", Title: "A",
			FSStandard: "openid4vci_wallet", FSVersion: "1.0", Suite: "ewc", File: "a.yaml",
			Standard: "openid4vci", Component: "wallet", Version: "1.0",
			VisibleIn: []string{"pipeline"},
		},
		{
			Path: "openid4vp_wallet/1.0/ewc/a", Title: "A",
			FSStandard: "openid4vp_wallet", FSVersion: "1.0", Suite: "ewc", File: "a.yaml",
			Standard: "openid4vp", Component: "wallet", Version: "1.0",
			VisibleIn: []string{"pipeline"},
		},
	}

	suites := ProjectSuites(checks, nil)
	require.Len(t, suites, 4)
	require.Equal(t, []string{"wallet", "wallet", "issuer", "verifier"}, []string{
		suites[0].Component, suites[1].Component, suites[2].Component, suites[3].Component,
	})
	require.Equal(t, "openid4vci", suites[0].Standard)
	require.Equal(t, "openid4vp", suites[1].Standard)
	require.Equal(t, "ewc", suites[0].Suite)
	require.Equal(t, "ewc", suites[1].Suite)
}
