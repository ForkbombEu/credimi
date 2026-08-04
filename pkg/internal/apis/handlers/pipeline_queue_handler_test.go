// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

type queueStub struct {
	cancelled       bool
	enqueueRequests []workflows.MobileDeviceSemaphoreEnqueueRunRequest
}

func setupPipelineQueueApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	ensureMobileDevicesCollection(t, app)
	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	return app
}

func setupPipelineQueueAppWithPipeline(t testing.TB, orgID string, yaml string) *tests.TestApp {
	app := setupPipelineQueueApp(t)

	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", "pipeline123")
	record.Set("description", "test-description")
	record.Set("steps", map[string]any{"rest-chain": map[string]any{"yaml": yaml}})
	record.Set("yaml", yaml)
	require.NoError(t, app.Save(record))

	createPipelineQueueMobileRunner(t, app, orgID, "runner-1", false)
	createPipelineQueueMobileRunner(t, app, orgID, "runner-2", false)

	return app
}

func createPipelineQueueMobileRunner(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
	published bool,
) {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("ip", "https://runner.example.test")
	record.Set("type", "android_phone")
	record.Set("published", published)
	require.NoError(t, app.Save(record))

	deviceColl, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(deviceColl)
	device.Set("owner", orgID)
	device.Set("runner", record.Id)
	device.Set("name", "device-1")
	device.Set("canonified_name", "device-1")
	device.Set("type", "android_phone")
	require.NoError(t, app.Save(device))
}

func ensureOrganizationsQueueLimitField(t testing.TB, app *tests.TestApp) {
	collection, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)

	if collection.Fields.GetByName("max_pipelines_in_queue") != nil {
		return
	}

	collection.Fields.Add(&core.NumberField{
		Name:    "max_pipelines_in_queue",
		OnlyInt: true,
	})
	require.NoError(t, app.Save(collection))
}

func installQueueStubs(t *testing.T, stub *queueStub) {
	origEnsure := ensureRunQueueSemaphoreWorkflow
	origEnqueue := enqueueRunTicket
	origQuery := queryRunTicketStatus
	origCancel := cancelRunTicket

	t.Cleanup(func() {
		ensureRunQueueSemaphoreWorkflow = origEnsure
		enqueueRunTicket = origEnqueue
		queryRunTicketStatus = origQuery
		cancelRunTicket = origCancel
	})

	ensureRunQueueSemaphoreWorkflow = func(ctx context.Context, deviceID string) error {
		return nil
	}
	enqueueRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreEnqueueRunRequest,
	) (workflows.MobileDeviceSemaphoreEnqueueRunResponse, error) {
		stub.enqueueRequests = append(stub.enqueueRequests, req)
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunQueued,
			Position: 0,
			LineLen:  1,
		}, nil
	}
	queryRunTicketStatus = func(
		ctx context.Context,
		deviceID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		if stub.cancelled || ticketID == "missing-ticket" {
			return workflows.MobileDeviceSemaphoreRunStatusView{
				TicketID: ticketID,
				Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
			}, nil
		}
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID:          ticketID,
			Status:            workflowengine.MobileDeviceSemaphoreRunQueued,
			Position:          0,
			LineLen:           1,
			LeaderDeviceID:    deviceID,
			RequiredDeviceIDs: []string{deviceID},
		}, nil
	}
	cancelRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreRunCancelRequest,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		stub.cancelled = true
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
		}, nil
	}
}

func TestPipelineQueueEnqueueAndPoll(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	stub := &queueStub{}
	installQueueStubs(t, stub)

	missingRunnerYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n"
	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-1/device-1\n"
	unknownRunnerYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/missing-runner/device-1\n"
	foreignPrivateRunnerYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      device_id: other-org/private-runner/device-1\n"

	scenarios := []tests.ApiScenario{
		{
			Name:   "enqueue requires auth",
			Method: http.MethodPost,
			URL:    "/api/pipeline/queue",
			Body: jsonBody(
				map[string]any{
					"pipeline_identifier": "usera-s-organization/pipeline123",
					"yaml":                validYaml,
				},
			),
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"authentication_required",
			},
			TestAppFactory: setupPipelineQueueApp,
		},
		{
			Name:   "enqueue missing runner selection",
			Method: http.MethodPost,
			URL:    "/api/pipeline/queue",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"yaml":                missingRunnerYaml,
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"device_ids",
				"device_ids are required",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return setupPipelineQueueAppWithPipeline(t, orgID, missingRunnerYaml)
			},
		},
		{
			Name:   "enqueue returns queued response",
			Method: http.MethodPost,
			URL:    "/api/pipeline/queue",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"yaml":                validYaml,
			}),
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"\"status\":\"queued\"",
				"\"device_ids\":[\"usera-s-organization/runner-1/device-1\"]",
				"\"pipeline_url\":\"https://credimi.test/my/pipelines/usera-s-organization/pipeline123\"",
			},
			NotExpectedContent: []string{
				"\"mode\"",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineQueueAppWithPipeline(t, orgID, validYaml)
				app.Settings().Meta.AppURL = "https://credimi.test"
				return app
			},
		},
		{
			Name:   "enqueue rejects foreign private runner",
			Method: http.MethodPost,
			URL:    "/api/pipeline/queue",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"yaml":                foreignPrivateRunnerYaml,
			}),
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"device_id is not accessible",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineQueueAppWithPipeline(t, orgID, foreignPrivateRunnerYaml)
				orgColl, err := app.FindCollectionByNameOrId("organizations")
				require.NoError(t, err)
				otherOrg := core.NewRecord(orgColl)
				otherOrg.Set("name", "Other Org")
				otherOrg.Set("canonified_name", "other-org")
				require.NoError(t, app.Save(otherOrg))
				createPipelineQueueMobileRunner(t, app, otherOrg.Id, "Private Runner", false)
				return app
			},
		},
		{
			Name:   "enqueue rejects missing runner",
			Method: http.MethodPost,
			URL:    "/api/pipeline/queue",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"yaml":                unknownRunnerYaml,
			}),
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				"runner not found",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return setupPipelineQueueAppWithPipeline(t, orgID, unknownRunnerYaml)
			},
		},
		{
			Name:   "poll returns not found",
			Method: http.MethodGet,
			URL:    "/api/pipeline/queue/missing-ticket?device_ids[]=runner-1/device-1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"\"status\":\"not_found\"",
			},
			NotExpectedContent: []string{
				"\"device_ids\"",
				"\"runners\"",
				"\"leader_device_id\"",
				"\"required_device_ids\"",
				"\"error_message\"",
			},
			TestAppFactory: setupPipelineQueueApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPipelineQueueEnqueuePassesQueueLimit(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	stub := &queueStub{}
	installQueueStubs(t, stub)

	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-1/device-1\n"

	scenario := tests.ApiScenario{
		Name:   "enqueue passes org queue limit",
		Method: http.MethodPost,
		URL:    "/api/pipeline/queue",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"yaml":                validYaml,
		}),
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"queued\"",
		},
		NotExpectedContent: []string{
			"\"mode\"",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupPipelineQueueAppWithPipeline(t, orgID, validYaml)
			ensureOrganizationsQueueLimitField(t, app)

			orgRecord, err := app.FindRecordById("organizations", orgID)
			require.NoError(t, err)
			orgRecord.Set("max_pipelines_in_queue", 7)
			require.NoError(t, app.Save(orgRecord))

			return app
		},
	}

	scenario.Test(t)

	require.Len(t, stub.enqueueRequests, 1)
	require.Equal(t, 7, stub.enqueueRequests[0].MaxPipelinesInQueue)
}

func TestPipelineQueueEnqueue_StartsNonRunnerPipeline(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	origStart := startPipelineWorkflow
	t.Cleanup(func() {
		startPipelineWorkflow = origStart
	})
	var capturedMemo map[string]any
	startPipelineWorkflow = func(
		yaml string,
		config map[string]any,
		memo map[string]any,
		pipelineIdentifier string,
	) (workflowengine.WorkflowResult, error) {
		capturedMemo = memo
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-123",
			WorkflowRunID: "run-456",
		}, nil
	}

	nonRunnerYaml := "name: test\nsteps: []\n"
	app := setupPipelineQueueAppWithPipeline(t, orgID, nonRunnerYaml)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/queue",
			jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"yaml":                nonRunnerYaml,
			}),
		)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("content-type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), "\"status\":\"running\"")
		require.Contains(t, rec.Body.String(), "\"workflow_id\":\"wf-123\"")
		require.Contains(t, rec.Body.String(), "\"run_id\":\"run-456\"")
		require.Contains(
			t,
			rec.Body.String(),
			"\"pipeline_url\":\"https://credimi.test/my/pipelines/usera-s-organization/pipeline123\"",
		)
		require.Contains(
			t,
			rec.Body.String(),
			"\"run_url\":\"https://credimi.test/my/tests/runs/wf-123/run-456\"",
		)
		require.NotContains(t, rec.Body.String(), "\"mode\"")
		return nil
	})
	require.NoError(t, serveErr)

	pipelineRecord, err := canonify.Resolve(app, "usera-s-organization/pipeline123")
	require.NoError(t, err)

	results, err := app.FindRecordsByFilter(
		"pipeline_results",
		"pipeline={:pipeline} && owner={:owner}",
		"",
		-1,
		0,
		dbx.Params{
			"pipeline": pipelineRecord.Id,
			"owner":    orgID,
		},
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "wf-123", results[0].GetString("workflow_id"))
	require.Equal(t, "run-456", results[0].GetString("run_id"))
	require.Equal(t, pipelineinternal.RunTypeManual, results[0].GetString("type"))
	require.Equal(t, false, capturedMemo[pipelineinternal.PublishedMemoKey])
}

func TestPipelineQueueStatusReturnsRunURL(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	origQuery := queryRunTicketStatus
	t.Cleanup(func() {
		queryRunTicketStatus = origQuery
	})

	queryRunTicketStatus = func(
		ctx context.Context,
		deviceID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID:          ticketID,
			Status:            workflowengine.MobileDeviceSemaphoreRunRunning,
			WorkflowID:        "wf-123",
			RunID:             "run-456",
			Position:          0,
			LineLen:           1,
			LeaderDeviceID:    deviceID,
			RequiredDeviceIDs: []string{deviceID},
		}, nil
	}

	scenario := tests.ApiScenario{
		Name:   "poll returns run url",
		Method: http.MethodGet,
		URL:    "/api/pipeline/queue/ticket-1?device_ids[]=runner-1/device-1",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"running\"",
			"\"workflow_id\":\"wf-123\"",
			"\"run_id\":\"run-456\"",
			"\"run_url\":\"https://credimi.test/my/tests/runs/wf-123/run-456\"",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupPipelineQueueApp(t)
			app.Settings().Meta.AppURL = "https://credimi.test"
			return app
		},
	}

	scenario.Test(t)
}

func TestPipelineQueueCancel(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	stub := &queueStub{}
	installQueueStubs(t, stub)

	scenarios := []tests.ApiScenario{
		{
			Name:   "cancel queued ticket",
			Method: http.MethodDelete,
			URL:    "/api/pipeline/queue/ticket-cancel?device_ids[]=runner-1/device-1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"\"ticket_id\":\"ticket-cancel\"",
				"\"status\":\"canceled\"",
			},
			NotExpectedContent: []string{
				"\"device_ids\"",
				"\"runners\"",
				"\"leader_device_id\"",
				"\"required_device_ids\"",
				"\"error_message\"",
			},
			TestAppFactory: setupPipelineQueueApp,
		},
		{
			Name:   "poll after cancel returns not found",
			Method: http.MethodGet,
			URL:    "/api/pipeline/queue/ticket-cancel?device_ids[]=runner-1/device-1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"\"status\":\"not_found\"",
			},
			NotExpectedContent: []string{
				"\"device_ids\"",
				"\"runners\"",
				"\"leader_device_id\"",
				"\"required_device_ids\"",
				"\"error_message\"",
			},
			TestAppFactory: setupPipelineQueueApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPipelineQueueCancelDeletesQueuedTempWalletVersion(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	stub := &queueStub{}
	installQueueStubs(t, stub)

	app := setupPipelineQueueApp(t)
	defer app.Cleanup()

	versionRecord := createQueueTempWalletVersion(t, app, orgID, "queued-temp-wallet", "abc123")

	cancelRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreRunCancelRequest,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
			Cleanup: &workflows.MobileDeviceSemaphoreCleanupMetadata{
				TempWalletVersionID:         versionRecord.Id,
				TempWalletVersionIdentifier: "usera-s-organization/queued-temp-wallet/abc123",
			},
		}, nil
	}

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/pipeline/queue/ticket-temp?device_ids[]=runner-1/device-1",
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"status":"canceled"`)
		return nil
	})
	require.NoError(t, serveErr)

	_, err = app.FindRecordById("wallet_versions", versionRecord.Id)
	require.Error(t, err)
}

func TestPipelineQueueCancelKeepsTempWalletVersionForRunningRun(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	stub := &queueStub{}
	installQueueStubs(t, stub)

	app := setupPipelineQueueApp(t)
	defer app.Cleanup()

	versionRecord := createQueueTempWalletVersion(t, app, orgID, "running-temp-wallet", "abc123")

	cancelRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreRunCancelRequest,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID:   req.TicketID,
			Status:     workflowengine.MobileDeviceSemaphoreRunRunning,
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Cleanup: &workflows.MobileDeviceSemaphoreCleanupMetadata{
				TempWalletVersionID:         versionRecord.Id,
				TempWalletVersionIdentifier: "usera-s-organization/running-temp-wallet/abc123",
			},
		}, nil
	}

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/pipeline/queue/ticket-temp?device_ids[]=runner-1/device-1",
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"status":"running"`)
		return nil
	})
	require.NoError(t, serveErr)

	_, err = app.FindRecordById("wallet_versions", versionRecord.Id)
	require.NoError(t, err)
}

func TestPipelineQueueEnqueue_RollbackOnPartialFailure(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	origEnsure := ensureRunQueueSemaphoreWorkflow
	origEnqueue := enqueueRunTicket
	origCancel := cancelRunTicket

	t.Cleanup(func() {
		ensureRunQueueSemaphoreWorkflow = origEnsure
		enqueueRunTicket = origEnqueue
		cancelRunTicket = origCancel
	})

	ensureRunQueueSemaphoreWorkflow = func(ctx context.Context, deviceID string) error {
		return nil
	}

	var ticketID string
	enqueueRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreEnqueueRunRequest,
	) (workflows.MobileDeviceSemaphoreEnqueueRunResponse, error) {
		if ticketID == "" {
			ticketID = req.TicketID
		}
		if deviceID == "usera-s-organization/runner-2/device-1" {
			return workflows.MobileDeviceSemaphoreEnqueueRunResponse{}, errors.New("enqueue failed")
		}
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunQueued,
			Position: 0,
			LineLen:  1,
		}, nil
	}

	type cancelCall struct {
		deviceID string
		ticketID string
	}
	cancelCalls := []cancelCall{}
	cancelRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreRunCancelRequest,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		cancelCalls = append(cancelCalls, cancelCall{
			deviceID: deviceID,
			ticketID: req.TicketID,
		})
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-1/device-1\n  - name: step2\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-2/device-1\n"

	scenario := tests.ApiScenario{
		Name:   "enqueue rollback on partial failure",
		Method: http.MethodPost,
		URL:    "/api/pipeline/queue",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"yaml":                validYaml,
		}),
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedContent: []string{
			"failed to enqueue pipeline run",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return setupPipelineQueueAppWithPipeline(t, orgID, validYaml)
		},
	}

	scenario.Test(t)

	require.NotEmpty(t, ticketID)
	require.Len(t, cancelCalls, 2)
	require.ElementsMatch(
		t,
		[]string{
			"usera-s-organization/runner-1/device-1",
			"usera-s-organization/runner-2/device-1",
		},
		[]string{
			cancelCalls[0].deviceID,
			cancelCalls[1].deviceID,
		},
	)
	for _, call := range cancelCalls {
		require.Equal(t, ticketID, call.ticketID)
	}
}

func createQueueTempWalletVersion(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	walletName string,
	tag string,
) *core.Record {
	t.Helper()

	identifier := createWalletAPKVersion(t, app, orgID, walletName, tag)
	walletRecord := createWalletAPKWallet(t, app, orgID, walletName)
	versionRecord, err := app.FindFirstRecordByFilter(
		"wallet_versions",
		"wallet = {:wallet} && owner = {:owner} && canonified_tag = {:tag}",
		dbx.Params{
			"wallet": walletRecord.Id,
			"owner":  orgID,
			"tag":    tag,
		},
	)
	require.NoError(t, err, "created wallet version %s should be queryable", identifier)
	return versionRecord
}

func TestPipelineQueueHelpers(t *testing.T) {
	t.Run("parseDeviceIDs prefers array param", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/?device_ids[]=runner-1/device-1&device_ids[]=runner-2/device-1",
			nil,
		)
		req.URL.RawQuery = "device_ids[]=runner-1/device-1&device_ids[]=runner-2/device-1&device_ids=runner-3/device-1"
		require.Equal(t, []string{"runner-1/device-1", "runner-2/device-1"}, parseDeviceIDs(req))
	})

	t.Run("parseDeviceIDs falls back to singular param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?device_ids=runner-3/device-1", nil)
		require.Equal(t, []string{"runner-3/device-1"}, parseDeviceIDs(req))
	})

	t.Run("normalizeDeviceIDs dedupes and trims", func(t *testing.T) {
		values := []string{" runner-2 , runner-1", "runner-2", "  ", "runner-3"}
		require.Equal(t, []string{"runner-1", "runner-2", "runner-3"}, normalizeDeviceIDs(values))
	})

	t.Run("normalizeDeviceIDs trims leading slash", func(t *testing.T) {
		values := []string{" /tenant/runner-2 , /tenant/runner-1", "tenant/runner-2"}
		require.Equal(t, []string{"tenant/runner-1", "tenant/runner-2"}, normalizeDeviceIDs(values))
	})

	t.Run("runTicketNotFoundView sets status", func(t *testing.T) {
		view := runTicketNotFoundView("ticket-123")
		require.Equal(t, "ticket-123", view.TicketID)
		require.Equal(t, workflowengine.MobileDeviceSemaphoreRunNotFound, view.Status)
	})

	t.Run("aggregateRunQueueStatus chooses highest priority", func(t *testing.T) {
		statuses := []pipelineQueueRunnerStatus{
			{
				DeviceID: "runner-1",
				Status:   workflowengine.MobileDeviceSemaphoreRunQueued,
				Position: 2,
				LineLen:  3,
			},
			{
				DeviceID:   "runner-2",
				Status:     workflowengine.MobileDeviceSemaphoreRunRunning,
				Position:   1,
				LineLen:    2,
				WorkflowID: "wf-1",
				RunID:      "run-1",
			},
		}

		status, position, lineLen, workflowID, runID, errMsg := aggregateRunQueueStatus(statuses)
		require.Equal(t, workflowengine.MobileDeviceSemaphoreRunRunning, status)
		require.Equal(t, 2, position)
		require.Equal(t, 3, lineLen)
		require.Equal(t, "wf-1", workflowID)
		require.Equal(t, "run-1", runID)
		require.Empty(t, errMsg)
	})

	t.Run("buildQueueStatusResponse keeps workflow details", func(t *testing.T) {
		response := buildQueueStatusResponse("ticket-1", []pipelineQueueRunnerStatus{
			{
				DeviceID:   "runner-1",
				Status:     workflowengine.MobileDeviceSemaphoreRunRunning,
				Position:   1,
				LineLen:    2,
				WorkflowID: "wf-99",
				RunID:      "run-99",
			},
		})
		require.Equal(t, "ticket-1", response.TicketID)
		require.Equal(t, workflowengine.MobileDeviceSemaphoreRunRunning, response.Status)
		require.NotNil(t, response.Position)
		require.NotNil(t, response.LineLen)
		require.Equal(t, "wf-99", response.WorkflowID)
		require.Equal(t, "run-99", response.RunID)
	})

	t.Run("buildQueueEnqueueResponse maps failed status", func(t *testing.T) {
		response := buildQueueEnqueueResponse(
			"ticket-1",
			time.Now(),
			[]string{"runner-1"},
			workflowengine.MobileDeviceSemaphoreRunFailed,
			0,
			1,
			"",
			"",
			"queue limit exceeded",
		)
		require.Equal(t, workflowengine.MobileDeviceSemaphoreRunFailed, response.Status)
		require.Equal(t, "queue limit exceeded", response.ErrorMessage)
		require.Empty(t, response.TicketID)
	})

	t.Run("buildQueueEnqueueResponse maps running status", func(t *testing.T) {
		response := buildQueueEnqueueResponse(
			"ticket-1",
			time.Now(),
			[]string{"runner-1"},
			workflowengine.MobileDeviceSemaphoreRunRunning,
			0,
			1,
			"wf-1",
			"run-1",
			"",
		)
		require.Equal(t, workflowengine.MobileDeviceSemaphoreRunRunning, response.Status)
		require.Equal(t, "wf-1", response.WorkflowID)
		require.Equal(t, "run-1", response.RunID)
		require.Empty(t, response.TicketID)
	})

	t.Run("canceledQueueCleanupMetadata rejects started status", func(t *testing.T) {
		cleanup := &workflows.MobileDeviceSemaphoreCleanupMetadata{
			TempWalletVersionID: "wallet-version-1",
		}
		metadata, ok := canceledQueueCleanupMetadata([]pipelineQueueRunnerStatus{
			{
				DeviceID: "runner-1",
				Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
				Cleanup:  cleanup,
			},
			{
				DeviceID:   "runner-2",
				Status:     workflowengine.MobileDeviceSemaphoreRunRunning,
				WorkflowID: "wf-1",
				RunID:      "run-1",
				Cleanup:    cleanup,
			},
		})
		require.False(t, ok)
		require.Nil(t, metadata)
	})
}

func TestDeleteTempWalletVersionForOwner(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	t.Run("missing record is no-op", func(t *testing.T) {
		app := setupPipelineQueueApp(t)
		defer app.Cleanup()

		apiErr := deleteTempWalletVersionForOwner(app, "missingrecord12", orgID)
		require.Nil(t, apiErr)
	})

	t.Run("owner mismatch is forbidden", func(t *testing.T) {
		app := setupPipelineQueueApp(t)
		defer app.Cleanup()

		orgColl, err := app.FindCollectionByNameOrId("organizations")
		require.NoError(t, err)
		otherOrg := core.NewRecord(orgColl)
		otherOrg.Set("name", "Other Org")
		otherOrg.Set("canonified_name", "other-org")
		require.NoError(t, app.Save(otherOrg))

		versionRecord := createQueueTempWalletVersion(
			t,
			app,
			otherOrg.Id,
			"other-temp-wallet",
			"abc123",
		)

		apiErr := deleteTempWalletVersionForOwner(app, versionRecord.Id, orgID)
		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusForbidden, apiErr.Code)

		_, err = app.FindRecordById("wallet_versions", versionRecord.Id)
		require.NoError(t, err)
	})
}

func TestPipelineQueueEnqueue_QueueLimitExceededRollsBack(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	origEnsure := ensureRunQueueSemaphoreWorkflow
	origEnqueue := enqueueRunTicket
	origCancel := cancelRunTicket

	t.Cleanup(func() {
		ensureRunQueueSemaphoreWorkflow = origEnsure
		enqueueRunTicket = origEnqueue
		cancelRunTicket = origCancel
	})

	ensureRunQueueSemaphoreWorkflow = func(ctx context.Context, deviceID string) error {
		return nil
	}

	var ticketID string
	enqueueRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreEnqueueRunRequest,
	) (workflows.MobileDeviceSemaphoreEnqueueRunResponse, error) {
		if ticketID == "" {
			ticketID = req.TicketID
		}
		if deviceID == "usera-s-organization/runner-2/device-1" {
			return workflows.MobileDeviceSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
				"queue limit exceeded for runner runner-2: 1 of 1",
				workflows.MobileDeviceSemaphoreErrQueueLimitExceeded,
			)
		}
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunQueued,
			Position: 0,
			LineLen:  1,
		}, nil
	}

	type cancelCall struct {
		deviceID string
		ticketID string
	}
	cancelCalls := []cancelCall{}
	cancelRunTicket = func(
		ctx context.Context,
		deviceID string,
		req workflows.MobileDeviceSemaphoreRunCancelRequest,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		cancelCalls = append(cancelCalls, cancelCall{
			deviceID: deviceID,
			ticketID: req.TicketID,
		})
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-1/device-1\n  - name: step2\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-2/device-1\n"

	scenario := tests.ApiScenario{
		Name:   "enqueue queue limit rollback",
		Method: http.MethodPost,
		URL:    "/api/pipeline/queue",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Body: jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"yaml":                validYaml,
		}),
		ExpectedStatus: http.StatusConflict,
		ExpectedContent: []string{
			"queue limit exceeded",
			"runner-2",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return setupPipelineQueueAppWithPipeline(t, orgID, validYaml)
		},
	}

	scenario.Test(t)

	require.NotEmpty(t, ticketID)
	require.Len(t, cancelCalls, 2)
	require.ElementsMatch(
		t,
		[]string{
			"usera-s-organization/runner-1/device-1",
			"usera-s-organization/runner-2/device-1",
		},
		[]string{
			cancelCalls[0].deviceID,
			cancelCalls[1].deviceID,
		},
	)
	for _, call := range cancelCalls {
		require.Equal(t, ticketID, call.ticketID)
	}
}

func TestPipelineQueueStatus_MultiRunnerDoesNot404WhenAnyRunnerFound(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	origQuery := queryRunTicketStatus
	t.Cleanup(func() {
		queryRunTicketStatus = origQuery
	})

	queryRunTicketStatus = func(
		ctx context.Context,
		deviceID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		if deviceID == "runner-1/device-1" {
			return workflows.MobileDeviceSemaphoreRunStatusView{
				TicketID:          ticketID,
				Status:            workflowengine.MobileDeviceSemaphoreRunFailed,
				ErrorMessage:      "boom",
				LeaderDeviceID:    "runner-1",
				RequiredDeviceIDs: []string{"runner-1", "runner-2"},
			}, nil
		}
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	scenario := tests.ApiScenario{
		Name:   "multi-runner status returns failure when any runner found",
		Method: http.MethodGet,
		URL:    "/api/pipeline/queue/ticket-1?device_ids[]=runner-1/device-1&device_ids[]=runner-2/device-1",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"failed\"",
		},
		NotExpectedContent: []string{
			"\"device_ids\"",
			"\"runners\"",
			"\"leader_device_id\"",
			"\"required_device_ids\"",
			"\"error_message\"",
		},
		TestAppFactory: setupPipelineQueueApp,
	}

	scenario.Test(t)
}

func TestPipelineQueueStatus_MultiRunnerIgnoresMissingRunnerWorkflow(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	origQuery := queryRunTicketStatus
	t.Cleanup(func() {
		queryRunTicketStatus = origQuery
	})

	queryRunTicketStatus = func(
		ctx context.Context,
		deviceID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		if deviceID == "runner-2/device-1" {
			return workflows.MobileDeviceSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID:          ticketID,
			Status:            workflowengine.MobileDeviceSemaphoreRunQueued,
			Position:          1,
			LineLen:           2,
			LeaderDeviceID:    "runner-1",
			RequiredDeviceIDs: []string{"runner-1", "runner-2"},
		}, nil
	}

	scenario := tests.ApiScenario{
		Name:   "multi-runner status ignores missing workflow",
		Method: http.MethodGet,
		URL:    "/api/pipeline/queue/ticket-2?device_ids[]=runner-1/device-1&device_ids[]=runner-2/device-1",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"queued\"",
		},
		NotExpectedContent: []string{
			"\"device_ids\"",
			"\"runners\"",
			"\"leader_device_id\"",
			"\"required_device_ids\"",
			"\"error_message\"",
		},
		TestAppFactory: setupPipelineQueueApp,
	}

	scenario.Test(t)
}

func TestPipelineQueueStatus_MultiRunnerAllMissingReturnsNotFound(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	origQuery := queryRunTicketStatus
	t.Cleanup(func() {
		queryRunTicketStatus = origQuery
	})

	queryRunTicketStatus = func(
		ctx context.Context,
		deviceID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
		return workflows.MobileDeviceSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	scenario := tests.ApiScenario{
		Name:   "multi-runner status returns not found when all missing",
		Method: http.MethodGet,
		URL:    "/api/pipeline/queue/ticket-3?device_ids[]=runner-1/device-1&device_ids[]=runner-2/device-1",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"not_found\"",
		},
		NotExpectedContent: []string{
			"\"device_ids\"",
			"\"runners\"",
			"\"leader_device_id\"",
			"\"required_device_ids\"",
			"\"error_message\"",
		},
		TestAppFactory: setupPipelineQueueApp,
	}

	scenario.Test(t)
}

type queueErrorEncodedValue struct{}

func (e queueErrorEncodedValue) HasValue() bool { return true }

func (e queueErrorEncodedValue) Get(interface{}) error { return errors.New("decode failed") }

func TestPipelineQueueTemporalHelpers(t *testing.T) {
	t.Run("ensureRunQueueSemaphoreWorkflowTemporal accepts already started", func(t *testing.T) {
		origClient := queueTemporalClient
		t.Cleanup(func() { queueTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"ExecuteWorkflow",
				mock.Anything,
				mock.Anything,
				workflows.MobileDeviceSemaphoreWorkflowName,
				mock.Anything,
			).
			Return(nil, &serviceerror.WorkflowExecutionAlreadyStarted{})

		queueTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		err := ensureRunQueueSemaphoreWorkflowTemporal(context.Background(), "runner-1")
		require.NoError(t, err)
	})

	t.Run("ensureRunQueueSemaphoreWorkflowTemporal bubbles errors", func(t *testing.T) {
		origClient := queueTemporalClient
		t.Cleanup(func() { queueTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"ExecuteWorkflow",
				mock.Anything,
				mock.Anything,
				workflows.MobileDeviceSemaphoreWorkflowName,
				mock.Anything,
			).
			Return(nil, errors.New("boom"))

		queueTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		err := ensureRunQueueSemaphoreWorkflowTemporal(context.Background(), "runner-1")
		require.Error(t, err)
	})

	t.Run("enqueueRunTicketTemporal returns response", func(t *testing.T) {
		origClient := queueTemporalClient
		t.Cleanup(func() { queueTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		handle := temporalmocks.NewWorkflowUpdateHandle(t)
		handle.
			On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				switch out := args.Get(1).(type) {
				case *workflows.MobileDeviceSemaphoreEnqueueRunResponse:
					*out = workflows.MobileDeviceSemaphoreEnqueueRunResponse{
						TicketID: "ticket-1",
						Status:   workflowengine.MobileDeviceSemaphoreRunQueued,
						Position: 1,
						LineLen:  2,
					}
				}
			}).
			Return(nil)

		mockClient.
			On("UpdateWorkflow", mock.Anything, mock.Anything).
			Return(handle, nil)

		queueTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		resp, err := enqueueRunTicketTemporal(
			context.Background(),
			"runner-1",
			workflows.MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "ticket-1"},
		)
		require.NoError(t, err)
		require.Equal(t, "ticket-1", resp.TicketID)
	})

	t.Run("queryRunTicketStatusTemporal handles not found", func(t *testing.T) {
		origClient := queueTemporalClient
		t.Cleanup(func() { queueTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"QueryWorkflow",
				mock.Anything,
				workflows.MobileDeviceSemaphoreWorkflowID("runner-1"),
				"",
				workflows.MobileDeviceSemaphoreRunStatusQuery,
				"org-1",
				"ticket-1",
			).
			Return(converter.EncodedValue(nil), &serviceerror.NotFound{Message: "missing"})

		queueTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		_, err := queryRunTicketStatusTemporal(
			context.Background(),
			"runner-1",
			"org-1",
			"ticket-1",
		)
		require.ErrorIs(t, err, errRunTicketNotFound)
	})

	t.Run("queryRunTicketStatusTemporal bubbles decode error", func(t *testing.T) {
		origClient := queueTemporalClient
		t.Cleanup(func() { queueTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"QueryWorkflow",
				mock.Anything,
				workflows.MobileDeviceSemaphoreWorkflowID("runner-2"),
				"",
				workflows.MobileDeviceSemaphoreRunStatusQuery,
				"org-1",
				"ticket-2",
			).
			Return(converter.EncodedValue(queueErrorEncodedValue{}), nil)

		queueTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		_, err := queryRunTicketStatusTemporal(
			context.Background(),
			"runner-2",
			"org-1",
			"ticket-2",
		)
		require.ErrorContains(t, err, "decode failed")
	})

	t.Run("cancelRunTicketTemporal handles not found", func(t *testing.T) {
		origClient := queueTemporalClient
		t.Cleanup(func() { queueTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On("UpdateWorkflow", mock.Anything, mock.Anything).
			Return(nil, &serviceerror.NotFound{Message: "missing"})

		queueTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		_, err := cancelRunTicketTemporal(
			context.Background(),
			"runner-1",
			workflows.MobileDeviceSemaphoreRunCancelRequest{TicketID: "ticket-1"},
		)
		require.ErrorIs(t, err, errRunTicketNotFound)
	})

	t.Run("cancelRunTicketTemporal returns status", func(t *testing.T) {
		origClient := queueTemporalClient
		t.Cleanup(func() { queueTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		handle := temporalmocks.NewWorkflowUpdateHandle(t)
		handle.
			On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				switch out := args.Get(1).(type) {
				case *workflows.MobileDeviceSemaphoreRunStatusView:
					*out = workflows.MobileDeviceSemaphoreRunStatusView{
						TicketID: "ticket-3",
						Status:   workflowengine.MobileDeviceSemaphoreRunCanceled,
					}
				}
			}).
			Return(nil)

		mockClient.
			On("UpdateWorkflow", mock.Anything, mock.Anything).
			Return(handle, nil)

		queueTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		resp, err := cancelRunTicketTemporal(
			context.Background(),
			"runner-3",
			workflows.MobileDeviceSemaphoreRunCancelRequest{TicketID: "ticket-3"},
		)
		require.NoError(t, err)
		require.Equal(t, workflowengine.MobileDeviceSemaphoreRunCanceled, resp.Status)
	})

	t.Run("runQueueUpdateID formats deterministically", func(t *testing.T) {
		require.Equal(
			t,
			"enqueue/tenant/runner-1/ticket-1",
			runQueueUpdateID("enqueue", "/tenant/runner-1", "ticket-1"),
		)
	})
}
