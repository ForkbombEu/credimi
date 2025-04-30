// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWalkConfigTemplates(t *testing.T) {
	// Mocked expected value for TestWalkConfigTemplates
	want := Standards{
		Standard{
			StandardMetadata: StandardMetadata{
				UID:          "openid4vp",
				Name:         "OpenID4VP Wallet",
				Description:  "OpenID for Verifiable Credential Issuance",
				StandardURL:  "https://openid.net/specs/openid-4-verifiable-presentations-1_0-24.html",
				LatestUpdate: "2024-02-08",
				ExternalLinks: map[string][]string{
					"reference": {},
				},
			},
			Versions: []Version{
				{
					VersionMetadata: VersionMetadata{
						UID:              "draft-24",
						Name:             "Draft 13",
						LatestUpdate:     "2024-02-08",
						SpecificationURL: "https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0-13.html",
					},
					Suites: []Suite{
						{
							SuiteMetadata: SuiteMetadata{
								UID:         "ewc",
								Name:        "OpenID Foundation Conformance Suite",
								Homepage:    "https://openid.net/certification/about-conformance-suite/",
								Repository:  "https://gitlab.com/openid/conformance-suite",
								Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
							},
							Files: []string{"ewc_file1.json", "ewc_file2.json"},
						},
						{
							SuiteMetadata: SuiteMetadata{
								UID:         "openid_conformance_suite",
								Name:        "OpenID Foundation Conformance Suite",
								Homepage:    "https://openid.net/certification/about-conformance-suite/",
								Repository:  "https://gitlab.com/openid/conformance-suite",
								Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
							},
							Files: []string{"conformance_file1.json", "conformance_file2.json"},
						},
						{
							SuiteMetadata: SuiteMetadata{
								UID:         "vuota_conformance_suite",
								Name:        "Vuota Conformance Suite",
								Homepage:    "https://vuota.com/certification/about-conformance-suite/",
								Repository:  "https://gitlab.com/vuota/conformance-suite",
								Help:        "https://vuota.com/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for Vuota",
							},
							Files: []string{},
						},
					},
				},
			},
		}}

	got := walkConfigTemplates()
	require.Equal(t, want, got, "Config templates do not match")
}
