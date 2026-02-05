// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"context"
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

func TestPipelineReportsSemaphoreDone(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	reportActivity := activities.NewReportMobileRunnerSemaphoreDoneActivity()
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
			steps []StepDefinition,
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
			payload, ok := input.Payload.(activities.ReportMobileRunnerSemaphoreDoneInput)
			if !ok {
				decoded, err := workflowengine.DecodePayload[activities.ReportMobileRunnerSemaphoreDoneInput](
					input.Payload,
				)
				if err != nil {
					return false
				}
				payload = decoded
			}
			return payload.TicketID == "ticket-1" &&
				payload.OwnerNamespace == "tenant-1" &&
				payload.LeaderRunnerID == "runner-1" &&
				payload.WorkflowID == "default-test-workflow-id" &&
				payload.RunID == "default-test-run-id"
		}),
	).
		Run(func(_ mock.Arguments) {
			runOrder = append(runOrder, "report")
		}).
		Return(workflowengine.ActivityResult{}, nil).
		Once()

	input := PipelineWorkflowInput{
		WorkflowDefinition: &WorkflowDefinition{
			Name:  "test-pipeline",
			Steps: []StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":                                  "https://example.test",
				"mobile_runner_semaphore_ticket_id":        "ticket-1",
				"mobile_runner_semaphore_leader_runner_id": "runner-1",
				"mobile_runner_semaphore_owner_namespace":  "tenant-1",
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	}

	env.ExecuteWorkflow(pipelineWf.Name(), input)

	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, []string{"cleanup", "report"}, runOrder)
	env.AssertExpectations(t)
}
