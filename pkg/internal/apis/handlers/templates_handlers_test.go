// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWalkConfigTemplates(t *testing.T) {
	testdataDir := t.TempDir()

	standardUID := "openid4vp"
	standardDir := filepath.Join(testdataDir, standardUID)
	require.NoError(t, os.Mkdir(standardDir, 0755))

	standardMeta := StandardMetadata{
		UID:          "openid4vp",
		Name:         "OpenID4VP Wallet",
		Description:  "OpenID for Verifiable Credential Issuance",
		StandardURL:  "https://openid.net/specs/openid-4-verifiable-presentations-1_0-24.html",
		LatestUpdate: "2024-02-08",
		ExternalLinks: map[string][]string{
			"reference": {},
		},
	}
	standardYaml, _ := yaml.Marshal(standardMeta)
	require.NoError(
		t,
		os.WriteFile(filepath.Join(standardDir, "standard.yaml"), standardYaml, 0644),
	)

	versionUID := "draft-24"
	versionDir := filepath.Join(standardDir, versionUID)
	require.NoError(t, os.Mkdir(versionDir, 0755))

	versionMeta := VersionMetadata{
		UID:              "draft-24",
		Name:             "Draft 13",
		LatestUpdate:     "2024-02-08",
		SpecificationURL: "https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0-13.html",
	}
	versionYaml, _ := yaml.Marshal(versionMeta)
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "version.yaml"), versionYaml, 0644))

	suite1UID := "ewc"
	suite1Dir := filepath.Join(versionDir, suite1UID)
	require.NoError(t, os.Mkdir(suite1Dir, 0755))
	suite1Meta := SuiteMetadata{
		UID:         "ewc",
		Name:        "OpenID Foundation Conformance Suite",
		Homepage:    "https://openid.net/certification/about-conformance-suite/",
		Repository:  "https://gitlab.com/openid/conformance-suite",
		Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
		Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
	}
	suite1Yaml, _ := yaml.Marshal(suite1Meta)
	require.NoError(t, os.WriteFile(filepath.Join(suite1Dir, "metadata.yaml"), suite1Yaml, 0644))
	ewcFiles := []string{"ewc_file1.json", "ewc_file2.json"}
	for _, fname := range ewcFiles {
		require.NoError(t, os.WriteFile(filepath.Join(suite1Dir, fname), []byte("{}"), 0644))
	}

	suite2UID := "openid_conformance_suite"
	suite2Dir := filepath.Join(versionDir, suite2UID)
	require.NoError(t, os.Mkdir(suite2Dir, 0755))
	suite2Meta := SuiteMetadata{
		UID:         "openid_conformance_suite",
		Name:        "OpenID Foundation Conformance Suite",
		Homepage:    "https://openid.net/certification/about-conformance-suite/",
		Repository:  "https://gitlab.com/openid/conformance-suite",
		Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
		Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
	}
	suite2Yaml, _ := yaml.Marshal(suite2Meta)
	require.NoError(t, os.WriteFile(filepath.Join(suite2Dir, "metadata.yaml"), suite2Yaml, 0644))
	conformanceFiles := []string{"conformance_file1.json", "conformance_file2.json"}
	for _, fname := range conformanceFiles {
		require.NoError(t, os.WriteFile(filepath.Join(suite2Dir, fname), []byte("{}"), 0644))
	}

	suite3UID := "vuota_conformance_suite"
	suite3Dir := filepath.Join(versionDir, suite3UID)
	require.NoError(t, os.Mkdir(suite3Dir, 0755))
	suite3Meta := SuiteMetadata{
		UID:         "vuota_conformance_suite",
		Name:        "Vuota Conformance Suite",
		Homepage:    "https://vuota.com/certification/about-conformance-suite/",
		Repository:  "https://gitlab.com/vuota/conformance-suite",
		Help:        "https://vuota.com/certification/conformance-testing-for-openid-for-verifiable-presentations/",
		Description: "Conformance suite for Vuota",
	}
	suite3Yaml, _ := yaml.Marshal(suite3Meta)
	require.NoError(t, os.WriteFile(filepath.Join(suite3Dir, "metadata.yaml"), suite3Yaml, 0644))

	want := Standards{
		Standard{
			StandardMetadata: standardMeta,
			Versions: []Version{
				{
					VersionMetadata: versionMeta,
					Suites: []Suite{
						{
							SuiteMetadata: suite1Meta,
							Files:         ewcFiles,
						},
						{
							SuiteMetadata: suite2Meta,
							Files:         conformanceFiles,
						},
						{
							SuiteMetadata: suite3Meta,
							Files:         []string{},
						},
					},
				},
			},
		},
	}

	t.Run("walkConfigTemplates", func(t *testing.T) {
		got, err := walkConfigTemplates(testdataDir)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("readDir error", func(t *testing.T) {
		_, err := walkConfigTemplates(filepath.Join(testdataDir, "doesnotexist"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "filesystem.readDir")
		require.Contains(t, err.Error(), "Failed to read directory")
	})

	t.Run("yaml unmarshal error", func(t *testing.T) {
		invalidYamlPath := filepath.Join(standardDir, "standard.yaml")
		require.NoError(t, os.WriteFile(invalidYamlPath, []byte("invalid: [unclosed"), 0644))
		_, err := walkConfigTemplates(testdataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "yaml.unmarshal")
		require.Contains(t, err.Error(), "Failed to unmarshal yaml")
	})
}
