// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestHandleSaveVariablesAndStartMissingProtocol(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	body, _ := json.Marshal(SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{},
		CustomChecks:      map[string]CustomCheck{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/compliance//save-variables-and-start", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{},
		CustomChecks:      map[string]CustomCheck{},
	}))
	rec := httptest.NewRecorder()

	err = HandleSaveVariablesAndStart()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSaveVariablesAndStartUnsupportedAuthor(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	rootDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "config_templates", "openid", "v1"), 0o755))
	t.Setenv("ROOT_DIR", rootDir)

	body, _ := json.Marshal(SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{"unknown/test-1": "{}"},
		CustomChecks:      map[string]CustomCheck{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/openid/v1/save-variables-and-start", bytes.NewBuffer(body))
	req.SetPathValue("protocol", "openid")
	req.SetPathValue("version", "v1")
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{"unknown/test-1": "{}"},
		CustomChecks:      map[string]CustomCheck{},
	}))
	rec := httptest.NewRecorder()

	err = HandleSaveVariablesAndStart()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSaveVariablesAndStartSuccessJSON(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	rootDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "config_templates", "ewc", "v1"), 0o755))
	t.Setenv("ROOT_DIR", rootDir)

	origRegistry := workflowRegistry
	t.Cleanup(func() {
		workflowRegistry = origRegistry
	})
	workflowRegistry = map[Author]WorkflowStarter{
		"ewc": func(params WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
			return workflowengine.WorkflowResult{
				WorkflowID:    "wf-1",
				WorkflowRunID: "run-1",
				Author:        string(params.Author),
			}, nil
		},
	}

	body, _ := json.Marshal(SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{"ewc/test-1": "{}"},
		CustomChecks:      map[string]CustomCheck{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/ewc/v1/save-variables-and-start", bytes.NewBuffer(body))
	req.SetPathValue("protocol", "ewc")
	req.SetPathValue("version", "v1")
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{"ewc/test-1": "{}"},
		CustomChecks:      map[string]CustomCheck{},
	}))
	rec := httptest.NewRecorder()

	err = HandleSaveVariablesAndStart()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "wf-1")
}

func TestReduceData(t *testing.T) {
	input := map[string]any{
		"trim": "  value  ",
		"json": `{"nested": " ok "}`,
		"list": []any{"  a ", `["b ", " c"]`},
	}
	out := reduceData(input).(map[string]any)
	require.Equal(t, "value", out["trim"])
	require.Equal(t, map[string]any{"nested": "ok"}, out["json"])

	list := out["list"].([]any)
	require.Equal(t, "a", list[0])
	require.Equal(t, []any{"b", "c"}, list[1])
}
