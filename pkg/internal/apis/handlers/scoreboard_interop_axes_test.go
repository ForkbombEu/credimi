// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInteropAxis_KnownHubs(t *testing.T) {
	t.Parallel()

	wallets, ok := getInteropAxis("wallets")
	require.True(t, ok)
	require.Equal(t, "wallets", wallets.HubCollection)
	require.Equal(t, "wallets", wallets.CacheField)
	require.False(t, wallets.PathBased)

	issuers, ok := getInteropAxis("credential_issuers")
	require.True(t, ok)
	require.Equal(t, "issuers", issuers.CacheField)
	require.False(t, issuers.PathBased)

	conformance, ok := getInteropAxis("conformance-checks")
	require.True(t, ok)
	require.Equal(t, "conformance_checks", conformance.CacheField)
	require.True(t, conformance.PathBased)
}

func TestGetInteropAxis_WalletsTiered(t *testing.T) {
	t.Parallel()

	axis, ok := getInteropAxis("wallets")
	require.True(t, ok)
	require.NotNil(t, axis.Tier)
	require.Equal(t, "wallet_versions", axis.Tier.LeafCacheField)
	require.Equal(t, "__no_version__", axis.Tier.NoLeafSentinel)
	require.True(t, axis.Tiered())
}

func TestGetInteropAxis_CredentialsFlat(t *testing.T) {
	t.Parallel()

	axis, ok := getInteropAxis("credentials")
	require.True(t, ok)
	require.Nil(t, axis.Tier)
	require.False(t, axis.Tiered())
}

func TestGetInteropAxis_Unknown(t *testing.T) {
	t.Parallel()
	_, ok := getInteropAxis("bad_hub")
	require.False(t, ok)
}

func TestSupportedInteropHubCollections(t *testing.T) {
	t.Parallel()
	got := supportedInteropHubCollections()
	require.Len(t, got, 6)
	require.Contains(t, got, "wallets")
	require.Contains(t, got, "conformance-checks")
}

func TestInteropHubsUsageHint(t *testing.T) {
	t.Parallel()
	require.Contains(t, interopHubsUsageHint(), "row=")
	require.Contains(t, interopHubsUsageHint(), "wallets")
}
