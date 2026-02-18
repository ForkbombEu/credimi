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
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
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
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compliance//save-variables-and-start",
		bytes.NewBuffer(body),
	)
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
	require.NoError(
		t,
		os.MkdirAll(filepath.Join(rootDir, "config_templates", "openid", "v1"), 0o755),
	)
	t.Setenv("ROOT_DIR", rootDir)

	body, _ := json.Marshal(SaveVariablesAndStartRequestInput{
		ConfigsWithFields: map[string][]Variable{},
		ConfigsWithJSON:   map[string]string{"unknown/test-1": "{}"},
		CustomChecks:      map[string]CustomCheck{},
	})
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compliance/openid/v1/save-variables-and-start",
		bytes.NewBuffer(body),
	)
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
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compliance/ewc/v1/save-variables-and-start",
		bytes.NewBuffer(body),
	)
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
	require.NoError(
		t,
		os.MkdirAll(filepath.Join(rootDir, "config_templates", "openid", "v1"), 0o755),
	)
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
			[]byte(
				"foo: {{ credimi \"{\\\"field_id\\\":\\\"foo\\\",\\\"credimi_id\\\":\\\"cred-1\\\"}\" }}\n",
			),
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

func TestStartOpenIDNetWorkflowInvalidVersion(t *testing.T) {
	params := WorkflowStarterParams{
		YAMLData: "variant: json\ntest: test-1\n",
		Version:  "invalid",
	}

	_, err := startOpenIDNetWorkflow(params)
	require.Error(t, err)
}

func TestStartOpenIDNetWorkflowSuccess(t *testing.T) {
	rootDir := t.TempDir()
	templatePath := filepath.Join(rootDir, workflows.OpenIDNetStepCITemplatePathv1_0)
	require.NoError(t, os.MkdirAll(filepath.Dir(templatePath), 0o755))
	require.NoError(t, os.WriteFile(templatePath, []byte("template"), 0o644))
	t.Setenv("ROOT_DIR", rootDir)

	origStart := openIDNetWorkflowStart
	t.Cleanup(func() {
		openIDNetWorkflowStart = origStart
	})

	openIDNetWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-openid",
			WorkflowRunID: "run-openid",
		}, nil
	}

	params := WorkflowStarterParams{
		YAMLData:  "variant: json\ntest: test-1\n",
		Email:     "user@example.com",
		AppURL:    "https://app.example.com",
		Namespace: "ns",
		Memo:      map[string]interface{}{"test": "test-1"},
		Author:    "openid_conformance_suite",
		Version:   "1.0",
		AppName:   "Credimi",
		LogoUrl:   "https://app.example.com/logo.png",
		UserName:  "User",
	}

	result, err := startOpenIDNetWorkflow(params)
	require.NoError(t, err)
	require.Equal(t, "wf-openid", result.WorkflowID)
	require.Equal(t, string(params.Author), result.Author)
}

func TestStartEWCWorkflowUnsupportedProtocol(t *testing.T) {
	rootDir := t.TempDir()
	filename := filepath.Join(rootDir, workflows.EWCTemplateFolderPath+"test.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte("template"), 0o644))
	t.Setenv("ROOT_DIR", rootDir)

	params := WorkflowStarterParams{
		YAMLData: "sessionId: session-1\n",
		Protocol: "unknown",
		TestName: "ewctest.yaml",
	}

	_, err := startEWCWorkflow(params)
	require.Error(t, err)
}

func TestStartEWCWorkflowSuccess(t *testing.T) {
	rootDir := t.TempDir()
	filename := filepath.Join(rootDir, workflows.EWCTemplateFolderPath+"test.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte("template"), 0o644))
	t.Setenv("ROOT_DIR", rootDir)

	origStart := ewcWorkflowStart
	t.Cleanup(func() {
		ewcWorkflowStart = origStart
	})
	ewcWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-ewc",
			WorkflowRunID: "run-ewc",
		}, nil
	}

	params := WorkflowStarterParams{
		YAMLData:  "sessionId: session-1\n",
		Email:     "user@example.com",
		AppURL:    "https://app.example.com",
		Namespace: "ns",
		Memo:      map[string]interface{}{"test": "ewc/test"},
		Author:    "ewc",
		Protocol:  "openid4vp_wallet",
		TestName:  "ewctest.yaml",
		AppName:   "Credimi",
		LogoUrl:   "https://app.example.com/logo.png",
		UserName:  "User",
	}

	result, err := startEWCWorkflow(params)
	require.NoError(t, err)
	require.Equal(t, "wf-ewc", result.WorkflowID)
	require.Equal(t, string(params.Author), result.Author)
}

func TestStartEudiwWorkflowSuccess(t *testing.T) {
	rootDir := t.TempDir()
	filename := filepath.Join(rootDir, workflows.EudiwTemplateFolderPath+"test.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte("template"), 0o644))
	t.Setenv("ROOT_DIR", rootDir)

	origStart := eudiwWorkflowStart
	t.Cleanup(func() {
		eudiwWorkflowStart = origStart
	})
	eudiwWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-eudiw",
			WorkflowRunID: "run-eudiw",
		}, nil
	}

	params := WorkflowStarterParams{
		YAMLData:  "nonce: n1\nid: id-1\n",
		Email:     "user@example.com",
		AppURL:    "https://app.example.com",
		Namespace: "ns",
		Memo:      map[string]interface{}{"test": "eudiw/test"},
		Author:    "eudiw",
		TestName:  "eudiwtest.yaml",
		AppName:   "Credimi",
		LogoUrl:   "https://app.example.com/logo.png",
		UserName:  "User",
	}

	result, err := startEudiwWorkflow(params)
	require.NoError(t, err)
	require.Equal(t, "wf-eudiw", result.WorkflowID)
	require.Equal(t, string(params.Author), result.Author)
}

func TestStartVLEIWorkflowSuccess(t *testing.T) {
	origStart := vleiWorkflowStart
	t.Cleanup(func() {
		vleiWorkflowStart = origStart
	})
	vleiWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-vlei",
			WorkflowRunID: "run-vlei",
		}, nil
	}

	params := WorkflowStarterParams{
		YAMLData:  "credentialID: cred-1\nserverURL: https://vlei.example.com\n",
		AppURL:    "https://app.example.com",
		Namespace: "ns",
		Memo:      map[string]interface{}{"test": "vlei/test"},
		Author:    "vlei",
	}

	result, err := startvLEIWorkflow(params)
	require.NoError(t, err)
	require.Equal(t, "wf-vlei", result.WorkflowID)
	require.Equal(t, string(params.Author), result.Author)
}

func TestProcessCustomChecksSuccess(t *testing.T) {
	origStart := customCheckWorkflowStart
	t.Cleanup(func() {
		customCheckWorkflowStart = origStart
	})
	customCheckWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-custom",
			WorkflowRunID: "run-custom",
		}, nil
	}

	result, err := processCustomChecks(
		"steps: []\n",
		"https://app.example.com",
		"ns",
		map[string]interface{}{"author": "custom"},
		"{\"foo\":\"bar\"}",
	)
	require.NoError(t, err)
	require.Equal(t, "wf-custom", result.WorkflowID)
	require.Equal(t, "custom", result.Author)
}

func TestReadTemplateFileError(t *testing.T) {
	_, err := readTemplateFile(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}
