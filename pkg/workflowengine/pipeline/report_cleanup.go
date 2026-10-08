// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"fmt"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func PipelineReportCleanupHook(
	ctx workflow.Context,
	wfDef *pipelineinternal.WorkflowDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData map[string]any,
	finalOutput *map[string]any,
) error {
	if wfDef == nil || !hasPipelineEvidenceStep(wfDef) {
		return nil
	}

	evidence, ok := pipelineEvidenceFromRunData(runData[pipelineEvidenceRunDataKey])
	if !ok {
		appendCleanupWarning(
			finalOutput,
			"pipeline report generation skipped: missing pipeline evidence",
		)
		return nil
	}

	baseAO := workflow.ActivityOptions{}
	if ao != nil {
		baseAO = *ao
	}
	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)

	workflowID, runID := pipelineWorkflowIDs(ctx, finalOutput)
	reportReq := workflowengine.ActivityInput{
		Payload: activities.PipelineReportGenerationInput{
			Namespace:          workflow.GetInfo(ctx).Namespace,
			WorkflowID:         workflowID,
			RunID:              runID,
			PipelineOutputMeta: pipelineOutputMeta(*finalOutput, wfDef),
			Evidence:           evidence,
		},
	}

	// The activity reads the run's history, generates the report and stores it.
	reportCtx := workflow.WithActivityOptions(
		cleanupCtx,
		evidenceActivityOptions(&baseAO, 5*time.Minute, 1),
	)
	var reportResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		reportCtx,
		activities.PipelineReportGenerationActivityName,
		reportReq,
	).Get(reportCtx, &reportResult); err != nil {
		if temporal.IsCanceledError(err) {
			return err
		}
		appendCleanupWarning(finalOutput, fmt.Sprintf("pipeline report generation failed: %v", err))
		return nil
	}

	reportOutput, err := workflowengine.DecodeOutput[activities.PipelineReportGenerationOutput](
		reportResult.Output,
	)
	if err != nil {
		appendCleanupWarning(
			finalOutput,
			fmt.Sprintf("pipeline report generation output invalid: %v", err),
		)
		return nil
	}
	for _, warning := range reportOutput.Warnings {
		appendCleanupWarning(finalOutput, warning)
	}
	return nil
}

// pipelineOutputMeta returns the finalOutput entries that are not main step outputs. Step
// outputs are recorded in history; these entries exist only in the workflow's memory.
func pipelineOutputMeta(
	finalOutput map[string]any,
	wfDef *pipelineinternal.WorkflowDefinition,
) map[string]any {
	mainStepIDs := make(map[string]struct{}, len(wfDef.Steps))
	for _, step := range wfDef.Steps {
		mainStepIDs[step.ID] = struct{}{}
	}
	meta := make(map[string]any)
	for key, value := range finalOutput {
		if _, isStep := mainStepIDs[key]; !isStep {
			meta[key] = value
		}
	}
	return meta
}

func pipelineEvidenceFromRunData(raw any) (activities.PipelineEvidenceExtractionOutput, bool) {
	if raw == nil {
		return activities.PipelineEvidenceExtractionOutput{}, false
	}
	switch evidence := raw.(type) {
	case activities.PipelineEvidenceExtractionOutput:
		return evidence, true
	case *activities.PipelineEvidenceExtractionOutput:
		if evidence == nil {
			return activities.PipelineEvidenceExtractionOutput{}, false
		}
		return *evidence, true
	default:
		var out activities.PipelineEvidenceExtractionOutput
		b, err := json.Marshal(raw)
		if err != nil {
			return out, false
		}
		if err := json.Unmarshal(b, &out); err != nil {
			return out, false
		}
		return out, true
	}
}

func pipelineWorkflowIDs(ctx workflow.Context, finalOutput *map[string]any) (string, string) {
	workflowID := stringFinalOutputValue(finalOutput, "workflow_id")
	runID := stringFinalOutputValue(finalOutput, "run_id")
	if (workflowID == "" || runID == "") && ctx != nil {
		info := workflow.GetInfo(ctx)
		if workflowID == "" {
			workflowID = info.WorkflowExecution.ID
		}
		if runID == "" {
			runID = info.WorkflowExecution.RunID
		}
	}
	return workflowID, runID
}

func stringFinalOutputValue(finalOutput *map[string]any, key string) string {
	value, _ := finalOutputValue(finalOutput, key).(string)
	return value
}
