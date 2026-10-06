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

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
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
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func covAPIRequireAPIError(t *testing.T, err error, code int, reason string) {
	t.Helper()
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, code, apiErr.Code)
	assert.Equal(t, reason, apiErr.Reason)
}

func covAPIStubComplianceClient(t *testing.T, c client.Client, clientErr error) {
	t.Helper()
	orig := complianceTemporalClient
	t.Cleanup(func() { complianceTemporalClient = orig })
	complianceTemporalClient = func(string) (client.Client, error) {
		if clientErr != nil {
			return nil, clientErr
		}
		return c, nil
	}
}

func covAPICaptureNotify(t *testing.T, notifyErr error) *[]string {
	t.Helper()
	orig := complianceNotifyLogsUpdate
	t.Cleanup(func() { complianceNotifyLogsUpdate = orig })
	var subscriptions []string
	complianceNotifyLogsUpdate = func(_ core.App, subscription string, _ []map[string]any) error {
		subscriptions = append(subscriptions, subscription)
		return notifyErr
	}
	return &subscriptions
}

func covAPISignalEvent(
	app core.App,
	auth *core.Record,
	input HandleSendTemporalSignalInput,
) (*core.RequestEvent, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodPost, "/api/compliance/send-temporal-signal", nil)
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, input))
	rec := httptest.NewRecorder()
	return &core.RequestEvent{
		App:   app,
		Auth:  auth,
		Event: router.Event{Request: req, Response: rec},
	}, rec
}

func TestHandleSendTemporalSignalDispatchesBySignal(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	running := &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
			Status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		},
	}

	scenarios := []struct {
		name   string
		signal string
		setup  func(m *temporalmocks.Client)
	}{
		{
			name:   "openid4vp wallet start signals the log workflow",
			signal: workflows.OpenID4VPWalletStartCheckSignal,
			setup: func(m *temporalmocks.Client) {
				m.On("SignalWorkflow", mock.Anything, "wf-1", "",
					workflows.OpenID4VPWalletStartCheckSignal, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:   "issuer start only describes a running workflow",
			signal: workflows.OpenID4VCIIssuerStartCheckSignal,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "").
					Return(running, nil).Once()
			},
		},
		{
			name:   "verifier start only describes a running workflow",
			signal: workflows.OpenID4VPVerifierStartCheckSignal,
			setup: func(m *temporalmocks.Client) {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "").
					Return(running, nil).Once()
			},
		},
		{
			name:   "issuer stop is a no-op",
			signal: workflows.OpenID4VCIIssuerStopCheckSignal,
			setup:  func(*temporalmocks.Client) {},
		},
		{
			name:   "verifier stop is a no-op",
			signal: workflows.OpenID4VPVerifierStopCheckSignal,
			setup:  func(*temporalmocks.Client) {},
		},
		{
			name:   "ewc start signals the workflow",
			signal: workflows.EwcStartCheckSignal,
			setup: func(m *temporalmocks.Client) {
				m.On("SignalWorkflow", mock.Anything, "wf-1", "",
					workflows.EwcStartCheckSignal, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:   "ewc stop signals the workflow",
			signal: workflows.EwcStopCheckSignal,
			setup: func(m *temporalmocks.Client) {
				m.On("SignalWorkflow", mock.Anything, "wf-1", "",
					workflows.EwcStopCheckSignal, mock.Anything).Return(nil).Once()
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			s.setup(m)
			covAPIStubComplianceClient(t, m, nil)

			e, rec := covAPISignalEvent(app, auth, HandleSendTemporalSignalInput{
				WorkflowID: "wf-1",
				Namespace:  "usera-s-organization",
				Signal:     s.signal,
			})
			require.NoError(t, HandleSendTemporalSignal()(e))
			require.Equal(t, http.StatusOK, rec.Code)
			assert.Contains(t, rec.Body.String(), "Signal sent successfully")
		})
	}
}

func TestHandleSendTemporalSignalRejectsBeforeSignalling(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	scenarios := []struct {
		name      string
		auth      *core.Record
		clientErr error
		code      int
		reason    string
	}{
		{
			name:   "unauthenticated",
			code:   http.StatusUnauthorized,
			reason: "authentication required",
		},
		{
			name:      "temporal client failure",
			auth:      auth,
			clientErr: errors.New("dial failed"),
			code:      http.StatusInternalServerError,
			reason:    "unable to create client",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			covAPIStubComplianceClient(t, temporalmocks.NewClient(t), s.clientErr)
			e, _ := covAPISignalEvent(app, s.auth, HandleSendTemporalSignalInput{
				WorkflowID: "wf-1",
				Namespace:  "usera-s-organization",
				Signal:     "sig",
			})
			covAPIRequireAPIError(t, HandleSendTemporalSignal()(e), s.code, s.reason)
		})
	}
}

func TestSendTemporalSignalGenericFailure(t *testing.T) {
	m := temporalmocks.NewClient(t)
	m.On("SignalWorkflow", mock.Anything, "wf-1", "", "sig", mock.Anything).
		Return(errors.New("unavailable")).Once()

	err := sendTemporalSignal(m, HandleSendTemporalSignalInput{WorkflowID: "wf-1", Signal: "sig"})
	covAPIRequireAPIError(t, err, http.StatusBadRequest, "failed to send signal: sig")
}

func TestSendOpenID4VPWalletLogUpdateStartErrorMapping(t *testing.T) {
	scenarios := []struct {
		name   string
		err    error
		code   int
		reason string
	}{
		{
			name:   "workflow missing",
			err:    &serviceerror.NotFound{Message: "workflow not found"},
			code:   http.StatusNotFound,
			reason: "workflow not found",
		},
		{
			name:   "invalid workflow id",
			err:    &serviceerror.InvalidArgument{Message: "bad id"},
			code:   http.StatusBadRequest,
			reason: "invalid workflow ID",
		},
		{
			name:   "other signal error",
			err:    errors.New("unavailable"),
			code:   http.StatusBadRequest,
			reason: "failed to send start logs update signal",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			m.On("SignalWorkflow", mock.Anything, "wf-log", "",
				workflows.OpenID4VPWalletStartCheckSignal, mock.Anything).Return(s.err).Once()

			err := sendOpenID4VPWalletLogUpdateStart(
				nil,
				m,
				HandleSendTemporalSignalInput{WorkflowID: "wf-log"},
			)
			covAPIRequireAPIError(t, err, s.code, s.reason)
		})
	}
}

func TestSendOpenID4VPWalletLogUpdateStartCanceledReplaysLogs(t *testing.T) {
	subscriptions := covAPICaptureNotify(t, nil)

	m := temporalmocks.NewClient(t)
	run := temporalmocks.NewWorkflowRun(t)
	m.On("SignalWorkflow", mock.Anything, "wf-9-log", "",
		workflows.OpenID4VPWalletStartCheckSignal, mock.Anything).
		Return(&serviceerror.Canceled{Message: "canceled"}).Once()
	m.On("GetWorkflow", mock.Anything, "wf-9-log", "").Return(run).Once()
	run.On("Get", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*workflowengine.WorkflowResult)
			out.Output = map[string]any{"logs": []any{map[string]any{"src": "runner"}}}
		}).
		Return(nil).Once()

	require.NoError(t, sendOpenID4VPWalletLogUpdateStart(
		nil,
		m,
		HandleSendTemporalSignalInput{WorkflowID: "wf-9-log"},
	))
	assert.Equal(t, []string{"wf-9" + workflows.OpenID4VPWalletSubscription}, *subscriptions)
}

func TestSendOpenIDNetConformanceLogUpdateStartDescribeErrors(t *testing.T) {
	scenarios := []struct {
		name   string
		err    error
		code   int
		reason string
	}{
		{
			name:   "workflow missing",
			err:    &serviceerror.NotFound{Message: "missing"},
			code:   http.StatusNotFound,
			reason: "workflow not found",
		},
		{
			name:   "invalid workflow id",
			err:    &serviceerror.InvalidArgument{Message: "bad"},
			code:   http.StatusBadRequest,
			reason: "invalid workflow ID",
		},
		{
			name:   "other describe error",
			err:    errors.New("unavailable"),
			code:   http.StatusBadRequest,
			reason: "failed to describe OpenIDNet workflow",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "").
				Return(nil, s.err).Once()

			err := sendOpenIDNetConformanceLogUpdateStart(
				nil,
				m,
				HandleSendTemporalSignalInput{WorkflowID: "wf-1"},
			)
			covAPIRequireAPIError(t, err, s.code, s.reason)
		})
	}
}

func TestSendCompletedWorkflowLogsUpdate(t *testing.T) {
	scenarios := []struct {
		name          string
		result        workflowengine.WorkflowResult
		notifyErr     error
		wantNotified  bool
		wantErrReason string
	}{
		{
			name:   "no logs skips notification",
			result: workflowengine.WorkflowResult{Output: map[string]any{"other": 1}},
		},
		{
			name: "notification failure maps to bad request",
			result: workflowengine.WorkflowResult{
				Log: []any{map[string]any{"msg": "done"}},
			},
			notifyErr:     errors.New("broker down"),
			wantNotified:  true,
			wantErrReason: "failed to send realtime logs update",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			subscriptions := covAPICaptureNotify(t, s.notifyErr)
			m := temporalmocks.NewClient(t)
			run := temporalmocks.NewWorkflowRun(t)
			m.On("GetWorkflow", mock.Anything, "wf-1", "").Return(run).Once()
			run.On("Get", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					*args.Get(1).(*workflowengine.WorkflowResult) = s.result
				}).
				Return(nil).Once()

			err := sendCompletedWorkflowLogsUpdate(
				nil,
				m,
				"wf-1",
				"topic",
				extractOpenIDNetLogsFromResultOrError,
			)
			if s.wantErrReason != "" {
				covAPIRequireAPIError(t, err, http.StatusBadRequest, s.wantErrReason)
			} else {
				require.NoError(t, err)
			}
			if s.wantNotified {
				assert.Equal(t, []string{"topic"}, *subscriptions)
			} else {
				assert.Empty(t, *subscriptions)
			}
		})
	}
}

func TestSendEWCLikeLogUpdateStartOutcomes(t *testing.T) {
	t.Run("canceled replays logs on the ewc topic", func(t *testing.T) {
		subscriptions := covAPICaptureNotify(t, nil)
		m := temporalmocks.NewClient(t)
		run := temporalmocks.NewWorkflowRun(t)
		m.On("SignalWorkflow", mock.Anything, "ewc-1-status", "",
			workflows.EwcStartCheckSignal, mock.Anything).
			Return(&serviceerror.Canceled{Message: "canceled"}).Once()
		m.On("GetWorkflow", mock.Anything, "ewc-1-status", "").Return(run).Once()
		run.On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				out := args.Get(1).(*workflowengine.WorkflowResult)
				out.Log = []any{map[string]any{"message": "m", "level": "info"}}
			}).
			Return(nil).Once()

		require.NoError(t, sendEWCLikeLogUpdateStart(
			nil,
			m,
			HandleSendTemporalSignalInput{WorkflowID: "ewc-1-status"},
		))
		assert.Equal(t, []string{"ewc-1" + workflows.EWCSubscription}, *subscriptions)
	})

	t.Run("non not-found error stops fallback", func(t *testing.T) {
		m := temporalmocks.NewClient(t)
		m.On("SignalWorkflow", mock.Anything, "ewc-1-status", "",
			workflows.EwcStartCheckSignal, mock.Anything).
			Return(errors.New("unavailable")).Once()

		err := sendEWCLikeLogUpdateStart(
			nil,
			m,
			HandleSendTemporalSignalInput{WorkflowID: "ewc-1-status"},
		)
		covAPIRequireAPIError(
			t,
			err,
			http.StatusBadRequest,
			"failed to send signal: "+workflows.EwcStartCheckSignal,
		)
	})

	t.Run("all candidates missing is not found", func(t *testing.T) {
		m := temporalmocks.NewClient(t)
		m.On("SignalWorkflow", mock.Anything, mock.Anything, "",
			workflows.EwcStartCheckSignal, mock.Anything).
			Return(&serviceerror.NotFound{Message: "missing"}).Twice()

		err := sendEWCLikeLogUpdateStart(
			nil,
			m,
			HandleSendTemporalSignalInput{WorkflowID: "ewc-1-status"},
		)
		covAPIRequireAPIError(t, err, http.StatusNotFound, "workflow not found")
	})
}

func TestSendEWCLikeLogUpdateStopOutcomes(t *testing.T) {
	scenarios := []struct {
		name      string
		err       error
		times     int
		wantCode  int
		wantError string
	}{
		{
			name:  "already completed is success",
			err:   &serviceerror.NotFound{Message: "workflow execution already completed"},
			times: 1,
		},
		{
			name:      "invalid workflow id",
			err:       &serviceerror.InvalidArgument{Message: "bad"},
			times:     1,
			wantCode:  http.StatusBadRequest,
			wantError: "invalid workflow ID",
		},
		{
			name:      "all candidates missing",
			err:       &serviceerror.NotFound{Message: "missing"},
			times:     2,
			wantCode:  http.StatusNotFound,
			wantError: "workflow not found",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			m.On("SignalWorkflow", mock.Anything, mock.Anything, "",
				workflows.EwcStopCheckSignal, mock.Anything).Return(s.err).Times(s.times)

			err := sendEWCLikeLogUpdateStop(
				m,
				HandleSendTemporalSignalInput{WorkflowID: "ewc-2-status"},
			)
			if s.wantError == "" {
				require.NoError(t, err)
				return
			}
			covAPIRequireAPIError(t, err, s.wantCode, s.wantError)
		})
	}
}

func TestExtractEWCLikeLogsFromResultOrError(t *testing.T) {
	entry := map[string]any{"message": "step", "timestamp": "2026-01-01T00:00:00Z"}
	scenarios := []struct {
		name    string
		result  workflowengine.WorkflowResult
		err     error
		wantMsg string
	}{
		{
			name:    "output logs key",
			result:  workflowengine.WorkflowResult{Output: map[string]any{"logs": []any{entry}}},
			wantMsg: "step",
		},
		{
			name: "output logs_response",
			result: workflowengine.WorkflowResult{Output: map[string]any{
				"logs_response": map[string]any{
					"logs": []any{map[string]any{"message": "nested", "metadata": 1}},
				},
			}},
			wantMsg: "nested",
		},
		{
			name: "output nested payload",
			result: workflowengine.WorkflowResult{Output: map[string]any{
				"payload": []any{map[string]any{"message": "deep", "data": "x"}},
			}},
			wantMsg: "deep",
		},
		{
			name: "list of payloads",
			result: workflowengine.WorkflowResult{Output: []any{
				"ignored",
				map[string]any{"logs": []any{map[string]any{"message": "item", "level": "info"}}},
			}},
			wantMsg: "item",
		},
		{
			name: "workflow error details",
			err: workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    "CRE999",
				Summary: "failed",
				Details: map[string]any{"logs": []any{entry}},
			}),
			wantMsg: "step",
		},
		{
			name: "entries without message are not logs",
			result: workflowengine.WorkflowResult{
				Log: []any{map[string]any{"timestamp": "t"}},
			},
		},
		{
			name: "message alone is not an ewc log",
			result: workflowengine.WorkflowResult{
				Output: map[string]any{"logs": []any{map[string]any{"message": "plain"}}},
			},
		},
		{
			name: "no logs and no error",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			logs := extractEWCLikeLogsFromResultOrError(s.result, s.err)
			if s.wantMsg == "" {
				assert.Empty(t, logs)
				return
			}
			require.NotEmpty(t, logs)
			assert.Equal(t, s.wantMsg, logs[0]["message"])
		})
	}
}

func TestExtractOpenIDNetLogsFromResultOrError(t *testing.T) {
	scenarios := []struct {
		name    string
		result  workflowengine.WorkflowResult
		err     error
		wantKey string
	}{
		{
			name: "output nested payload with src",
			result: workflowengine.WorkflowResult{Output: map[string]any{
				"payload": map[string]any{"logs": []any{map[string]any{"src": "a"}}},
			}},
			wantKey: "src",
		},
		{
			name: "list of payloads with result",
			result: workflowengine.WorkflowResult{Output: []any{
				"ignored",
				[]any{map[string]any{"result": "PASSED"}},
			}},
			wantKey: "result",
		},
		{
			name: "workflow error payload",
			err: workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    "CRE999",
				Summary: "failed",
				Details: map[string]any{"payload": []any{map[string]any{"msg": "x"}}},
			}),
			wantKey: "msg",
		},
		{
			name: "unrelated entries are ignored",
			result: workflowengine.WorkflowResult{
				Output: map[string]any{"logs": []any{map[string]any{"other": 1}}},
			},
		},
		{
			name: "unstructured error",
			err:  errors.New("plain"),
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			logs := extractOpenIDNetLogsFromResultOrError(s.result, s.err)
			if s.wantKey == "" {
				assert.Empty(t, logs)
				return
			}
			require.NotEmpty(t, logs)
			assert.Contains(t, logs[0], s.wantKey)
		})
	}
}

func covAPIAuthorDescription(
	t *testing.T,
	author string,
) *workflowservice.DescribeWorkflowExecutionResponse {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(author)
	require.NoError(t, err)
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
			Memo: &common.Memo{Fields: map[string]*common.Payload{"author": payload}},
		},
	}
}

func covAPICompletedActivityEvent(t *testing.T, output map[string]any) *historypb.HistoryEvent {
	t.Helper()
	data, err := json.Marshal(map[string]any{"Output": output})
	require.NoError(t, err)
	return &historypb.HistoryEvent{
		EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
		Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
			ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
				Result: &common.Payloads{Payloads: []*common.Payload{{Data: data}}},
			},
		},
	}
}

func TestHandleDeeplinkResolvesFromHistory(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	auth, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	eudiwLink, err := workflows.BuildQRDeepLink("client-1", "https://verifier.test/req")
	require.NoError(t, err)

	scenarios := []struct {
		name        string
		author      string
		describeErr error
		events      []*historypb.HistoryEvent
		iterErr     error
		code        int
		reason      string
		deeplink    string
	}{
		{
			name:   "openid conformance suite",
			author: workflows.OpenIDConformanceSuite,
			events: []*historypb.HistoryEvent{
				{EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
				covAPICompletedActivityEvent(t, map[string]any{
					"captures": map[string]any{"deeplink": "openid://link"},
				}),
			},
			code:     http.StatusOK,
			deeplink: "openid://link",
		},
		{
			name:   "webuild suite",
			author: workflows.WebuildSuite,
			events: []*historypb.HistoryEvent{covAPICompletedActivityEvent(t, map[string]any{
				"captures": map[string]any{"deeplink": "webuild://link"},
			})},
			code:     http.StatusOK,
			deeplink: "webuild://link",
		},
		{
			name:   "eudiw suite builds QR deeplink",
			author: workflows.EudiwSuite,
			events: []*historypb.HistoryEvent{covAPICompletedActivityEvent(t, map[string]any{
				"captures": map[string]any{
					"client_id":   "client-1",
					"request_uri": "https://verifier.test/req",
				},
			})},
			code:     http.StatusOK,
			deeplink: eudiwLink,
		},
		{
			name:   "unsupported author",
			author: "someone-else",
			events: []*historypb.HistoryEvent{covAPICompletedActivityEvent(t, map[string]any{})},
			code:   http.StatusBadRequest,
			reason: "unsupported suite",
		},
		{
			name:   "no completed activity",
			author: workflows.EWCSuite,
			events: []*historypb.HistoryEvent{
				{EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
			},
			code:   http.StatusNotFound,
			reason: "no matching activity found",
		},
		{
			name:    "history iteration error",
			author:  workflows.EWCSuite,
			iterErr: errors.New("history unavailable"),
			code:    http.StatusInternalServerError,
			reason:  "failed to iterate workflow history",
		},
		{
			name:        "describe failure",
			describeErr: errors.New("unavailable"),
			code:        http.StatusInternalServerError,
			reason:      "failed to describe workflow execution",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			m := temporalmocks.NewClient(t)
			if s.describeErr != nil {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(nil, s.describeErr).Once()
			} else {
				m.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
					Return(covAPIAuthorDescription(t, s.author), nil).Once()
				m.On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false,
					enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
					Return(&fakeHistoryIterator{events: s.events, nextErr: s.iterErr}).Once()
			}
			var namespaces []string
			orig := complianceTemporalClient
			t.Cleanup(func() { complianceTemporalClient = orig })
			complianceTemporalClient = func(namespace string) (client.Client, error) {
				namespaces = append(namespaces, namespace)
				return m, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/api/compliance/deeplink/wf-1/run-1", nil)
			req.SetPathValue("workflowId", "wf-1")
			req.SetPathValue("runId", "run-1")
			rec := httptest.NewRecorder()
			err := HandleDeeplink()(&core.RequestEvent{
				App:   app,
				Auth:  auth,
				Event: router.Event{Request: req, Response: rec},
			})
			assert.Equal(t, []string{"usera-s-organization"}, namespaces)

			if s.code != http.StatusOK {
				covAPIRequireAPIError(t, err, s.code, s.reason)
				return
			}
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, s.deeplink, body["deeplink"])
		})
	}
}

func TestGetDeeplinkFromYAMLRejectsUnusableWorkflowOutput(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	scenarios := []struct {
		name      string
		startErr  error
		clientErr error
		output    any
		reason    string
		message   string
	}{
		{
			name:     "start failure",
			startErr: errors.New("no worker"),
			reason:   "failed to start get deeplink check",
			message:  "no worker",
		},
		{
			name:      "temporal unavailable",
			clientErr: errors.New("no temporal"),
			reason:    "failed to get temporal client",
			message:   "no temporal",
		},
		{
			name:    "output not a list",
			output:  map[string]any{},
			reason:  "failed to get workflow output",
			message: "output is not an array",
		},
		{
			name:    "empty output",
			output:  []any{},
			reason:  "failed to get workflow output",
			message: "output is empty",
		},
		{
			name:    "no steps",
			output:  []any{map[string]any{"steps": []any{}}},
			reason:  "failed to get workflow output",
			message: "steps are not present or empty",
		},
		{
			name:    "no captures",
			output:  []any{map[string]any{"steps": []any{map[string]any{}}}},
			reason:  "failed to get workflow output",
			message: "captures are not present in step",
		},
		{
			name: "empty deeplink",
			output: []any{map[string]any{"steps": []any{
				map[string]any{"captures": map[string]any{"deeplink": ""}},
			}}},
			reason:  "failed to get workflow output",
			message: "deeplink is not present in captures",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			origStart := deeplinkStartWorkflow
			origClient := deeplinkTemporalClient
			origWait := deeplinkWaitForWorkflowResult
			t.Cleanup(func() {
				deeplinkStartWorkflow = origStart
				deeplinkTemporalClient = origClient
				deeplinkWaitForWorkflowResult = origWait
			})
			deeplinkStartWorkflow = func(workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				return workflowengine.WorkflowResult{
					WorkflowID:    "wf",
					WorkflowRunID: "run",
				}, s.startErr
			}
			deeplinkTemporalClient = func(string) (client.Client, error) {
				if s.clientErr != nil {
					return nil, s.clientErr
				}
				return temporalmocks.NewClient(t), nil
			}
			deeplinkWaitForWorkflowResult = func(_ client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
				assert.Equal(t, "wf", workflowID)
				assert.Equal(t, "run", runID)
				return workflowengine.WorkflowResult{Output: s.output}, nil
			}

			_, err := getDeeplinkFromYAML(app, "yaml", nil, false)
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, s.reason, apiErr.Reason)
			assert.Equal(t, s.message, apiErr.Message)
		})
	}
}
