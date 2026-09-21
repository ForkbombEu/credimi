// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTestIDDelegatesToTaxonomy(t *testing.T) {
	code, subgroup, label := parseTestID(
		"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
	)
	require.Equal(t, "DM", code)
	require.Equal(t, "addressdata", subgroup)
	require.Equal(t, "Address data", label)
	require.Equal(t, "Data model", categoryName(code))
	require.Equal(t, [3]int{78, 91, 210}, categoryColor(code))
}
