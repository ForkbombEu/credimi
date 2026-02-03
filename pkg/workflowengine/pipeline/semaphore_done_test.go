// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

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
	env.AssertExpectations(t)
}
