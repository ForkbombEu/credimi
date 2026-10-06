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
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
)

const covAPIQueueDeviceYAML = "name: test\nsteps:\n  - name: step1\n    use: mobile-automation\n    with:\n      device_id: usera-s-organization/runner-1/device-1\n"

func covAPIQueueEvent(
	app core.App,
	auth *core.Record,
	req *http.Request,
) (*core.RequestEvent, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	return &core.RequestEvent{
		App:   app,
		Auth:  auth,
		Event: router.Event{Request: req, Response: rec},
	}, rec
}

func covAPIUserA(t *testing.T, app core.App) (*core.Record, *core.Record) {
	t.Helper()
	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	org, err := app.FindFirstRecordByData(
		"organizations",
		"canonified_name",
		"usera-s-organization",
	)
	require.NoError(t, err)
	return auth, org
}

func covAPIOtherOrg(t *testing.T, app core.App) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	org := core.NewRecord(coll)
	org.Set("name", "Other Org")
	org.Set("canonified_name", "other-org")
	require.NoError(t, app.Save(org))
	return org
}

func TestHandlePipelineQueueEnqueueValidatesRequest(t *testing.T) {
	app := setupPipelineQueueApp(t)
	defer app.Cleanup()
	auth, _ := covAPIUserA(t, app)

	scenarios := []struct {
		name   string
		auth   *core.Record
		input  PipelineQueueInput
		code   int
		reason string
	}{
		{
			name:   "unauthenticated",
			input:  PipelineQueueInput{PipelineIdentifier: "p", YAML: "y"},
			code:   http.StatusUnauthorized,
			reason: "authentication required",
		},
		{
			name:   "blank pipeline identifier",
			auth:   auth,
			input:  PipelineQueueInput{PipelineIdentifier: "  ", YAML: "y"},
			code:   http.StatusBadRequest,
			reason: "pipeline_identifier is required",
		},
		{
			name:   "blank yaml",
			auth:   auth,
			input:  PipelineQueueInput{PipelineIdentifier: "usera-s-organization/p", YAML: " \n"},
			code:   http.StatusBadRequest,
			reason: "yaml is required",
		},
		{
			name: "unknown pipeline",
			auth: auth,
			input: PipelineQueueInput{
				PipelineIdentifier: "usera-s-organization/missing",
				YAML:               "y",
			},
			code:   http.StatusNotFound,
			reason: "pipeline not found",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/pipeline/queue", nil)
			req = req.WithContext(
				context.WithValue(req.Context(), middlewares.ValidatedInputKey, s.input),
			)
			e, _ := covAPIQueueEvent(app, s.auth, req)
			covAPIRequireAPIError(t, HandlePipelineQueueEnqueue()(e), s.code, s.reason)
		})
	}
}

func TestEnqueuePipelineRunFailures(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []struct {
		name   string
		yaml   string
		setup  func(t *testing.T, app *tests.TestApp)
		noOrg  bool
		code   int
		reason string
	}{
		{
			name:   "organization without namespace",
			yaml:   covAPIQueueDeviceYAML,
			noOrg:  true,
			code:   http.StatusInternalServerError,
			reason: "unable to get user organization canonified name",
		},
		{
			name:   "unparseable yaml",
			yaml:   "steps: [",
			code:   http.StatusBadRequest,
			reason: "failed to parse pipeline yaml",
		},
		{
			name: "semaphore cannot be ensured",
			yaml: covAPIQueueDeviceYAML,
			setup: func(t *testing.T, _ *tests.TestApp) {
				installQueueStubs(t, &queueStub{})
				ensureRunQueueSemaphoreWorkflow = func(context.Context, string) error {
					return errors.New("temporal down")
				}
			},
			code:   http.StatusInternalServerError,
			reason: "failed to ensure runner semaphore",
		},
		{
			name: "direct start fails",
			yaml: "name: test\nsteps: []\n",
			setup: func(t *testing.T, _ *tests.TestApp) {
				orig := startPipelineWorkflow
				t.Cleanup(func() { startPipelineWorkflow = orig })
				startPipelineWorkflow = func(string, map[string]any, map[string]any, string) (workflowengine.WorkflowResult, error) {
					return workflowengine.WorkflowResult{}, errors.New("temporal down")
				}
			},
			code:   http.StatusInternalServerError,
			reason: "failed to start workflow",
		},
		{
			name: "pipeline result cannot be saved",
			yaml: "name: test\nsteps: []\n",
			setup: func(t *testing.T, app *tests.TestApp) {
				orig := startPipelineWorkflow
				t.Cleanup(func() { startPipelineWorkflow = orig })
				startPipelineWorkflow = func(string, map[string]any, map[string]any, string) (workflowengine.WorkflowResult, error) {
					return workflowengine.WorkflowResult{
						WorkflowID:    "wf-1",
						WorkflowRunID: "run-1",
					}, nil
				}
				app.OnRecordCreate("pipeline_results").BindFunc(func(*core.RecordEvent) error {
					return errors.New("boom")
				})
			},
			code:   http.StatusInternalServerError,
			reason: "failed to save pipeline record",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			app := setupPipelineQueueAppWithPipeline(t, orgID, s.yaml)
			defer app.Cleanup()
			if s.setup != nil {
				s.setup(t, app)
			}
			pipelineRecord, err := canonify.Resolve(app, "usera-s-organization/pipeline123")
			require.NoError(t, err)
			org, err := app.FindRecordById("organizations", orgID)
			require.NoError(t, err)
			if s.noOrg {
				org.Set("canonified_name", "")
			}

			req := httptest.NewRequest(http.MethodPost, "/api/pipeline/queue", nil)
			e, _ := covAPIQueueEvent(app, nil, req)
			_, apiErr := enqueuePipelineRun(e, pipelineQueueRunContext{
				pipelineRecord:     pipelineRecord,
				pipelineIdentifier: "usera-s-organization/pipeline123",
				organizationRecord: org,
				userID:             "user-1",
				yaml:               s.yaml,
			})
			require.NotNil(t, apiErr)
			assert.Equal(t, s.code, apiErr.Code)
			assert.Equal(t, s.reason, apiErr.Reason)
		})
	}
}

func TestPipelineQueueStatusAndCancelValidateRequest(t *testing.T) {
	app := setupPipelineQueueApp(t)
	defer app.Cleanup()
	auth, _ := covAPIUserA(t, app)

	handlers := map[string]func() func(*core.RequestEvent) error{
		"status": HandlePipelineQueueStatus,
		"cancel": HandlePipelineQueueCancel,
	}
	scenarios := []struct {
		name   string
		auth   *core.Record
		ticket string
		query  string
		code   int
		reason string
	}{
		{
			name:   "unauthenticated",
			ticket: "t-1",
			query:  "?device_ids=d",
			code:   http.StatusUnauthorized,
			reason: "authentication required",
		},
		{
			name:   "missing ticket",
			auth:   auth,
			query:  "?device_ids=d",
			code:   http.StatusBadRequest,
			reason: "ticket is required",
		},
		{
			name:   "missing devices",
			auth:   auth,
			ticket: "t-1",
			code:   http.StatusBadRequest,
			reason: "device_ids are required",
		},
	}

	for handlerName, handler := range handlers {
		for _, s := range scenarios {
			t.Run(handlerName+"/"+s.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/api/pipeline/queue/x"+s.query, nil)
				req.SetPathValue("ticket", s.ticket)
				e, _ := covAPIQueueEvent(app, s.auth, req)
				covAPIRequireAPIError(t, handler()(e), s.code, s.reason)
			})
		}
	}
}

func TestPipelineQueueStatusAndCancelSemaphoreOutcomes(t *testing.T) {
	app := setupPipelineQueueApp(t)
	defer app.Cleanup()
	auth, _ := covAPIUserA(t, app)

	t.Run("status query failure", func(t *testing.T) {
		installQueueStubs(t, &queueStub{})
		queryRunTicketStatus = func(context.Context, string, string, string) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
			return workflows.MobileDeviceSemaphoreRunStatusView{}, errors.New("temporal down")
		}
		req := httptest.NewRequest(http.MethodGet, "/api/pipeline/queue/t-1?device_ids=d-1", nil)
		req.SetPathValue("ticket", "t-1")
		e, _ := covAPIQueueEvent(app, auth, req)
		covAPIRequireAPIError(
			t,
			HandlePipelineQueueStatus()(e),
			http.StatusInternalServerError,
			"failed to query ticket status",
		)
	})

	t.Run("cancel failure", func(t *testing.T) {
		installQueueStubs(t, &queueStub{})
		cancelRunTicket = func(context.Context, string, workflows.MobileDeviceSemaphoreRunCancelRequest) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
			return workflows.MobileDeviceSemaphoreRunStatusView{}, errors.New("temporal down")
		}
		req := httptest.NewRequest(http.MethodDelete, "/api/pipeline/queue/t-1?device_ids=d-1", nil)
		req.SetPathValue("ticket", "t-1")
		e, _ := covAPIQueueEvent(app, auth, req)
		covAPIRequireAPIError(
			t,
			HandlePipelineQueueCancel()(e),
			http.StatusInternalServerError,
			"failed to cancel ticket",
		)
	})

	t.Run("cancel of unknown ticket reports canceled", func(t *testing.T) {
		installQueueStubs(t, &queueStub{})
		var namespaces []string
		cancelRunTicket = func(_ context.Context, _ string, req workflows.MobileDeviceSemaphoreRunCancelRequest) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
			namespaces = append(namespaces, req.OwnerNamespace)
			return workflows.MobileDeviceSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/pipeline/queue/t-1?device_ids=d-1,d-2",
			nil,
		)
		req.SetPathValue("ticket", "t-1")
		e, rec := covAPIQueueEvent(app, auth, req)
		require.NoError(t, HandlePipelineQueueCancel()(e))
		require.Equal(t, http.StatusOK, rec.Code)

		var body PipelineQueueResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, workflowengine.MobileDeviceSemaphoreRunCanceled, body.Status)
		assert.Equal(t, []string{"usera-s-organization", "usera-s-organization"}, namespaces)
	})
}

func TestCleanupCanceledQueueResourcesDeletesOwnedTempRecords(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineQueueApp(t)
	defer app.Cleanup()

	credential, err := canonify.Resolve(
		app,
		createIssuerCICredential(t, app, orgID, "cov-issuer", "cov-credential"),
	)
	require.NoError(t, err)
	useCase, err := canonify.Resolve(
		app,
		createVerifierCIUseCase(t, app, orgID, "cov-verifier", "cov-use-case"),
	)
	require.NoError(t, err)

	statuses := []pipelineQueueRunnerStatus{
		{Status: workflowengine.MobileDeviceSemaphoreRunNotFound},
		{
			Status: workflowengine.MobileDeviceSemaphoreRunCanceled,
			Cleanup: &workflows.MobileDeviceSemaphoreCleanupMetadata{
				TempCredentials: []workflows.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
					{RecordID: " "},
					{RecordID: credential.Id},
					{RecordID: "missingrecord12"},
				},
				TempUseCaseVerifications: []workflows.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
					{RecordID: ""},
					{RecordID: useCase.Id},
				},
			},
		},
	}

	require.Nil(t, cleanupCanceledQueueResources(app, orgID, statuses))

	_, err = app.FindRecordById("credentials", credential.Id)
	assert.Error(t, err, "temporary credential must be deleted")
	_, err = app.FindRecordById("use_cases_verifications", useCase.Id)
	assert.Error(t, err, "temporary use case verification must be deleted")
}

func TestCleanupCanceledQueueResourcesRejectsForeignTempRecords(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []struct {
		name    string
		cleanup func(t *testing.T, app *tests.TestApp, foreignID string) (*workflows.MobileDeviceSemaphoreCleanupMetadata, string, string)
		domain  string
	}{
		{
			name: "credential",
			cleanup: func(t *testing.T, app *tests.TestApp, foreignID string) (*workflows.MobileDeviceSemaphoreCleanupMetadata, string, string) {
				record, err := canonify.Resolve(
					app,
					covAPIForeignIdentifier(
						createIssuerCICredential(
							t,
							app,
							foreignID,
							"foreign-issuer",
							"foreign-cred",
						),
					),
				)
				require.NoError(t, err)
				return &workflows.MobileDeviceSemaphoreCleanupMetadata{
					TempCredentials: []workflows.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
						{RecordID: record.Id},
					},
				}, "credentials", record.Id
			},
		},
		{
			name: "use case verification",
			cleanup: func(t *testing.T, app *tests.TestApp, foreignID string) (*workflows.MobileDeviceSemaphoreCleanupMetadata, string, string) {
				record, err := canonify.Resolve(
					app,
					covAPIForeignIdentifier(
						createVerifierCIUseCase(
							t,
							app,
							foreignID,
							"foreign-verifier",
							"foreign-uc",
						),
					),
				)
				require.NoError(t, err)
				return &workflows.MobileDeviceSemaphoreCleanupMetadata{
					TempUseCaseVerifications: []workflows.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
						{RecordID: record.Id},
					},
				}, "use_cases_verifications", record.Id
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			app := setupPipelineQueueApp(t)
			defer app.Cleanup()
			foreign := covAPIOtherOrg(t, app)

			cleanup, collection, recordID := s.cleanup(t, app, foreign.Id)
			apiErr := cleanupCanceledQueueResources(app, orgID, []pipelineQueueRunnerStatus{
				{Status: workflowengine.MobileDeviceSemaphoreRunCanceled, Cleanup: cleanup},
			})
			require.NotNil(t, apiErr)
			assert.Equal(t, http.StatusForbidden, apiErr.Code)

			_, err := app.FindRecordById(collection, recordID)
			require.NoError(t, err, "foreign record must be kept")
		})
	}
}

func TestCleanupCanceledQueueResourcesSkipsStartedRuns(t *testing.T) {
	app := setupPipelineQueueApp(t)
	defer app.Cleanup()

	cleanup := &workflows.MobileDeviceSemaphoreCleanupMetadata{
		TempUseCaseVerifications: []workflows.MobileDeviceSemaphoreTempCredentialCleanupMetadata{
			{RecordID: "would-fail-lookup"},
		},
	}
	scenarios := []struct {
		name   string
		status pipelineQueueRunnerStatus
	}{
		{
			name: "run already has a workflow",
			status: pipelineQueueRunnerStatus{
				Status:     workflowengine.MobileDeviceSemaphoreRunCanceled,
				WorkflowID: "wf-1",
				Cleanup:    cleanup,
			},
		},
		{
			name: "run is running",
			status: pipelineQueueRunnerStatus{
				Status:  workflowengine.MobileDeviceSemaphoreRunRunning,
				Cleanup: cleanup,
			},
		},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			require.Nil(
				t,
				cleanupCanceledQueueResources(app, "owner", []pipelineQueueRunnerStatus{s.status}),
			)
		})
	}
}

func covAPIStubQueueClient(t *testing.T, c client.Client, clientErr error) {
	t.Helper()
	orig := queueTemporalClient
	t.Cleanup(func() { queueTemporalClient = orig })
	queueTemporalClient = func(string) (client.Client, error) {
		if clientErr != nil {
			return nil, clientErr
		}
		return c, nil
	}
}

func TestPipelineQueueTemporalHelperFailures(t *testing.T) {
	ctx := context.Background()
	clientErr := errors.New("no temporal")

	t.Run("client unavailable", func(t *testing.T) {
		covAPIStubQueueClient(t, nil, clientErr)

		require.ErrorIs(t, ensureRunQueueSemaphoreWorkflowTemporal(ctx, "d-1"), clientErr)
		_, err := enqueueRunTicketTemporal(
			ctx,
			"d-1",
			workflows.MobileDeviceSemaphoreEnqueueRunRequest{},
		)
		require.ErrorIs(t, err, clientErr)
		_, err = queryRunTicketStatusTemporal(ctx, "d-1", "ns", "t-1")
		require.ErrorIs(t, err, clientErr)
		_, err = cancelRunTicketTemporal(
			ctx,
			"d-1",
			workflows.MobileDeviceSemaphoreRunCancelRequest{},
		)
		require.ErrorIs(t, err, clientErr)
	})

	t.Run("enqueue update rejected", func(t *testing.T) {
		m := temporalmocks.NewClient(t)
		m.On("UpdateWorkflow", mock.Anything, mock.Anything).
			Return(nil, errors.New("update rejected")).Once()
		covAPIStubQueueClient(t, m, nil)

		_, err := enqueueRunTicketTemporal(
			ctx,
			"d-1",
			workflows.MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "t-1"},
		)
		require.EqualError(t, err, "update rejected")
	})

	t.Run("enqueue result unreadable", func(t *testing.T) {
		m := temporalmocks.NewClient(t)
		handle := temporalmocks.NewWorkflowUpdateHandle(t)
		handle.On("Get", mock.Anything, mock.Anything).Return(errors.New("decode")).Once()
		m.On("UpdateWorkflow", mock.Anything, mock.Anything).Return(handle, nil).Once()
		covAPIStubQueueClient(t, m, nil)

		_, err := enqueueRunTicketTemporal(
			ctx,
			"d-1",
			workflows.MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "t-1"},
		)
		require.EqualError(t, err, "decode")
	})

	t.Run("query outcomes", func(t *testing.T) {
		scenarios := []struct {
			name       string
			value      converter.EncodedValue
			queryErr   error
			wantErr    error
			wantErrMsg string
			wantStatus workflows.MobileDeviceSemaphoreRunStatus
		}{
			{
				name:     "semaphore workflow missing",
				queryErr: &serviceerror.NotFound{Message: "missing"},
				wantErr:  errRunTicketNotFound,
			},
			{
				name:       "query failure",
				queryErr:   errors.New("unavailable"),
				wantErrMsg: "unavailable",
			},
			{
				name:       "undecodable status",
				value:      queueErrorEncodedValue{},
				wantErrMsg: "decode failed",
			},
			{
				name: "status decoded",
				value: covAPIStatusEncodedValue{value: workflows.MobileDeviceSemaphoreRunStatusView{
					TicketID: "t-1",
					Status:   workflowengine.MobileDeviceSemaphoreRunQueued,
				}},
				wantStatus: workflowengine.MobileDeviceSemaphoreRunQueued,
			},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				m := temporalmocks.NewClient(t)
				m.On("QueryWorkflow", mock.Anything,
					workflows.MobileDeviceSemaphoreWorkflowID("d-1"), "",
					workflows.MobileDeviceSemaphoreRunStatusQuery, "ns", "t-1").
					Return(s.value, s.queryErr).Once()
				covAPIStubQueueClient(t, m, nil)

				status, err := queryRunTicketStatusTemporal(ctx, "d-1", "ns", "t-1")
				switch {
				case s.wantErr != nil:
					require.ErrorIs(t, err, s.wantErr)
				case s.wantErrMsg != "":
					require.EqualError(t, err, s.wantErrMsg)
				default:
					require.NoError(t, err)
					assert.Equal(t, s.wantStatus, status.Status)
				}
			})
		}
	})

	t.Run("cancel outcomes", func(t *testing.T) {
		scenarios := []struct {
			name       string
			updateErr  error
			getErr     error
			wantErr    error
			wantErrMsg string
		}{
			{
				name:      "semaphore workflow missing",
				updateErr: &serviceerror.NotFound{Message: "missing"},
				wantErr:   errRunTicketNotFound,
			},
			{
				name:       "update failure",
				updateErr:  errors.New("unavailable"),
				wantErrMsg: "unavailable",
			},
			{
				name:       "result unreadable",
				getErr:     errors.New("decode"),
				wantErrMsg: "decode",
			},
		}
		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				m := temporalmocks.NewClient(t)
				if s.updateErr != nil {
					m.On("UpdateWorkflow", mock.Anything, mock.Anything).
						Return(nil, s.updateErr).
						Once()
				} else {
					handle := temporalmocks.NewWorkflowUpdateHandle(t)
					handle.On("Get", mock.Anything, mock.Anything).Return(s.getErr).Once()
					m.On("UpdateWorkflow", mock.Anything, mock.Anything).Return(handle, nil).Once()
				}
				covAPIStubQueueClient(t, m, nil)

				_, err := cancelRunTicketTemporal(
					ctx,
					"d-1",
					workflows.MobileDeviceSemaphoreRunCancelRequest{TicketID: "t-1"},
				)
				if s.wantErr != nil {
					require.ErrorIs(t, err, s.wantErr)
					return
				}
				require.EqualError(t, err, s.wantErrMsg)
			})
		}
	})
}

type covAPIStatusEncodedValue struct {
	value workflows.MobileDeviceSemaphoreRunStatusView
}

func (v covAPIStatusEncodedValue) HasValue() bool { return true }

func (v covAPIStatusEncodedValue) Get(valuePtr interface{}) error {
	*valuePtr.(*workflows.MobileDeviceSemaphoreRunStatusView) = v.value
	return nil
}

func covAPICreateWalletAction(t *testing.T, app *tests.TestApp, orgID, code string) string {
	t.Helper()
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", orgID)
	wallet.Set("name", "cov-wallet")
	require.NoError(t, app.Save(wallet))

	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	action := core.NewRecord(actions)
	action.Set("owner", orgID)
	action.Set("wallet", wallet.Id)
	action.Set("name", "cov-action")
	action.Set("category", "onboarding")
	action.Set("code", code)
	require.NoError(t, app.Save(action))

	return "usera-s-organization/" + wallet.GetString("canonified_name") + "/" +
		action.GetString("canonified_name")
}

func covAPIMobileFlowClient(t *testing.T, device map[string]any) *temporalmocks.Client {
	t.Helper()
	m := temporalmocks.NewClient(t)
	m.On("DescribeWorkflowExecution", mock.Anything, "pipeline-1", "run-1").
		Return(pipelineMobileFlowDescription(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, []string{"tenant/runner-1"}), nil).
		Once()
	m.On("QueryWorkflow", mock.Anything, "pipeline-1", "run-1", pipeline.PipelineMobileDevicesQuery).
		Return(converter.EncodedValue(pipelineMobileFlowEncodedValue{
			devices: map[string]any{"tenant/runner-1": device},
		}), nil).
		Once()
	return m
}

func TestHandlePipelineMobileFlow(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineQueueApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"
	actionID := covAPICreateWalletAction(t, app, orgID, "appId: com.example\n---\n- launchApp\n")

	device := map[string]any{
		"serial":    "emulator-5554",
		"type":      "android_emulator",
		"runner_id": "usera-s-organization/runner-1",
	}

	scenarios := []struct {
		name       string
		actionID   string
		clientErr  error
		setup      func(t *testing.T) *temporalmocks.Client
		code       int
		reason     string
		wantOK     bool
		wantOutput any
		wantError  string
	}{
		{
			name:      "temporal client unavailable",
			actionID:  actionID,
			clientErr: errors.New("no temporal"),
			setup:     func(*testing.T) *temporalmocks.Client { return nil },
			code:      http.StatusInternalServerError,
			reason:    "failed to get temporal client",
		},
		{
			name:     "pipeline workflow missing",
			actionID: actionID,
			setup: func(t *testing.T) *temporalmocks.Client {
				m := temporalmocks.NewClient(t)
				m.On("DescribeWorkflowExecution", mock.Anything, "pipeline-1", "run-1").
					Return(nil, errors.New("not found")).Once()
				return m
			},
			code:   http.StatusNotFound,
			reason: "pipeline workflow not found",
		},
		{
			name:     "unknown action",
			actionID: "usera-s-organization/cov-wallet/missing",
			setup:    func(t *testing.T) *temporalmocks.Client { return covAPIMobileFlowClient(t, device) },
			code:     http.StatusNotFound,
			reason:   "wallet action not found",
		},
		{
			name:     "identifier of another collection",
			actionID: "usera-s-organization",
			setup:    func(t *testing.T) *temporalmocks.Client { return covAPIMobileFlowClient(t, device) },
			code:     http.StatusNotFound,
			reason:   "wallet action not found",
		},
		{
			name:     "workflow start failure",
			actionID: actionID,
			setup: func(t *testing.T) *temporalmocks.Client {
				m := covAPIMobileFlowClient(t, device)
				m.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("start failed")).Once()
				return m
			},
			code:   http.StatusInternalServerError,
			reason: "failed to start mobile automation workflow",
		},
		{
			name:     "workflow output returned",
			actionID: actionID,
			setup: func(t *testing.T) *temporalmocks.Client {
				m := covAPIMobileFlowClient(t, device)
				run := temporalmocks.NewWorkflowRun(t)
				run.On("Get", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						args.Get(1).(*workflowengine.WorkflowResult).Output = "flow done"
					}).
					Return(nil).Once()
				m.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						opts := args.Get(1).(client.StartWorkflowOptions)
						assert.Equal(t, pipeline.PipelineTaskQueue, opts.TaskQueue)
						input := args.Get(3).(workflowengine.WorkflowInput)
						assert.Equal(
							t,
							"usera-s-organization/runner-1-TaskQueue",
							input.Config["taskqueue"],
						)
						assert.Equal(
							t,
							"https://credimi.test",
							input.Config[workflowengine.AppURLConfigKey],
						)
						payload := input.Payload.(workflows.MobileAutomationWorkflowPayload)
						assert.Equal(t, "emulator-5554", payload.Serial)
						assert.Equal(t, "tenant/runner-1", payload.DeviceID)
						assert.Equal(t, map[string]string{"user": "alice"}, payload.Parameters)
						assert.Contains(t, payload.ActionCode, "launchApp")
					}).
					Return(run, nil).Once()
				return m
			},
			code:       http.StatusOK,
			wantOK:     true,
			wantOutput: "flow done",
		},
		{
			name:     "workflow failure is reported in body",
			actionID: actionID,
			setup: func(t *testing.T) *temporalmocks.Client {
				m := covAPIMobileFlowClient(t, device)
				run := temporalmocks.NewWorkflowRun(t)
				run.On("Get", mock.Anything, mock.Anything).Return(errors.New("device lost")).Once()
				m.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(run, nil).Once()
				return m
			},
			code:      http.StatusOK,
			wantError: "device lost",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := s.setup(t)
			orig := pipelineMobileFlowTemporalClient
			t.Cleanup(func() { pipelineMobileFlowTemporalClient = orig })
			var namespaces []string
			pipelineMobileFlowTemporalClient = func(namespace string) (client.Client, error) {
				namespaces = append(namespaces, namespace)
				if s.clientErr != nil {
					return nil, s.clientErr
				}
				return m, nil
			}

			input := PipelineMobileFlowInput{
				WorkflowID:       " pipeline-1 ",
				RunID:            "run-1",
				OrganizationID:   " usera-s-organization ",
				ActionID:         s.actionID,
				ActionParameters: map[string]string{"user": "alice"},
			}
			req := httptest.NewRequest(http.MethodPost, "/api/pipeline/mobile-flow", nil)
			req = req.WithContext(
				context.WithValue(req.Context(), middlewares.ValidatedInputKey, input),
			)
			e, rec := covAPIQueueEvent(app, nil, req)

			err := HandlePipelineMobileFlow()(e)
			assert.Equal(t, []string{"usera-s-organization"}, namespaces)
			if s.code != http.StatusOK {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, s.code, apiErr.Code)
				assert.Equal(t, s.reason, apiErr.Reason)
				return
			}
			require.NoError(t, err)
			var body PipelineMobileFlowResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, s.wantOK, body.Success)
			assert.Equal(t, s.wantOutput, body.Output)
			if s.wantError != "" {
				assert.Equal(t, s.wantError, body.Error)
			}
		})
	}
}

func TestHandlePipelineMobileFlowRejectsActionWithoutCode(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineQueueApp(t)
	defer app.Cleanup()
	actionID := covAPICreateWalletAction(t, app, orgID, "   ")

	m := covAPIMobileFlowClient(t, map[string]any{"type": "android_emulator"})
	orig := pipelineMobileFlowTemporalClient
	t.Cleanup(func() { pipelineMobileFlowTemporalClient = orig })
	pipelineMobileFlowTemporalClient = func(string) (client.Client, error) { return m, nil }

	input := PipelineMobileFlowInput{
		WorkflowID:     "pipeline-1",
		RunID:          "run-1",
		OrganizationID: "usera-s-organization",
		ActionID:       actionID,
	}
	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/mobile-flow", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	e, _ := covAPIQueueEvent(app, nil, req)

	covAPIRequireAPIError(
		t,
		HandlePipelineMobileFlow()(e),
		http.StatusUnprocessableEntity,
		"wallet action has no code",
	)
}

// covAPIForeignIdentifier rewrites a CI fixture identifier, which always uses
// userA's namespace, to the namespace of covAPIOtherOrg.
func covAPIForeignIdentifier(identifier string) string {
	return strings.Replace(identifier, "usera-s-organization/", "other-org/", 1)
}

func TestHandleGetPipelineExecutionRejectsRunsOfOtherPipelines(t *testing.T) {
	app := setupPipelineStartApp(t)
	defer app.Cleanup()
	auth, org := covAPIUserA(t, app)
	pipelineRecord := createPipelineExecutionTestPipeline(t, app, org.Id)
	identifier := pipelineIdentifierForTest(t, app, pipelineRecord)

	describe := func(info *workflowpb.WorkflowExecutionInfo, err error) func(m *temporalmocks.Client) {
		return func(m *temporalmocks.Client) {
			var resp *workflowservice.DescribeWorkflowExecutionResponse
			if info != nil {
				resp = &workflowservice.DescribeWorkflowExecutionResponse{
					WorkflowExecutionInfo: info,
				}
			}
			m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
				Return(resp, err).
				Once()
		}
	}
	otherType := buildPipelineExecutionInfo("wf-1", "run-1", identifier)
	otherType.Type = &commonpb.WorkflowType{Name: "Some Other Workflow"}

	scenarios := []struct {
		name      string
		auth      *core.Record
		runID     string
		clientErr error
		setup     func(m *temporalmocks.Client)
		code      int
		reason    string
	}{
		{
			name:   "unauthenticated",
			runID:  "run-1",
			code:   http.StatusUnauthorized,
			reason: "authentication required",
		},
		{
			name:   "missing run id",
			auth:   auth,
			code:   http.StatusBadRequest,
			reason: "workflow ID and run ID are required",
		},
		{
			name:      "temporal unavailable",
			auth:      auth,
			runID:     "run-1",
			clientErr: errors.New("no temporal"),
			code:      http.StatusInternalServerError,
			reason:    "unable to create temporal client",
		},
		{
			name:   "unknown run",
			auth:   auth,
			runID:  "run-1",
			setup:  describe(nil, &serviceerror.NotFound{Message: "missing"}),
			code:   http.StatusNotFound,
			reason: "workflow not found",
		},
		{
			name:   "run of a non-pipeline workflow",
			auth:   auth,
			runID:  "run-1",
			setup:  describe(otherType, nil),
			code:   http.StatusNotFound,
			reason: "pipeline execution not found",
		},
		{
			name:  "run of another pipeline",
			auth:  auth,
			runID: "run-1",
			setup: describe(
				buildPipelineExecutionInfo("wf-1", "run-1", "other-org/other-pipeline"),
				nil,
			),
			code:   http.StatusNotFound,
			reason: "pipeline execution not found",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			if s.setup != nil {
				s.setup(m)
			}
			orig := pipelineTemporalClient
			t.Cleanup(func() { pipelineTemporalClient = orig })
			pipelineTemporalClient = func(string) (client.Client, error) {
				if s.clientErr != nil {
					return nil, s.clientErr
				}
				return m, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetPathValue("id", pipelineRecord.Id)
			req.SetPathValue("workflow_id", "wf-1")
			req.SetPathValue("run_id", s.runID)
			e, _ := covAPIQueueEvent(app, s.auth, req)
			covAPIRequireAPIError(t, HandleGetPipelineExecution()(e), s.code, s.reason)
		})
	}
}

func TestHandleListPipelineExecutionHistoryBackendFailures(t *testing.T) {
	app := setupPipelineStartApp(t)
	defer app.Cleanup()
	auth, org := covAPIUserA(t, app)
	pipelineRecord := createPipelineExecutionTestPipeline(t, app, org.Id)

	scenarios := []struct {
		name      string
		status    string
		queuedErr error
		reason    string
	}{
		{
			name:      "queued runs unavailable",
			queuedErr: errors.New("semaphore down"),
			reason:    "failed to list queued runs",
		},
		{
			name:   "temporal unavailable",
			status: "completed",
			reason: "unable to create temporal client",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			origQueued := pipelineListQueuedRuns
			origClient := pipelineTemporalClient
			t.Cleanup(func() {
				pipelineListQueuedRuns = origQueued
				pipelineTemporalClient = origClient
			})
			pipelineListQueuedRuns = func(context.Context, string) (map[string]QueuedPipelineRunAggregate, error) {
				return nil, s.queuedErr
			}
			pipelineTemporalClient = func(string) (client.Client, error) {
				return nil, errors.New("no temporal")
			}

			req := httptest.NewRequest(http.MethodGet, "/?status="+s.status, nil)
			req.SetPathValue("id", pipelineRecord.Id)
			e, _ := covAPIQueueEvent(app, auth, req)
			covAPIRequireAPIError(
				t,
				HandleListPipelineExecutionHistory()(e),
				http.StatusInternalServerError,
				s.reason,
			)
		})
	}
}
