// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/utils"
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

	appURL, _ := config["app_url"].(string)
	if strings.TrimSpace(appURL) == "" {
		appendCleanupWarning(finalOutput, "pipeline report generation skipped: missing app_url")
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

	reportActivity := activities.NewPipelineReportGenerationActivity()
	workflowID, runID := pipelineWorkflowIDs(ctx, finalOutput)
	reportReq := workflowengine.ActivityInput{
		Payload: activities.PipelineReportGenerationInput{
			WorkflowDefinition: wfDef,
			PipelineOutput:     copyStringAnyMap(finalOutput),
			Evidence:           evidence,
			WorkflowID:         workflowID,
			RunID:              runID,
		},
	}

	reportCtx := workflow.WithActivityOptions(
		cleanupCtx,
		evidenceActivityOptions(&baseAO, 2*time.Minute, 1),
	)
	var reportResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(reportCtx, reportActivity.Name(), reportReq).
		Get(reportCtx, &reportResult); err != nil {
		if temporal.IsCanceledError(err) {
			return err
		}
		appendCleanupWarning(finalOutput, fmt.Sprintf("pipeline report generation failed: %v", err))
		return nil
	}

	reportOutput, err := decodePipelineReportOutput(reportResult)
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
	if strings.TrimSpace(reportOutput.Markdown) == "" {
		return nil
	}
	if workflowID == "" || runID == "" {
		appendCleanupWarning(
			finalOutput,
			"pipeline report storage skipped: missing workflow_id or run_id",
		)
		return nil
	}

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	updateReq := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodPost,
			URL: utils.JoinURL(
				appURL,
				"api",
				"pipeline",
				"pipeline-execution-results",
				"report",
			),
			ExpectedStatus: http.StatusOK,
			Timeout:        "30",
			Body: map[string]any{
				"workflow_id": workflowID,
				"run_id":      runID,
				"filename":    reportOutput.Filename,
				"markdown":    reportOutput.Markdown,
			},
		},
	}

	updateCtx := workflow.WithActivityOptions(
		cleanupCtx,
		evidenceActivityOptions(&baseAO, 2*time.Minute, 5),
	)
	var updateResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(updateCtx, internalHTTPActivity.Name(), updateReq).
		Get(updateCtx, &updateResult); err != nil {
		if temporal.IsCanceledError(err) {
			return err
		}
		appendCleanupWarning(finalOutput, fmt.Sprintf("pipeline report storage failed: %v", err))
	}

	return nil
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

func decodePipelineReportOutput(
	result workflowengine.ActivityResult,
) (activities.PipelineReportGenerationOutput, error) {
	var out activities.PipelineReportGenerationOutput
	raw, err := json.Marshal(result.Output)
	if err != nil {
		return out, fmt.Errorf("marshal output: %w", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode output: %w", err)
	}
	return out, nil
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

func copyStringAnyMap(value *map[string]any) map[string]any {
	if value == nil || *value == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(*value))
	for k, v := range *value {
		out[k] = v
	}
	return out
}

func stringFinalOutputValue(finalOutput *map[string]any, key string) string {
	value, _ := finalOutputValue(finalOutput, key).(string)
	return value
}
