package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWalkConfigTemplates(t *testing.T) {
	// Mocked expected value for TestWalkConfigTemplates
	want := Configs{
		"openid4vp": Config{
			Standard: StandardMetadata{
				UID:          "openid4vp",
				Name:         "OpenID4VP Wallet",
				Description:  "OpenID for Verifiable Credential Issuance",
				StandardURL:  "https://openid.net/specs/openid-4-verifiable-presentations-1_0-24.html",
				LatestUpdate: "2024-02-08",
				ExternalLinks: map[string][]string{
					"reference": {},
				},
			},
			Drafts: map[string]Draft{
				"draft-24": {
					Version: VersionMetadata{
						Name:             "Draft 13",
						LatestUpdate:     "2024-02-08",
						SpecificationURL: "https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0-13.html",
					},
					Suites: map[string]Suite{
						"ewc": {
							Metadata: SuiteMetadata{
								Name:        "OpenID Foundation Conformance Suite",
								Homepage:    "https://openid.net/certification/about-conformance-suite/",
								GitHub:      "https://gitlab.com/openid/conformance-suite",
								Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
							},
							Files: []string{"ewc_file1.json", "ewc_file2.json"},
						},
						"openid_conformance_suite": {
							Metadata: SuiteMetadata{
								Name:        "OpenID Foundation Conformance Suite",
								Homepage:    "https://openid.net/certification/about-conformance-suite/",
								GitHub:      "https://gitlab.com/openid/conformance-suite",
								Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
							},
							Files: []string{"conformance_file1.json", "conformance_file2.json"},
						},
					},
				},
			},
		}}

	// Assuming you have a function to walk the config templates
	got := walkConfigTemplates()
	require.Equal(t, want, got, "Config templates do not match")
}
