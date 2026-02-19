// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestHandleGetDeeplinkInvalidJSON(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
}

func TestHandleGetDeeplinkWaitError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	restore := installDeeplinkSeams(t)
	defer restore()

	deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	deeplinkTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	deeplinkWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("wait failed")
	}

	body, _ := json.Marshal(CredentialDeeplinkRequest{Yaml: "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "failed to get workflow result")
}

func TestHandleGetDeeplinkMalformedOutput(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	restore := installDeeplinkSeams(t)
	defer restore()

	deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	deeplinkTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	deeplinkWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{Output: "invalid"}, nil
	}

	body, _ := json.Marshal(CredentialDeeplinkRequest{Yaml: "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "output is not an array")
}

func TestHandleGetDeeplinkSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	restore := installDeeplinkSeams(t)
	defer restore()

	deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	deeplinkTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	deeplinkWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			Output: []any{
				map[string]any{
					"steps": []any{
						map[string]any{
							"captures": map[string]any{
								"deeplink": "credimi://link",
							},
						},
					},
				},
			},
		}, nil
	}

	body, _ := json.Marshal(CredentialDeeplinkRequest{Yaml: "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/get-deeplink", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	err = HandleGetDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "credimi://link")
}

func installDeeplinkSeams(t testing.TB) func() {
	t.Helper()

	origStart := deeplinkStartWorkflow
	origClient := deeplinkTemporalClient
	origWait := deeplinkWaitForWorkflowResult

	return func() {
		deeplinkStartWorkflow = origStart
		deeplinkTemporalClient = origClient
		deeplinkWaitForWorkflowResult = origWait
	}
}
