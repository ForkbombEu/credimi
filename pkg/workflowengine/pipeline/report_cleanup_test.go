// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestPipelineReportCleanupHookStoresReport(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	reportActivity := activities.NewPipelineReportGenerationActivity(
		nil,
		registry.StepActivityOutputKind,
	)
	env.RegisterActivityWithOptions(
		reportActivity.Execute,
		activity.RegisterOptions{Name: reportActivity.Name()},
	)
	env.RegisterWorkflowWithOptions(
		pipelineReportCleanupHookTestWorkflow,
		workflow.RegisterOptions{Name: "test-pipeline-report-cleanup-hook"},
	)

	var rawInput map[string]any
	env.OnActivity(
		reportActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			encoded, err := json.Marshal(input.Payload)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(encoded, &rawInput))
			return true
		}),
	).Return(
		workflowengine.ActivityResult{
			Output: activities.PipelineReportGenerationOutput{
				MarkdownSHA256: "abc",
				Filename:       "workflow-1.md",
				Warnings:       []string{"pipeline report storage failed: boom"},
			},
		},
		nil,
	)

	env.ExecuteWorkflow("test-pipeline-report-cleanup-hook")
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)

	require.Equal(t, "default-test-namespace", rawInput["namespace"])
	require.Equal(t, "default-test-workflow-id", rawInput["workflow_id"])
	require.Equal(t, "default-test-run-id", rawInput["run_id"])
	require.NotContains(t, rawInput, "app_url")
	require.NotContains(t, rawInput, "workflow_definition")
	require.NotContains(t, rawInput, "pipeline_output")
	require.Equal(t, map[string]any{
		setupWarningsOutputKey: []any{"evidence warning"},
	}, rawInput["pipeline_output_meta"])
	require.Len(t, rawInput["evidence"].(map[string]any)["credential_offers"], 1)

	var finalOutput map[string]any
	require.NoError(t, env.GetWorkflowResult(&finalOutput))
	require.Equal(
		t,
		[]any{"pipeline report storage failed: boom"},
		finalOutput[cleanupWarningsOutputKey],
	)
}

func TestPipelineReportCleanupHookWarnsWhenEvidenceMissing(t *testing.T) {
	finalOutput := map[string]any{"workflow_id": "workflow-1", "run_id": "run-1"}
	evidence, ok := pipelineEvidenceFromRunData(nil)
	require.False(t, ok)
	require.Empty(t, evidence)

	appendCleanupWarning(
		&finalOutput,
		"pipeline report generation skipped: missing pipeline evidence",
	)
	require.Equal(
		t,
		[]string{"pipeline report generation skipped: missing pipeline evidence"},
		finalOutput[cleanupWarningsOutputKey],
	)
}

func TestPipelineEvidenceFromRunDataDecodesMap(t *testing.T) {
	evidence, ok := pipelineEvidenceFromRunData(map[string]any{
		"credential_offers": []map[string]any{
			{"step_id": "credential-step"},
		},
	})
	require.True(t, ok)
	require.Len(t, evidence.CredentialOffers, 1)
}

func TestPipelineReportCleanupHelpers(t *testing.T) {
	result, err := decodePipelineReportOutput(workflowengine.ActivityResult{
		Output: map[string]any{
			"markdown_sha256": "abc",
			"filename":        "workflow-1.md",
			"fixture":         "workflow-1",
			"slug":            "workflow-1",
			"passed_count":    float64(3),
		},
	})
	require.NoError(t, err)
	require.Equal(t, "abc", result.MarkdownSHA256)
	require.Equal(t, "workflow-1.md", result.Filename)

	finalOutput := map[string]any{"workflow_id": "workflow-1"}
	require.Equal(t, "workflow-1", stringFinalOutputValue(&finalOutput, "workflow_id"))
	require.Empty(t, stringFinalOutputValue(nil, "workflow_id"))
	workflowID, runID := pipelineWorkflowIDs(nil, &map[string]any{
		"workflow_id": "workflow-1",
		"run_id":      "run-1",
	})
	require.Equal(t, "workflow-1", workflowID)
	require.Equal(t, "run-1", runID)
}

func TestPipelineReportCleanupHookSkipsWithoutEvidenceSteps(t *testing.T) {
	finalOutput := map[string]any{}
	err := PipelineReportCleanupHook(
		nil,
		&pipelineinternal.WorkflowDefinition{
			Steps: []pipelineinternal.StepDefinition{
				{StepSpec: pipelineinternal.StepSpec{Use: "http-request"}},
			},
		},
		nil,
		map[string]any{},
		map[string]any{},
		&finalOutput,
	)
	require.NoError(t, err)
	require.Empty(t, finalOutput)
}

func pipelineReportCleanupHookTestWorkflow(ctx workflow.Context) (map[string]any, error) {
	ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
	ctx = workflow.WithActivityOptions(ctx, ao)
	finalOutput := map[string]any{
		"credential-step":      map[string]any{"outputs": map[string]any{"offer": "large"}},
		setupWarningsOutputKey: []string{"evidence warning"},
	}
	runData := map[string]any{
		pipelineEvidenceRunDataKey: activities.PipelineEvidenceExtractionOutput{
			CredentialOffers: []map[string]any{
				{
					"step_id":          "credential-step",
					"credential_offer": map[string]any{"credential_issuer": "issuer"},
				},
			},
		},
	}
	wfDef := &pipelineinternal.WorkflowDefinition{
		Name: "report-pipeline",
		Steps: []pipelineinternal.StepDefinition{
			{
				StepSpec: pipelineinternal.StepSpec{
					ID:  "credential-step",
					Use: "credential-offer",
				},
			},
		},
	}
	err := PipelineReportCleanupHook(
		ctx,
		wfDef,
		&ao,
		map[string]any{"app_url": "https://credimi.test"},
		runData,
		&finalOutput,
	)
	return finalOutput, err
}
