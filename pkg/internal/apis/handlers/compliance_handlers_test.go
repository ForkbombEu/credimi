// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
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
