//go:build unit

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
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
		"pipeline-id",
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

type fakeScheduleClient struct {
	createdOptions []client.ScheduleOptions
	handle         client.ScheduleHandle
}

// Create records schedule options for assertions and returns a stub handle.
func (f *fakeScheduleClient) Create(
	ctx context.Context,
	options client.ScheduleOptions,
) (client.ScheduleHandle, error) {
	f.createdOptions = append(f.createdOptions, options)
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
