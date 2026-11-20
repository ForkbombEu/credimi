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
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestPipelineReportsSemaphoreDone(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	reportActivity := activities.NewReportMobileDeviceSemaphoreDoneActivity(nil)
	env.RegisterActivityWithOptions(
		reportActivity.Execute,
		activity.RegisterOptions{Name: reportActivity.Name()},
	)

	originalSetupHooks := setupHooks
	originalCleanupHooks := cleanupHooks
	setupHooks = []SetupFunc{}
	cleanupHooks = []CleanupFunc{
		func(
			ctx workflow.Context,
			wfDef *pipeline.WorkflowDefinition,
			ao *workflow.ActivityOptions,
			config map[string]any,
			runData map[string]any,
			output *map[string]any,
		) error {
			return workflow.ExecuteActivity(
				ctx,
				"test-cleanup",
				workflowengine.ActivityInput{},
			).Get(ctx, nil)
		},
	}
	t.Cleanup(func() {
		setupHooks = originalSetupHooks
		cleanupHooks = originalCleanupHooks
	})

	env.RegisterActivityWithOptions(
		func(
			ctx context.Context,
			input workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{}, nil
		},
		activity.RegisterOptions{Name: "test-cleanup"},
	)

	runOrder := []string{}
	env.OnActivity(
		"test-cleanup",
		mock.Anything,
		mock.Anything,
	).
		Run(func(_ mock.Arguments) {
			runOrder = append(runOrder, "cleanup")
		}).
		Return(workflowengine.ActivityResult{}, nil).
		Once()

	env.OnActivity(
		reportActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(activities.ReportMobileDeviceSemaphoreDoneInput)
			if !ok {
				decoded, err := workflowengine.DecodePayload[activities.ReportMobileDeviceSemaphoreDoneInput](
					input.Payload,
				)
				if err != nil {
					return false
				}
				payload = decoded
			}
			return payload.TicketID == "ticket-1" &&
				payload.OwnerNamespace == "tenant-1" &&
				payload.LeaderDeviceID == "runner-1/device-1" &&
				payload.WorkflowID == "default-test-workflow-id" &&
				payload.RunID == "default-test-run-id" &&
				payload.WorkflowResult == resultSuccess
		}),
	).
		Run(func(_ mock.Arguments) {
			runOrder = append(runOrder, "report")
		}).
		Return(workflowengine.ActivityResult{}, nil).
		Once()

	input := PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{
			Name:  "test-pipeline",
			Steps: []pipeline.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":                                  "https://example.test",
				"mobile_device_semaphore_ticket_id":        "ticket-1",
				"mobile_device_semaphore_leader_device_id": "runner-1/device-1",
				"mobile_device_semaphore_owner_namespace":  "tenant-1",
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	}

	env.ExecuteWorkflow(pipelineWf.Name(), input)

	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, []string{"cleanup", "report"}, runOrder)
	env.AssertExpectations(t)
}

func TestPipelineSendsFinalReportsAfterCancellation(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	reportActivity := activities.NewReportMobileDeviceSemaphoreDoneActivity(nil)
	env.RegisterActivityWithOptions(
		reportActivity.Execute,
		activity.RegisterOptions{Name: reportActivity.Name()},
	)
	notificationActivity := activities.NewSendPipelineCompletionNotificationActivity(nil)
	env.RegisterActivityWithOptions(
		notificationActivity.Execute,
		activity.RegisterOptions{Name: notificationActivity.Name()},
	)

	originalSetupHooks := setupHooks
	originalCleanupHooks := cleanupHooks
	setupHooks = []SetupFunc{
		func(
			ctx workflow.Context,
			_ *pipeline.WorkflowDefinition,
			_ map[string]any,
			_ *map[string]any,
			_ *map[string]any,
			_ log.Logger,
		) error {
			return workflow.Sleep(ctx, time.Hour)
		},
	}
	cleanupHooks = []CleanupFunc{}
	t.Cleanup(func() {
		setupHooks = originalSetupHooks
		cleanupHooks = originalCleanupHooks
	})

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Minute)

	var reportedResult, notifiedResult string
	env.OnActivity(reportActivity.Name(), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			payload, err := workflowengine.DecodePayload[activities.ReportMobileDeviceSemaphoreDoneInput](
				args.Get(1).(workflowengine.ActivityInput).Payload,
			)
			require.NoError(t, err)
			reportedResult = payload.WorkflowResult
		}).
		Return(workflowengine.ActivityResult{}, nil).
		Once()
	env.OnActivity(notificationActivity.Name(), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			payload, err := workflowengine.DecodePayload[activities.SendPipelineCompletionNotificationInput](
				args.Get(1).(workflowengine.ActivityInput).Payload,
			)
			require.NoError(t, err)
			notifiedResult = payload.Result
		}).
		Return(workflowengine.ActivityResult{}, nil).
		Once()

	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{
			Name:  "test-pipeline",
			Steps: []pipeline.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":                                  "https://example.test",
				"mobile_device_semaphore_ticket_id":        "ticket-1",
				"mobile_device_semaphore_leader_device_id": "runner-1/device-1",
				"mobile_device_semaphore_owner_namespace":  "tenant-1",
				CompletionNotificationConfigKey:            true,
			},
		},
	})

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.True(t, temporal.IsCanceledError(err))
	require.Equal(t, resultCanceled, reportedResult)
	require.Equal(t, resultCanceled, notifiedResult)
	env.AssertExpectations(t)
}

func TestChildPipelineDoesNotSendTheRunReports(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)
	captured := []string{}
	registerRuntimeCaptureActivity(t, env, "capture-runtime", &captured)
	registerStubActivity(env, activities.ResolveRecordActivityName)
	semaphoreDone := activities.NewReportMobileDeviceSemaphoreDoneActivity(nil)
	registerStubActivity(env, semaphoreDone.Name())
	registerStubActivity(env, activities.SendPipelineCompletionNotificationActivityName)
	githubUpdate := activities.NewUpdateGitHubPRCommentActivity()
	registerStubActivity(env, githubUpdate.Name())
	env.OnActivity(activities.ResolveRecordActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"yaml": `
name: Child Pipeline
steps:
  - id: inner
    use: capture-runtime
    with:
      payload:
        text: inner
`}}, nil).Once()

	reportedBy := map[string][]string{}
	recordReport := func(name string, workflowID func(workflowengine.ActivityInput) string) {
		env.OnActivity(name, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				input, ok := args.Get(1).(workflowengine.ActivityInput)
				require.True(t, ok)
				reportedBy[name] = append(reportedBy[name], workflowID(input))
			}).
			Return(workflowengine.ActivityResult{}, nil)
	}
	recordReport(semaphoreDone.Name(), func(input workflowengine.ActivityInput) string {
		payload, err := workflowengine.DecodePayload[activities.ReportMobileDeviceSemaphoreDoneInput](
			input.Payload,
		)
		require.NoError(t, err)
		return payload.WorkflowID
	})
	recordReport(
		activities.SendPipelineCompletionNotificationActivityName,
		func(input workflowengine.ActivityInput) string {
			payload, err := workflowengine.DecodePayload[activities.SendPipelineCompletionNotificationInput](
				input.Payload,
			)
			require.NoError(t, err)
			return payload.WorkflowID
		},
	)
	recordReport(githubUpdate.Name(), func(input workflowengine.ActivityInput) string {
		payload, err := workflowengine.DecodePayload[activities.UpdateGitHubPRCommentInput](
			input.Payload,
		)
		require.NoError(t, err)
		return payload.WorkflowID
	})

	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{
			Name: "Parent Pipeline",
			Steps: []pipeline.StepDefinition{
				{StepSpec: pipeline.StepSpec{
					ID:   "child",
					Use:  "child-pipeline",
					With: pipeline.StepInputs{Payload: map[string]any{"pipeline_id": "child"}},
				}},
			},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":                           "https://example.test",
				"namespace":                         "tenant-1",
				"mobile_device_semaphore_ticket_id": "ticket-1",
				"mobile_device_semaphore_leader_device_id": "runner-1/device-1",
				"mobile_device_semaphore_owner_namespace":  "tenant-1",
				CompletionNotificationConfigKey:            true,
				GitHubPRCommentConfigKey: map[string]any{
					GitHubPRCommentConfigRepositoryKey:        "forkbombeu/issuer",
					GitHubPRCommentConfigPullRequestNumberKey: 17,
				},
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	})

	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, []string{"inner"}, captured)
	parentOnly := []string{"default-test-workflow-id"}
	require.Equal(t, map[string][]string{
		semaphoreDone.Name(): parentOnly,
		activities.SendPipelineCompletionNotificationActivityName: parentOnly,
		githubUpdate.Name(): parentOnly,
	}, reportedBy)
}
