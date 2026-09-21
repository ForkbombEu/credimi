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

	"github.com/forkbombeu/credimi/pkg/conformancecatalog"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestHandleGetConfigsTemplatesInvalidSurface(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/template/blueprints?surface=notasurface",
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
	require.NoError(
		t,
		os.WriteFile(filepath.Join(standardDir, "standard.yaml"), standardYaml, 0644),
	)

	versionMeta := VersionMetadata{
		UID:              "v1",
		Name:             "Version 1",
		SpecificationURL: "https://example.org/spec",
	}
	versionYaml, _ := yaml.Marshal(versionMeta)
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "version.yaml"), versionYaml, 0644))

	suiteMeta := SuiteMetadata{
		UID:       "suite-a",
		Name:      "Suite A",
		VisibleIn: []string{TemplateSurfaceManual, TemplateSurfacePipeline},
	}
	suiteYaml, _ := yaml.Marshal(suiteMeta)
	require.NoError(t, os.WriteFile(filepath.Join(suiteDir, "metadata.yaml"), suiteYaml, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(suiteDir, "test.yaml"), []byte("{}"), 0644))

	require.NoError(t, conformancecatalog.Default().LoadFromDir(templatesDir))

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
	req = req.WithContext(
		context.WithValue(
			req.Context(),
			middlewares.ValidatedInputKey,
			GetPlaceholdersByFilenamesRequestInput{},
		),
	)
	rec := httptest.NewRecorder()

	err := HandlePlaceholdersByFilenames()(&core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
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
