// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
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
	stubLiveViewTemporalOfType(t, "Dynamic Pipeline Workflow", status, devices)
}

func stubLiveViewTemporalOfType(
	t *testing.T,
	workflowType string,
	status enums.WorkflowExecutionStatus,
	devices map[string]any,
) {
	t.Helper()
	stubLiveViewWorkflow(t, "pipeline-1", workflowType, status, devices)
}

// stubLiveViewWorkflow answers the run run-1 of workflowID and returns the
// namespaces of the clients the handler asked for.
func stubLiveViewWorkflow(
	t *testing.T,
	workflowID string,
	workflowType string,
	status enums.WorkflowExecutionStatus,
	devices map[string]any,
) *[]string {
	t.Helper()

	description := pipelineMobileFlowDescription(t, status, []string{"tenant/runner-1"})
	description.WorkflowExecutionInfo.Type.Name = workflowType
	mockClient := temporalmocks.NewClient(t)
	mockClient.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		workflowID,
		"run-1",
	).Return(description, nil).Once()
	if status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		mockClient.On(
			"QueryWorkflow",
			mock.Anything,
			workflowID,
			"run-1",
			pipeline.PipelineMobileDevicesQuery,
		).Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{devices: devices}), nil).Once()
	}

	namespaces := []string{}
	original := pipelineLiveViewTemporalClient
	t.Cleanup(func() { pipelineLiveViewTemporalClient = original })
	pipelineLiveViewTemporalClient = func(namespace string) (client.Client, error) {
		namespaces = append(namespaces, namespace)
		return mockClient, nil
	}
	return &namespaces
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
	// Set to prove Credimi never forwards an internal admin key to a runner.
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "internal-admin-key")
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
		"run_identifier":    pipeline.RunIdentifier(namespace, "pipeline-1", "run-1"),
	}, call.body)
}

// liveViewSession is the action author session authoringSessionID that the
// live view looks up through actionAuthorSessionTemporalClient.
type liveViewSession struct {
	authoringSessionGrant
	// unsigned leaves the start input without Credimi's grant.
	unsigned bool
	// runs are the session's runs, oldest first; the first is the trusted one.
	runs   []string
	status enums.WorkflowExecutionStatus
	// clientErr fails the session namespace's client.
	clientErr error
}

// stubLiveViewSession makes actionAuthorSessionTemporalClient answer s and
// returns the namespaces asked for.
func stubLiveViewSession(t *testing.T, app *tests.TestApp, s liveViewSession) *[]string {
	t.Helper()
	setAuthoringSessionKey(t)
	c := &temporalmocks.Client{}
	stubSessionRuns(t, c, s.sessionID, s.runs...)
	if len(s.runs) > 0 {
		c.On(
			"GetWorkflowHistory",
			mock.Anything,
			s.sessionID,
			s.runs[0],
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).Return(s.history(t, app, !s.unsigned)).Maybe()
		c.On("DescribeWorkflowExecution", mock.Anything, s.sessionID, s.runs[0]).
			Return(authoringExecution(activities.ActionAuthorSessionWorkflowName, s.status, 0), nil).
			Maybe()
	}
	namespaces := []string{}
	original := actionAuthorSessionTemporalClient
	t.Cleanup(func() { actionAuthorSessionTemporalClient = original })
	actionAuthorSessionTemporalClient = func(namespace string) (client.Client, error) {
		namespaces = append(namespaces, namespace)
		return c, s.clientErr
	}
	return &namespaces
}

// refuseLiveViewTemporal fails the test if the handler reads the workflow.
func refuseLiveViewTemporal(t *testing.T) *temporalmocks.Client {
	t.Helper()
	c := &temporalmocks.Client{}
	original := pipelineLiveViewTemporalClient
	t.Cleanup(func() { pipelineLiveViewTemporalClient = original })
	pipelineLiveViewTemporalClient = func(string) (client.Client, error) { return c, nil }
	return c
}

// An action author session prepares its device with its workflow ID as the
// run identifier, so the runner must look the live view up by it, in the
// runner namespace the session runs in.
func TestPipelineLiveViewOpensActionAuthorSessionDevice(t *testing.T) {
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
	app, authRecord, namespace := pipelineLiveViewTestApp(t)
	server, calls := newLiveViewRunner(t, http.StatusOK, map[string]any{"path": "/live/tok"})
	deviceID, _ := createLiveViewDevice(t, app, authRecord, server.URL, true)
	stubLiveViewSession(t, app, liveViewSession{
		authoringSessionGrant: authoringSessionGrant{
			sessionID:    authoringSessionID,
			organization: namespace,
			device:       deviceID,
			version:      authoringAndroidVersion,
			actorID:      authRecord.Id,
		},
		runs:   []string{"run-1"},
		status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
	})
	namespaces := stubLiveViewWorkflow(
		t,
		authoringSessionID,
		activities.ActionAuthorSessionWorkflowName,
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		map[string]any{
			deviceID: map[string]any{"serial": "emulator-5554", "type": "android_emulator"},
		},
	)

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: authoringSessionID,
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, []string{authoringSessionNamespace}, *namespaces)
	require.Len(t, *calls, 1)
	require.Equal(t, map[string]any{
		"device_identifier": deviceID,
		"serial":            "emulator-5554",
		"namespace":         authoringSessionNamespace,
		"workflow_id":       authoringSessionID,
		"run_id":            "run-1",
		"run_identifier":    authoringSessionID,
	}, (*calls)[0].body)
}

// Only the user who started a session opens the live view of its trusted
// run; anyone else gets the answer of a session that does not exist.
func TestPipelineLiveViewRefusesActionAuthorSessionsOfOthers(t *testing.T) {
	for _, tc := range []struct {
		name string
		// runs are the session's runs, oldest first; none is not visible.
		runs     []string
		unsigned bool
		member   bool
		// organization and namespace override the ones the grant names, the
		// caller's organization and the session's runner namespace.
		organization string
		namespace    string
		clientErr    error
		code         int
		reason       string
	}{
		{name: "not visible in Temporal", code: http.StatusNotFound, reason: "session_not_found"},
		{
			name:         "started for another organization",
			runs:         []string{"run-1"},
			organization: "other-organization",
			code:         http.StatusNotFound,
			reason:       "session_not_found",
		},
		{
			name:      "granted for another namespace",
			runs:      []string{"run-1"},
			namespace: "runner-otherrunner0001",
			code:      http.StatusNotFound,
			reason:    "session_not_found",
		},
		{
			name:     "not signed by Credimi",
			runs:     []string{"run-1"},
			unsigned: true,
			code:     http.StatusNotFound,
			reason:   "session_not_found",
		},
		{
			name:   "a later run",
			runs:   []string{"run-0", "run-1"},
			code:   http.StatusNotFound,
			reason: "session_not_found",
		},
		{
			name:   "another member",
			runs:   []string{"run-1"},
			member: true,
			code:   http.StatusNotFound,
			reason: "session_not_found",
		},
		{
			name:      "Temporal unavailable",
			runs:      []string{"run-1"},
			clientErr: errors.New("dial failed"),
			code:      http.StatusInternalServerError,
			reason:    "session_lookup_failed",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
			app, authRecord, namespace := pipelineLiveViewTestApp(t)
			server, calls := newLiveViewRunner(
				t,
				http.StatusOK,
				map[string]any{"path": "/live/tok"},
			)
			deviceID, _ := createLiveViewDevice(t, app, authRecord, server.URL, true)
			organization := namespace
			if tc.organization != "" {
				organization = tc.organization
			}
			stubLiveViewSession(t, app, liveViewSession{
				authoringSessionGrant: authoringSessionGrant{
					sessionID:    authoringSessionID,
					organization: organization,
					device:       deviceID,
					version:      authoringAndroidVersion,
					actorID:      authRecord.Id,
					namespace:    tc.namespace,
				},
				unsigned:  tc.unsigned,
				runs:      tc.runs,
				status:    enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
				clientErr: tc.clientErr,
			})
			caller := authRecord
			if tc.member {
				caller = joinAgentRunnerOrganization(t, app, "member")
			}
			workflow := refuseLiveViewTemporal(t)

			rec := servePipelineLiveView(t, app, caller, PipelineLiveViewInput{
				WorkflowID: authoringSessionID,
				RunID:      "run-1",
			})

			require.Equal(t, tc.code, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), tc.reason)
			require.Empty(t, *calls)
			workflow.AssertNotCalled(
				t,
				"DescribeWorkflowExecution",
				mock.Anything,
				mock.Anything,
				mock.Anything,
			)
		})
	}
}

// A workflow of the session type under an ID that names no session is no
// session anyone started.
func TestPipelineLiveViewRefusesSessionWorkflowsOfOtherIDs(t *testing.T) {
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
	app, authRecord, _ := pipelineLiveViewTestApp(t)
	server, calls := newLiveViewRunner(t, http.StatusOK, map[string]any{"path": "/live/tok"})
	deviceID, _ := createLiveViewDevice(t, app, authRecord, server.URL, true)
	namespaces := stubLiveViewSession(t, app, liveViewSession{})
	stubLiveViewTemporalOfType(
		t,
		activities.ActionAuthorSessionWorkflowName,
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		map[string]any{
			deviceID: map[string]any{"serial": "emulator-5554", "type": "android_emulator"},
		},
	)

	rec := servePipelineLiveView(t, app, authRecord, PipelineLiveViewInput{
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	})

	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "session_not_found")
	require.Empty(t, *calls)
	require.Empty(t, *namespaces)
}

// A session's status is the starter's only: anyone else gets the answer of
// a session that does not exist whether it runs or has closed.
func TestPipelineLiveViewChecksTheSessionOwnerBeforeItsStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status enums.WorkflowExecutionStatus
		member bool
		// organization overrides the one the grant names, the caller's.
		organization string
		code         int
		reason       string
	}{
		{
			name:         "another organization, running",
			status:       enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			organization: "other-organization",
			code:         http.StatusNotFound,
			reason:       "session_not_found",
		},
		{
			name:   "another member, closed",
			status: enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			member: true,
			code:   http.StatusNotFound,
			reason: "session_not_found",
		},
		{
			name:   "another member, running",
			status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			member: true,
			code:   http.StatusNotFound,
			reason: "session_not_found",
		},
		{
			name:   "starter, closed",
			status: enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			code:   http.StatusConflict,
			reason: "pipeline workflow is not running",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(mobilerunner.CredentialSecretEnvVar, "runner-credential-secret")
			app, authRecord, namespace := pipelineLiveViewTestApp(t)
			server, calls := newLiveViewRunner(
				t,
				http.StatusOK,
				map[string]any{"path": "/live/tok"},
			)
			deviceID, _ := createLiveViewDevice(t, app, authRecord, server.URL, true)
			organization := namespace
			if tc.organization != "" {
				organization = tc.organization
			}
			stubLiveViewSession(t, app, liveViewSession{
				authoringSessionGrant: authoringSessionGrant{
					sessionID:    authoringSessionID,
					organization: organization,
					device:       deviceID,
					version:      authoringAndroidVersion,
					actorID:      authRecord.Id,
				},
				runs:   []string{"run-1"},
				status: tc.status,
			})
			caller := authRecord
			if tc.member {
				caller = joinAgentRunnerOrganization(t, app, "member")
			}
			description := pipelineMobileFlowDescription(t, tc.status, []string{"tenant/runner-1"})
			description.WorkflowExecutionInfo.Type.Name = activities.ActionAuthorSessionWorkflowName
			mockClient := &temporalmocks.Client{}
			mockClient.On("DescribeWorkflowExecution", mock.Anything, authoringSessionID, "run-1").
				Return(description, nil).
				Maybe()
			original := pipelineLiveViewTemporalClient
			t.Cleanup(func() { pipelineLiveViewTemporalClient = original })
			pipelineLiveViewTemporalClient = func(string) (client.Client, error) {
				return mockClient, nil
			}

			rec := servePipelineLiveView(t, app, caller, PipelineLiveViewInput{
				WorkflowID: authoringSessionID,
				RunID:      "run-1",
			})

			require.Equal(t, tc.code, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), tc.reason)
			require.Empty(t, *calls)
			if tc.code == http.StatusNotFound {
				mockClient.AssertNotCalled(
					t,
					"DescribeWorkflowExecution",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				)
			}
		})
	}
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
