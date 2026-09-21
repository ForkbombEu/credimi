// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProjectBlueprintsNestedShapeAndSurfaces(t *testing.T) {
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

	// Non-standard data directories must be excluded from blueprints.
	for _, skipUID := range []string{"fcaf_sources"} {
		skipDir := filepath.Join(testdataDir, skipUID)
		require.NoError(t, os.Mkdir(skipDir, 0755))
		skipYaml, _ := yaml.Marshal(StandardMetadata{UID: skipUID, Name: skipUID})
		require.NoError(
			t,
			os.WriteFile(filepath.Join(skipDir, "standard.yaml"), skipYaml, 0644),
		)
	}

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
	ewcPaths := []string{"openid4vp/draft-24/ewc/ewc_file1", "openid4vp/draft-24/ewc/ewc_file2"}
	for _, fname := range ewcFiles {
		require.NoError(t, os.WriteFile(filepath.Join(suite1Dir, fname), []byte("{}"), 0644))
	}

	suite2UID := "openid_conformance_suite"
	suite2Dir := filepath.Join(versionDir, suite2UID)
	require.NoError(t, os.Mkdir(suite2Dir, 0755))
	suite2Meta := SuiteMetadata{
		UID:       "openid_conformance_suite",
		Name:      "OpenID Foundation Conformance Suite",
		VisibleIn: []string{SurfaceManual},
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

	suite3UID := "empty_conformance_suite"
	suite3Dir := filepath.Join(versionDir, suite3UID)
	require.NoError(t, os.Mkdir(suite3Dir, 0755))
	suite3Meta := SuiteMetadata{
		UID:       "empty_conformance_suite",
		Name:      "empty Conformance Suite",
		VisibleIn: []string{SurfacePipeline},
	}
	suite3Yaml, _ := yaml.Marshal(suite3Meta)
	require.NoError(t, os.WriteFile(filepath.Join(suite3Dir, "metadata.yaml"), suite3Yaml, 0644))

	noVersionDir := filepath.Join(standardDir, "no_version_dir")
	require.NoError(t, os.Mkdir(noVersionDir, 0755))

	noMetaSuiteDir := filepath.Join(versionDir, "no_metadata_suite")
	require.NoError(t, os.Mkdir(noMetaSuiteDir, 0755))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(noMetaSuiteDir, "orphan.json"), []byte("{}"), 0644),
	)

	cat := &Catalog{}
	require.NoError(t, cat.LoadFromDir(testdataDir))

	wantUnfiltered := Blueprints{
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

	t.Run("unfiltered", func(t *testing.T) {
		got := cat.Blueprints("")
		require.Equal(t, wantUnfiltered, got)
	})

	wantManual := Blueprints{
		Standard{
			StandardMetadata: standardMeta,
			Versions: []Version{
				{
					VersionMetadata: versionMeta,
					Suites: []Suite{
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

	t.Run("manual surface", func(t *testing.T) {
		got := cat.Blueprints(SurfaceManual)
		require.Equal(t, wantManual, got)
	})

	wantPipeline := Blueprints{
		Standard{
			StandardMetadata: standardMeta,
			Versions: []Version{
				{
					VersionMetadata: versionMeta,
					Suites: []Suite{
						{SuiteMetadata: suite3Meta, Files: []string{}, Paths: []string{}},
						{SuiteMetadata: suite1Meta, Files: ewcFiles, Paths: ewcPaths},
					},
				},
			},
		},
	}

	t.Run("pipeline surface", func(t *testing.T) {
		got := cat.Blueprints(SurfacePipeline)
		require.Equal(t, wantPipeline, got)
	})

	t.Run("empty suite preserved without checks", func(t *testing.T) {
		snap := cat.Snapshot()
		for _, ch := range snap {
			require.NotEqual(t, "empty_conformance_suite", ch.Suite)
		}
		unfiltered := cat.Blueprints("")
		require.Len(t, unfiltered[0].Versions[0].Suites, 3)
		require.Equal(t, "empty_conformance_suite", unfiltered[0].Versions[0].Suites[0].UID)
		require.Empty(t, unfiltered[0].Versions[0].Suites[0].Files)
	})

	t.Run("missing dir", func(t *testing.T) {
		err := (&Catalog{}).LoadFromDir(filepath.Join(testdataDir, "doesnotexist"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "read templates dir")
	})

	t.Run("yaml unmarshal error", func(t *testing.T) {
		invalidYamlPath := filepath.Join(standardDir, "standard.yaml")
		require.NoError(t, os.WriteFile(invalidYamlPath, []byte("invalid: [unclosed"), 0644))
		err := (&Catalog{}).LoadFromDir(testdataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unmarshal")
	})
}
