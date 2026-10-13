// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuitePathPrefixFromCheckID(t *testing.T) {
	t.Parallel()
	require.Equal(
		t,
		"openid4vci_wallet/draft-15/ewc",
		SuitePathPrefixFromCheckID("openid4vci_wallet/draft-15/ewc/check"),
	)
	require.Empty(t, SuitePathPrefixFromCheckID("ewc/check"))
	require.Empty(t, SuitePathPrefixFromCheckID(""))
}

func TestSuiteHasQRFromRebuiltCatalog(t *testing.T) {
	require.NoError(t, Rebuild(realTemplatesDir(t)))

	require.True(t, SuiteHasQR("openid4vci_wallet/draft-15/ewc"))
	require.True(t, SuiteHasQR("openid4vp_wallet/draft-23/ewc"))
	require.True(t, SuiteHasQR("openid4vci_wallet/1.0/webuild"))
	require.True(t, SuiteHasQR("openid4vp_wallet/1.0/webuild"))

	require.False(t, SuiteHasQR("openid4vp_wallet/1.0/openid_conformance_suite"))
	require.False(t, SuiteHasQR("openid4vci_issuer/1.0/webuild"))
	require.False(t, SuiteHasQR("missing/suite/path"))
	require.False(t, SuiteHasQR(""))
}
