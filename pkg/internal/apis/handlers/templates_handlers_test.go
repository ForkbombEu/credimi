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

	// Suite 1 (has files + GUI)
	suite1UID := "ewc"
	suite1Dir := filepath.Join(versionDir, suite1UID)
	require.NoError(t, os.Mkdir(suite1Dir, 0755))
	suite1Meta := SuiteMetadata{
		UID:               "ewc",
		Name:              "OpenID Foundation Conformance Suite",
		Homepage:          "https://openid.net/certification/about-conformance-suite/",
		Repository:        "https://gitlab.com/openid/conformance-suite",
		Help:              "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
		Description:       "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
		ShowInPipelineGUI: true,
	}
	suite1Yaml, _ := yaml.Marshal(suite1Meta)
	require.NoError(t, os.WriteFile(filepath.Join(suite1Dir, "metadata.yaml"), suite1Yaml, 0644))
	ewcFiles := []string{"ewc_file1.json", "ewc_file2.json"}
	ewcPaths := []string{"openid4vp/draft-24/ewc/ewc_file1", "openid4vp/draft-24/ewc/ewc_file2"}
	for _, fname := range ewcFiles {
		require.NoError(t, os.WriteFile(filepath.Join(suite1Dir, fname), []byte("{}"), 0644))
	}

	// Suite 2 (has files + NOT visible in GUI)
	suite2UID := "openid_conformance_suite"
	suite2Dir := filepath.Join(versionDir, suite2UID)
	require.NoError(t, os.Mkdir(suite2Dir, 0755))
	suite2Meta := SuiteMetadata{
		UID:               "openid_conformance_suite",
		Name:              "OpenID Foundation Conformance Suite",
		ShowInPipelineGUI: false,
	}
	suite2Yaml, _ := yaml.Marshal(suite2Meta)
	require.NoError(t, os.WriteFile(filepath.Join(suite2Dir, "metadata.yaml"), suite2Yaml, 0644))
	conformanceFiles := []string{"conformance_file1.json", "conformance_file2.json"}
	conformancePaths := []string{
		"openid4vp/draft-24/openid_conformance_suite/conformance_file1",
		"openid4vp/draft-24/openid_conformance_suite/conformance_file2",
	}
	for _, fname := range conformanceFiles {
		require.NoError(t, os.WriteFile(filepath.Join(suite2Dir, fname), []byte("{}"), 0644))
	}

	// Suite 3 (empty files + NOT visible in GUI)
	suite3UID := "empty_conformance_suite"
	suite3Dir := filepath.Join(versionDir, suite3UID)
	require.NoError(t, os.Mkdir(suite3Dir, 0755))
	suite3Meta := SuiteMetadata{
		UID:               "empty_conformance_suite",
		Name:              "empty Conformance Suite",
		ShowInPipelineGUI: false,
	}
	suite3Yaml, _ := yaml.Marshal(suite3Meta)
	require.NoError(t, os.WriteFile(filepath.Join(suite3Dir, "metadata.yaml"), suite3Yaml, 0644))

	wantUnfiltered := Standards{
		Standard{
			StandardMetadata: standardMeta,
			Versions: []Version{
				{
					VersionMetadata: versionMeta,
					Suites: []Suite{
						{SuiteMetadata: suite3Meta, Files: []string{}, Paths: []string{}},
						{SuiteMetadata: suite1Meta, Files: ewcFiles, Paths: ewcPaths},
						{
							SuiteMetadata: suite2Meta,
							Files:         conformanceFiles,
							Paths:         conformancePaths,
						},
					},
				},
			},
		},
	}

	t.Run("walkConfigTemplates unfiltered", func(t *testing.T) {
		got, err := walkConfigTemplates(testdataDir, false)
		require.NoError(t, err)
		require.Equal(t, wantUnfiltered, got)
	})

	wantFiltered := Standards{
		Standard{
			StandardMetadata: standardMeta,
			Versions: []Version{
				{
					VersionMetadata: versionMeta,
					Suites: []Suite{
						{SuiteMetadata: suite1Meta, Files: ewcFiles, Paths: ewcPaths},
					},
				},
			},
		},
	}

	t.Run("walkConfigTemplates filtered", func(t *testing.T) {
		got, err := walkConfigTemplates(testdataDir, true)
		require.NoError(t, err)
		require.Equal(t, wantFiltered, got)
	})

	t.Run("readDir error", func(t *testing.T) {
		_, err := walkConfigTemplates(filepath.Join(testdataDir, "doesnotexist"), false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "filesystem.readDir")
		require.Contains(t, err.Error(), "Failed to read directory")
	})

	t.Run("yaml unmarshal error", func(t *testing.T) {
		invalidYamlPath := filepath.Join(standardDir, "standard.yaml")
		require.NoError(t, os.WriteFile(invalidYamlPath, []byte("invalid: [unclosed"), 0644))
		_, err := walkConfigTemplates(testdataDir, false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "yaml.unmarshal")
		require.Contains(t, err.Error(), "Failed to unmarshal yaml")
	})
}
