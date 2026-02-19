// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
)

type fakeHistoryIterator struct {
	events  []*historypb.HistoryEvent
	nextErr error
	index   int
}

func (f *fakeHistoryIterator) HasNext() bool {
	return f.nextErr != nil || f.index < len(f.events)
}

func (f *fakeHistoryIterator) Next() (*historypb.HistoryEvent, error) {
	if f.nextErr != nil {
		err := f.nextErr
		f.nextErr = nil
		return nil, err
	}
	if f.index >= len(f.events) {
		return nil, errors.New("no more events")
	}
	event := f.events[f.index]
	f.index++
	return event, nil
}

func TestHandleGetWorkflowMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	t.Run("missing workflowId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks//run", nil)
		rec := httptest.NewRecorder()

		err := HandleGetWorkflow()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing runId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf/run", nil)
		req.SetPathValue("workflowId", "wf-1")
		rec := httptest.NewRecorder()

		err := HandleGetWorkflow()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandleGetWorkflowSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	originalClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = originalClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{
					WorkflowId: "wf-1",
					RunId:      "run-1",
				},
			},
		}, nil).
		Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-1/run-1", nil)
	req.SetPathValue("workflowId", "wf-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflow()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.NotEmpty(t, payload["workflowExecutionInfo"])

	mockClient.AssertExpectations(t)
}

func TestGetWorkflowExecutionWithFallback(t *testing.T) {
	originalClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = originalClient
	})

	mockOrg := &temporalmocks.Client{}
	mockDefault := &temporalmocks.Client{}
	mockOrg.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return((*workflowservice.DescribeWorkflowExecutionResponse)(nil), &serviceerror.NotFound{Message: "missing"}).
		Once()
	mockDefault.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil).
		Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		if namespace == "default" {
			return mockDefault, nil
		}
		return mockOrg, nil
	}

	exec, err := getWorkflowExecutionWithFallback("acme", "wf-1", "run-1")
	require.NoError(t, err)
	require.NotNil(t, exec)

	mockOrg.AssertExpectations(t)
	mockDefault.AssertExpectations(t)
}

func TestGetWorkflowHistoryWithFallback(t *testing.T) {
	originalClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = originalClient
	})

	mockOrg := &temporalmocks.Client{}
	mockDefault := &temporalmocks.Client{}
	mockOrg.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			mock.Anything,
		).
		Return(&fakeHistoryIterator{nextErr: &serviceerror.NotFound{Message: "missing"}}, nil).
		Once()
	mockDefault.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			mock.Anything,
		).
		Return(&fakeHistoryIterator{
			events: []*historypb.HistoryEvent{
				{EventId: 1},
			},
		}, nil).
		Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		if namespace == "default" {
			return mockDefault, nil
		}
		return mockOrg, nil
	}

	history, err := getWorkflowHistoryWithFallback("acme", "wf-1", "run-1")
	require.NoError(t, err)
	require.Len(t, history, 1)

	mockOrg.AssertExpectations(t)
	mockDefault.AssertExpectations(t)
}

func TestSendTemporalSignalErrorMapping(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On("SignalWorkflow", mock.Anything, "wf-1", "", "signal-1", mock.Anything).
			Return(&serviceerror.NotFound{Message: "missing"}).
			Once()

		err := sendTemporalSignal(mockClient, HandleSendTemporalSignalInput{
			WorkflowID: "wf-1",
			Signal:     "signal-1",
		})
		require.Error(t, err)
		var apiErr *apierror.APIError
		require.True(t, errors.As(err, &apiErr))
		require.Equal(t, http.StatusNotFound, apiErr.Code)

		mockClient.AssertExpectations(t)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On("SignalWorkflow", mock.Anything, "wf-1", "", "signal-1", mock.Anything).
			Return(&serviceerror.InvalidArgument{Message: "bad"}).
			Once()

		err := sendTemporalSignal(mockClient, HandleSendTemporalSignalInput{
			WorkflowID: "wf-1",
			Signal:     "signal-1",
		})
		require.Error(t, err)
		var apiErr *apierror.APIError
		require.True(t, errors.As(err, &apiErr))
		require.Equal(t, http.StatusBadRequest, apiErr.Code)

		mockClient.AssertExpectations(t)
	})
}

func TestSendOpenIDNetLogUpdateStartAlreadyCompleted(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	originalNotify := complianceNotifyLogsUpdate
	t.Cleanup(func() {
		complianceNotifyLogsUpdate = originalNotify
	})

	var capturedSubscription string
	var capturedLogs []map[string]any
	complianceNotifyLogsUpdate = func(_ core.App, subscription string, data []map[string]any) error {
		capturedSubscription = subscription
		capturedLogs = data
		return nil
	}

	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockClient.
		On(
			"SignalWorkflow",
			mock.Anything,
			"wf-1-log",
			"",
			workflows.OpenIDNetStartCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.NotFound{Message: "workflow execution already completed"}).
		Once()
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*workflowengine.WorkflowResult)
			out.Log = []any{map[string]any{"step": "done"}}
		}).
		Return(nil).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "wf-1-log", "").Return(mockRun).Once()

	err = sendOpenIDNetLogUpdateStart(
		app,
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "wf-1-log"},
	)
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusNotFound, apiErr.Code)
	require.Equal(t, "wf-1"+workflows.OpenIDNetSubscription, capturedSubscription)
	require.Len(t, capturedLogs, 1)
	require.Equal(t, "done", capturedLogs[0]["step"])

	mockClient.AssertExpectations(t)
	mockRun.AssertExpectations(t)
}

func TestHandleGetWorkflowsHistorySuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{
			events: []*historypb.HistoryEvent{
				{EventId: 1},
			},
		}).
		Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-1/run-1/history", nil)
	req.SetPathValue("workflowId", "wf-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflowsHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"eventId\":\"1\"")
}

func TestHandleGetWorkflowsHistoryMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks//run-1/history", nil)
	rec := httptest.NewRecorder()

	err = HandleGetWorkflowsHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-1//history", nil)
	req.SetPathValue("workflowId", "wf-1")
	rec = httptest.NewRecorder()

	err = HandleGetWorkflowsHistory()(&core.RequestEvent{
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

func TestHandleGetWorkflowsHistoryNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockOrg := &temporalmocks.Client{}
	mockDefault := &temporalmocks.Client{}
	mockOrg.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-2",
			"run-2",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{nextErr: &serviceerror.NotFound{Message: "missing"}}).
		Once()
	mockDefault.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-2",
			"run-2",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{nextErr: &serviceerror.NotFound{Message: "missing"}}).
		Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		if namespace == "default" {
			return mockDefault, nil
		}
		return mockOrg, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-2/run-2/history", nil)
	req.SetPathValue("workflowId", "wf-2")
	req.SetPathValue("runId", "run-2")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflowsHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleGetWorkflowNamespaceEmpty(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgAuth, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user={:user}",
		dbx.Params{"user": authRecord.Id},
	)
	require.NoError(t, err)

	orgRecord, err := app.FindRecordById("organizations", orgAuth.GetString("organization"))
	require.NoError(t, err)

	orgRecord.Set("canonified_name", "")
	require.NoError(t, app.Save(orgRecord))

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-3/run-3", nil)
	req.SetPathValue("workflowId", "wf-3")
	req.SetPathValue("runId", "run-3")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflow()(&core.RequestEvent{
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

func TestHandleGetWorkflowNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockOrg := &temporalmocks.Client{}
	mockDefault := &temporalmocks.Client{}
	mockOrg.
		On("DescribeWorkflowExecution", mock.Anything, "wf-4", "run-4").
		Return((*workflowservice.DescribeWorkflowExecutionResponse)(nil), &serviceerror.NotFound{}).
		Once()
	mockDefault.
		On("DescribeWorkflowExecution", mock.Anything, "wf-4", "run-4").
		Return((*workflowservice.DescribeWorkflowExecutionResponse)(nil), &serviceerror.NotFound{}).
		Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		if namespace == "default" {
			return mockDefault, nil
		}
		return mockOrg, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-4/run-4", nil)
	req.SetPathValue("workflowId", "wf-4")
	req.SetPathValue("runId", "run-4")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflow()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleGetWorkflowClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		return nil, errors.New("client error")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-5/run-5", nil)
	req.SetPathValue("workflowId", "wf-5")
	req.SetPathValue("runId", "run-5")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflow()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleGetWorkflowResultNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Return(&serviceerror.NotFound{}).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "wf-2", "run-2").Return(mockRun).Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-2/run-2/result", nil)
	req.SetPathValue("workflowId", "wf-2")
	req.SetPathValue("runId", "run-2")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflowResult()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusNotFound, apiErr.Code)
}

func TestHandleGetWorkflowResultMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks//run/result", nil)
	rec := httptest.NewRecorder()

	err = HandleGetWorkflowResult()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestHandleGetWorkflowResultClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		return nil, errors.New("client error")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-6/run-6/result", nil)
	req.SetPathValue("workflowId", "wf-6")
	req.SetPathValue("runId", "run-6")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflowResult()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestHandleGetWorkflowResultSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*workflowengine.WorkflowResult)
			out.Log = []any{"ok"}
		}).
		Return(nil).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "wf-7", "run-7").Return(mockRun).Once()

	complianceTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/checks/wf-7/run-7/result", nil)
	req.SetPathValue("workflowId", "wf-7")
	req.SetPathValue("runId", "run-7")
	rec := httptest.NewRecorder()

	err = HandleGetWorkflowResult()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"ok\"")
}

func TestHandleSendOpenIDNetLogUpdateSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	origNotify := complianceNotifyLogsUpdate
	t.Cleanup(func() {
		complianceNotifyLogsUpdate = origNotify
	})

	var capturedSubscription string
	complianceNotifyLogsUpdate = func(_ core.App, subscription string, data []map[string]any) error {
		capturedSubscription = subscription
		return nil
	}

	input := HandleSendLogUpdateRequestInput{
		WorkflowID: "wf-3",
		Logs:       []map[string]any{{"step": "ok"}},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/compliance/send-openidnet-log-update", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err = HandleSendOpenIDNetLogUpdate()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "wf-3"+workflows.OpenIDNetSubscription, capturedSubscription)
}

func TestHandleSendEudiwLogUpdateError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	origNotify := complianceNotifyLogsUpdate
	t.Cleanup(func() {
		complianceNotifyLogsUpdate = origNotify
	})

	complianceNotifyLogsUpdate = func(_ core.App, subscription string, data []map[string]any) error {
		return errors.New("boom")
	}

	input := HandleSendLogUpdateRequestInput{
		WorkflowID: "wf-1",
		Logs:       []map[string]any{{"step": "ok"}},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/compliance/send-eudiw-log-update", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err = HandleSendEudiwLogUpdate()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleDeeplinkMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink/", nil)
	rec := httptest.NewRecorder()

	err = HandleDeeplink()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink/wf-1/", nil)
	req.SetPathValue("workflowId", "wf-1")
	rec = httptest.NewRecorder()

	err = HandleDeeplink()(&core.RequestEvent{
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

func TestHandleDeeplinkTemporalClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})
	complianceTemporalClient = func(_ string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink/wf-1/run-1", nil)
	req.SetPathValue("workflowId", "wf-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleDeeplink()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleSendTemporalSignalMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("SignalWorkflow", mock.Anything, "", "", "", mock.Anything).
		Return(&serviceerror.InvalidArgument{Message: "bad"})

	complianceTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/compliance/signal", nil)
	req = req.WithContext(
		context.WithValue(
			req.Context(),
			middlewares.ValidatedInputKey,
			HandleSendTemporalSignalInput{},
		),
	)
	rec := httptest.NewRecorder()

	err = HandleSendTemporalSignal()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSendTemporalSignalNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("SignalWorkflow", mock.Anything, "wf-1", "", "sig", mock.Anything).
		Return(&serviceerror.NotFound{Message: "missing"})

	complianceTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	input := HandleSendTemporalSignalInput{
		WorkflowID: "wf-1",
		Namespace:  "ns",
		Signal:     "sig",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/signal", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err = HandleSendTemporalSignal()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleSendTemporalSignalSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	origClient := complianceTemporalClient
	t.Cleanup(func() {
		complianceTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("SignalWorkflow", mock.Anything, "wf-2", "", "sig", mock.Anything).
		Return(nil)

	complianceTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	input := HandleSendTemporalSignalInput{
		WorkflowID: "wf-2",
		Namespace:  "ns",
		Signal:     "sig",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/signal", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err = HandleSendTemporalSignal()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestNotifyLogsUpdateNoSubscribers(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	err = notifyLogsUpdate(app, "subscription-1", []map[string]any{{"step": "ok"}})
	require.NoError(t, err)
}

func TestGetDeeplinkOpenIDConformanceSuite(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	payload := base64.StdEncoding.EncodeToString(
		[]byte(`{"Output":{"captures":{"deeplink":"link-1"}}}`),
	)
	req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink", nil)
	rec := httptest.NewRecorder()

	err = getDeeplinkOpenIDConformanceSuite(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}, map[string]any{"data": payload})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "link-1")
}

func TestGetDeeplinkEudiw(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	payload := base64.StdEncoding.EncodeToString(
		[]byte(
			`{"Output":{"captures":{"client_id":"client-1","request_uri":"https://example.com/req"}}}`,
		),
	)
	req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink", nil)
	rec := httptest.NewRecorder()

	err = getDeeplinkEudiw(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}, map[string]any{"data": payload})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	expected, err := workflows.BuildQRDeepLink("client-1", "https://example.com/req")
	require.NoError(t, err)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, expected, body["deeplink"])
}

func TestGetWorkflowAuthorFromMemo(t *testing.T) {
	payload, err := converter.GetDefaultDataConverter().ToPayload("ewc")
	require.NoError(t, err)

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-4", "run-4").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Memo: &common.Memo{
					Fields: map[string]*common.Payload{
						"author": payload,
					},
				},
			},
		}, nil).
		Once()

	author, err := getWorkflowAuthor(mockClient, "wf-4", "run-4")
	require.NoError(t, err)
	require.Equal(t, "ewc", author)
}

func TestHandleDeeplinkFromHistoryEWC(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	payloadData, err := json.Marshal(map[string]any{
		"Output": map[string]any{
			"Captures": map[string]any{
				"deeplink": "ewc://link",
			},
		},
	})
	require.NoError(t, err)
	payloads := &common.Payloads{
		Payloads: []*common.Payload{
			{Data: payloadData},
		},
	}

	iter := &fakeHistoryIterator{
		events: []*historypb.HistoryEvent{
			{
				EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
				Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
					ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
						Result: payloads,
					},
				},
			},
		},
	}

	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-5",
			"run-5",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(iter).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink/wf-5/run-5", nil)
	rec := httptest.NewRecorder()
	err = handleDeeplinkFromHistory(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}, mockClient, "wf-5", "run-5", "ewc")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "ewc://link")
}
