// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

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
