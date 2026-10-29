// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// recordCleanupPayloads registers the runner cleanup activity and records the
// payloads and task queues it is called with.
func recordCleanupPayloads(
	env *testsuite.TestWorkflowEnvironment,
	payloads *[]map[string]any,
	taskQueues *[]string,
) {
	env.RegisterActivityWithOptions(
		func(ctx context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			*payloads = append(*payloads, workflowengine.AsMap(input.Payload))
			*taskQueues = append(*taskQueues, activity.GetInfo(ctx).TaskQueue)
			return workflowengine.ActivityResult{Output: "cleaned"}, nil
		},
		activity.RegisterOptions{Name: activities.NewCleanupDeviceActivity().Name()},
	)
}

// TestMobileAutomationSetupHookTracksDeviceWhoseSetupFails covers an emulator
// that started but whose app inventory failed: its clone must still be
// deleted, which the pipeline's cleanup hook only does for devices it knows.
func TestMobileAutomationSetupHookTracksDeviceWhoseSetupFails(t *testing.T) {
	tests := []struct {
		name         string
		beforeChange bool
		wantCleanups int
	}{
		{name: "cleans the started emulator", wantCleanups: 1},
		{name: "does not know the device before the change", beforeChange: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			var payloads []map[string]any
			var taskQueues []string
			recordCleanupPayloads(env, &payloads, &taskQueues)
			registerStubActivity(env, activities.GetMobileDeviceActivityName)
			registerStubActivity(env, activities.ResolveRecordActivityName)
			startActivity := activities.NewSetupMobileDeviceActivity()
			env.RegisterActivityWithOptions(
				startActivity.Execute,
				activity.RegisterOptions{Name: startActivity.Name()},
			)
			listAppsActivity := activities.NewListInstalledAppsActivity()
			env.RegisterActivityWithOptions(
				listAppsActivity.Execute,
				activity.RegisterOptions{Name: listAppsActivity.Name()},
			)

			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) (bool, error) {
					ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
					ctx = workflow.WithActivityOptions(ctx, ao)
					runData := map[string]any{"run_identifier": "tenant/run-1"}
					setupErr := MobileAutomationSetupHook(
						ctx,
						&pipeline.WorkflowDefinition{Steps: mobileAutomationSetupSteps()},
						map[string]any{
							"namespace":                            "tenant",
							mobileDeviceSemaphoreTicketIDConfigKey: "ticket-1",
						},
						&runData,
						&map[string]any{},
						log.Logger(noopLogger{}),
					)
					output := map[string]any{}
					_ = MobileAutomationCleanupHook(ctx, nil, &ao, nil, runData, &output)
					return setupErr != nil, nil
				},
				workflow.RegisterOptions{Name: "test-setup-failure-tracked"},
			)
			if tt.beforeChange {
				env.OnGetVersion(mobileEarlyDeviceTrackingChange, workflow.DefaultVersion, 1).
					Return(workflow.DefaultVersion)
			}
			onResolveMobileAction(env, "action-1", map[string]any{"code": "code-1"})
			onGetMobileDevice(env, "runner-1/device-1", map[string]any{
				"runner_id":  "tenant/runner-1",
				"runner_url": "https://runner.example",
				"type":       "emulator",
			})
			env.OnActivity(startActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"serial": "emulator-5554",
					"name":   "clone-1",
				}}, nil)
			env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{}, assert.AnError)

			env.ExecuteWorkflow("test-setup-failure-tracked")
			require.NoError(t, env.GetWorkflowError())
			var setupFailed bool
			require.NoError(t, env.GetWorkflowResult(&setupFailed))
			require.True(t, setupFailed)
			require.Len(t, payloads, tt.wantCleanups)
			if tt.wantCleanups > 0 {
				require.Equal(t, "clone-1", payloads[0]["name"])
				require.Equal(t, "emulator-5554", payloads[0]["serial"])
				require.Equal(t, "android_emulator", payloads[0]["type"])
			}
		})
	}
}

func TestCleanupDeviceSkipsEmulatorThatNeverStarted(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	var payloads []map[string]any
	var taskQueues []string
	recordCleanupPayloads(env, &payloads, &taskQueues)
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			output := map[string]any{}
			var cleanupErrs []error
			err := cleanupDevice(cleanupDeviceInput{
				ctx:      workflow.WithActivityOptions(ctx, ao),
				deviceID: "tenant/runner-1/device-1",
				raw: map[string]any{
					"type":      "android_emulator",
					"runner_id": "tenant/runner-1",
					"installed": map[string]string{},
				},
				mobileAo:      &ao,
				runIdentifier: "tenant/run-1",
				output:        &output,
				cleanupErrs:   &cleanupErrs,
				logger:        workflow.GetLogger(ctx),
			})
			require.Empty(t, cleanupErrs)
			return output, err
		},
		workflow.RegisterOptions{Name: "test-cleanup-unstarted-emulator"},
	)

	env.ExecuteWorkflow("test-cleanup-unstarted-emulator")
	require.NoError(t, env.GetWorkflowError())
	var output map[string]any
	require.NoError(t, env.GetWorkflowResult(&output))
	require.Empty(t, payloads)
	require.Contains(
		t,
		workflowengine.AsSliceOfStrings(output[cleanupWarningsOutputKey]),
		"device cleanup skipped for tenant/runner-1/device-1 because it never started",
	)
}
