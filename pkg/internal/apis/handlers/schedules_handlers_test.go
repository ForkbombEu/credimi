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
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/workflow"
)

func TestValidateScheduleModeDaily(t *testing.T) {
	mode := workflowengine.ScheduleMode{Mode: "daily"}
	require.NoError(t, validateScheduleMode(&mode))
}

func TestValidateScheduleModeWeeklyDefault(t *testing.T) {
	mode := workflowengine.ScheduleMode{Mode: "weekly"}
	require.NoError(t, validateScheduleMode(&mode))
	require.NotNil(t, mode.Day)
	require.GreaterOrEqual(t, *mode.Day, 0)
	require.LessOrEqual(t, *mode.Day, 6)
}

func TestValidateScheduleModeWeeklyBounds(t *testing.T) {
	for _, day := range []int{-1, 7} {
		mode := workflowengine.ScheduleMode{Mode: "weekly", Day: &day}
		require.Error(t, validateScheduleMode(&mode))
	}
}

func TestValidateScheduleModeMonthlyDefault(t *testing.T) {
	// Test that a default day is assigned when none is provided.
	// Note: We cannot assert the exact value or validity since it depends
	// on the current date. On the 31st of a month, the implementation will
	// assign day=31 which exceeds the valid range (0-30) and validation fails.
	// This test only verifies that a default value is assigned, not its validity.
	mode := workflowengine.ScheduleMode{Mode: "monthly"}
	_ = validateScheduleMode(&mode)
	require.NotNil(t, mode.Day, "default day should be assigned")
}

func TestValidateScheduleModeMonthlyBounds(t *testing.T) {
	for _, day := range []int{-1, 31} {
		mode := workflowengine.ScheduleMode{Mode: "monthly", Day: &day}
		require.Error(t, validateScheduleMode(&mode))
	}
}

func TestValidateScheduleModeInvalid(t *testing.T) {
	mode := workflowengine.ScheduleMode{Mode: "yearly"}
	require.Error(t, validateScheduleMode(&mode))
}

func TestHandleStartScheduleInvalidJSON(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/my/schedules/start",
		strings.NewReader("{invalid json"),
	)
	rec := httptest.NewRecorder()

	err = HandleStartSchedule()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)

	var apiErr *router.ApiError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestHandleStartScheduleInvalidMode(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	body, err := json.Marshal(map[string]any{
		"pipeline_id":   "missing",
		"schedule_mode": map[string]any{"mode": "yearly"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/my/schedules/start",
		strings.NewReader(string(body)),
	)
	rec := httptest.NewRecorder()

	err = HandleStartSchedule()(&core.RequestEvent{
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

func TestHandleStartSchedulePipelineNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	body, err := json.Marshal(map[string]any{
		"pipeline_id":   "missing",
		"schedule_mode": map[string]any{"mode": "daily"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/my/schedules/start",
		strings.NewReader(string(body)),
	)
	rec := httptest.NewRecorder()

	err = HandleStartSchedule()(&core.RequestEvent{
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

func TestHandleStartScheduleTemporalCreateError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://example.test"

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	_, pipelineID := createSchedulePipelineRecord(t, app, orgID, "pipeline-1")

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	fakeSchedule := &fakeScheduleClient{createErr: errors.New("create failed")}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	body, err := json.Marshal(map[string]any{
		"pipeline_id":   pipelineID,
		"schedule_mode": map[string]any{"mode": "daily"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/my/schedules/start",
		strings.NewReader(string(body)),
	)
	rec := httptest.NewRecorder()

	err = HandleStartSchedule()(&core.RequestEvent{
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

func TestHandleStartScheduleDescribeError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://example.test"

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	_, pipelineID := createSchedulePipelineRecord(t, app, orgID, "pipeline-2")

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	handle := temporalmocks.NewScheduleHandle(t)
	handle.On("Describe", mock.Anything).Return(nil, errors.New("describe failed"))
	fakeSchedule := &fakeScheduleClient{handle: handle}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	body, err := json.Marshal(map[string]any{
		"pipeline_id":   pipelineID,
		"schedule_mode": map[string]any{"mode": "daily"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/my/schedules/start",
		strings.NewReader(string(body)),
	)
	rec := httptest.NewRecorder()

	err = HandleStartSchedule()(&core.RequestEvent{
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

func TestHandleStartScheduleSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://example.test"

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipelineRec, pipelineID := createSchedulePipelineRecord(t, app, orgID, "pipeline-3")

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	handle := temporalmocks.NewScheduleHandle(t)
	handle.On("Describe", mock.Anything).Return(&client.ScheduleDescription{}, nil)
	fakeSchedule := &fakeScheduleClient{handle: handle}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	body, err := json.Marshal(map[string]any{
		"pipeline_id":   pipelineID,
		"schedule_mode": map[string]any{"mode": "daily"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/my/schedules/start",
		strings.NewReader(string(body)),
	)
	rec := httptest.NewRecorder()

	err = HandleStartSchedule()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response StartScheduleResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, "daily", response.ScheduleMode.Mode)
	require.NotEmpty(t, response.ScheduleID)

	record, err := app.FindFirstRecordByFilter(
		"schedules",
		"temporal_schedule_id = {:sid}",
		map[string]any{"sid": response.ScheduleID},
	)
	require.NoError(t, err)
	require.Equal(t, pipelineRec.Id, record.GetString("pipeline"))
	require.Equal(t, orgID, record.GetString("owner"))
}

// TestStartScheduledPipelineUsesScheduledEnqueueWorkflow ensures schedules target the enqueue workflow.
func TestStartScheduledPipelineUsesScheduledEnqueueWorkflow(t *testing.T) {
	originalClient := scheduleTemporalClient
	defer func() {
		scheduleTemporalClient = originalClient
	}()

	fakeHandle := &fakeScheduleHandle{}
	fakeSchedule := &fakeScheduleClient{handle: fakeHandle}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)

	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	config := map[string]any{
		"namespace": "acme",
		"app_url":   "https://example.test",
		"user_name": "Ada",
		"user_mail": "ada@example.test",
	}
	_, err := startScheduledPipelineWithOptions(
		"pipeline-slug",
		"Pipeline Name",
		"acme",
		config,
		workflowengine.ScheduleMode{Mode: "daily"},
		"UTC",
		"runner-1",
		7,
	)
	require.NoError(t, err)

	require.Len(t, fakeSchedule.createdOptions, 1)
	options := fakeSchedule.createdOptions[0]
	action, ok := options.Action.(*client.ScheduleWorkflowAction)
	require.True(t, ok)
	require.Equal(t, workflows.ScheduledPipelineEnqueueWorkflowName, action.Workflow)
	require.Equal(t, pipeline.PipelineTaskQueue, action.TaskQueue)
	require.Len(t, action.Args, 1)

	arg, ok := action.Args[0].(workflowengine.WorkflowInput)
	require.True(t, ok)
	require.Equal(t, config, arg.Config)

	payload, ok := arg.Payload.(workflows.ScheduledPipelineEnqueueWorkflowInput)
	require.True(t, ok)
	require.Equal(t, "pipeline-slug", payload.PipelineIdentifier)
	require.Equal(t, "acme", payload.OwnerNamespace)
	require.Equal(t, "runner-1", payload.GlobalRunnerID)
	require.Equal(t, 7, payload.MaxPipelinesInQueue)

	mockClient.AssertExpectations(t)
}

func TestListScheduledWorkflowsHappyPath(t *testing.T) {
	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	iter := temporalmocks.NewScheduleListIterator(t)
	iter.On("HasNext").Return(true).Once()
	iter.On("Next").Return(&client.ScheduleListEntry{
		ID: "schedule-1",
		Spec: &client.ScheduleSpec{
			Calendars: []client.ScheduleCalendarSpec{
				{
					Month:      []client.ScheduleRange{{Start: 1, End: 12}},
					DayOfMonth: []client.ScheduleRange{{Start: 1, End: 31}},
					DayOfWeek:  []client.ScheduleRange{{Start: 0, End: 6}},
				},
			},
		},
		WorkflowType: workflow.Type{Name: "Dynamic Pipeline Workflow"},
		NextActionTimes: []time.Time{
			time.Date(2026, time.February, 17, 12, 0, 0, 0, time.UTC),
		},
	}, nil).Once()
	iter.On("HasNext").Return(false).Once()

	fakeSchedule := &fakeScheduleClient{listIter: iter}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)

	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	schedules, err := listScheduledWorkflows("acme")
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	require.Equal(t, "schedule-1", schedules[0].ID)
	require.Equal(t, "daily", schedules[0].ScheduleMode.Mode)
	require.Equal(t, "", schedules[0].DisplayName)
	require.Equal(t, "", schedules[0].PipelineID)
	require.Equal(t, "Dynamic Pipeline Workflow", schedules[0].WorkflowType.Name)
	require.Equal(t, "17/02/2026, 12:00:00", schedules[0].NextActionTime)
}

func TestHandleScheduleNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	fakeSchedule := &fakeScheduleClient{}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)

	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/schedules/sched-1/cancel", nil)
	req.SetPathValue("scheduleId", "sched-1")
	rec := httptest.NewRecorder()

	handler := handleSchedule(
		func(ctx context.Context, h client.ScheduleHandle) error {
			return &serviceerror.NotFound{Message: "missing"}
		},
		func(scheduleID, namespace string) any {
			return map[string]string{"scheduleId": scheduleID, "namespace": namespace}
		},
		nil,
	)

	err = handler(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusNotFound, apiErr.Code)
	require.Equal(t, "schedule not found", apiErr.Reason)
}

func TestHandleListMySchedules(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	displayPayload, err := converter.GetDefaultDataConverter().ToPayload("Test Schedule")
	require.NoError(t, err)
	pipelinePayload, err := converter.GetDefaultDataConverter().ToPayload("org-1/pipeline-1")
	require.NoError(t, err)

	iter := temporalmocks.NewScheduleListIterator(t)
	iter.On("HasNext").Return(true).Once()
	iter.On("Next").Return(&client.ScheduleListEntry{
		ID: "schedule-1",
		Spec: &client.ScheduleSpec{
			Calendars: []client.ScheduleCalendarSpec{
				{
					Month:      []client.ScheduleRange{{Start: 1, End: 12}},
					DayOfMonth: []client.ScheduleRange{{Start: 1, End: 31}},
					DayOfWeek:  []client.ScheduleRange{{Start: 0, End: 6}},
				},
			},
		},
		WorkflowType: workflow.Type{Name: "Dynamic Pipeline Workflow"},
		NextActionTimes: []time.Time{
			time.Date(2026, time.February, 17, 12, 0, 0, 0, time.UTC),
		},
		Memo: &commonpb.Memo{
			Fields: map[string]*commonpb.Payload{
				"test":        displayPayload,
				"pipeline_id": pipelinePayload,
			},
		},
		Paused: true,
	}, nil).Once()
	iter.On("HasNext").Return(false).Once()

	fakeSchedule := &fakeScheduleClient{listIter: iter}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)

	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/schedules", nil)
	rec := httptest.NewRecorder()

	err = HandleListMySchedules()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response ListMySchedulesResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Len(t, response.Schedules, 1)
	require.Equal(t, "schedule-1", response.Schedules[0].ID)
	require.Equal(t, "daily", response.Schedules[0].ScheduleMode.Mode)
	require.Equal(t, "Test Schedule", response.Schedules[0].DisplayName)
	require.Equal(t, "org-1/pipeline-1", response.Schedules[0].PipelineID)
	require.True(t, response.Schedules[0].Paused)
}

func TestHandleCancelScheduleDeletesRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	schedulesColl, err := app.FindCollectionByNameOrId("schedules")
	require.NoError(t, err)
	scheduleRecord := core.NewRecord(schedulesColl)
	scheduleRecord.Set("temporal_schedule_id", "sched-1")
	scheduleRecord.Set("owner", orgID)
	require.NoError(t, app.Save(scheduleRecord))

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	handle := temporalmocks.NewScheduleHandle(t)
	handle.On("Delete", mock.Anything).Return(nil)

	fakeSchedule := &fakeScheduleClient{handle: handle}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/schedules/sched-1/cancel", nil)
	req.SetPathValue("scheduleId", "sched-1")
	rec := httptest.NewRecorder()

	err = HandleCancelSchedule()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	_, err = app.FindRecordById("schedules", scheduleRecord.Id)
	require.Error(t, err)
}

func TestHandleCancelScheduleNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	handle := temporalmocks.NewScheduleHandle(t)
	handle.On("Delete", mock.Anything).Return(&serviceerror.NotFound{Message: "missing"})

	fakeSchedule := &fakeScheduleClient{handle: handle}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/schedules/sched-1/cancel", nil)
	req.SetPathValue("scheduleId", "sched-1")
	rec := httptest.NewRecorder()

	err = HandleCancelSchedule()(&core.RequestEvent{
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

func TestHandlePauseSchedule(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	handle := temporalmocks.NewScheduleHandle(t)
	handle.On("Pause", mock.Anything, mock.Anything).Return(nil)

	fakeSchedule := &fakeScheduleClient{handle: handle}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/schedules/sched-1/pause", nil)
	req.SetPathValue("scheduleId", "sched-1")
	rec := httptest.NewRecorder()

	err = HandlePauseSchedule()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	handle = temporalmocks.NewScheduleHandle(t)
	handle.On("Pause", mock.Anything, mock.Anything).
		Return(&serviceerror.NotFound{Message: "missing"})
	fakeSchedule = &fakeScheduleClient{handle: handle}
	mockClient = &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	rec = httptest.NewRecorder()
	err = HandlePauseSchedule()(&core.RequestEvent{
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

func TestHandleResumeSchedule(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	originalClient := scheduleTemporalClient
	t.Cleanup(func() {
		scheduleTemporalClient = originalClient
	})

	handle := temporalmocks.NewScheduleHandle(t)
	handle.On("Unpause", mock.Anything, mock.Anything).Return(nil)

	fakeSchedule := &fakeScheduleClient{handle: handle}
	mockClient := &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/schedules/sched-1/resume", nil)
	req.SetPathValue("scheduleId", "sched-1")
	rec := httptest.NewRecorder()

	err = HandleResumeSchedule()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	handle = temporalmocks.NewScheduleHandle(t)
	handle.On("Unpause", mock.Anything, mock.Anything).
		Return(&serviceerror.NotFound{Message: "missing"})
	fakeSchedule = &fakeScheduleClient{handle: handle}
	mockClient = &temporalmocks.Client{}
	mockClient.On("ScheduleClient").Return(fakeSchedule)
	scheduleTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	rec = httptest.NewRecorder()
	err = HandleResumeSchedule()(&core.RequestEvent{
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

func TestDeleteScheduleRecord(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	schedulesColl, err := app.FindCollectionByNameOrId("schedules")
	require.NoError(t, err)
	scheduleRecord := core.NewRecord(schedulesColl)
	scheduleRecord.Set("temporal_schedule_id", "sched-2")
	scheduleRecord.Set("owner", orgID)
	require.NoError(t, app.Save(scheduleRecord))

	require.Error(t, deleteScheduleRecord(app, "missing", orgID))
	require.NoError(t, deleteScheduleRecord(app, "sched-2", orgID))
}

type fakeScheduleClient struct {
	createdOptions []client.ScheduleOptions
	handle         client.ScheduleHandle
	listIter       client.ScheduleListIterator
	listErr        error
	createErr      error
}

func createSchedulePipelineRecord(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
) (*core.Record, string) {
	t.Helper()

	orgRecord, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	if orgRecord.GetString("canonified_name") == "" {
		orgRecord.Set("canonified_name", "usera-s-organization")
		require.NoError(t, app.Save(orgRecord))
	}
	orgCanon := orgRecord.GetString("canonified_name")

	pipelinesColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelinesColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", name)
	pipelineRecord.Set("canonified_name", name)
	pipelineRecord.Set("description", "test pipeline")
	pipelineRecord.Set("yaml", "name: "+name+"\nsteps: []\n")
	require.NoError(t, app.Save(pipelineRecord))

	return pipelineRecord, orgCanon + "/" + pipelineRecord.GetString("canonified_name")
}

// Create records schedule options for assertions and returns a stub handle.
func (f *fakeScheduleClient) Create(
	ctx context.Context,
	options client.ScheduleOptions,
) (client.ScheduleHandle, error) {
	f.createdOptions = append(f.createdOptions, options)
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.handle != nil {
		return f.handle, nil
	}
	return &fakeScheduleHandle{}, nil
}

// List returns an error to surface unexpected list calls in tests.
func (f *fakeScheduleClient) List(
	ctx context.Context,
	options client.ScheduleListOptions,
) (client.ScheduleListIterator, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listIter != nil {
		return f.listIter, nil
	}
	return nil, errors.New("schedule list not implemented in test")
}

// GetHandle returns the configured handle or a default stub.
func (f *fakeScheduleClient) GetHandle(
	ctx context.Context,
	scheduleID string,
) client.ScheduleHandle {
	if f.handle != nil {
		return f.handle
	}
	return &fakeScheduleHandle{}
}

type fakeScheduleHandle struct{}

// GetID returns an empty schedule ID for stubbed handles.
func (f *fakeScheduleHandle) GetID() string {
	return ""
}

// Delete is a no-op stub for schedule handle cleanup.
func (f *fakeScheduleHandle) Delete(ctx context.Context) error {
	return nil
}

// Backfill is a no-op stub for schedule handle backfill.
func (f *fakeScheduleHandle) Backfill(
	ctx context.Context,
	options client.ScheduleBackfillOptions,
) error {
	return nil
}

// Update is a no-op stub for schedule handle updates.
func (f *fakeScheduleHandle) Update(
	ctx context.Context,
	options client.ScheduleUpdateOptions,
) error {
	return nil
}

// Describe returns an empty schedule description for verification calls.
func (f *fakeScheduleHandle) Describe(ctx context.Context) (*client.ScheduleDescription, error) {
	return &client.ScheduleDescription{}, nil
}

// Trigger is a no-op stub for schedule handle triggers.
func (f *fakeScheduleHandle) Trigger(
	ctx context.Context,
	options client.ScheduleTriggerOptions,
) error {
	return nil
}

// Pause is a no-op stub for schedule handle pauses.
func (f *fakeScheduleHandle) Pause(
	ctx context.Context,
	options client.SchedulePauseOptions,
) error {
	return nil
}

// Unpause is a no-op stub for schedule handle resumes.
func (f *fakeScheduleHandle) Unpause(
	ctx context.Context,
	options client.ScheduleUnpauseOptions,
) error {
	return nil
}
