// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"net/http"
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

func TestPipelineEvidenceSetupHookAddsWarningsWithoutFailing(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	evidenceActivity := activities.NewPipelineEvidenceExtractionActivity()
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
		activity.RegisterOptions{Name: evidenceActivity.Name()},
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

	evidenceActivity := activities.NewPipelineEvidenceExtractionActivity()
	env.RegisterActivityWithOptions(
		func(
			_ context.Context,
			_ workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
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
			}, nil
		},
		activity.RegisterOptions{Name: evidenceActivity.Name()},
	)

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	env.RegisterActivityWithOptions(
		func(
			_ context.Context,
			_ workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: map[string]any{"status": http.StatusOK},
			}, nil
		},
		activity.RegisterOptions{Name: internalHTTPActivity.Name()},
	)
	env.OnActivity(
		internalHTTPActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, err := workflowengine.DecodePayload[activities.InternalHTTPActivityPayload](
				input.Payload,
			)
			require.NoError(t, err)
			require.Equal(t, http.MethodPost, payload.Method)
			require.Equal(
				t,
				"https://credimi.test/api/pipeline/pipeline-execution-results/evidence",
				payload.URL,
			)
			body, ok := payload.Body.(map[string]any)
			require.True(t, ok)
			require.Equal(t, "workflow-1", body["workflow_id"])
			require.Equal(t, "run-1", body["run_id"])
			require.Len(t, body["credential_well_knowns"], 1)
			require.Len(t, body["presentation_results"], 1)
			return true
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{"status": http.StatusOK}}, nil).Once()

	env.ExecuteWorkflow(testPipelineEvidenceSetupWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, true, result["run_data_has_evidence"])
	require.Equal(t, false, result["final_output_has_evidence"])
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

	finalOutput := map[string]any{"workflow-id": "workflow-1"}
	appendSetupWarning(&finalOutput, "warning-1")
	appendSetupWarnings(&finalOutput, []string{"warning-2"})
	require.Equal(t, "workflow-1", finalOutputValue(&finalOutput, "workflow-id"))
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
		},
	}
	runData := map[string]any{}
	finalOutput := map[string]any{
		"workflow-id":     "workflow-1",
		"workflow-run-id": "run-1",
	}
	err := PipelineEvidenceSetupHook(
		ctx,
		wfDef,
		map[string]any{"app_url": "https://credimi.test"},
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
