//go:build credimi_extra

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileAutomationWorkflowPropagatesActivityCancellation(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileAutomationWorkflow()
	env.RegisterWorkflowWithOptions(
		w.Workflow,
		workflow.RegisterOptions{Name: w.Name()},
	)

	mobileActivity := activities.NewRunMobileFlowActivity()
	env.RegisterActivityWithOptions(
		mobileActivity.Execute,
		activity.RegisterOptions{Name: mobileActivity.Name()},
	)

	env.OnActivity(
		mobileActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{}, temporal.NewCanceledError("mobile flow canceled"))

	env.ExecuteWorkflow(
		w.Name(),
		workflowengine.WorkflowInput{
			Payload: MobileAutomationWorkflowPayload{
				Serial:     "emulator-5554",
				ActionCode: "steps: []",
			},
			Config: map[string]any{
				"app_url":   "https://example.test",
				"taskqueue": "runner-1-TaskQueue",
			},
			ActivityOptions: &workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			},
		},
	)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.True(t, temporal.IsCanceledError(err))
}

func TestMobileAutomationWorkflowPropagatesExternalInstallDetectionCancellation(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileAutomationWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	listAppsActivity := activities.NewListInstalledAppsActivity()
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)
	env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, temporal.NewCanceledError("listing canceled"))

	env.ExecuteWorkflow(w.Name(), externalInstallWorkflowInput())

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.True(t, temporal.IsCanceledError(err))
}

func externalInstallWorkflowInput() workflowengine.WorkflowInput {
	return workflowengine.WorkflowInput{
		Payload: MobileAutomationWorkflowPayload{
			Serial:     "emulator-5554",
			ActionCode: "steps: []",
		},
		Config: map[string]any{
			"app_url":                         "https://example.test",
			"taskqueue":                       "runner-1-TaskQueue",
			externalInstallDetectionConfigKey: true,
		},
		ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	}
}

func TestMobileAutomationWorkflowStoresStepScreenshots(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileAutomationWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})
	mobileActivity := activities.NewRunMobileFlowActivity()
	httpActivity := activities.NewInternalHTTPActivity()
	env.RegisterActivityWithOptions(
		mobileActivity.Execute,
		activity.RegisterOptions{Name: mobileActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.OnActivity(mobileActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"output":                   "Maestro output",
			"maestro_screenshot_paths": []string{"/tmp/checkout.png"},
		}}, nil)
	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			request, err := workflowengine.DecodePayload[activities.InternalHTTPActivityPayload](
				input.Payload,
			)
			if err != nil {
				return false
			}
			body := workflowengine.AsMap(request.Body)
			return request.URL == "https://runner.example/credimi/execution-screenshots" &&
				workflowengine.AsString(body["step_id"]) == "scan-credential" &&
				workflowengine.AsString(body["run_identifier"]) == "org/workflow-run" &&
				workflowengine.AsString(body["runner_identifier"]) == "org/runner" &&
				requireScreenshotPaths(body["screenshot_paths"], "/tmp/checkout.png")
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"screenshot_urls": []string{"https://app.example/api/files/checkout.png"},
		},
	}}, nil)

	env.ExecuteWorkflow(w.Name(), mobileScreenshotWorkflowInput())
	require.NoError(t, env.GetWorkflowError())
	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output := workflowengine.AsMap(result.Output)
	flowOutput := workflowengine.AsMap(output["flow_output"])
	require.Equal(t, "Maestro output", flowOutput["output"])
	require.NotContains(t, flowOutput, "maestro_screenshot_paths")
	require.Equal(
		t,
		[]string{"https://app.example/api/files/checkout.png"},
		workflowengine.AsSliceOfStrings(flowOutput["maestro_screenshot_urls"]),
	)
	env.AssertExpectations(t)
}

func TestMobileAutomationWorkflowSkipsScreenshotAPIWithoutPaths(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	w := NewMobileAutomationWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})
	mobileActivity := activities.NewRunMobileFlowActivity()
	env.RegisterActivityWithOptions(
		mobileActivity.Execute,
		activity.RegisterOptions{Name: mobileActivity.Name()},
	)
	env.OnActivity(mobileActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"output": "no screenshots"}}, nil)

	env.ExecuteWorkflow(w.Name(), mobileScreenshotWorkflowInput())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, activities.NewInternalHTTPActivity().Name(), mock.Anything, mock.Anything)
}

func mobileScreenshotWorkflowInput() workflowengine.WorkflowInput {
	return workflowengine.WorkflowInput{
		Payload: MobileAutomationWorkflowPayload{
			Serial:     "emulator-5554",
			ActionCode: "steps: []",
			RunnerID:   "org/runner",
		},
		Config: map[string]any{
			"app_url":        "https://app.example",
			"taskqueue":      "org-runner-TaskQueue",
			"runner_url":     "https://runner.example",
			"step_id":        "scan-credential",
			"run_identifier": "org/workflow-run",
		},
		ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	}
}

func requireScreenshotPaths(raw any, expected ...string) bool {
	actual := workflowengine.AsSliceOfStrings(raw)
	if len(actual) != len(expected) {
		return false
	}
	for index := range expected {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func TestMobileActivityOptionsAddHeartbeatAndPreserveRetryPolicy(t *testing.T) {
	options := mobileActivityOptions(
		&workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 5,
			},
		},
		"runner-1-TaskQueue",
	)

	require.Equal(t, time.Minute, options.StartToCloseTimeout)
	require.Equal(t, 30*time.Second, options.HeartbeatTimeout)
	require.Equal(t, 30*time.Second, options.ScheduleToStartTimeout)
	require.Equal(t, "runner-1-TaskQueue", options.TaskQueue)
	require.NotNil(t, options.RetryPolicy)
	require.Equal(t, int32(5), options.RetryPolicy.MaximumAttempts)
}
