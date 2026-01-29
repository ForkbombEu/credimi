// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
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

	env.RegisterWorkflowWithOptions(testAcquireRunnerPermitsWorkflow, workflow.RegisterOptions{Name: "test-acquire-permits"})

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

	env.RegisterWorkflowWithOptions(testProcessStepWithoutPermitWorkflow, workflow.RegisterOptions{Name: "test-missing-permit"})

	env.ExecuteWorkflow("test-missing-permit")

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing runner permit")
}

func testAcquireRunnerPermitsWorkflow(ctx workflow.Context, steps []StepDefinition) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Second})
	acquireActivity := activities.NewAcquireMobileRunnerPermitActivity()
	runnerIDs, err := collectMobileRunnerIDs(steps)
	if err != nil {
		return err
	}

	_, err = acquireRunnerPermits(ctx, runnerIDs, acquireActivity)
	return err
}

func testProcessStepWithoutPermitWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Second})
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
		ctx,
		&step,
		config,
		&activityOptions,
		settedDevices,
		&runData,
		activities.NewHTTPActivity(),
		activities.NewStartEmulatorActivity(),
		activities.NewApkInstallActivity(),
		workflow.GetLogger(ctx),
	)
}
