// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveFacetsPrecedenceAndKnownStandards(t *testing.T) {
	t.Parallel()

	got := resolveFacets("openid4vp_wallet", "ewc", facetFields{}, facetFields{})
	require.Equal(t, "openid4vp", got.Protocol)
	require.Equal(t, "wallet", got.Role)
	require.Equal(t, "ewc", got.Provider)
	require.Empty(t, got.SUT)

	got = resolveFacets("openid4vp_wallet", "ewc", facetFields{
		Protocol: "from-suite",
		Role:     "suite-role",
		Provider: "suite-provider",
		SUT:      "suite-sut",
	}, facetFields{})
	require.Equal(t, "from-suite", got.Protocol)
	require.Equal(t, "suite-role", got.Role)
	require.Equal(t, "suite-provider", got.Provider)
	require.Equal(t, "suite-sut", got.SUT)

	got = resolveFacets("openid4vp_wallet", "ewc", facetFields{
		Protocol: "from-suite",
	}, facetFields{
		Protocol: "from-file",
		Role:     "file-role",
	})
	require.Equal(t, "from-file", got.Protocol)
	require.Equal(t, "file-role", got.Role)
	require.Equal(t, "ewc", got.Provider)

	got = resolveFacets("fcaf", "relying_party", facetFields{}, facetFields{
		SUT:  "wallet_solution",
		Role: "relying_party",
	})
	require.Equal(t, "wallet_solution", got.SUT)
	require.Equal(t, "relying_party", got.Role)
	require.Equal(t, "fcaf", got.Provider)
	require.Empty(t, got.Protocol)
}
