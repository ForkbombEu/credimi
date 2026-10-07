// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// evidenceDeeplinkStubs registers stand-ins for the registry child workflows
// that resolve evidence deeplinks and records how they were started.
type evidenceDeeplinkStubs struct {
	credentialErr error
	childIDs      []string
	payloads      []map[string]any
}

func (s *evidenceDeeplinkStubs) register(env *testsuite.TestWorkflowEnvironment) {
	stub := func(deeplink string, err func() error) func(
		workflow.Context, workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		return func(
			ctx workflow.Context,
			input workflowengine.WorkflowInput,
		) (workflowengine.WorkflowResult, error) {
			s.childIDs = append(s.childIDs, workflow.GetInfo(ctx).WorkflowExecution.ID)
			payload, _ := input.Payload.(map[string]any)
			s.payloads = append(s.payloads, payload)
			if e := err(); e != nil {
				return workflowengine.WorkflowResult{}, e
			}
			return workflowengine.WorkflowResult{Output: deeplink}, nil
		}
	}
	env.RegisterWorkflowWithOptions(
		stub("openid-credential-offer://?credential_offer=%7B%7D", func() error {
			return s.credentialErr
		}),
		workflow.RegisterOptions{Name: "Get a credential offer"},
	)
	env.RegisterWorkflowWithOptions(
		stub("haip-vp://?request_uri=https%3A%2F%2Fverifier.example%2Frequest", func() error {
			return nil
		}),
		workflow.RegisterOptions{Name: "Get use case verification deeplink"},
	)
}

func TestPipelineEvidenceSetupHookAddsWarningsWithoutFailing(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	(&evidenceDeeplinkStubs{}).register(env)

	evidenceActivity := activities.NewPipelineEvidenceExtractionActivity(nil)
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
	stubs := &evidenceDeeplinkStubs{}
	stubs.register(env)

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
			require.Equal(t, map[string]string{
				"cred-step": "openid-credential-offer://?credential_offer=%7B%7D",
				"vp-step":   "haip-vp://?request_uri=https%3A%2F%2Fverifier.example%2Frequest",
			}, payload.Deeplinks)
			require.Empty(t, payload.DeeplinkErrors)
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
	require.Equal(t, []string{
		"default-test-workflow-id-evidence-cred-step",
		"default-test-workflow-id-evidence-vp-step",
	}, stubs.childIDs)
	require.Equal(t, "tenant/credential-1", stubs.payloads[0]["credential_id"])
	require.Equal(t, "tenant/use-case-1", stubs.payloads[1]["use_case_id"])
	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, true, result["run_data_has_evidence"])
	require.Equal(t, false, result["final_output_has_evidence"])
}

func TestPipelineEvidenceSetupHookSendsOnlyDiscoverySteps(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	(&evidenceDeeplinkStubs{}).register(env)

	var definition *pipelineinternal.WorkflowDefinition
	evidenceActivity := activities.NewPipelineEvidenceExtractionActivity(nil)
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
		activity.RegisterOptions{Name: evidenceActivity.Name()},
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
}

func TestPipelineEvidenceSetupHookReportsDeeplinkFailures(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	stubs := &evidenceDeeplinkStubs{
		credentialErr: temporal.NewNonRetryableApplicationError(
			"credentials record tenant/credential-1 not found", "CRE233", nil,
		),
	}
	stubs.register(env)

	var input activities.PipelineEvidenceExtractionInput
	env.RegisterActivityWithOptions(
		func(
			_ context.Context,
			activityInput workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.PipelineEvidenceExtractionInput](
				activityInput.Payload,
			)
			require.NoError(t, err)
			input = payload
			return workflowengine.ActivityResult{
				Output: activities.PipelineEvidenceExtractionOutput{},
			}, nil
		},
		activity.RegisterOptions{Name: activities.PipelineEvidenceExtractionActivityName},
	)

	env.ExecuteWorkflow(testPipelineEvidenceSetupWorkflow)
	require.NoError(t, env.GetWorkflowError())

	require.Equal(t, map[string]string{
		"vp-step": "haip-vp://?request_uri=https%3A%2F%2Fverifier.example%2Frequest",
	}, input.Deeplinks)
	require.Contains(
		t,
		input.DeeplinkErrors["cred-step"],
		"credentials record tenant/credential-1 not found",
	)
}

func TestResolveEvidenceDeeplinksKeepsStepInputs(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	stubs := &evidenceDeeplinkStubs{}
	stubs.register(env)

	env.ExecuteWorkflow(testResolveTemplatedEvidenceDeeplinkWorkflow)
	require.NoError(t, env.GetWorkflowError())
	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))

	require.Empty(t, result["deeplink_errors"])
	require.Equal(t, []string{"default-test-workflow-id-evidence-cred-step"}, stubs.childIDs)
	require.Equal(t, "tenant/credential-9", stubs.payloads[0]["credential_id"])
	require.Equal(t, "${{ credential.path }}", result["step_credential_id"])
}

// testResolveTemplatedEvidenceDeeplinkWorkflow resolves the deeplink of a step
// whose input is a template and reports the step's input afterwards.
func testResolveTemplatedEvidenceDeeplinkWorkflow(ctx workflow.Context) (map[string]any, error) {
	payload := map[string]any{"credential_id": "${{ credential.path }}"}
	def := &pipelineinternal.WorkflowDefinition{
		Steps: []pipelineinternal.StepDefinition{{StepSpec: pipelineinternal.StepSpec{
			ID:   "cred-step",
			Use:  "credential-offer",
			With: pipelineinternal.StepInputs{Payload: payload},
		}}},
	}
	runData := map[string]any{"credential": map[string]any{"path": "tenant/credential-9"}}
	ao := workflow.ActivityOptions{StartToCloseTimeout: time.Minute}
	_, deeplinkErrors, err := resolveEvidenceDeeplinks(ctx, def, map[string]any{}, &runData, ao)
	return map[string]any{
		"deeplink_errors":    deeplinkErrors,
		"step_credential_id": payload["credential_id"],
	}, err
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
