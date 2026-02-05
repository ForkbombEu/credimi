// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

const testDataDir = "../../../test_pb_data"

func setupSchedulesTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	RegisterSchedulesHooks(app)
	return app
}

func TestSchedulesEnrichMissingOwner(t *testing.T) {
	app := setupSchedulesTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("schedules")
	require.NoError(t, err)

	record := core.NewRecord(collection)
	record.Set("owner", "")

	event := &core.RecordEnrichEvent{App: app}
	event.Record = record
	err = app.OnRecordEnrich("schedules").Trigger(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch owner organization")
}

func TestSchedulesEnrichInvalidOwner(t *testing.T) {
	app := setupSchedulesTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("schedules")
	require.NoError(t, err)

	record := core.NewRecord(collection)
	record.Set("owner", "missing-owner")

	event := &core.RecordEnrichEvent{App: app}
	event.Record = record
	err = app.OnRecordEnrich("schedules").Trigger(event)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch owner organization")
}

// TestReadGlobalRunnerIDFromScheduledInput verifies scheduled inputs surface global runner IDs.
func TestReadGlobalRunnerIDFromScheduledInput(t *testing.T) {
	t.Run("direct-field", func(t *testing.T) {
		desc := &client.ScheduleDescription{
			Schedule: client.Schedule{
				Action: &client.ScheduleWorkflowAction{
					Args: []any{
						workflows.ScheduledPipelineEnqueueWorkflowInput{
							GlobalRunnerID: "runner-1",
							PipelineConfig: map[string]any{
								"global_runner_id": "runner-fallback",
							},
						},
					},
				},
			},
		}
		require.Equal(t, "runner-1", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("config-fallback", func(t *testing.T) {
		desc := &client.ScheduleDescription{
			Schedule: client.Schedule{
				Action: &client.ScheduleWorkflowAction{
					Args: []any{
						workflows.ScheduledPipelineEnqueueWorkflowInput{
							PipelineConfig: map[string]any{
								"global_runner_id": "runner-2",
							},
						},
					},
				},
			},
		}
		require.Equal(t, "runner-2", readGlobalRunnerIDFromScheduleDescription(desc))
	})

	t.Run("payload", func(t *testing.T) {
		dc := converter.GetDefaultDataConverter()
		payload, err := dc.ToPayload(workflows.ScheduledPipelineEnqueueWorkflowInput{
			GlobalRunnerID: "runner-3",
		})
		require.NoError(t, err)

		desc := &client.ScheduleDescription{
			Schedule: client.Schedule{
				Action: &client.ScheduleWorkflowAction{
					Args: []any{payload},
				},
			},
		}
		require.Equal(t, "runner-3", readGlobalRunnerIDFromScheduleDescription(desc))
	})
}
