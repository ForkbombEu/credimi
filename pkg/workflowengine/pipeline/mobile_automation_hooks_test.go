// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileAutomationSetupHookAcquiresUniqueRunnerPermits(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testAcquireRunnerPermitsWorkflow,
		workflow.RegisterOptions{Name: "test-acquire-permits"},
	)

	acquireActivity := activities.NewAcquireMobileRunnerPermitActivity()
	env.RegisterActivityWithOptions(
		acquireActivity.Execute,
		activity.RegisterOptions{Name: acquireActivity.Name()},
	)
	env.OnActivity(acquireActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: workflows.MobileRunnerSemaphorePermit{RunnerID: "runner"}}, nil).
		Times(2)

	steps := []StepDefinition{
		{
			StepSpec: StepSpec{
				ID:  "step-1",
				Use: "mobile-automation",
				With: StepInputs{
					Payload: map[string]any{
						"runner_id": "runner-a",
						"action_id": "action-1",
					},
				},
			},
		},
		{
			StepSpec: StepSpec{
				ID:  "step-2",
				Use: "mobile-automation",
				With: StepInputs{
					Payload: map[string]any{
						"runner_id": "runner-b",
						"action_id": "action-2",
					},
				},
			},
		},
		{
			StepSpec: StepSpec{
				ID:  "step-3",
				Use: "mobile-automation",
				With: StepInputs{
					Payload: map[string]any{
						"runner_id": "runner-a",
						"action_id": "action-3",
					},
				},
			},
		},
	}

	env.ExecuteWorkflow("test-acquire-permits", steps)

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestProcessStepFailsWithoutPermit(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testProcessStepWithoutPermitWorkflow,
		workflow.RegisterOptions{Name: "test-missing-permit"},
	)

	env.ExecuteWorkflow("test-missing-permit")

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing runner permit")
}

func TestMobileAutomationCleanupReleasesPermitsOnFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testCleanupReleasesPermitsWorkflow,
		workflow.RegisterOptions{Name: "test-cleanup-release"},
	)

	cleanupActivity := activities.NewCleanupDeviceActivity()
	releaseActivity := activities.NewReleaseMobileRunnerPermitActivity()

	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		releaseActivity.Execute,
		activity.RegisterOptions{Name: releaseActivity.Name()},
	)

	env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("cleanup failed"))
	env.OnActivity(releaseActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil).
		Times(2)

	env.ExecuteWorkflow("test-cleanup-release")

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mobile automation cleanup")
	env.AssertExpectations(t)
}

func testAcquireRunnerPermitsWorkflow(ctx workflow.Context, steps []StepDefinition) error {
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	acquireActivity := activities.NewAcquireMobileRunnerPermitActivity()
	runnerIDs, err := collectMobileRunnerIDs(steps, "")
	if err != nil {
		return err
	}

	_, err = acquireRunnerPermits(ctx, runnerIDs, acquireActivity)
	return err
}

func testProcessStepWithoutPermitWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	step := StepDefinition{
		StepSpec: StepSpec{
			ID:  "step-1",
			Use: "mobile-automation",
			With: StepInputs{
				Payload: map[string]any{
					"runner_id": "runner-1",
					"action_id": "action-1",
				},
			},
		},
	}
	config := map[string]any{"app_url": "https://example.test"}
	settedDevices := map[string]any{}
	runData := map[string]any{}
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}

	return processStep(
		processStepInput{
			ctx:              ctx,
			step:             &step,
			config:           config,
			ao:               &activityOptions,
			settedDevices:    settedDevices,
			runData:          &runData,
			httpActivity:     activities.NewHTTPActivity(),
			startEmuActivity: activities.NewStartEmulatorActivity(),
			installActivity:  activities.NewApkInstallActivity(),
			logger:           workflow.GetLogger(ctx),
			globalRunnerID:   "",
		},
	)
}

func testCleanupReleasesPermitsWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	config := map[string]any{"app_url": "https://example.test"}
	output := map[string]any{}
	runData := map[string]any{
		"setted_devices": map[string]any{
			"runner-1": map[string]any{
				"serial":     "serial-1",
				"clone_name": "clone-1",
				"installed":  map[string]string{},
				"runner_url": "https://runner.test",
				"recording":  false,
			},
		},
		"run_identifier": "run-1",
		"mobile_runner_permits": map[string]workflows.MobileRunnerSemaphorePermit{
			"runner-1": {RunnerID: "runner-1", LeaseID: "lease-1"},
			"runner-2": {RunnerID: "runner-2", LeaseID: "lease-2"},
		},
	}
	steps := []StepDefinition{}
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}

	return MobileAutomationCleanupHook(ctx, steps, &activityOptions, config, runData, &output)
}
