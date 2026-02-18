// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
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
)

const testDataDir = "../../../test_pb_data"

func getOrgIDfromName(app core.App, name string) (string, error) {
	record, err := app.FindFirstRecordByFilter(
		"organizations",
		"name = {:name}",
		dbx.Params{"name": name},
	)
	if err != nil {
		return "", err
	}
	return record.Id, nil
}

func TestReadGlobalRunnerIDFromScheduleDescription(t *testing.T) {
	t.Run("scheduled input value", func(t *testing.T) {
		desc := scheduleDescWithArg(workflows.ScheduledPipelineEnqueueWorkflowInput{
			GlobalRunnerID: "runner-1",
		})
		require.Equal(t, "runner-1", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("scheduled input pointer", func(t *testing.T) {
		desc := scheduleDescWithArg(&workflows.ScheduledPipelineEnqueueWorkflowInput{
			GlobalRunnerID: "runner-2",
		})
		require.Equal(t, "runner-2", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("workflow input value", func(t *testing.T) {
		desc := scheduleDescWithArg(workflowengine.WorkflowInput{
			Payload: workflows.ScheduledPipelineEnqueueWorkflowInput{GlobalRunnerID: "runner-3"},
		})
		require.Equal(t, "runner-3", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("workflow input pointer with map payload", func(t *testing.T) {
		desc := scheduleDescWithArg(&workflowengine.WorkflowInput{
			Payload: map[string]any{
				"global_runner_id": "runner-4",
			},
		})
		require.Equal(t, "runner-4", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("pipeline input value", func(t *testing.T) {
		desc := scheduleDescWithArg(pipeline.PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"global_runner_id": "runner-5"},
			},
		})
		require.Equal(t, "runner-5", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("pipeline input pointer", func(t *testing.T) {
		desc := scheduleDescWithArg(&pipeline.PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"global_runner_id": "runner-6"},
			},
		})
		require.Equal(t, "runner-6", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("payload scheduled input", func(t *testing.T) {
		payload := mustPayload(t, workflows.ScheduledPipelineEnqueueWorkflowInput{
			GlobalRunnerID: "runner-7",
		})
		desc := scheduleDescWithArg(payload)
		require.Equal(t, "runner-7", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("payload workflow input", func(t *testing.T) {
		payload := mustPayload(t, workflowengine.WorkflowInput{
			Config: map[string]any{"global_runner_id": "runner-8"},
		})
		desc := scheduleDescWithArg(payload)
		require.Equal(t, "runner-8", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("payload pipeline input", func(t *testing.T) {
		payload := mustPayload(t, pipeline.PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"global_runner_id": "runner-9"},
			},
		})
		desc := scheduleDescWithArg(payload)
		require.Equal(t, "runner-9", readGlobalRunnerIDFromScheduleDescription(desc))
	})
}

func TestResolveScheduleRunnerRecordsFallbackToCanonify(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app, "userA's organization")
	require.NoError(t, err)

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
	pipelineRecord.Set("name", "pipeline-1")
	pipelineRecord.Set("canonified_name", "pipeline-1")
	pipelineRecord.Set("description", "test pipeline")
	pipelineRecord.Set(
		"yaml",
		"name: pipeline-1\nsteps:\n  - use: mobile-automation\n    with:\n      payload:\n        runner_id: \""+orgCanon+"/runner-1\"\n",
	)
	require.NoError(t, app.Save(pipelineRecord))

	runnersColl, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runnerRecord := core.NewRecord(runnersColl)
	runnerRecord.Set("owner", orgID)
	runnerRecord.Set("name", "runner-1")
	runnerRecord.Set("canonified_name", "runner-1")
	runnerRecord.Set("ip", "127.0.0.1")
	runnerRecord.Set("runner_url", "https://runner.test")
	runnerRecord.Set("serial", "serial-1")
	require.NoError(t, app.Save(runnerRecord))

	records, err := resolveScheduleRunnerRecords(app, orgCanon+"/pipeline-1", nil)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, runnerRecord.Id, records[0]["id"])
}

func TestResolveScheduleRunnerRecordsParseError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName(app, "userA's organization")
	require.NoError(t, err)

	pipelinesColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelinesColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "pipeline-bad")
	pipelineRecord.Set("canonified_name", "pipeline-bad")
	pipelineRecord.Set("description", "broken pipeline")
	pipelineRecord.Set("yaml", "name: [")
	require.NoError(t, app.Save(pipelineRecord))

	_, err = resolveScheduleRunnerRecords(app, pipelineRecord.Id, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse pipeline yaml")
}

func TestRegisterSchedulesHooksNotFoundEnrich(t *testing.T) {
	origClient := schedulesTemporalClient
	t.Cleanup(func() { schedulesTemporalClient = origClient })

	mockClient := temporalmocks.NewClient(t)
	scheduleClient := temporalmocks.NewScheduleClient(t)
	scheduleHandle := temporalmocks.NewScheduleHandle(t)
	scheduleHandle.
		On("Describe", mock.Anything).
		Return(nil, &serviceerror.NotFound{Message: "missing"})
	scheduleClient.
		On("GetHandle", mock.Anything, "temporal-1").
		Return(scheduleHandle)
	mockClient.On("ScheduleClient").Return(scheduleClient)

	schedulesTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	RegisterSchedulesHooks(app)

	orgID, err := getOrgIDfromName(app, "userA's organization")
	require.NoError(t, err)
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
	pipelineRecord.Set("name", "pipeline-1")
	pipelineRecord.Set("canonified_name", "pipeline-1")
	pipelineRecord.Set("description", "test pipeline")
	pipelineRecord.Set(
		"yaml",
		"name: pipeline-1\nsteps:\n  - use: mobile-automation\n    with:\n      payload:\n        runner_id: \""+orgCanon+"/runner-1\"\n",
	)
	require.NoError(t, app.Save(pipelineRecord))

	runnersColl, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runnerRecord := core.NewRecord(runnersColl)
	runnerRecord.Set("owner", orgID)
	runnerRecord.Set("name", "runner-1")
	runnerRecord.Set("canonified_name", "runner-1")
	runnerRecord.Set("ip", "127.0.0.1")
	runnerRecord.Set("runner_url", "https://runner.test")
	runnerRecord.Set("serial", "serial-1")
	require.NoError(t, app.Save(runnerRecord))

	schedulesColl, err := app.FindCollectionByNameOrId("schedules")
	require.NoError(t, err)
	scheduleRecord := core.NewRecord(schedulesColl)
	scheduleRecord.Set("owner", orgID)
	scheduleRecord.Set("pipeline", pipelineRecord.Id)
	scheduleRecord.Set("temporal_schedule_id", "temporal-1")
	require.NoError(t, app.Save(scheduleRecord))

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/collections/schedules/records/"+scheduleRecord.Id,
		nil,
	)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	require.NoError(t, apis.EnrichRecord(event, scheduleRecord))
	export := scheduleRecord.PublicExport()
	statusRaw, ok := export["__schedule_status__"]
	require.True(t, ok)

	var status ScheduleStatus
	encoded, err := json.Marshal(statusRaw)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &status))
	require.NotEmpty(t, status.Runners)
}

func TestDecodeScheduledEnqueueInput(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		input, err := decodeScheduledEnqueueInput(map[string]any{
			"pipeline_identifier": "pipeline-1",
			"global_runner_id":    "runner-1",
		})
		require.NoError(t, err)
		require.Equal(t, "pipeline-1", input.PipelineIdentifier)
		require.Equal(t, "runner-1", input.GlobalRunnerID)
	})

	t.Run("invalid payload types", func(t *testing.T) {
		_, err := decodeScheduledEnqueueInput(map[string]any{
			"pipeline_identifier": 123,
		})
		require.Error(t, err)
	})
}

func scheduleDescWithArg(arg any) *client.ScheduleDescription {
	return &client.ScheduleDescription{
		Schedule: client.Schedule{
			Action: &client.ScheduleWorkflowAction{
				Args: []any{arg},
			},
		},
	}
}

func mustPayload(t testing.TB, v any) *commonpb.Payload {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(v)
	require.NoError(t, err)
	return payload
}
