// SPDX-FileCopyrightText: 2025 Forkbomb BV
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestPipelineStartMissingNamespace(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()
	result, err := pipelineWf.Start(
		"name: test-pipeline\nsteps: []\n",
		map[string]any{},
		map[string]any{},
		"tenant-1/test-pipeline",
	)
	require.Error(t, err)
	require.Empty(t, result.WorkflowID)
	require.Contains(t, err.Error(), "namespace is required")
}

func TestPipelineStartInvalidYAML(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()
	_, err := pipelineWf.Start("name: [", map[string]any{}, map[string]any{}, "tenant-1/pipeline")
	require.Error(t, err)
}

func TestPipelineStartScheduled(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()

	originalClient := pipelineTemporalClient
	defer func() {
		pipelineTemporalClient = originalClient
	}()

	mockClient := temporalmocks.NewClient(t)
	scheduleClient := temporalmocks.NewScheduleClient(t)
	scheduleHandle := temporalmocks.NewScheduleHandle(t)
	var capturedAction *client.ScheduleWorkflowAction

	scheduleHandle.On("Describe", mock.Anything).Return(&client.ScheduleDescription{}, nil)
	scheduleHandle.On("GetID").Return("schedule-123")
	scheduleClient.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		opts := args.Get(1).(client.ScheduleOptions)
		action, ok := opts.Action.(*client.ScheduleWorkflowAction)
		require.True(t, ok)
		capturedAction = action
	}).Return(scheduleHandle, nil)
	mockClient.On("ScheduleClient").Return(scheduleClient)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}
	yaml := `name: scheduled-pipeline
runtime:
  schedule:
    interval: 1m
steps:
  - id: step1
    use: mobile-automation
    with:
      payload:
        runner_id: "runner-android"
  - id: step2
    use: mobile-automation
    with:
      payload:
        runner_id: "runner-ios"
`
	result, err := pipelineWf.Start(
		yaml,
		map[string]any{"namespace": "default"},
		map[string]any{},
		"tenant-1/scheduled-pipeline",
	)
	require.NoError(t, err)
	require.Equal(t, "schedule-123", result.WorkflowID)
	require.Contains(t, result.Message, "scheduled successfully")

	expectedRunnerIDs := []string{"runner-android", "runner-ios"}
	expectedSearchAttrs := workflowengine.PipelineTypedSearchAttributes(
		"tenant-1/scheduled-pipeline",
		expectedRunnerIDs,
		workflowengine.EntityIDs{},
	)
	require.Equal(
		t,
		expectedSearchAttrs,
		capturedAction.TypedSearchAttributes,
	)
}

func TestPipelineStartImmediate(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()

	originalClient := pipelineTemporalClient
	defer func() {
		pipelineTemporalClient = originalClient
	}()

	mockClient := temporalmocks.NewClient(t)
	workflowRun := temporalmocks.NewWorkflowRun(t)
	var capturedOptions client.StartWorkflowOptions
	var capturedInput PipelineWorkflowInput

	workflowRun.On("GetID").Return("workflow-123")
	workflowRun.On("GetRunID").Return("run-456")
	mockClient.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.Anything,
		pipelineWf.Name(),
		mock.Anything,
	).Run(func(args mock.Arguments) {
		capturedOptions = args.Get(1).(client.StartWorkflowOptions)
		capturedInput = args.Get(3).(PipelineWorkflowInput)
	}).Return(workflowRun, nil)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	result, err := pipelineWf.Start(
		"name: immediate-pipeline\nsteps: []\n",
		map[string]any{"namespace": "default"},
		map[string]any{},
		"tenant-1/immediate-pipeline",
	)
	require.NoError(t, err)
	require.Equal(t, "workflow-123", result.WorkflowID)
	require.Equal(t, "run-456", result.WorkflowRunID)
	key := temporal.NewSearchAttributeKeyKeyword(workflowengine.PipelineIdentifierSearchAttribute)
	value, ok := capturedOptions.TypedSearchAttributes.GetKeyword(key)
	require.True(t, ok)
	require.Equal(t, "tenant-1/immediate-pipeline", value)
	require.NotContains(t, capturedInput.WorkflowInput.Config, tempWalletVersionConfigKey)
}

func TestPipelineStartIgnoresReservedYAMLConfig(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()

	originalClient := pipelineTemporalClient
	defer func() {
		pipelineTemporalClient = originalClient
	}()

	mockClient := temporalmocks.NewClient(t)
	workflowRun := temporalmocks.NewWorkflowRun(t)
	var capturedInput PipelineWorkflowInput

	workflowRun.On("GetID").Return("workflow-123")
	workflowRun.On("GetRunID").Return("run-456")
	mockClient.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.Anything,
		pipelineWf.Name(),
		mock.Anything,
	).Run(func(args mock.Arguments) {
		capturedInput = args.Get(3).(PipelineWorkflowInput)
	}).Return(workflowRun, nil)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	_, err := pipelineWf.Start(
		`name: reserved-config
config:
  keep: value
  temp_wallet_version:
    record_id: malicious
steps: []
`,
		map[string]any{"namespace": "default"},
		map[string]any{},
		"tenant-1/reserved-config",
	)
	require.NoError(t, err)
	require.Equal(t, "value", capturedInput.WorkflowInput.Config["keep"])
	require.NotContains(t, capturedInput.WorkflowInput.Config, tempWalletVersionConfigKey)
}

func TestPipelineWorkflowSuccessWithNoSteps(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{
			Name:  "empty-steps",
			Steps: []pipeline.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": "https://example.test",
			},
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, output["workflow-id"])
	require.NotEmpty(t, output["workflow-run-id"])
}

func TestPipelineWorkflowReportsGitHubPRCommentDone(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	update := capturePipelineGitHubPRCommentUpdate(env)

	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{
			Name:  "empty-steps",
			Steps: []pipeline.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": "https://credimi.test",
				GitHubPRCommentConfigKey: map[string]any{
					"repository":          "forkbombeu/issuer",
					"pull_request_number": 17,
					"commit_sha":          "abc123",
					"pipeline_id":         "tenant-a/issuer",
					"pipeline_url":        "https://credimi.test/my/pipelines/tenant-a/issuer",
					"app_url":             "https://credimi.test",
					"section_title":       activities.GitHubPRCommentSectionIssuer,
				},
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, "forkbombeu/issuer", update.Repository)
	require.Equal(t, 17, update.PullRequestNumber)
	require.Equal(t, "abc123", update.CommitSHA)
	require.Equal(t, "success", update.WorkflowStatus)
	require.Equal(t, "tenant-a/issuer", update.PipelineID)
	require.Equal(t, activities.GitHubPRCommentSectionIssuer, update.SectionTitle)
	require.NotEmpty(t, update.WorkflowID)
	require.NotEmpty(t, update.RunID)
	env.AssertExpectations(t)
}

func TestPipelineWorkflowReportsGitHubPRCommentFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	update := capturePipelineGitHubPRCommentUpdate(env)

	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": "https://credimi.test",
				GitHubPRCommentConfigKey: map[string]any{
					"repository":          "forkbombeu/issuer",
					"pull_request_number": 17,
					"commit_sha":          "abc123",
					"pipeline_id":         "tenant-a/issuer",
					"pipeline_url":        "https://credimi.test/my/pipelines/tenant-a/issuer",
					"app_url":             "https://credimi.test",
					"section_title":       activities.GitHubPRCommentSectionIssuer,
				},
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	require.Equal(t, "forkbombeu/issuer", update.Repository)
	require.Equal(t, "failed", update.WorkflowStatus)
	require.Equal(t, activities.GitHubPRCommentSectionIssuer, update.SectionTitle)
	require.NotEmpty(t, update.WorkflowID)
	require.NotEmpty(t, update.RunID)
	env.AssertExpectations(t)
}

func capturePipelineGitHubPRCommentUpdate(
	env *testsuite.TestWorkflowEnvironment,
) *activities.UpdateGitHubPRCommentInput {
	var update activities.UpdateGitHubPRCommentInput
	env.RegisterActivityWithOptions(
		func(
			ctx context.Context,
			input workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{}, nil
		},
		activity.RegisterOptions{Name: "Update GitHub PR comment"},
	)
	env.OnActivity(
		"Update GitHub PR comment",
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			decoded, err := workflowengine.DecodePayload[activities.UpdateGitHubPRCommentInput](
				input.Payload,
			)
			if err != nil {
				return false
			}
			update = decoded
			return true
		}),
	).Return(workflowengine.ActivityResult{}, nil).Once()
	return &update
}
