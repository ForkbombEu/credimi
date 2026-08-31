// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTestID(t *testing.T) {
	tests := []struct {
		name     string
		testID   string
		code     string
		subgroup string
		label    string
	}{
		{
			name:     "data model single underscore",
			testID:   "WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
			code:     "DM",
			subgroup: "addressdata",
			label:    "Address data",
		},
		{
			name:     "interaction double underscore",
			testID:   "WS_RP_IA_MainInteraction__003",
			code:     "IA",
			subgroup: "maininteraction",
			label:    "Main interaction",
		},
		{
			name:     "credential metadata casing variants normalize",
			testID:   "WS_RP_DM_Credentialmetadata_X_001",
			code:     "DM",
			subgroup: "credentialmetadata",
			label:    "Credential metadata",
		},
		{
			name:     "rp integrity acronym",
			testID:   "WS_RP_SM_RpIntegrity_Signed_001",
			code:     "SM",
			subgroup: "rpintegrity",
			label:    "RP integrity",
		},
		{
			name:     "unknown category falls back to other",
			testID:   "WS_RP_TEST__001",
			code:     "OTHER",
			subgroup: "other",
			label:    "Other",
		},
		{
			name:     "non ws rp prefix falls back to other",
			testID:   "SOMETHING_ELSE",
			code:     "OTHER",
			subgroup: "other",
			label:    "Other",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, subgroup, label := parseTestID(tc.testID)
			require.Equal(t, tc.code, code)
			require.Equal(t, tc.subgroup, subgroup)
			require.Equal(t, tc.label, label)
		})
	}
}
