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

func withValidatedInput(
	req *http.Request,
	input SaveVariablesAndStartRequestInput,
) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
}

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
	req = withValidatedInput(req, SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{},
		CustomChecks:      map[string]CustomCheck{},
	})
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
	req = withValidatedInput(req, SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{"unknown/test-1": "{}"},
		CustomChecks:      map[string]CustomCheck{},
	})
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
	req = withValidatedInput(req, SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{"ewc/test-1": "{}"},
		CustomChecks:      map[string]CustomCheck{},
	})
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

func TestHandleSaveVariablesAndStartCustomCheckMissingYAML(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	rootDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "config_templates", "openid", "v1"), 0o755))
	t.Setenv("ROOT_DIR", rootDir)

	input := SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{},
		CustomChecks: map[string]CustomCheck{
			"custom-1": {Yaml: ""},
		},
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compliance/openid/v1/save-variables-and-start",
		bytes.NewBuffer(body),
	)
	req.SetPathValue("protocol", "openid")
	req.SetPathValue("version", "v1")
	req = withValidatedInput(req, input)
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
	require.Contains(t, rec.Body.String(), "yaml is required")
}

func TestHandleSaveVariablesAndStartVariablesFlow(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	rootDir := t.TempDir()
	templateDir := filepath.Join(rootDir, "config_templates", "ewc", "v1", "ewc")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))
	templatePath := filepath.Join(templateDir, "test-1.yaml")
	require.NoError(
		t,
		os.WriteFile(
			templatePath,
			[]byte("foo: {{ credimi \"{\\\"field_id\\\":\\\"foo\\\",\\\"credimi_id\\\":\\\"cred-1\\\"}\" }}\n"),
			0o644,
		),
	)
	t.Setenv("ROOT_DIR", rootDir)

	var captured WorkflowStarterParams
	origRegistry := workflowRegistry
	t.Cleanup(func() {
		workflowRegistry = origRegistry
	})
	workflowRegistry = map[Author]WorkflowStarter{
		"ewc": func(params WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
			captured = params
			return workflowengine.WorkflowResult{
				WorkflowID:    "wf-variables",
				WorkflowRunID: "run-variables",
			}, nil
		},
	}

	input := SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{
			"ewc/test-1.yaml": {
				{FieldName: "foo", Value: "bar", CredimiID: "cred-1"},
			},
		},
		ConfigsWithJSON: map[string]string{},
		CustomChecks:    map[string]CustomCheck{},
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compliance/ewc/v1/save-variables-and-start",
		bytes.NewBuffer(body),
	)
	req.SetPathValue("protocol", "ewc")
	req.SetPathValue("version", "v1")
	req = withValidatedInput(req, input)
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
	require.Contains(t, rec.Body.String(), "wf-variables")
	require.Contains(t, captured.YAMLData, "foo: bar")

	orgID, err := GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)

	records, err := app.FindRecordsByFilter(
		"config_values",
		"template_path={:template_path} && owner={:owner}",
		"",
		-1,
		0,
		map[string]any{
			"template_path": "ewc/test-1.yaml",
			"owner":         orgID,
		},
	)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "cred-1", records[0].GetString("credimi_id"))
	require.Equal(t, "foo", records[0].GetString("field_name"))
}

func TestHandleSaveVariablesAndStartVariablesMissingTemplate(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	rootDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "config_templates", "ewc", "v1"), 0o755))
	t.Setenv("ROOT_DIR", rootDir)

	input := SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{
			"ewc/missing.yaml": {
				{FieldName: "foo", Value: "bar", CredimiID: "cred-1"},
			},
		},
		ConfigsWithJSON: map[string]string{},
		CustomChecks:    map[string]CustomCheck{},
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compliance/ewc/v1/save-variables-and-start",
		bytes.NewBuffer(body),
	)
	req.SetPathValue("protocol", "ewc")
	req.SetPathValue("version", "v1")
	req = withValidatedInput(req, input)
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
	require.Contains(t, rec.Body.String(), "failed to open template")
}
