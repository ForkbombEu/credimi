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

func TestMobileAutomationWorkflowChecksExternallyInstalledApp(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileAutomationWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	mobileActivity := activities.NewRunMobileFlowActivity()
	listAppsActivity := activities.NewListInstalledAppsActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	for _, registeredActivity := range []struct {
		name string
		fn   any
	}{
		{mobileActivity.Name(), mobileActivity.Execute},
		{listAppsActivity.Name(), listAppsActivity.Execute},
		{postInstallActivity.Name(), postInstallActivity.Execute},
	} {
		env.RegisterActivityWithOptions(registeredActivity.fn, activity.RegisterOptions{Name: registeredActivity.name})
	}

	env.OnActivity(mobileActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"status": "ok"}}, nil).Once()
	for _, apps := range [][]string{
		{"com.example.old"},
		{"com.example.installed", "com.example.old"},
	} {
		env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
			Return(workflowengine.ActivityResult{Output: apps}, nil).Once()
	}
	env.OnActivity(postInstallActivity.Name(), mock.Anything, mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
		payload := workflowengine.AsMap(input.Payload)
		return payload["serial"] == "serial-1" && payload["package_id"] == "com.example.installed"
	})).Return(workflowengine.ActivityResult{Output: map[string]any{
		"package_id": "com.example.installed",
	}}, nil).Once()

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileAutomationWorkflowPayload{Serial: "serial-1", Type: "android_phone", ActionCode: "steps: []"},
		Config: map[string]any{
			"app_url":                         "https://example.test",
			"taskqueue":                       "runner-1-TaskQueue",
			externalInstallDetectionConfigKey: true,
		},
		ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	})

	require.NoError(t, env.GetWorkflowError())
	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	flowOutput := workflowengine.AsMap(workflowengine.AsMap(result.Output)["flow_output"])
	postInstallOutput := workflowengine.AsMap(flowOutput["post_install"])
	require.Equal(t, "com.example.installed", postInstallOutput["package_id"])
}
