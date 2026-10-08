// SPDX-FileCopyrightText: 2025 Forkbomb BV
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

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
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

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/compliance/signal", nil)
	req = req.WithContext(
		context.WithValue(
			req.Context(),
			middlewares.ValidatedInputKey,
			HandleSendTemporalSignalInput{Namespace: "usera-s-organization"},
		),
	)
	rec := httptest.NewRecorder()

	err = HandleSendTemporalSignal()(&core.RequestEvent{
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

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	input := HandleSendTemporalSignalInput{
		WorkflowID: "wf-1",
		Namespace:  "usera-s-organization",
		Signal:     "sig",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/signal", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err = HandleSendTemporalSignal()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
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

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	input := HandleSendTemporalSignalInput{
		WorkflowID: "wf-2",
		Namespace:  "usera-s-organization",
		Signal:     "sig",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/signal", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()

	err = HandleSendTemporalSignal()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleSendTemporalSignalNamespaceAuthorization(t *testing.T) {
	scenarios := []struct {
		name           string
		collection     string
		email          string
		namespace      string
		expectedStatus int
	}{
		{
			name:           "other organization namespace is forbidden",
			collection:     "users",
			email:          "userA@example.org",
			namespace:      "userb-s-organization",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "superuser may signal any namespace",
			collection:     core.CollectionNameSuperusers,
			email:          "admin@example.org",
			namespace:      "userb-s-organization",
			expectedStatus: http.StatusOK,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			authRecord, err := app.FindAuthRecordByEmail(s.collection, s.email)
			require.NoError(t, err)

			origClient := complianceTemporalClient
			t.Cleanup(func() { complianceTemporalClient = origClient })

			mockClient := &temporalmocks.Client{}
			mockClient.
				On("SignalWorkflow", mock.Anything, "wf-b", "", "sig", mock.Anything).
				Return(nil).
				Maybe()
			var requestedNamespaces []string
			complianceTemporalClient = func(namespace string) (client.Client, error) {
				requestedNamespaces = append(requestedNamespaces, namespace)
				return mockClient, nil
			}

			input := HandleSendTemporalSignalInput{
				WorkflowID: "wf-b",
				Namespace:  s.namespace,
				Signal:     "sig",
			}
			req := httptest.NewRequest(http.MethodPost, "/api/compliance/signal", nil)
			req = req.WithContext(
				context.WithValue(req.Context(), middlewares.ValidatedInputKey, input),
			)
			rec := httptest.NewRecorder()

			err = HandleSendTemporalSignal()(&core.RequestEvent{
				App:  app,
				Auth: authRecord,
				Event: router.Event{
					Request:  req,
					Response: rec,
				},
			})
			if s.expectedStatus == http.StatusOK {
				require.NoError(t, err)
				require.Equal(t, []string{s.namespace}, requestedNamespaces)
				mockClient.AssertCalled(
					t,
					"SignalWorkflow",
					mock.Anything,
					"wf-b",
					"",
					"sig",
					mock.Anything,
				)
			} else {
				requireHandlerErrorHandled(t, rec, err)
				require.Empty(t, requestedNamespaces)
				mockClient.AssertNotCalled(
					t,
					"SignalWorkflow",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				)
			}
			require.Equal(t, s.expectedStatus, rec.Code)
		})
	}
}

func TestGetDeeplinkOpenIDConformanceSuite(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	captures := map[string]any{"deeplink": "link-1"}
	req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink", nil)
	rec := httptest.NewRecorder()

	err = getDeeplinkOpenIDConformanceSuite(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}, captures)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "link-1")
}

func TestGetDeeplinkEudiw(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	captures := map[string]any{
		"client_id":   "client-1",
		"request_uri": "https://example.com/req",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink", nil)
	rec := httptest.NewRecorder()

	err = getDeeplinkEudiw(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}, captures)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	expected, err := workflows.BuildQRDeepLink("client-1", "https://example.com/req")
	require.NoError(t, err)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, expected, body["deeplink"])
}

func TestGetWorkflowAuthorFromMemo(t *testing.T) {
	payload, err := temporalcrypto.DataConverter().ToPayload("ewc")
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

	payloads, err := temporalcrypto.DataConverter().ToPayloads(workflowengine.ActivityResult{
		Output: map[string]any{
			"captures": map[string]any{
				"deeplink": "ewc://link",
			},
		},
	})
	require.NoError(t, err)

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

func deeplinkHistoryIterator(
	t *testing.T,
	result workflowengine.ActivityResult,
) *fakeHistoryIterator {
	t.Helper()

	payloads, err := temporalcrypto.DataConverter().ToPayloads(result)
	require.NoError(t, err)
	return &fakeHistoryIterator{
		events: []*historypb.HistoryEvent{
			// Events without an activity result must be skipped.
			{EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
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
}

func describeWithAuthor(
	t *testing.T,
	author string,
) *workflowservice.DescribeWorkflowExecutionResponse {
	t.Helper()

	payload, err := temporalcrypto.DataConverter().ToPayload(author)
	require.NoError(t, err)
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
			Memo: &common.Memo{Fields: map[string]*common.Payload{"author": payload}},
		},
	}
}

func TestHandleDeeplink(t *testing.T) {
	openIDResult := workflowengine.ActivityResult{
		Output: map[string]any{"captures": map[string]any{"deeplink": "openid://link"}},
	}

	cases := []struct {
		name         string
		describe     *workflowservice.DescribeWorkflowExecutionResponse
		describeErr  error
		history      func(t *testing.T) client.HistoryEventIterator
		wantStatus   int
		wantReason   string
		wantDeeplink string
	}{
		{
			name:     "openid conformance suite deeplink from the run history",
			describe: describeWithAuthor(t, workflows.OpenIDConformanceSuite),
			history: func(t *testing.T) client.HistoryEventIterator {
				return deeplinkHistoryIterator(t, openIDResult)
			},
			wantStatus:   http.StatusOK,
			wantDeeplink: "openid://link",
		},
		{
			name:     "webuild suite returns the captured deeplink",
			describe: describeWithAuthor(t, workflows.WebuildSuite),
			history: func(t *testing.T) client.HistoryEventIterator {
				return deeplinkHistoryIterator(t, openIDResult)
			},
			wantStatus:   http.StatusOK,
			wantDeeplink: "openid://link",
		},
		{
			name:        "describe failure",
			describeErr: errors.New("temporal down"),
			wantStatus:  http.StatusInternalServerError,
			wantReason:  "failed to describe workflow execution",
		},
		{
			name:     "unsupported author",
			describe: describeWithAuthor(t, workflows.VLEISuite),
			history: func(t *testing.T) client.HistoryEventIterator {
				return deeplinkHistoryIterator(t, openIDResult)
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "unsupported suite",
		},
		{
			name:       "missing history",
			describe:   describeWithAuthor(t, workflows.EWCSuite),
			history:    func(*testing.T) client.HistoryEventIterator { return nil },
			wantStatus: http.StatusNotFound,
			wantReason: "workflow history not found",
		},
		{
			name:     "history iteration failure",
			describe: describeWithAuthor(t, workflows.EWCSuite),
			history: func(*testing.T) client.HistoryEventIterator {
				return &fakeHistoryIterator{nextErr: errors.New("history unavailable")}
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed to iterate workflow history",
		},
		{
			name:     "history without activity results",
			describe: describeWithAuthor(t, workflows.EWCSuite),
			history: func(*testing.T) client.HistoryEventIterator {
				return &fakeHistoryIterator{events: []*historypb.HistoryEvent{
					{EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
				}}
			},
			wantStatus: http.StatusNotFound,
			wantReason: "no matching activity found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)

			mockClient := &temporalmocks.Client{}
			mockClient.
				On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
				Return(tc.describe, tc.describeErr).
				Once()
			if tc.history != nil {
				mockClient.
					On(
						"GetWorkflowHistory",
						mock.Anything,
						"wf-1",
						"run-1",
						false,
						enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
					).
					Return(tc.history(t)).
					Once()
			}

			origClient := complianceTemporalClient
			t.Cleanup(func() { complianceTemporalClient = origClient })
			var requestedNamespace string
			complianceTemporalClient = func(namespace string) (client.Client, error) {
				requestedNamespace = namespace
				return mockClient, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink/wf-1/run-1", nil)
			req.SetPathValue("workflowId", "wf-1")
			req.SetPathValue("runId", "run-1")
			rec := httptest.NewRecorder()

			err = HandleDeeplink()(&core.RequestEvent{
				App:   app,
				Auth:  authRecord,
				Event: router.Event{Request: req, Response: rec},
			})
			requireHandlerErrorHandled(t, rec, err)
			require.Equal(t, "usera-s-organization", requestedNamespace)
			require.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantStatus == http.StatusOK {
				var body map[string]string
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
				require.Equal(t, tc.wantDeeplink, body["deeplink"])
			} else {
				require.Equal(t, tc.wantReason, decodeHandlerErrorResponse(t, rec).Error.Reason)
			}
			mockClient.AssertExpectations(t)
		})
	}
}

func performComplianceSignal(
	t *testing.T,
	app *tests.TestApp,
	auth *core.Record,
	input HandleSendTemporalSignalInput,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/compliance/send-temporal-signal", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()
	err := HandleSendTemporalSignal()(&core.RequestEvent{
		App:   app,
		Auth:  auth,
		Event: router.Event{Request: req, Response: rec},
	})
	requireHandlerErrorHandled(t, rec, err)
	return rec
}

func TestHandleSendTemporalSignalDispatch(t *testing.T) {
	t.Run("unauthenticated caller never reaches temporal", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		origClient := complianceTemporalClient
		t.Cleanup(func() { complianceTemporalClient = origClient })
		complianceTemporalClient = func(string) (client.Client, error) {
			t.Fatal("temporal client must not be created")
			return nil, nil
		}

		rec := performComplianceSignal(t, app, nil, HandleSendTemporalSignalInput{
			WorkflowID: "wf-1",
			Namespace:  "usera-s-organization",
			Signal:     "sig",
		})
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Equal(t, "authentication required", decodeHandlerErrorResponse(t, rec).Error.Reason)
	})

	cases := []struct {
		name       string
		signal     string
		setup      func(*temporalmocks.Client)
		clientErr  error
		wantStatus int
		wantReason string
	}{
		{
			name:       "temporal client creation failure",
			signal:     "sig",
			clientErr:  errors.New("dial failed"),
			wantStatus: http.StatusInternalServerError,
			wantReason: "unable to create client",
		},
		{
			name:       "issuer stop signal is acknowledged without contacting temporal",
			signal:     workflows.OpenID4VCIIssuerStopCheckSignal,
			wantStatus: http.StatusOK,
		},
		{
			name:       "verifier stop signal is acknowledged without contacting temporal",
			signal:     workflows.OpenID4VPVerifierStopCheckSignal,
			wantStatus: http.StatusOK,
		},
		{
			name:   "issuer start on a running workflow sends no log replay",
			signal: workflows.OpenID4VCIIssuerStartCheckSignal,
			setup: func(c *temporalmocks.Client) {
				c.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "").
					Return(&workflowservice.DescribeWorkflowExecutionResponse{
						WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
							Status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
						},
					}, nil).
					Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "verifier start on an unknown workflow is not found",
			signal: workflows.OpenID4VPVerifierStartCheckSignal,
			setup: func(c *temporalmocks.Client) {
				c.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "").
					Return(nil, &serviceerror.NotFound{Message: "missing"}).
					Once()
			},
			wantStatus: http.StatusNotFound,
			wantReason: "workflow not found",
		},
		{
			name:   "wallet start signal is forwarded to the workflow",
			signal: workflows.OpenID4VPWalletStartCheckSignal,
			setup: func(c *temporalmocks.Client) {
				c.On(
					"SignalWorkflow",
					mock.Anything,
					"wf-1",
					"",
					workflows.OpenID4VPWalletStartCheckSignal,
					mock.Anything,
				).Return(nil).Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "ewc start signal on an unknown workflow is not found",
			signal: workflows.EwcStartCheckSignal,
			setup: func(c *temporalmocks.Client) {
				c.On("SignalWorkflow", mock.Anything, "wf-1", "", workflows.EwcStartCheckSignal, mock.Anything).
					Return(&serviceerror.NotFound{Message: "missing"}).
					Once()
			},
			wantStatus: http.StatusNotFound,
			wantReason: "workflow not found",
		},
		{
			name:   "ewc stop signal on a completed workflow succeeds",
			signal: workflows.EwcStopCheckSignal,
			setup: func(c *temporalmocks.Client) {
				c.On("SignalWorkflow", mock.Anything, "wf-1", "", workflows.EwcStopCheckSignal, mock.Anything).
					Return(&serviceerror.NotFound{Message: "workflow execution already completed"}).
					Once()
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)

			mockClient := &temporalmocks.Client{}
			if tc.setup != nil {
				tc.setup(mockClient)
			}
			origClient := complianceTemporalClient
			t.Cleanup(func() { complianceTemporalClient = origClient })
			complianceTemporalClient = func(string) (client.Client, error) {
				if tc.clientErr != nil {
					return nil, tc.clientErr
				}
				return mockClient, nil
			}

			rec := performComplianceSignal(t, app, authRecord, HandleSendTemporalSignalInput{
				WorkflowID: "wf-1",
				Namespace:  "usera-s-organization",
				Signal:     tc.signal,
			})
			require.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantReason != "" {
				require.Equal(t, tc.wantReason, decodeHandlerErrorResponse(t, rec).Error.Reason)
			}
			// Unexpected temporal calls panic in the mock; expected ones must happen.
			mockClient.AssertExpectations(t)
		})
	}
}

func TestSendOpenID4VPWalletLogUpdateStartErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantReason string
	}{
		{
			name:       "unknown workflow",
			err:        &serviceerror.NotFound{Message: "workflow not found"},
			wantStatus: http.StatusNotFound,
			wantReason: "workflow not found",
		},
		{
			name:       "malformed workflow id",
			err:        &serviceerror.InvalidArgument{Message: "bad id"},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid workflow ID",
		},
		{
			name:       "transport failure",
			err:        errors.New("unavailable"),
			wantStatus: http.StatusBadRequest,
			wantReason: "failed to send start logs update signal",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &temporalmocks.Client{}
			mockClient.
				On(
					"SignalWorkflow",
					mock.Anything,
					"wf-log",
					"",
					workflows.OpenID4VPWalletStartCheckSignal,
					mock.Anything,
				).
				Return(tc.err).
				Once()

			err := sendOpenID4VPWalletLogUpdateStart(
				nil,
				mockClient,
				HandleSendTemporalSignalInput{WorkflowID: "wf-log"},
			)
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, tc.wantStatus, apiErr.Code)
			require.Equal(t, tc.wantReason, apiErr.Reason)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestSendOpenID4VPWalletLogUpdateStartCanceledReplaysLogs(t *testing.T) {
	origNotify := complianceNotifyLogsUpdate
	t.Cleanup(func() { complianceNotifyLogsUpdate = origNotify })
	var capturedSubscription string
	var capturedLogs []map[string]any
	complianceNotifyLogsUpdate = func(_ core.App, subscription string, logs []map[string]any) error {
		capturedSubscription = subscription
		capturedLogs = logs
		return nil
	}

	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockClient.
		On(
			"SignalWorkflow",
			mock.Anything,
			"wf-3-log",
			"",
			workflows.OpenID4VPWalletStartCheckSignal,
			mock.Anything,
		).
		Return(&serviceerror.Canceled{Message: "canceled"}).
		Once()
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*workflowengine.WorkflowResult)
			out.Output = map[string]any{"logs": []any{map[string]any{"src": "check"}}}
		}).
		Return(nil).
		Once()
	mockClient.On("GetWorkflow", mock.Anything, "wf-3-log", "").Return(mockRun).Once()

	require.NoError(t, sendOpenID4VPWalletLogUpdateStart(
		nil,
		mockClient,
		HandleSendTemporalSignalInput{WorkflowID: "wf-3-log"},
	))
	require.Equal(t, "wf-3"+workflows.OpenID4VPWalletSubscription, capturedSubscription)
	require.Equal(t, []map[string]any{{"src": "check"}}, capturedLogs)
	mockClient.AssertExpectations(t)
	mockRun.AssertExpectations(t)
}

func TestSendOpenIDNetConformanceLogUpdateStartDescribeErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantReason string
	}{
		{
			name:       "malformed workflow id",
			err:        &serviceerror.InvalidArgument{Message: "bad id"},
			wantReason: "invalid workflow ID",
		},
		{
			name:       "transport failure",
			err:        errors.New("unavailable"),
			wantReason: "failed to describe OpenIDNet workflow",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &temporalmocks.Client{}
			mockClient.
				On("DescribeWorkflowExecution", mock.Anything, "wf-1", "").
				Return(nil, tc.err).
				Once()

			err := sendOpenIDNetConformanceLogUpdateStart(
				nil,
				mockClient,
				HandleSendTemporalSignalInput{WorkflowID: "wf-1"},
			)
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.Code)
			require.Equal(t, tc.wantReason, apiErr.Reason)
		})
	}
}

func TestSendEWCLikeLogUpdateErrors(t *testing.T) {
	cases := []struct {
		name       string
		signal     string
		send       func(*temporalmocks.Client) error
		errs       map[string]error
		wantStatus int
		wantReason string
	}{
		{
			name:   "start reports not found when neither id exists",
			signal: workflows.EwcStartCheckSignal,
			send: func(c *temporalmocks.Client) error {
				return sendEWCLikeLogUpdateStart(
					nil,
					c,
					HandleSendTemporalSignalInput{WorkflowID: "ewc-status"},
				)
			},
			errs: map[string]error{
				"ewc-status": &serviceerror.NotFound{Message: "missing status"},
				"ewc":        &serviceerror.NotFound{Message: "missing direct"},
			},
			wantStatus: http.StatusNotFound,
			wantReason: "workflow not found",
		},
		{
			name:   "start stops at a non not-found error",
			signal: workflows.EwcStartCheckSignal,
			send: func(c *temporalmocks.Client) error {
				return sendEWCLikeLogUpdateStart(
					nil,
					c,
					HandleSendTemporalSignalInput{WorkflowID: "ewc-status"},
				)
			},
			errs: map[string]error{
				"ewc-status": &serviceerror.InvalidArgument{Message: "bad"},
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid workflow ID",
		},
		{
			name:   "stop reports not found when neither id exists",
			signal: workflows.EwcStopCheckSignal,
			send: func(c *temporalmocks.Client) error {
				return sendEWCLikeLogUpdateStop(
					c,
					HandleSendTemporalSignalInput{WorkflowID: "ewc-status"},
				)
			},
			errs: map[string]error{
				"ewc-status": &serviceerror.NotFound{Message: "missing status"},
				"ewc":        &serviceerror.NotFound{Message: "missing direct"},
			},
			wantStatus: http.StatusNotFound,
			wantReason: "workflow not found",
		},
		{
			name:   "stop surfaces transport failures",
			signal: workflows.EwcStopCheckSignal,
			send: func(c *temporalmocks.Client) error {
				return sendEWCLikeLogUpdateStop(c, HandleSendTemporalSignalInput{WorkflowID: "ewc"})
			},
			errs:       map[string]error{"ewc": errors.New("unavailable")},
			wantStatus: http.StatusBadRequest,
			wantReason: "failed to send signal: " + workflows.EwcStopCheckSignal,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &temporalmocks.Client{}
			for workflowID, err := range tc.errs {
				mockClient.
					On("SignalWorkflow", mock.Anything, workflowID, "", tc.signal, mock.Anything).
					Return(err).
					Once()
			}

			var apiErr *apierror.APIError
			require.ErrorAs(t, tc.send(mockClient), &apiErr)
			require.Equal(t, tc.wantStatus, apiErr.Code)
			require.Equal(t, tc.wantReason, apiErr.Reason)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestSendCompletedWorkflowLogsUpdate(t *testing.T) {
	newClient := func(result workflowengine.WorkflowResult) *temporalmocks.Client {
		mockClient := &temporalmocks.Client{}
		mockRun := &temporalmocks.WorkflowRun{}
		mockRun.
			On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
			Run(func(args mock.Arguments) {
				*args.Get(1).(*workflowengine.WorkflowResult) = result
			}).
			Return(nil).
			Once()
		mockClient.On("GetWorkflow", mock.Anything, "wf-1", "").Return(mockRun).Once()
		return mockClient
	}
	origNotify := complianceNotifyLogsUpdate
	t.Cleanup(func() { complianceNotifyLogsUpdate = origNotify })

	t.Run("no recognizable logs skips the realtime notification", func(t *testing.T) {
		complianceNotifyLogsUpdate = func(core.App, string, []map[string]any) error {
			t.Fatal("notification must not be sent without logs")
			return nil
		}
		err := sendCompletedWorkflowLogsUpdate(
			nil,
			newClient(workflowengine.WorkflowResult{Output: map[string]any{"status": "ok"}}),
			"wf-1",
			"sub",
			extractEWCLikeLogsFromResultOrError,
		)
		require.NoError(t, err)
	})

	t.Run("notification failure is reported", func(t *testing.T) {
		complianceNotifyLogsUpdate = func(core.App, string, []map[string]any) error {
			return errors.New("broker down")
		}
		err := sendCompletedWorkflowLogsUpdate(
			nil,
			newClient(workflowengine.WorkflowResult{
				Log: []any{map[string]any{"message": "done", "level": "info"}},
			}),
			"wf-1",
			"sub",
			extractEWCLikeLogsFromResultOrError,
		)
		var apiErr *apierror.APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.Code)
		require.Equal(t, "failed to send realtime logs update", apiErr.Reason)
		require.Equal(t, "broker down", apiErr.Message)
	})
}

func TestExtractEWCLikeLogsFromPayload(t *testing.T) {
	entry := map[string]any{"message": "step", "metadata": map[string]any{}}
	cases := []struct {
		name    string
		payload any
		want    []map[string]any
	}{
		{name: "nil payload", payload: nil},
		{name: "direct log list", payload: []any{entry}, want: []map[string]any{entry}},
		{
			name:    "first matching element of a payload list",
			payload: []any{"noise", map[string]any{"logs": []any{entry}}},
			want:    []map[string]any{entry},
		},
		{
			name:    "nested payload",
			payload: map[string]any{"payload": map[string]any{"logs": []any{entry}}},
			want:    []map[string]any{entry},
		},
		{
			name: "logs_response wrapper",
			payload: map[string]any{
				"logs_response": map[string]any{"logs": []any{entry}},
			},
			want: []map[string]any{entry},
		},
		{
			name:    "entries need a message",
			payload: []any{map[string]any{"level": "info"}},
		},
		{
			name:    "message alone is not an ewc log",
			payload: map[string]any{"logs": []any{map[string]any{"message": "plain"}}},
		},
		{
			name:    "data marks an ewc log",
			payload: []any{map[string]any{"message": "m", "data": 1}},
			want:    []map[string]any{{"message": "m", "data": 1}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractEWCLikeLogsFromPayload(tc.payload))
		})
	}
}

func TestExtractOpenIDNetLogsFromPayload(t *testing.T) {
	entry := map[string]any{"src": "check", "result": "SUCCESS"}
	cases := []struct {
		name    string
		payload any
		want    []map[string]any
	}{
		{name: "nil payload", payload: nil},
		{name: "direct log list", payload: []any{entry}, want: []map[string]any{entry}},
		{
			name:    "first matching element of a payload list",
			payload: []any{1, map[string]any{"logs": []any{entry}}},
			want:    []map[string]any{entry},
		},
		{
			name:    "nested payload",
			payload: map[string]any{"payload": []any{entry}},
			want:    []map[string]any{entry},
		},
		{
			name:    "entries without msg, src or result are ignored",
			payload: map[string]any{"logs": []any{map[string]any{"message": "ewc style"}}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractOpenIDNetLogsFromPayload(tc.payload))
		})
	}
}

func TestExtractLogsFromResultOrErrorPrecedence(t *testing.T) {
	logEntry := map[string]any{"msg": "from log", "message": "from log", "level": "info"}
	outputEntry := map[string]any{"msg": "from output", "message": "from output", "level": "info"}
	errorEntry := map[string]any{"msg": "from error", "message": "from error", "level": "info"}
	workflowErr := workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    "CRE999",
		Summary: "failed",
		Details: map[string]any{"payload": []any{errorEntry}},
	})

	extractors := map[string]workflowLogsExtractor{
		"openidnet": extractOpenIDNetLogsFromResultOrError,
		"ewc":       extractEWCLikeLogsFromResultOrError,
	}
	for name, extract := range extractors {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, "from log", extract(workflowengine.WorkflowResult{
				Log:    []any{logEntry},
				Output: []any{outputEntry},
			}, workflowErr)[0]["msg"])
			require.Equal(t, "from output", extract(workflowengine.WorkflowResult{
				Output: []any{outputEntry},
			}, workflowErr)[0]["msg"])
			require.Equal(
				t,
				"from error",
				extract(workflowengine.WorkflowResult{}, workflowErr)[0]["msg"],
			)
			require.Nil(t, extract(workflowengine.WorkflowResult{}, nil))
		})
	}
}
