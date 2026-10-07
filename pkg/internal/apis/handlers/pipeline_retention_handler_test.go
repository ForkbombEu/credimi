// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func setupPipelineRetentionApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineInternalRoutes.Add(app)
	seedInternalAdminKey(t, app)

	return app
}

// buildAppMux builds the app router and mux exactly once per app instance.
// PocketBase 0.40 binds its UI extension routes inside apis.NewRouter, so
// calling NewRouter twice on the same app registers duplicate routes.
func buildAppMux(t testing.TB, app *tests.TestApp) http.Handler {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	var mux http.Handler
	err = app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		var err error
		mux, err = e.Router.BuildMux()
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, mux, "OnServe trigger must have built the request mux")
	return mux
}

func TestSchedulePipelineRetentionWorkflow(t *testing.T) {
	app := setupPipelineRetentionApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"

	originalScheduleTemporalClient := scheduleTemporalClient
	defer func() { scheduleTemporalClient = originalScheduleTemporalClient }()

	mux := buildAppMux(t, app)

	t.Run("success - defaults create schedule", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Trigger", mock.Anything, pipelineRetentionImmediateTriggerOptions).
			Return(nil).
			Once()

		mockClient := &temporalmocks.Client{}
		mockScheduleClient := &fakeScheduleClient{handle: mockHandle}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/schedule",
			strings.NewReader(`{}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response SchedulePipelineRetentionResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.Equal(t, workflows.DefaultNamespace, response.WorkflowNamespace)
		require.Contains(t, response.Message, "every 1 day(s)")
		require.Contains(t, response.Message, "older_than_days=30")
		require.Contains(t, response.Message, "triggered now")
		require.Equal(t, pipelineRetentionScheduleID, response.ScheduleID)

		require.Len(t, mockScheduleClient.createdOptions, 1)
		opts := mockScheduleClient.createdOptions[0]
		require.Equal(t, pipelineRetentionScheduleID, opts.ID)
		require.Len(t, opts.Spec.Intervals, 1)
		require.Equal(t, 24*time.Hour, opts.Spec.Intervals[0].Every)
		require.Equal(t, enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE, opts.Overlap)

		action, ok := opts.Action.(*client.ScheduleWorkflowAction)
		require.True(t, ok)
		require.Equal(t, workflows.PipelineRetentionTaskQueue, action.TaskQueue)
		require.Equal(t, pipelineRetentionScheduleID, action.ID)
		require.Equal(t, workflows.NewPipelineRetentionWorkflow().Name(), action.Workflow)
		require.Len(t, action.Args, 1)
		input, ok := action.Args[0].(workflowengine.WorkflowInput)
		require.True(t, ok)
		require.Equal(
			t,
			workflows.PipelineRetentionWorkflowInput{
				OlderThanDays: 30,
				DryRun:        false,
			},
			input.Payload,
		)
	})

	t.Run("success - custom values update existing schedule", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		var capturedUpdateOptions client.ScheduleUpdateOptions
		mockHandle.On("Update", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedUpdateOptions = args.Get(1).(client.ScheduleUpdateOptions)
		}).Return(nil).Once()
		mockHandle.On("Trigger", mock.Anything, pipelineRetentionImmediateTriggerOptions).
			Return(nil).
			Once()

		mockScheduleClient := &fakeScheduleClient{
			createErr: serviceerror.NewAlreadyExists("schedule exists"),
			handle:    mockHandle,
		}
		mockClient := &temporalmocks.Client{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/schedule",
			strings.NewReader(`{"older_than_days":45,"interval_days":2}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, mockScheduleClient.createdOptions, 1)
		require.Equal(t, pipelineRetentionScheduleID, mockScheduleClient.createdOptions[0].ID)
		require.NotNil(t, capturedUpdateOptions.DoUpdate)

		update, err := capturedUpdateOptions.DoUpdate(client.ScheduleUpdateInput{})
		require.NoError(t, err)
		require.NotNil(t, update)
		require.NotNil(t, update.Schedule)
		require.Len(t, update.Schedule.Spec.Intervals, 1)
		require.Equal(t, 48*time.Hour, update.Schedule.Spec.Intervals[0].Every)
		require.NotNil(t, update.Schedule.Policy)
		require.Equal(
			t,
			enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
			update.Schedule.Policy.Overlap,
		)
		require.NotNil(t, update.Schedule.State)

		action, ok := update.Schedule.Action.(*client.ScheduleWorkflowAction)
		require.True(t, ok)
		require.Equal(t, pipelineRetentionScheduleID, action.ID)
		require.Equal(t, workflows.NewPipelineRetentionWorkflow().Name(), action.Workflow)
		input, ok := action.Args[0].(workflowengine.WorkflowInput)
		require.True(t, ok)
		require.Equal(
			t,
			workflows.PipelineRetentionWorkflowInput{
				OlderThanDays: 45,
				DryRun:        false,
			},
			input.Payload,
		)
	})

	t.Run("success - update existing schedule on already registered error", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		var capturedUpdateOptions client.ScheduleUpdateOptions
		mockHandle.On("Update", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			capturedUpdateOptions = args.Get(1).(client.ScheduleUpdateOptions)
		}).Return(nil).Once()
		mockHandle.On("Trigger", mock.Anything, pipelineRetentionImmediateTriggerOptions).
			Return(nil).
			Once()

		mockScheduleClient := &fakeScheduleClient{
			createErr: errors.New("schedule with this ID is already registered"),
			handle:    mockHandle,
		}
		mockClient := &temporalmocks.Client{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/schedule",
			strings.NewReader(`{"older_than_days":30,"interval_days":1}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.NotNil(t, capturedUpdateOptions.DoUpdate)
	})

	t.Run("fail - temporal client error", func(t *testing.T) {
		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return nil, errors.New("temporal connection failed")
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/schedule",
			strings.NewReader(`{}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("fail - update existing schedule fails", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Update", mock.Anything, mock.Anything).
			Return(errors.New("update failed")).
			Once()

		mockScheduleClient := &fakeScheduleClient{
			createErr: serviceerror.NewAlreadyExists("schedule exists"),
			handle:    mockHandle,
		}
		mockClient := &temporalmocks.Client{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/schedule",
			strings.NewReader(`{}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("fail - immediate trigger fails after create", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Trigger", mock.Anything, pipelineRetentionImmediateTriggerOptions).
			Return(errors.New("trigger failed")).
			Once()

		mockScheduleClient := &fakeScheduleClient{handle: mockHandle}
		mockClient := &temporalmocks.Client{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/schedule",
			strings.NewReader(`{}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestDeletePipelineRetentionSchedule(t *testing.T) {
	app := setupPipelineRetentionApp(t)
	defer app.Cleanup()

	originalScheduleTemporalClient := scheduleTemporalClient
	defer func() { scheduleTemporalClient = originalScheduleTemporalClient }()

	mux := buildAppMux(t, app)

	t.Run("success", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Delete", mock.Anything).Return(nil).Once()

		mockScheduleClient := &fakeScheduleClient{handle: mockHandle}
		mockClient := &temporalmocks.Client{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/pipeline/retention/schedule", nil)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var response DeletePipelineRetentionScheduleResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.True(t, response.Success)
		require.Equal(t, pipelineRetentionScheduleID, response.ScheduleID)
	})

	t.Run("not found", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Delete", mock.Anything).
			Return(&serviceerror.NotFound{Message: "missing"}).
			Once()

		mockScheduleClient := &fakeScheduleClient{handle: mockHandle}
		mockClient := &temporalmocks.Client{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/pipeline/retention/schedule", nil)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
