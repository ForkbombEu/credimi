// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func setupNewsAgentApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	NewsAgentInternalRoutes.Add(app)
	seedInternalAdminKey(t, app)

	return app
}

func useNewsAgentScheduleClient(scheduleClient *fakeScheduleClient) {
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(scheduleClient)
	mockClient.On("Close").Return()
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}
}

func newsAgentRequest(method, body string, withKey bool) *http.Request {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, "/api/news-agent/schedule", nil)
	} else {
		req = httptest.NewRequest(method, "/api/news-agent/schedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if withKey {
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
	}
	return req
}

func TestNewsAgentScheduleNewsScan(t *testing.T) {
	app := setupNewsAgentApp(t)
	defer app.Cleanup()

	originalScheduleTemporalClient := scheduleTemporalClient
	defer func() { scheduleTemporalClient = originalScheduleTemporalClient }()

	mux := buildAppMux(t, app)

	t.Run("success - defaults create schedule without trigger", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		scheduleClient := &fakeScheduleClient{handle: mockHandle}
		useNewsAgentScheduleClient(scheduleClient)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newsAgentRequest(http.MethodPost, `{}`, true))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var response ScheduleNewsScanResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.Equal(t, workflows.NewsDailyScanScheduleID, response.ScheduleID)
		require.Equal(t, workflows.DefaultNamespace, response.WorkflowNamespace)
		require.Equal(t, "0 6 * * *", response.Cron)
		require.Equal(t, "News scan scheduled daily at 06:00 UTC", response.Message)

		require.Len(t, scheduleClient.createdOptions, 1)
		opts := scheduleClient.createdOptions[0]
		require.Equal(t, "news-daily-scan", opts.ID)
		require.Equal(t, []string{"0 6 * * *"}, opts.Spec.CronExpressions)
		require.Equal(t, enumspb.SCHEDULE_OVERLAP_POLICY_SKIP, opts.Overlap)
		require.Equal(t, 12*time.Hour, opts.CatchupWindow)

		action, ok := opts.Action.(*client.ScheduleWorkflowAction)
		require.True(t, ok)
		require.Equal(t, "news-daily-scan", action.ID)
		require.Equal(t, "NewsDailyScan", action.Workflow)
		require.Equal(t, "news-agent", action.TaskQueue)
		require.Equal(
			t,
			[]interface{}{workflows.NewsDailyScanInput{SourceIDs: []string{}, MaxDrafts: 3}},
			action.Args,
		)
		mockHandle.AssertNotCalled(t, "Trigger", mock.Anything, mock.Anything)
	})

	t.Run("success - midnight with immediate trigger", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Trigger", mock.Anything, newsScanTriggerOptions).Return(nil).Once()
		scheduleClient := &fakeScheduleClient{handle: mockHandle}
		useNewsAgentScheduleClient(scheduleClient)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newsAgentRequest(
			http.MethodPost,
			`{"hour_utc":0,"trigger_now":true,"source_ids":["openid4vp"],"max_drafts":5}`,
			true,
		))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var response ScheduleNewsScanResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.Equal(t, "0 0 * * *", response.Cron)
		require.Equal(
			t,
			"News scan scheduled daily at 00:00 UTC and triggered now",
			response.Message,
		)
		require.Len(t, scheduleClient.createdOptions, 1)
		opts := scheduleClient.createdOptions[0]
		require.Equal(t, []string{"0 0 * * *"}, opts.Spec.CronExpressions)
		action, ok := opts.Action.(*client.ScheduleWorkflowAction)
		require.True(t, ok)
		require.Equal(
			t,
			[]interface{}{
				workflows.NewsDailyScanInput{SourceIDs: []string{"openid4vp"}, MaxDrafts: 5},
			},
			action.Args,
		)
		mockHandle.AssertNumberOfCalls(t, "Trigger", 1)
	})

	t.Run("success - update keeps existing state", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		var capturedUpdateOptions client.ScheduleUpdateOptions
		mockHandle.On("Update", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedUpdateOptions = args.Get(1).(client.ScheduleUpdateOptions)
		}).Return(nil).Once()
		scheduleClient := &fakeScheduleClient{
			createErr: serviceerror.NewAlreadyExists("schedule exists"),
			handle:    mockHandle,
		}
		useNewsAgentScheduleClient(scheduleClient)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newsAgentRequest(http.MethodPost, `{"hour_utc":7}`, true))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotNil(t, capturedUpdateOptions.DoUpdate)
		paused := &client.ScheduleState{Paused: true, Note: "paused by operator"}
		update, err := capturedUpdateOptions.DoUpdate(client.ScheduleUpdateInput{
			Description: client.ScheduleDescription{Schedule: client.Schedule{State: paused}},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"0 7 * * *"}, update.Schedule.Spec.CronExpressions)
		require.Equal(t, enumspb.SCHEDULE_OVERLAP_POLICY_SKIP, update.Schedule.Policy.Overlap)
		require.Equal(t, 12*time.Hour, update.Schedule.Policy.CatchupWindow)
		require.Same(t, paused, update.Schedule.State)
		action, ok := update.Schedule.Action.(*client.ScheduleWorkflowAction)
		require.True(t, ok)
		require.Equal(t, "NewsDailyScan", action.Workflow)
		mockHandle.AssertNotCalled(t, "Trigger", mock.Anything, mock.Anything)
	})

	t.Run("fail - hour out of range", func(t *testing.T) {
		scheduleClient := &fakeScheduleClient{handle: &temporalmocks.ScheduleHandle{}}
		useNewsAgentScheduleClient(scheduleClient)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newsAgentRequest(http.MethodPost, `{"hour_utc":24}`, true))

		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		require.Empty(t, scheduleClient.createdOptions)
	})

	t.Run("fail - missing api key", func(t *testing.T) {
		scheduleClient := &fakeScheduleClient{handle: &temporalmocks.ScheduleHandle{}}
		useNewsAgentScheduleClient(scheduleClient)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newsAgentRequest(http.MethodPost, `{}`, false))

		require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
		require.Empty(t, scheduleClient.createdOptions)
	})
}

func TestNewsAgentDeleteNewsScanSchedule(t *testing.T) {
	app := setupNewsAgentApp(t)
	defer app.Cleanup()

	originalScheduleTemporalClient := scheduleTemporalClient
	defer func() { scheduleTemporalClient = originalScheduleTemporalClient }()

	mux := buildAppMux(t, app)

	t.Run("success", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Delete", mock.Anything).Return(nil).Once()
		useNewsAgentScheduleClient(&fakeScheduleClient{handle: mockHandle})

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newsAgentRequest(http.MethodDelete, "", true))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var response DeleteNewsScanScheduleResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.True(t, response.Success)
		require.Equal(t, workflows.NewsDailyScanScheduleID, response.ScheduleID)
		mockHandle.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Delete", mock.Anything).
			Return(&serviceerror.NotFound{Message: "missing"}).
			Once()
		useNewsAgentScheduleClient(&fakeScheduleClient{handle: mockHandle})

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newsAgentRequest(http.MethodDelete, "", true))

		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	})
}
