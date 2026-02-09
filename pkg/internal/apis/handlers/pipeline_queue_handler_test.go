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

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

type queueStub struct {
	cancelled       bool
	enqueueRequests []workflows.MobileRunnerSemaphoreEnqueueRunRequest
}

func setupPipelineQueueApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

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

	return app
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

	ensureRunQueueSemaphoreWorkflow = func(ctx context.Context, runnerID string) error {
		return nil
	}
	enqueueRunTicket = func(
		ctx context.Context,
		runnerID string,
		req workflows.MobileRunnerSemaphoreEnqueueRunRequest,
	) (workflows.MobileRunnerSemaphoreEnqueueRunResponse, error) {
		stub.enqueueRequests = append(stub.enqueueRequests, req)
		return workflows.MobileRunnerSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunQueued,
			Position: 0,
			LineLen:  1,
		}, nil
	}
	queryRunTicketStatus = func(
		ctx context.Context,
		runnerID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
		if stub.cancelled || ticketID == "missing-ticket" {
			return workflows.MobileRunnerSemaphoreRunStatusView{
				TicketID: ticketID,
				Status:   workflowengine.MobileRunnerSemaphoreRunNotFound,
			}, nil
		}
		return workflows.MobileRunnerSemaphoreRunStatusView{
			TicketID:          ticketID,
			Status:            workflowengine.MobileRunnerSemaphoreRunQueued,
			Position:          0,
			LineLen:           1,
			LeaderRunnerID:    runnerID,
			RequiredRunnerIDs: []string{runnerID},
		}, nil
	}
	cancelRunTicket = func(
		ctx context.Context,
		runnerID string,
		req workflows.MobileRunnerSemaphoreRunCancelRequest,
	) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
		stub.cancelled = true
		return workflows.MobileRunnerSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunNotFound,
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
	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      runner_id: runner-1\n"

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
			ExpectedStatus: http.StatusInternalServerError,
			ExpectedContent: []string{
				"valid record authorization token",
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
				"runner_ids",
				"runner_ids are required",
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
				"\"mode\":\"queued\"",
				"\"runner_ids\":[\"runner-1\"]",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return setupPipelineQueueAppWithPipeline(t, orgID, validYaml)
			},
		},
		{
			Name:   "poll returns not found",
			Method: http.MethodGet,
			URL:    "/api/pipeline/queue/missing-ticket?runner_ids[]=runner-1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"\"status\":\"not_found\"",
			},
			NotExpectedContent: []string{
				"\"runner_ids\"",
				"\"runners\"",
				"\"leader_runner_id\"",
				"\"required_runner_ids\"",
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

	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      runner_id: runner-1\n"

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
			"\"mode\":\"queued\"",
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
	startPipelineWorkflow = func(
		yaml string,
		config map[string]any,
		memo map[string]any,
	) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-123",
			WorkflowRunID: "run-456",
		}, nil
	}

	nonRunnerYaml := "name: test\nsteps: []\n"
	app := setupPipelineQueueAppWithPipeline(t, orgID, nonRunnerYaml)
	defer app.Cleanup()

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
		require.Contains(t, rec.Body.String(), "\"mode\":\"started\"")
		require.Contains(t, rec.Body.String(), "\"workflow_id\":\"wf-123\"")
		require.Contains(t, rec.Body.String(), "\"run_id\":\"run-456\"")
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
			URL:    "/api/pipeline/queue/ticket-cancel?runner_ids[]=runner-1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"\"ticket_id\":\"ticket-cancel\"",
				"\"status\":\"not_found\"",
			},
			NotExpectedContent: []string{
				"\"runner_ids\"",
				"\"runners\"",
				"\"leader_runner_id\"",
				"\"required_runner_ids\"",
				"\"error_message\"",
			},
			TestAppFactory: setupPipelineQueueApp,
		},
		{
			Name:   "poll after cancel returns not found",
			Method: http.MethodGet,
			URL:    "/api/pipeline/queue/ticket-cancel?runner_ids[]=runner-1",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"\"status\":\"not_found\"",
			},
			NotExpectedContent: []string{
				"\"runner_ids\"",
				"\"runners\"",
				"\"leader_runner_id\"",
				"\"required_runner_ids\"",
				"\"error_message\"",
			},
			TestAppFactory: setupPipelineQueueApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
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

	ensureRunQueueSemaphoreWorkflow = func(ctx context.Context, runnerID string) error {
		return nil
	}

	var ticketID string
	enqueueRunTicket = func(
		ctx context.Context,
		runnerID string,
		req workflows.MobileRunnerSemaphoreEnqueueRunRequest,
	) (workflows.MobileRunnerSemaphoreEnqueueRunResponse, error) {
		if ticketID == "" {
			ticketID = req.TicketID
		}
		if runnerID == "runner-2" {
			return workflows.MobileRunnerSemaphoreEnqueueRunResponse{}, errors.New("enqueue failed")
		}
		return workflows.MobileRunnerSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunQueued,
			Position: 0,
			LineLen:  1,
		}, nil
	}

	type cancelCall struct {
		runnerID string
		ticketID string
	}
	cancelCalls := []cancelCall{}
	cancelRunTicket = func(
		ctx context.Context,
		runnerID string,
		req workflows.MobileRunnerSemaphoreRunCancelRequest,
	) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
		cancelCalls = append(cancelCalls, cancelCall{
			runnerID: runnerID,
			ticketID: req.TicketID,
		})
		return workflows.MobileRunnerSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunNotFound,
		}, nil
	}

	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      runner_id: runner-1\n  - name: step2\n    use: mobile-automation\n    with:\n      runner_id: runner-2\n"

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
	require.ElementsMatch(t, []string{"runner-1", "runner-2"}, []string{
		cancelCalls[0].runnerID,
		cancelCalls[1].runnerID,
	})
	for _, call := range cancelCalls {
		require.Equal(t, ticketID, call.ticketID)
	}
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

	ensureRunQueueSemaphoreWorkflow = func(ctx context.Context, runnerID string) error {
		return nil
	}

	var ticketID string
	enqueueRunTicket = func(
		ctx context.Context,
		runnerID string,
		req workflows.MobileRunnerSemaphoreEnqueueRunRequest,
	) (workflows.MobileRunnerSemaphoreEnqueueRunResponse, error) {
		if ticketID == "" {
			ticketID = req.TicketID
		}
		if runnerID == "runner-2" {
			return workflows.MobileRunnerSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
				"queue limit exceeded for runner runner-2: 1 of 1",
				workflows.MobileRunnerSemaphoreErrQueueLimitExceeded,
			)
		}
		return workflows.MobileRunnerSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunQueued,
			Position: 0,
			LineLen:  1,
		}, nil
	}

	type cancelCall struct {
		runnerID string
		ticketID string
	}
	cancelCalls := []cancelCall{}
	cancelRunTicket = func(
		ctx context.Context,
		runnerID string,
		req workflows.MobileRunnerSemaphoreRunCancelRequest,
	) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
		cancelCalls = append(cancelCalls, cancelCall{
			runnerID: runnerID,
			ticketID: req.TicketID,
		})
		return workflows.MobileRunnerSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunNotFound,
		}, nil
	}

	validYaml := "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      runner_id: runner-1\n  - name: step2\n    use: mobile-automation\n    with:\n      runner_id: runner-2\n"

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
	require.ElementsMatch(t, []string{"runner-1", "runner-2"}, []string{
		cancelCalls[0].runnerID,
		cancelCalls[1].runnerID,
	})
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
		runnerID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
		if runnerID == "runner-1" {
			return workflows.MobileRunnerSemaphoreRunStatusView{
				TicketID:          ticketID,
				Status:            workflowengine.MobileRunnerSemaphoreRunFailed,
				ErrorMessage:      "boom",
				LeaderRunnerID:    "runner-1",
				RequiredRunnerIDs: []string{"runner-1", "runner-2"},
			}, nil
		}
		return workflows.MobileRunnerSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunNotFound,
		}, nil
	}

	scenario := tests.ApiScenario{
		Name:   "multi-runner status returns failure when any runner found",
		Method: http.MethodGet,
		URL:    "/api/pipeline/queue/ticket-1?runner_ids[]=runner-1&runner_ids[]=runner-2",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"failed\"",
		},
		NotExpectedContent: []string{
			"\"runner_ids\"",
			"\"runners\"",
			"\"leader_runner_id\"",
			"\"required_runner_ids\"",
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
		runnerID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
		if runnerID == "runner-2" {
			return workflows.MobileRunnerSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		return workflows.MobileRunnerSemaphoreRunStatusView{
			TicketID:          ticketID,
			Status:            workflowengine.MobileRunnerSemaphoreRunQueued,
			Position:          1,
			LineLen:           2,
			LeaderRunnerID:    "runner-1",
			RequiredRunnerIDs: []string{"runner-1", "runner-2"},
		}, nil
	}

	scenario := tests.ApiScenario{
		Name:   "multi-runner status ignores missing workflow",
		Method: http.MethodGet,
		URL:    "/api/pipeline/queue/ticket-2?runner_ids[]=runner-1&runner_ids[]=runner-2",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"queued\"",
		},
		NotExpectedContent: []string{
			"\"runner_ids\"",
			"\"runners\"",
			"\"leader_runner_id\"",
			"\"required_runner_ids\"",
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
		runnerID string,
		ownerNamespace string,
		ticketID string,
	) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
		return workflows.MobileRunnerSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   workflowengine.MobileRunnerSemaphoreRunNotFound,
		}, nil
	}

	scenario := tests.ApiScenario{
		Name:   "multi-runner status returns not found when all missing",
		Method: http.MethodGet,
		URL:    "/api/pipeline/queue/ticket-3?runner_ids[]=runner-1&runner_ids[]=runner-2",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"status\":\"not_found\"",
		},
		NotExpectedContent: []string{
			"\"runner_ids\"",
			"\"runners\"",
			"\"leader_runner_id\"",
			"\"required_runner_ids\"",
			"\"error_message\"",
		},
		TestAppFactory: setupPipelineQueueApp,
	}

	scenario.Test(t)
}
