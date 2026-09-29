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

func TestEWCLikeSignalError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedReason string
	}{
		{
			name:           "invalid workflow ID",
			err:            &serviceerror.InvalidArgument{Message: "bad workflow ID"},
			expectedReason: "invalid workflow ID",
		},
		{
			name:           "signal failure",
			err:            errors.New("unavailable"),
			expectedReason: "failed to send signal: start",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ewcLikeSignalError(test.err, "start")
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.Code)
			require.Equal(t, test.expectedReason, apiErr.Reason)
		})
	}
}

func TestExtractEWCLikeLogsFromWorkflowError(t *testing.T) {
	expected := []any{map[string]any{
		"message":   "check completed",
		"timestamp": "2026-07-15T12:00:00Z",
	}}
	workflowErr := workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    "CRE999",
		Summary: "failed",
		Details: map[string]any{
			"payload": map[string]any{"logs": expected},
		},
	})

	logs := extractEWCLikeLogsFromWorkflowError(errors.Join(errors.New("outer"), workflowErr))
	require.Len(t, logs, 1)
	require.Equal(t, "check completed", logs[0]["message"])

	require.Nil(t, extractEWCLikeLogsFromWorkflowError(errors.New("unstructured")))
}

func TestSendOpenID4VPWalletLogUpdateStartAlreadyCompleted(t *testing.T) {
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
			workflows.OpenID4VPWalletStartCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.NotFound{Message: "workflow execution already completed"}).
		Once()
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*workflowengine.WorkflowResult)
			out.Log = []any{map[string]any{"msg": "done", "result": "FINISHED"}}
		}).
		Return(nil).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "wf-1-log", "").Return(mockRun).Once()

	err = sendOpenID4VPWalletLogUpdateStart(
		app,
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "wf-1-log"},
	)
	require.NoError(t, err)
	require.Equal(t, "wf-1"+workflows.OpenID4VPWalletSubscription, capturedSubscription)
	require.Len(t, capturedLogs, 1)
	require.Equal(t, "done", capturedLogs[0]["msg"])

	mockClient.AssertExpectations(t)
	mockRun.AssertExpectations(t)
}

func TestSendOpenID4VPWalletLogUpdateStartAlreadyCompletedErrorLogs(t *testing.T) {
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
			"wf-2-log",
			"",
			workflows.OpenID4VPWalletStartCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.NotFound{Message: "workflow execution already completed"}).
		Once()
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Return(workflowengine.NewAppError(workflowengine.WorkflowError{
			Code:    "CRE999",
			Summary: "failed",
			Details: map[string]any{
				"payload": []any{map[string]any{"msg": "finished", "result": "FINISHED"}},
			},
		})).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "wf-2-log", "").Return(mockRun).Once()

	err = sendOpenID4VPWalletLogUpdateStart(
		app,
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "wf-2-log"},
	)
	require.NoError(t, err)
	require.Equal(t, "wf-2"+workflows.OpenID4VPWalletSubscription, capturedSubscription)
	require.Len(t, capturedLogs, 1)
	require.Equal(t, "FINISHED", capturedLogs[0]["result"])

	mockClient.AssertExpectations(t)
	mockRun.AssertExpectations(t)
}

func TestSendOpenIDNetConformanceLogUpdateStartCompleted(t *testing.T) {
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
		On("DescribeWorkflowExecution", mock.Anything, "verifier-wf", "").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Status: enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
		}, nil).
		Once()
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*workflowengine.WorkflowResult)
			out.Log = []map[string]any{{"msg": "ok", "result": "FINISHED"}}
		}).
		Return(nil).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "verifier-wf", "").Return(mockRun).Once()

	err = sendOpenIDNetConformanceLogUpdateStart(
		app,
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "verifier-wf"},
	)
	require.NoError(t, err)
	require.Equal(t, "verifier-wf"+workflows.OpenID4VPWalletSubscription, capturedSubscription)
	require.Len(t, capturedLogs, 1)
	require.Equal(t, "FINISHED", capturedLogs[0]["result"])

	mockClient.AssertExpectations(t)
	mockRun.AssertExpectations(t)
}

func TestSendEWCLikeLogUpdateStartFallsBackToDirectWorkflow(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"SignalWorkflow",
			mock.Anything,
			"ewc-wf-status",
			"",
			workflows.EwcStartCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.NotFound{Message: "workflow not found"}).
		Once()
	mockClient.
		On(
			"SignalWorkflow",
			mock.Anything,
			"ewc-wf",
			"",
			workflows.EwcStartCheckSignal,
			mock.Anything,
		).
		Return(nil).
		Once()

	err := sendEWCLikeLogUpdateStart(
		nil,
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "ewc-wf-status"},
	)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestEWCLikeWorkflowIDsDoesNotAppendStatusSuffix(t *testing.T) {
	require.Equal(t, []string{"ewc-wf"}, ewcLikeWorkflowIDs("ewc-wf"))
	require.Equal(t, []string{"ewc-wf-status", "ewc-wf"}, ewcLikeWorkflowIDs("ewc-wf-status"))
}

func TestSendEWCLikeLogUpdateStartReplaysCompletedDirectWorkflowLogs(t *testing.T) {
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
			"ewc-wf-status",
			"",
			workflows.EwcStartCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.NotFound{Message: "workflow not found"}).
		Once()
	mockClient.
		On(
			"SignalWorkflow",
			mock.Anything,
			"ewc-wf",
			"",
			workflows.EwcStartCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.NotFound{Message: "workflow execution already completed"}).
		Once()
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*workflowengine.WorkflowResult)
			out.Log = []map[string]any{{"message": "done", "level": "info"}}
		}).
		Return(nil).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "ewc-wf", "").Return(mockRun).Once()

	err = sendEWCLikeLogUpdateStart(
		app,
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "ewc-wf-status"},
	)
	require.NoError(t, err)
	require.Equal(t, "ewc-wf"+workflows.EWCSubscription, capturedSubscription)
	require.Len(t, capturedLogs, 1)
	require.Equal(t, "done", capturedLogs[0]["message"])
	mockClient.AssertExpectations(t)
	mockRun.AssertExpectations(t)
}

func TestSendEWCLikeLogUpdateStopFallsBackToDirectWorkflow(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"SignalWorkflow",
			mock.Anything,
			"ewc-wf-status",
			"",
			workflows.EwcStopCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.NotFound{Message: "workflow not found"}).
		Once()
	mockClient.
		On(
			"SignalWorkflow",
			mock.Anything,
			"ewc-wf",
			"",
			workflows.EwcStopCheckSignal,
			mock.Anything,
		).
		Return(nil).
		Once()

	err := sendEWCLikeLogUpdateStop(
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "ewc-wf-status"},
	)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestHandleSendOpenID4VPWalletLogUpdateSuccess(t *testing.T) {
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

	err = HandleSendOpenID4VPWalletLogUpdate()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "wf-3"+workflows.OpenID4VPWalletSubscription, capturedSubscription)
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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
