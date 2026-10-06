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

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func covAPIStubWorkflowClients(t *testing.T, c client.Client, clientErr error) {
	t.Helper()
	origWorkflow := workflowTemporalClient
	origList := listWorkflowsTemporalClient
	t.Cleanup(func() {
		workflowTemporalClient = origWorkflow
		listWorkflowsTemporalClient = origList
	})
	stub := func(string) (client.Client, error) {
		if clientErr != nil {
			return nil, clientErr
		}
		return c, nil
	}
	workflowTemporalClient = stub
	listWorkflowsTemporalClient = stub
}

func covAPIWorkflowEvent(
	app core.App,
	auth *core.Record,
	workflowID, runID string,
) (*core.RequestEvent, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, "/api/my/workflows", nil)
	req.SetPathValue("workflowId", workflowID)
	req.SetPathValue("runId", runID)
	rec := httptest.NewRecorder()
	return &core.RequestEvent{
		App:   app,
		Auth:  auth,
		Event: router.Event{Request: req, Response: rec},
	}, rec
}

var covAPIWorkflowHandlers = map[string]func() func(*core.RequestEvent) error{
	"list":     HandleListMyWorkflows,
	"get":      HandleGetMyWorkflowRun,
	"history":  HandleGetMyWorkflowRunHistory,
	"listRuns": HandleListMyWorkflowRuns,
	"rerun":    HandleRerunMyWorkflow,
	"cancel":   HandleCancelMyWorkflowRun,
	"export":   HandleExportMyWorkflowRun,
	"logs":     HandleMyWorkflowLogs,
}

func TestMyWorkflowHandlersRequireAuthAndTemporal(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	for name, handler := range covAPIWorkflowHandlers {
		t.Run(name+"/unauthenticated", func(t *testing.T) {
			covAPIStubWorkflowClients(t, temporalmocks.NewClient(t), nil)
			e, _ := covAPIWorkflowEvent(app, nil, "wf-1", "run-1")
			covAPIRequireAPIError(
				t,
				handler()(e),
				http.StatusUnauthorized,
				"authentication required",
			)
		})
		if name == "logs" {
			continue
		}
		t.Run(name+"/temporal unavailable", func(t *testing.T) {
			covAPIStubWorkflowClients(t, nil, errors.New("no temporal"))
			e, _ := covAPIWorkflowEvent(app, auth, "wf-1", "run-1")
			covAPIRequireAPIError(
				t,
				handler()(e),
				http.StatusInternalServerError,
				"unable to create client",
			)
		})
	}

	missingParams := []struct {
		name       string
		handler    func() func(*core.RequestEvent) error
		workflowID string
		runID      string
		reason     string
	}{
		{
			name:       "get without run",
			handler:    HandleGetMyWorkflowRun,
			workflowID: "wf-1",
			reason:     "runId is required",
		},
		{
			name:    "history without workflow",
			handler: HandleGetMyWorkflowRunHistory,
			runID:   "run-1",
			reason:  "workflowId and runId are required",
		},
		{
			name:    "runs without workflow",
			handler: HandleListMyWorkflowRuns,
			reason:  "workflowId is required",
		},
		{
			name:       "rerun without run",
			handler:    HandleRerunMyWorkflow,
			workflowID: "wf-1",
			reason:     "workflowId and runId are required",
		},
		{
			name:       "cancel without run",
			handler:    HandleCancelMyWorkflowRun,
			workflowID: "wf-1",
			reason:     "workflowId and runId are required",
		},
		{
			name:       "export without run",
			handler:    HandleExportMyWorkflowRun,
			workflowID: "wf-1",
			reason:     "workflowId and runId are required",
		},
	}
	for _, s := range missingParams {
		t.Run(s.name, func(t *testing.T) {
			covAPIStubWorkflowClients(t, temporalmocks.NewClient(t), nil)
			e, _ := covAPIWorkflowEvent(app, auth, s.workflowID, s.runID)
			covAPIRequireAPIError(t, s.handler()(e), http.StatusBadRequest, s.reason)
		})
	}
}

func TestMyWorkflowHandlersMapTemporalErrors(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	notFound := &serviceerror.NotFound{Message: "missing"}
	invalid := &serviceerror.InvalidArgument{Message: "bad"}
	generic := errors.New("unavailable")

	scenarios := []struct {
		name    string
		handler func() func(*core.RequestEvent) error
		setup   func(m *temporalmocks.Client)
		code    int
		reason  string
	}{
		{
			name:    "get missing run",
			handler: HandleGetMyWorkflowRun,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(nil, notFound).
					Once()
			},
			code:   http.StatusNotFound,
			reason: "workflow not found",
		},
		{
			name:    "get invalid id",
			handler: HandleGetMyWorkflowRun,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(nil, invalid).
					Once()
			},
			code:   http.StatusBadRequest,
			reason: "invalid workflow ID",
		},
		{
			name:    "get describe failure",
			handler: HandleGetMyWorkflowRun,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(nil, generic).
					Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to describe workflow execution",
		},
		{
			name:    "history iteration failure",
			handler: HandleGetMyWorkflowRunHistory,
			setup: func(m *temporalmocks.Client) {
				m.On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false,
					enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
					Return(&fakeHistoryIterator{nextErr: generic}).Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to get workflow history",
		},
		{
			name:    "runs listing failure",
			handler: HandleListMyWorkflowRuns,
			setup: func(m *temporalmocks.Client) {
				m.On("ListWorkflow", mock.Anything, mock.Anything).Return(nil, generic).Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to list workflow executions",
		},
		{
			name:    "rerun missing run",
			handler: HandleRerunMyWorkflow,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(nil, notFound).
					Once()
			},
			code:   http.StatusNotFound,
			reason: "workflow execution not found",
		},
		{
			name:    "rerun describe failure",
			handler: HandleRerunMyWorkflow,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(nil, generic).
					Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to describe workflow execution",
		},
		{
			name:    "rerun unreadable input",
			handler: HandleRerunMyWorkflow,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil).Once()
				m.On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false,
					enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
					Return(&fakeHistoryIterator{nextErr: generic}).Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to get workflow input",
		},
		{
			name:    "cancel missing run",
			handler: HandleCancelMyWorkflowRun,
			setup: func(m *temporalmocks.Client) {
				m.On("CancelWorkflow", mock.Anything, "wf-1", "run-1").Return(notFound).Once()
			},
			code:   http.StatusNotFound,
			reason: "workflow execution not found",
		},
		{
			name:    "cancel failure",
			handler: HandleCancelMyWorkflowRun,
			setup: func(m *temporalmocks.Client) {
				m.On("CancelWorkflow", mock.Anything, "wf-1", "run-1").Return(generic).Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to cancel workflow execution",
		},
		{
			name:    "export unreadable input",
			handler: HandleExportMyWorkflowRun,
			setup: func(m *temporalmocks.Client) {
				m.On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false,
					enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
					Return(&fakeHistoryIterator{nextErr: generic}).Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to get workflow input",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			s.setup(m)
			covAPIStubWorkflowClients(t, m, nil)
			e, _ := covAPIWorkflowEvent(app, auth, "wf-1", "run-1")
			covAPIRequireAPIError(t, s.handler()(e), s.code, s.reason)
		})
	}
}

func TestHandleRerunMyWorkflowStartFailure(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	m := temporalmocks.NewClient(t)
	m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil).Once()
	covAPIStubWorkflowClients(t, m, nil)

	origInput := workflowRunInputGetter
	origStart := workflowStartWithOptions
	t.Cleanup(func() {
		workflowRunInputGetter = origInput
		workflowStartWithOptions = origStart
	})
	workflowRunInputGetter = func(string, string, client.Client) (workflowengine.WorkflowInput, error) {
		return workflowengine.WorkflowInput{}, nil
	}
	workflowStartWithOptions = func(string, client.StartWorkflowOptions, string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("start rejected")
	}

	e, _ := covAPIWorkflowEvent(app, auth, "wf-1", "run-1")
	covAPIRequireAPIError(
		t,
		HandleRerunMyWorkflow()(e),
		http.StatusInternalServerError,
		"failed to start workflow",
	)
}

func TestHandleExportMyWorkflowRunDefaultsEmptyInput(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	covAPIStubWorkflowClients(t, temporalmocks.NewClient(t), nil)
	orig := workflowRunInputGetter
	t.Cleanup(func() { workflowRunInputGetter = orig })
	workflowRunInputGetter = func(string, string, client.Client) (workflowengine.WorkflowInput, error) {
		return workflowengine.WorkflowInput{}, nil
	}

	e, rec := covAPIWorkflowEvent(app, auth, "wf-1", "run-1")
	require.NoError(t, HandleExportMyWorkflowRun()(e))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(
		t,
		`{"export":{"workflowId":"wf-1","runId":"run-1","input":{},"config":{}}}`,
		rec.Body.String(),
	)
}

func TestHandleListMyWorkflowsFailures(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	scenarios := []struct {
		name   string
		list   *workflowservice.ListWorkflowExecutionsResponse
		setup  func(m *temporalmocks.Client)
		code   int
		reason string
	}{
		{
			name:   "listing fails",
			code:   http.StatusInternalServerError,
			reason: "failed to list workflows",
		},
		{
			name: "parent lookup fails",
			list: &workflowservice.ListWorkflowExecutionsResponse{
				Executions: []*workflow.WorkflowExecutionInfo{
					{
						Execution: &common.WorkflowExecution{
							WorkflowId: "child",
							RunId:      "child-run",
						},
						Type: &common.WorkflowType{Name: "Child"},
						ParentExecution: &common.WorkflowExecution{
							WorkflowId: "parent",
							RunId:      "parent-run",
						},
					},
				},
			},
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "parent", "parent-run").
					Return(nil, errors.New("unavailable")).Once()
			},
			code:   http.StatusInternalServerError,
			reason: "failed to resolve workflow parents",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			if s.setup != nil {
				s.setup(m)
			}
			covAPIStubWorkflowClients(t, m, nil)
			orig := listWorkflows
			t.Cleanup(func() { listWorkflows = orig })
			listWorkflows = func(context.Context, client.Client, string, string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
				if s.list == nil {
					return nil, errors.New("unavailable")
				}
				return s.list, nil
			}

			e, _ := covAPIWorkflowEvent(app, auth, "", "")
			covAPIRequireAPIError(t, HandleListMyWorkflows()(e), s.code, s.reason)
		})
	}
}

func TestFilterNonPipelineExecutionsHidesPipelineInternals(t *testing.T) {
	pipelineName := pipeline.NewPipelineWorkflow().Name()
	m := temporalmocks.NewClient(t)
	m.On("DescribeWorkflowExecution", mock.Anything, "remote-pipeline", "rp-run").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Type: &common.WorkflowType{Name: pipelineName},
			},
		}, nil).Once()
	m.On("DescribeWorkflowExecution", mock.Anything, "remote-other", "ro-run").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil).Once()

	executions := []*WorkflowExecution{
		nil,
		{},
		{
			Execution: &WorkflowIdentifier{WorkflowID: "child-of-remote-pipeline", RunID: "c1"},
			ParentExecution: &WorkflowIdentifier{
				WorkflowID: "remote-pipeline",
				RunID:      "rp-run",
			},
		},
		{
			Execution: &WorkflowIdentifier{WorkflowID: "child-of-remote-other", RunID: "c2"},
			ParentExecution: &WorkflowIdentifier{
				WorkflowID: "remote-other",
				RunID:      "ro-run",
			},
		},
	}

	filtered, err := filterNonPipelineExecutions(context.Background(), m, executions)
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, "child-of-remote-other", filtered[0].Execution.WorkflowID)
}

func TestGetWorkflowTypeNameWithoutLookup(t *testing.T) {
	name, err := getWorkflowTypeName(context.Background(), nil, "wf", "run")
	require.NoError(t, err)
	assert.Empty(t, name)

	m := temporalmocks.NewClient(t)
	name, err = getWorkflowTypeName(context.Background(), m, "", "run")
	require.NoError(t, err)
	assert.Empty(t, name)
}

func TestPaginateWorkflowExecutionSummariesClampsNegativeOffset(t *testing.T) {
	summaries := []*WorkflowExecutionSummary{{}, {}, {}}
	assert.Len(t, paginateWorkflowExecutionSummaries(summaries, 2, -5), 2)
}

func covAPIStartedEvent(data []byte) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: &common.Payloads{Payloads: []*common.Payload{{Data: data}}},
			},
		},
	}
}

func TestGetWorkflowInputValidatesStartedPayload(t *testing.T) {
	scenarios := []struct {
		name       string
		data       string
		wantErr    string
		wantConfig map[string]any
	}{
		{name: "not json", data: "not json", wantErr: "failed to unmarshal workflow input payload"},
		{name: "payload missing", data: `{"Config":{}}`, wantErr: "missing workflow input payload"},
		{
			name:    "payload not a map",
			data:    `{"Payload":1,"Config":{}}`,
			wantErr: "invalid workflow input payload format",
		},
		{name: "config missing", data: `{"Payload":{}}`, wantErr: "missing workflow input config"},
		{
			name:    "config not a map",
			data:    `{"Payload":{},"Config":[1]}`,
			wantErr: "invalid workflow input config format",
		},
		{
			name:       "valid input",
			data:       `{"Payload":{"a":1},"Config":{"namespace":"ns"}}`,
			wantConfig: map[string]any{"namespace": "ns"},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			m.On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false,
				enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
				Return(&fakeHistoryIterator{events: []*historypb.HistoryEvent{
					{EventType: enums.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED},
					covAPIStartedEvent([]byte(s.data)),
				}}).Once()

			input, err := getWorkflowInput("wf-1", "run-1", m)
			if s.wantErr != "" {
				require.ErrorContains(t, err, s.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, s.wantConfig, input.Config)
			raw, err := json.Marshal(input.Payload)
			require.NoError(t, err)
			assert.JSONEq(t, `{"a":1}`, string(raw))
		})
	}
}
