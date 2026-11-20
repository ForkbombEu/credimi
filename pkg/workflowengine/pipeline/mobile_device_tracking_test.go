// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
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
