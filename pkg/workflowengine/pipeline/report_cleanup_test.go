// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"strings"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestPipelineReportCleanupHookStoresReport(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	reportActivity := activities.NewPipelineReportGenerationActivity()
	internalHTTPActivity := activities.NewInternalHTTPActivity()
	env.RegisterActivityWithOptions(
		reportActivity.Execute,
		activity.RegisterOptions{Name: reportActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		internalHTTPActivity.Execute,
		activity.RegisterOptions{Name: internalHTTPActivity.Name()},
	)
	env.RegisterWorkflowWithOptions(
		pipelineReportCleanupHookTestWorkflow,
		workflow.RegisterOptions{Name: "test-pipeline-report-cleanup-hook"},
	)

	env.OnActivity(
		reportActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(activities.PipelineReportGenerationInput)
			if !ok {
				decoded, err := workflowengine.DecodePayload[activities.PipelineReportGenerationInput](
					input.Payload,
				)
				if err != nil {
					return false
				}
				payload = decoded
			}
			return payload.WorkflowID == "default-test-workflow-id" &&
				len(payload.Evidence.CredentialOffers) == 1
		}),
	).Return(
		workflowengine.ActivityResult{
			Output: activities.PipelineReportGenerationOutput{
				Markdown: "# Report",
				Filename: "workflow-1.md",
			},
		},
		nil,
	)
	env.OnActivity(
		internalHTTPActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(activities.InternalHTTPActivityPayload)
			if !ok {
				decoded, err := workflowengine.DecodePayload[activities.InternalHTTPActivityPayload](
					input.Payload,
				)
				if err != nil {
					return false
				}
				payload = decoded
			}
			require.Contains(t, payload.URL, "/api/pipeline/pipeline-execution-results/report")
			body, ok := payload.Body.(map[string]any)
			if !ok {
				return false
			}
			return strings.Contains(
				payload.URL,
				"/api/pipeline/pipeline-execution-results/report",
			) &&
				body["workflow_id"] == "default-test-workflow-id" &&
				body["run_id"] == "default-test-run-id" &&
				body["filename"] == "workflow-1.md" &&
				body["markdown"] == "# Report"
		}),
	).Return(workflowengine.ActivityResult{}, nil)

	env.ExecuteWorkflow("test-pipeline-report-cleanup-hook")
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
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

func TestPipelineReportCleanupHookWarnsWhenAppURLMissing(t *testing.T) {
	finalOutput := map[string]any{"workflow_id": "workflow-1", "run_id": "run-1"}
	err := PipelineReportCleanupHook(
		nil,
		&pipelineinternal.WorkflowDefinition{
			Steps: []pipelineinternal.StepDefinition{
				{StepSpec: pipelineinternal.StepSpec{Use: "credential-offer"}},
			},
		},
		nil,
		map[string]any{},
		map[string]any{},
		&finalOutput,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{"pipeline report generation skipped: missing app_url"},
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
			"markdown":     "# Report",
			"filename":     "workflow-1.md",
			"fixture":      "workflow-1",
			"slug":         "workflow-1",
			"passed_count": float64(3),
		},
	})
	require.NoError(t, err)
	require.Equal(t, "# Report", result.Markdown)
	require.Equal(t, "workflow-1.md", result.Filename)

	finalOutput := map[string]any{"workflow_id": "workflow-1"}
	copied := copyStringAnyMap(&finalOutput)
	copied["workflow_id"] = "changed"
	require.Equal(t, "workflow-1", finalOutput["workflow_id"])
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
	finalOutput := map[string]any{}
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
