// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"testing"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// The test environments below register no child workflows: the hook must
// extract evidence with the extraction activity alone, so any child workflow
// start would fail the workflow.

func TestPipelineEvidenceSetupHookAddsWarningsWithoutFailing(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(
			_ context.Context,
			_ workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: activities.PipelineEvidenceExtractionOutput{
					Warnings: []string{
						"no credential well-knowns or presentation results were extracted",
					},
				},
			}, nil
		},
		activity.RegisterOptions{Name: activities.PipelineEvidenceExtractionActivityName},
	)

	env.ExecuteWorkflow(testPipelineEvidenceSetupWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var output map[string]any
	require.NoError(t, env.GetWorkflowResult(&output))
	require.Equal(
		t,
		[]any{"no credential well-knowns or presentation results were extracted"},
		output[setupWarningsOutputKey],
	)
}

func TestPipelineEvidenceSetupHookStoresEvidence(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	evidenceActivity := activities.NewPipelineEvidenceExtractionActivity(nil)
	env.RegisterActivityWithOptions(
		evidenceActivity.Execute,
		activity.RegisterOptions{Name: evidenceActivity.Name()},
	)
	env.OnActivity(
		activities.PipelineEvidenceExtractionActivityName,
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, err := workflowengine.DecodePayload[activities.PipelineEvidenceExtractionInput](
				input.Payload,
			)
			require.NoError(t, err)
			require.Equal(t, "default-test-workflow-id", payload.WorkflowID)
			require.Equal(t, "default-test-run-id", payload.RunID)
			require.NotNil(t, payload.WorkflowDefinition)
			require.Empty(t, payload.InputErrors)
			return true
		}),
	).Return(workflowengine.ActivityResult{
		Output: activities.PipelineEvidenceExtractionOutput{
			CredentialWellKnowns: []map[string]any{
				{
					"step_id":       "cred-step",
					"credential_id": "tenant/credential-1",
					"well_known":    map[string]any{"credential_issuer": "issuer-1"},
				},
			},
			PresentationResults: []map[string]any{
				{
					"step_id":     "vp-step",
					"use_case_id": "tenant/use-case-1",
					"result":      map[string]any{"format": "jwt"},
				},
			},
		},
	}, nil).Once()

	env.ExecuteWorkflow(testPipelineEvidenceSetupWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, true, result["run_data_has_evidence"])
	require.Equal(t, false, result["final_output_has_evidence"])
}

func TestPipelineEvidenceSetupHookSendsOnlyDiscoverySteps(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	var definition *pipelineinternal.WorkflowDefinition
	env.RegisterActivityWithOptions(
		func(
			_ context.Context,
			input workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.PipelineEvidenceExtractionInput](
				input.Payload,
			)
			require.NoError(t, err)
			definition = payload.WorkflowDefinition
			return workflowengine.ActivityResult{
				Output: activities.PipelineEvidenceExtractionOutput{},
			}, nil
		},
		activity.RegisterOptions{Name: activities.PipelineEvidenceExtractionActivityName},
	)

	env.ExecuteWorkflow(testPipelineEvidenceSetupWorkflow)
	require.NoError(t, env.GetWorkflowError())

	require.NotNil(t, definition)
	require.Equal(t, "evidence-pipeline", definition.Name)
	stepIDs := make([]string, 0, len(definition.Steps))
	for _, step := range definition.Steps {
		stepIDs = append(stepIDs, step.ID)
	}
	require.Equal(t, []string{"cred-step", "vp-step"}, stepIDs)
	require.Equal(t, "tenant/credential-1", definition.Steps[0].With.Payload["credential_id"])
	require.Equal(t, "tenant/use-case-1", definition.Steps[1].With.Payload["use_case_id"])
}

func TestResolveEvidenceInputs(t *testing.T) {
	templated := map[string]any{"credential_id": "${{ credential.path }}"}
	fromEarlierStep := map[string]any{"use_case_id": "${{ create-session.outputs.id }}"}
	wfDef := &pipelineinternal.WorkflowDefinition{
		Name: "evidence-pipeline",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{
				ID:   "cred-step",
				Use:  "credential-offer",
				With: pipelineinternal.StepInputs{Payload: templated},
			}},
			{StepSpec: pipelineinternal.StepSpec{
				ID:   "create-session",
				Use:  "http-request",
				With: pipelineinternal.StepInputs{Payload: map[string]any{"url": "https://x"}},
			}},
			{StepSpec: pipelineinternal.StepSpec{
				ID:   "vp-step",
				Use:  "use-case-verification-deeplink",
				With: pipelineinternal.StepInputs{Payload: fromEarlierStep},
			}},
		},
	}
	runData := map[string]any{"credential": map[string]any{"path": "tenant/credential-9"}}

	def, inputErrors := resolveEvidenceInputs(wfDef, &runData)

	require.Len(t, def.Steps, 2)
	require.Equal(t, "tenant/credential-9", def.Steps[0].With.Payload["credential_id"])
	require.Equal(
		t,
		"${{ create-session.outputs.id }}",
		def.Steps[1].With.Payload["use_case_id"],
	)
	require.NotContains(t, inputErrors, "cred-step")
	require.Contains(t, inputErrors["vp-step"], "create-session.outputs.id")
	// The pipeline steps keep their own inputs for their own resolution.
	require.Equal(t, "${{ credential.path }}", templated["credential_id"])
	require.Equal(t, "${{ create-session.outputs.id }}", fromEarlierStep["use_case_id"])
}

func TestPipelineEvidenceSetupHelpers(t *testing.T) {
	wfDef := &pipelineinternal.WorkflowDefinition{
		Steps: []pipelineinternal.StepDefinition{
			{
				StepSpec: pipelineinternal.StepSpec{Use: "credential-offer"},
			},
		},
	}
	require.True(t, hasPipelineEvidenceStep(wfDef))
	require.False(t, hasPipelineEvidenceStep(&pipelineinternal.WorkflowDefinition{}))
	require.False(t, hasPipelineEvidenceStep(nil))

	finalOutput := map[string]any{"workflow_id": "workflow-1"}
	appendSetupWarning(&finalOutput, "warning-1")
	appendSetupWarnings(&finalOutput, []string{"warning-2"})
	require.Equal(t, "workflow-1", finalOutputValue(&finalOutput, "workflow_id"))
	require.Nil(t, finalOutputValue(&finalOutput, "missing"))
	require.Equal(t, []string{"warning-1", "warning-2"}, finalOutput[setupWarningsOutputKey])
}

func testPipelineEvidenceSetupWorkflow(ctx workflow.Context) (map[string]any, error) {
	wfDef := &pipelineinternal.WorkflowDefinition{
		Name: "evidence-pipeline",
		Steps: []pipelineinternal.StepDefinition{
			{
				StepSpec: pipelineinternal.StepSpec{
					ID:  "cred-step",
					Use: "credential-offer",
					With: pipelineinternal.StepInputs{
						Payload: map[string]any{"credential_id": "tenant/credential-1"},
					},
				},
			},
			{
				StepSpec: pipelineinternal.StepSpec{
					ID:   "fetch-step",
					Use:  "http-request",
					With: pipelineinternal.StepInputs{Payload: map[string]any{"url": "https://x"}},
				},
			},
			{
				StepSpec: pipelineinternal.StepSpec{
					ID:  "vp-step",
					Use: "use-case-verification-deeplink",
					With: pipelineinternal.StepInputs{
						Payload: map[string]any{"use_case_id": "tenant/use-case-1"},
					},
				},
			},
		},
	}
	runData := map[string]any{}
	finalOutput := map[string]any{}
	err := PipelineEvidenceSetupHook(
		ctx,
		wfDef,
		map[string]any{},
		&runData,
		&finalOutput,
		workflow.GetLogger(ctx),
	)
	_, runDataHasEvidence := runData[pipelineEvidenceRunDataKey]
	_, finalOutputHasEvidence := finalOutput[pipelineEvidenceRunDataKey]
	finalOutput["run_data_has_evidence"] = runDataHasEvidence
	finalOutput["final_output_has_evidence"] = finalOutputHasEvidence
	return finalOutput, err
}
