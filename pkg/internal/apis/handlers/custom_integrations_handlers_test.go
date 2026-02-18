// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func withValidatedCustomIntegrationInput(
	req *http.Request,
	input RunCustomIntegrationRequestInput,
) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
}

func TestHandleRunCustomIntegrationMissingYAML(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	input := RunCustomIntegrationRequestInput{
		Yaml: "",
	}
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/custom-integrations/run",
		bytes.NewBufferString("{}"),
	)
	req = withValidatedCustomIntegrationInput(req, input)
	rec := httptest.NewRecorder()

	err = HandleRunCustomIntegration()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "yaml is empty")
}

func TestHandleRunCustomIntegrationWorkflowStartError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origStart := customCheckWorkflowStart
	t.Cleanup(func() {
		customCheckWorkflowStart = origStart
	})
	customCheckWorkflowStart = func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("boom")
	}

	input := RunCustomIntegrationRequestInput{
		Yaml: "steps: []\n",
	}
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/custom-integrations/run",
		bytes.NewBufferString("{}"),
	)
	req = withValidatedCustomIntegrationInput(req, input)
	rec := httptest.NewRecorder()

	err = HandleRunCustomIntegration()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "failed to process custom integration")
}

func TestHandleRunCustomIntegrationDataMarshalError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	input := RunCustomIntegrationRequestInput{
		Yaml: "steps: []\n",
		Data: map[string]any{
			"bad": func() {},
		},
	}
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/custom-integrations/run",
		bytes.NewBufferString("{}"),
	)
	req = withValidatedCustomIntegrationInput(req, input)
	rec := httptest.NewRecorder()

	err = HandleRunCustomIntegration()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "failed to serialize data to JSON")
}

func TestHandleRunCustomIntegrationSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origStart := customCheckWorkflowStart
	t.Cleanup(func() {
		customCheckWorkflowStart = origStart
	})

	var capturedNamespace string
	var capturedInput workflowengine.WorkflowInput
	customCheckWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedInput = input
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-custom",
			WorkflowRunID: "run-custom",
		}, nil
	}

	timeoutSeconds := 7
	input := RunCustomIntegrationRequestInput{
		Yaml: "steps: []\n",
		Data: map[string]any{
			"foo": "bar",
		},
		TimeoutSeconds: &timeoutSeconds,
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/custom-integrations/run",
		bytes.NewBufferString("{}"),
	)
	req = withValidatedCustomIntegrationInput(req, input)
	rec := httptest.NewRecorder()

	err = HandleRunCustomIntegration()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "wf-custom")

	expectedNamespace, err := GetUserOrganizationCanonifiedName(app, authRecord.Id)
	require.NoError(t, err)
	require.Equal(t, expectedNamespace, capturedNamespace)

	payload, ok := capturedInput.Payload.(workflows.CustomCheckWorkflowPayload)
	require.True(t, ok)
	require.Equal(t, "steps: []\n", payload.Yaml)

	require.Equal(t, app.Settings().Meta.AppURL, capturedInput.Config["app_url"])
	require.Equal(t, `{"foo":"bar"}`, capturedInput.Config["env"])

	memo, ok := capturedInput.Config["memo"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "custom-integration", memo["test"])
	require.Equal(t, authRecord.Id, memo["author"])

	require.NotNil(t, capturedInput.ActivityOptions)
	require.Equal(t, 7*time.Second, capturedInput.ActivityOptions.StartToCloseTimeout)
	require.Equal(t, 35*time.Second, capturedInput.ActivityOptions.ScheduleToCloseTimeout)
	require.NotNil(t, capturedInput.ActivityOptions.RetryPolicy)
	require.Equal(t, int32(5), capturedInput.ActivityOptions.RetryPolicy.MaximumAttempts)
	require.Equal(t, 1.0, capturedInput.ActivityOptions.RetryPolicy.BackoffCoefficient)
}

func TestProcessCustomChecksEmptyYAML(t *testing.T) {
	result, err := processCustomChecks(
		"",
		"https://app.example",
		"ns-1",
		map[string]interface{}{"author": "custom"},
		"",
		5*time.Second,
	)
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Empty(t, result.WorkflowID)
}

func TestProcessCustomChecksStartError(t *testing.T) {
	origStart := customCheckWorkflowStart
	t.Cleanup(func() {
		customCheckWorkflowStart = origStart
	})
	customCheckWorkflowStart = func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("boom")
	}

	result, err := processCustomChecks(
		"steps: []\n",
		"https://app.example",
		"ns-1",
		map[string]interface{}{"author": "custom"},
		"",
		5*time.Second,
	)
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Empty(t, result.WorkflowID)
}

func TestProcessCustomChecksSuccess(t *testing.T) {
	origStart := customCheckWorkflowStart
	t.Cleanup(func() {
		customCheckWorkflowStart = origStart
	})

	var capturedNamespace string
	var capturedInput workflowengine.WorkflowInput
	customCheckWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedInput = input
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-custom",
			WorkflowRunID: "run-custom",
		}, nil
	}

	result, err := processCustomChecks(
		"steps: []\n",
		"https://app.example.com",
		"ns",
		map[string]interface{}{"author": "custom-user"},
		`{"foo":"bar"}`,
		12*time.Second,
	)
	require.NoError(t, err)
	require.Equal(t, "wf-custom", result.WorkflowID)
	require.Equal(t, "custom-user", result.Author)

	require.Equal(t, "ns", capturedNamespace)

	payload, ok := capturedInput.Payload.(workflows.CustomCheckWorkflowPayload)
	require.True(t, ok)
	require.Equal(t, "steps: []\n", payload.Yaml)

	require.Equal(t, "https://app.example.com", capturedInput.Config["app_url"])
	require.Equal(t, `{"foo":"bar"}`, capturedInput.Config["env"])

	memo, ok := capturedInput.Config["memo"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "custom-user", memo["author"])

	require.NotNil(t, capturedInput.ActivityOptions)
	require.Equal(t, 12*time.Second, capturedInput.ActivityOptions.StartToCloseTimeout)
	require.Equal(t, 60*time.Second, capturedInput.ActivityOptions.ScheduleToCloseTimeout)
	require.NotNil(t, capturedInput.ActivityOptions.RetryPolicy)
	require.Equal(t, int32(5), capturedInput.ActivityOptions.RetryPolicy.MaximumAttempts)
	require.Equal(t, 1.0, capturedInput.ActivityOptions.RetryPolicy.BackoffCoefficient)
}
