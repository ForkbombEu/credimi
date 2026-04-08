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
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileExternalInstallWorkflowWaitsForInstalledAppVisibility(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileExternalInstallWorkflow()
	env.RegisterWorkflowWithOptions(
		w.Workflow,
		workflow.RegisterOptions{Name: w.Name()},
	)

	mobileActivity := activities.NewRunMobileFlowActivity()
	listAppsActivity := activities.NewListInstalledAppsActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	env.RegisterActivityWithOptions(
		mobileActivity.Execute,
		activity.RegisterOptions{Name: mobileActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		postInstallActivity.Execute,
		activity.RegisterOptions{Name: postInstallActivity.Name()},
	)

	env.OnActivity(
		mobileActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"status": "ok",
	}}, nil).Once()

	env.OnActivity(
		listAppsActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: []string{"com.example.old"}}, nil).Once()
	env.OnActivity(
		listAppsActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: []string{"com.example.old"}}, nil).Once()
	env.OnActivity(
		listAppsActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: []string{"com.example.old"}}, nil).Once()
	env.OnActivity(
		listAppsActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: []string{
		"com.example.installed",
		"com.example.old",
	}}, nil).Once()

	env.OnActivity(
		postInstallActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			return ok &&
				workflowengine.AsString(payload["serial"]) == "serial-1" &&
				workflowengine.AsString(payload["package_id"]) == "com.example.installed"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"package_id": "com.example.installed",
	}}, nil).Once()

	env.ExecuteWorkflow(
		w.Name(),
		workflowengine.WorkflowInput{
			Payload: MobileAutomationWorkflowPayload{
				Serial:     "serial-1",
				Type:       "android_phone",
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

	require.NoError(t, env.GetWorkflowError())

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))

	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	flowOutput, ok := output["flow_output"].(map[string]any)
	require.True(t, ok)
	postInstallOutput, ok := flowOutput["post_install"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "com.example.installed", postInstallOutput["package_id"])
}
