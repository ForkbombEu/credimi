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

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
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
	canonify.RegisterCanonifyHooks(app)

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	namespace, err := pbutils.GetUserOrganizationCanonifiedName(app, authRecord.Id)
	require.NoError(t, err)

	return app, authRecord, namespace
}

// createLiveViewDevice stores a runner at runnerURL holding an emulator and
// returns the emulator device ID with its runner.
func createLiveViewDevice(
	t *testing.T,
	app *tests.TestApp,
	authRecord *core.Record,
	runnerURL string,
	adminManaged bool,
) (string, *core.Record) {
	t.Helper()

	orgID, err := pbutils.GetUserOrganizationID(app, authRecord.Id)
	require.NoError(t, err)
	runner := createRunnerRecord(t, app, orgID, "live-runner")
	runner.Set("ip", runnerURL)
	runner.Set("admin_managed", adminManaged)
	require.NoError(t, app.Save(runner))
	createDeviceRecord(t, app, orgID, runner.Id, "pixel")

	device, err := app.FindFirstRecordByFilter(
		"mobile_devices",
		"runner = {:runner}",
		map[string]any{"runner": runner.Id},
	)
	require.NoError(t, err)
	deviceID, err := mobileDeviceIdentifier(app, device)
	require.NoError(t, err)

	return deviceID, runner
}

func TestPipelineLiveViewOpensAndroidDevice(t *testing.T) {
	t.Setenv(InternalAdminAPIKeyEnvVar, "internal-admin-key")
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
	app, authRecord, namespace := pipelineLiveViewTestApp(t)
	server, calls := newLiveViewRunner(t, http.StatusOK, map[string]any{"path": "/live/tok"})
	deviceID, runner := createLiveViewDevice(t, app, authRecord, server.URL, true)
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
		deviceID: map[string]any{
			"serial": "emulator-5554",
			"type":   "android_emulator",
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
		DeviceID:   deviceID,
		DeviceName: "pixel",
		URL:        server.URL + "/live/tok",
	}}, resp.Streams)

	credential, err := mobilerunner.Credential(runner)
	require.NoError(t, err)
	require.Len(t, *calls, 1)
	call := (*calls)[0]
	require.Equal(t, credential, call.apiKey)
	require.NotEqual(t, "internal-admin-key", call.apiKey)
	require.Equal(t, map[string]any{
		"device_identifier": deviceID,
		"serial":            "emulator-5554",
		"namespace":         namespace,
		"workflow_id":       "pipeline-1",
		"run_id":            "run-1",
	}, call.body)
}

// A tenant chooses its runner's address, so Credimi must never connect to a
// loopback or private one on its behalf.
func TestPipelineLiveViewNeverContactsTenantLoopbackRunner(t *testing.T) {
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	server, calls := newLiveViewRunner(t, http.StatusOK, map[string]any{"path": "/live/tok"})
	deviceID, _ := createLiveViewDevice(t, app, authRecord, server.URL, false)
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
		deviceID: map[string]any{"serial": "emulator-5554", "type": "android_emulator"},
	})

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
		DeviceID:   deviceID,
	})

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "device runner is offline", decodeAPIError(t, rec).Reason)
	require.Empty(t, *calls)
}

func TestPipelineLiveViewRequiresRunnerCredentialSecret(t *testing.T) {
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "")
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	server, calls := newLiveViewRunner(t, http.StatusOK, map[string]any{"path": "/live/tok"})
	deviceID, _ := createLiveViewDevice(t, app, authRecord, server.URL, true)
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
		deviceID: map[string]any{"serial": "emulator-5554", "type": "android_emulator"},
	})

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
		DeviceID:   deviceID,
	})

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, decodeAPIError(t, rec).Message, mobilerunner.CredentialSecretEnvVar)
	require.Empty(t, *calls)
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
	require.Equal(
		t,
		"other is still being prepared by the pipeline (for emulators and Redroid "+
			"this includes creating and booting it). Live view becomes available as "+
			"soon as it is ready; try again in a moment.",
		decodeAPIError(t, rec).Message,
	)
}

// Devices appear in the pipeline only once its mobile setup has finished, so an
// emulator still being created or booted must not look like a broken request.
func TestPipelineLiveViewExplainsDevicesStillBeingPrepared(t *testing.T) {
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{})

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, decodeAPIError(t, rec).Message, "still preparing its devices")
}

func TestPipelineLiveViewForwardsRunnerRefusals(t *testing.T) {
	cases := []struct {
		name         string
		runnerStatus int
		message      string
		wantMessage  string
		wantStatus   int
	}{
		{
			name:         "unavailable",
			runnerStatus: http.StatusServiceUnavailable,
			message:      "The runner image has no scrcpy. Live stream needs the linux/amd64 runner image.",
			wantMessage:  "The runner image has no scrcpy. Live stream needs the linux/amd64 runner image.",
			wantStatus:   http.StatusServiceUnavailable,
		},
		{
			name:         "unsupported device",
			runnerStatus: http.StatusBadRequest,
			message:      "Live stream is not enabled for this device on its runner.",
			wantMessage:  "Live stream is not enabled for this device on its runner.",
			wantStatus:   http.StatusUnprocessableEntity,
		},
		{
			name:         "runner failure",
			runnerStatus: http.StatusInternalServerError,
			message:      "The runner could not start the live stream (boom).",
			wantMessage:  "The runner could not start the live stream (boom).",
			wantStatus:   http.StatusBadGateway,
		},
		{
			name:         "unauthorized",
			runnerStatus: http.StatusUnauthorized,
			message:      "invalid api key",
			wantMessage: "Credimi is not authorized on the runner that holds this device. " +
				"Check that the runner registered with this Credimi instance " +
				"and uses its current runner credential.",
			wantStatus: http.StatusBadGateway,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
			app, authRecord, _ := pipelineLiveViewTestApp(t)
			server, _ := newLiveViewRunner(t, tc.runnerStatus, map[string]any{
				"name":    "runner_error",
				"code":    tc.runnerStatus,
				"domain":  "live_view",
				"reason":  tc.message,
				"message": tc.message,
			})
			deviceID, _ := createLiveViewDevice(t, app, authRecord, server.URL, true)
			stubPipelineLiveViewTemporal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, map[string]any{
				deviceID: map[string]any{"serial": "emulator-5554", "type": "android_emulator"},
			})

			rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
				WorkflowID: "pipeline-1",
				RunID:      "run-1",
				DeviceID:   deviceID,
			})

			require.Equal(t, tc.wantStatus, rec.Code)
			require.Equal(t, tc.wantMessage, decodeAPIError(t, rec).Message)
		})
	}
}

func TestPipelineLiveViewRequiresAuth(t *testing.T) {
	rec := servePipelineLiveView(t, nil, nil, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
