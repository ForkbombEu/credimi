// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestRunStepCIAndSendMailNoMail(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"captures": map[string]any{
					"deeplink": "link",
				},
			},
		}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				SendMail:      false,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-no-mail"},
	)

	env.ExecuteWorkflow("test-stepci-no-mail")
	require.NoError(t, env.GetWorkflowError())

	var result StepCIAndEmailResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "link", result.Captures["deeplink"])
}

func TestRunStepCIAndSendMailMissingCaptures(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				SendMail:      false,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-missing-captures"},
	)

	env.ExecuteWorkflow("test-stepci-missing-captures")
	require.Error(t, env.GetWorkflowError())
}

func TestRunStepCIAndSendMailMissingDeeplink(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"captures": map[string]any{
					"no_deeplink": "missing",
				},
			},
		}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				AppName:       "Credimi",
				AppLogo:       "logo.png",
				UserName:      "User",
				UserMail:      "user@example.org",
				Namespace:     "ns-1",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				Suite:         OpenIDConformanceSuite,
				SendMail:      true,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-missing-deeplink"},
	)

	env.ExecuteWorkflow("test-stepci-missing-deeplink")
	require.Error(t, env.GetWorkflowError())
}

func TestStartCheckWorkflowStart(t *testing.T) {
	origStart := startCheckWorkflowWithOptions
	t.Cleanup(func() {
		startCheckWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string

	startCheckWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		_ workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}

	w := NewStartCheckWorkflow()
	result, err := w.Start("ns-1", workflowengine.WorkflowInput{})
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, ConformanceCheckTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "conformance-check-"))
}
