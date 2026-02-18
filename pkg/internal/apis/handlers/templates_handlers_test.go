// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
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

func TestHandleGetConfigsTemplatesInvalidFilter(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/template/blueprints?only_show_in_pipeline_gui=notabool",
		nil,
	)
	rec := httptest.NewRecorder()

	err := HandleGetConfigsTemplates()(&core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestHandleGetConfigsTemplatesSuccess(t *testing.T) {
	rootDir := t.TempDir()
	t.Setenv("ROOT_DIR", rootDir)

	templatesDir := filepath.Join(rootDir, "config_templates")
	standardDir := filepath.Join(templatesDir, "standard-1")
	versionDir := filepath.Join(standardDir, "v1")
	suiteDir := filepath.Join(versionDir, "suite-a")

	require.NoError(t, os.MkdirAll(suiteDir, 0755))

	standardMeta := StandardMetadata{
		UID:         "standard-1",
		Name:        "Standard One",
		StandardURL: "https://example.org",
	}
	standardYaml, _ := yaml.Marshal(standardMeta)
	require.NoError(t, os.WriteFile(filepath.Join(standardDir, "standard.yaml"), standardYaml, 0644))

	versionMeta := VersionMetadata{
		UID:              "v1",
		Name:             "Version 1",
		SpecificationURL: "https://example.org/spec",
	}
	versionYaml, _ := yaml.Marshal(versionMeta)
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "version.yaml"), versionYaml, 0644))

	suiteMeta := SuiteMetadata{
		UID:               "suite-a",
		Name:              "Suite A",
		ShowInPipelineGUI: true,
	}
	suiteYaml, _ := yaml.Marshal(suiteMeta)
	require.NoError(t, os.WriteFile(filepath.Join(suiteDir, "metadata.yaml"), suiteYaml, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(suiteDir, "test.yaml"), []byte("{}"), 0644))

	req := httptest.NewRequest(http.MethodGet, "/api/template/blueprints", nil)
	rec := httptest.NewRecorder()

	err := HandleGetConfigsTemplates()(&core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "standard-1")
}

func TestHandlePlaceholdersByFilenamesValidation(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/template/placeholders", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, GetPlaceholdersByFilenamesRequestInput{}))
	rec := httptest.NewRecorder()

	err := HandlePlaceholdersByFilenames()(&core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandlePlaceholdersByFilenamesSuccess(t *testing.T) {
	rootDir := t.TempDir()
	t.Setenv("ROOT_DIR", rootDir)

	templatesDir := filepath.Join(rootDir, "config_templates", "test-suite", "nested")
	require.NoError(t, os.MkdirAll(templatesDir, 0755))

	templateStr := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_type\":\"string\"}" }}`
	filePath := filepath.Join(templatesDir, "template.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(templateStr), 0644))

	input := GetPlaceholdersByFilenamesRequestInput{
		TestID:    "test-suite",
		Filenames: []string{"nested/template.yaml"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/template/placeholders", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err := HandlePlaceholdersByFilenames()(&core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.Contains(t, payload, "specific_fields")
}
