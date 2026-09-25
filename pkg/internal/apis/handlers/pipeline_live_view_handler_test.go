// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
)

type liveViewRunnerCall struct {
	apiKey string
	body   map[string]any
}

func newLiveViewRunner(
	t *testing.T,
	status int,
	response map[string]any,
) (*httptest.Server, *[]liveViewRunnerCall) {
	t.Helper()

	calls := []liveViewRunnerCall{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/credimi/live-view" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		calls = append(calls, liveViewRunnerCall{
			apiKey: r.Header.Get("Credimi-Api-Key"),
			body:   body,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)

	return server, &calls
}

func stubPipelineLiveViewTemporal(
	t *testing.T,
	status enums.WorkflowExecutionStatus,
	devices map[string]any,
) {
	t.Helper()

	mockClient := temporalmocks.NewClient(t)
	mockClient.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		"pipeline-1",
		"run-1",
	).Return(pipelineMobileFlowDescription(t, status, []string{"tenant/runner-1"}), nil).Once()
	if status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		mockClient.On(
			"QueryWorkflow",
			mock.Anything,
			"pipeline-1",
			"run-1",
			pipeline.PipelineMobileDevicesQuery,
		).Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{devices: devices}), nil).Once()
	}

	original := pipelineLiveViewTemporalClient
	t.Cleanup(func() { pipelineLiveViewTemporalClient = original })
	pipelineLiveViewTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}
}

func servePipelineLiveView(
	t *testing.T,
	app core.App,
	auth *core.Record,
	input PipelineLiveViewInput,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/live-view", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err := HandlePipelineLiveView()(&core.RequestEvent{
		App:  app,
		Auth: auth,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)

	return rec
}

func pipelineLiveViewTestApp(t *testing.T) (*tests.TestApp, *core.Record, string) {
	t.Helper()

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	namespace, err := pbutils.GetUserOrganizationCanonifiedName(app, authRecord.Id)
	require.NoError(t, err)

	return app, authRecord, namespace
}

func TestPipelineLiveViewOpensAndroidDevice(t *testing.T) {
	t.Setenv(InternalAdminAPIKeyEnvVar, "k")
	app, authRecord, namespace := pipelineLiveViewTestApp(t)
	runner, calls := newLiveViewRunner(t, http.StatusOK, map[string]any{"path": "/live/tok"})
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
		"tenant/runner-1/pixel": map[string]any{
			"serial":     "emulator-5554",
			"type":       "android_emulator",
			"runner_url": runner.URL,
		},
	})

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp PipelineLiveViewResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, []PipelineLiveViewStream{{
		DeviceID:   "tenant/runner-1/pixel",
		DeviceName: "pixel",
		URL:        runner.URL + "/live/tok",
	}}, resp.Streams)

	require.Len(t, *calls, 1)
	call := (*calls)[0]
	require.Equal(t, "k", call.apiKey)
	require.Equal(t, map[string]any{
		"device_identifier": "tenant/runner-1/pixel",
		"serial":            "emulator-5554",
		"namespace":         namespace,
		"workflow_id":       "pipeline-1",
		"run_id":            "run-1",
	}, call.body)
}

func TestPipelineLiveViewRejectsCompletedRun(t *testing.T) {
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, nil)

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "pipeline workflow is not running")
}

func TestPipelineLiveViewRejectsIOSOnlyExecution(t *testing.T) {
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
		"tenant/runner-1/iphone": map[string]any{
			"serial":     "sim-1",
			"type":       "ios_simulator",
			"runner_url": "http://runner.example",
		},
	})

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "no live-view capable device is initialized")
}

func TestPipelineLiveViewRejectsUnknownExplicitDevice(t *testing.T) {
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
		"tenant/runner-1/pixel": map[string]any{
			"serial":     "emulator-5554",
			"type":       "android_emulator",
			"runner_url": "http://runner.example",
		},
	})

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
		DeviceID:   "tenant/runner-1/other",
	})

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "pipeline mobile device is not initialized")
}

func TestPipelineLiveViewForwardsRunnerUnavailable(t *testing.T) {
	t.Setenv(InternalAdminAPIKeyEnvVar, "k")
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	runner, _ := newLiveViewRunner(t, http.StatusServiceUnavailable, map[string]any{
		"name":    "service_unavailable",
		"code":    http.StatusServiceUnavailable,
		"domain":  "live_view",
		"reason":  "live view unavailable",
		"message": "scrcpy is not installed",
	})
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
		"tenant/runner-1/pixel": map[string]any{
			"serial":     "emulator-5554",
			"type":       "android_emulator",
			"runner_url": runner.URL,
		},
	})

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
		DeviceID:   "tenant/runner-1/pixel",
	})

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "scrcpy is not installed")
}

func TestPipelineLiveViewRequiresAuth(t *testing.T) {
	rec := servePipelineLiveView(t, nil, nil, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
