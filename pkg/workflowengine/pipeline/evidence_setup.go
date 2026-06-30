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
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	setupWarningsOutputKey     = "setup_warnings"
	cleanupWarningsOutputKey   = "cleanup_warnings"
	pipelineEvidenceRunDataKey = "pipeline_evidence"
)

func PipelineEvidenceSetupHook(
	ctx workflow.Context,
	wfDef *pipelineinternal.WorkflowDefinition,
	config map[string]any,
	runData *map[string]any,
	finalOutput *map[string]any,
	logger log.Logger,
) error {
	if !hasPipelineEvidenceStep(wfDef) {
		return nil
	}
	baseAO := PrepareWorkflowOptions(wfDef.Runtime).ActivityOptions

	appURL, ok := config["app_url"].(string)
	if !ok || strings.TrimSpace(appURL) == "" {
		appendSetupWarning(finalOutput, "pipeline evidence extraction skipped: missing app_url")
		return nil
	}

	extractionActivity := activities.NewPipelineEvidenceExtractionActivity()
	extractionReq := workflowengine.ActivityInput{
		Payload: activities.PipelineEvidenceExtractionInput{
			WorkflowDefinition: wfDef,
			CredimiBaseURL:     appURL,
		},
	}

	extractionCtx := workflow.WithActivityOptions(
		ctx,
		evidenceActivityOptions(&baseAO, 5*time.Minute, 1),
	)
	var extractionResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(extractionCtx, extractionActivity.Name(), extractionReq).
		Get(extractionCtx, &extractionResult); err != nil {
		if temporal.IsCanceledError(err) {
			return err
		}
		appendSetupWarning(finalOutput, fmt.Sprintf("pipeline evidence extraction failed: %v", err))
		logger.Warn("Pipeline evidence extraction failed", "error", err)
		return nil
	}

	output, err := decodePipelineEvidenceOutput(extractionResult)
	if err != nil {
		appendSetupWarning(
			finalOutput,
			fmt.Sprintf("pipeline evidence extraction output invalid: %v", err),
		)
		logger.Warn("Pipeline evidence extraction output invalid", "error", err)
		return nil
	}
	appendSetupWarnings(finalOutput, output.Warnings)
	SetRunDataValue(runData, pipelineEvidenceRunDataKey, output)
	if len(output.CredentialWellKnowns) == 0 && len(output.PresentationResults) == 0 {
		return nil
	}
	workflowID, runID := pipelineWorkflowIDs(ctx, finalOutput)
	if workflowID == "" || runID == "" {
		appendSetupWarning(
			finalOutput,
			"pipeline evidence storage skipped: missing workflow_id or run_id",
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
				"evidence",
			),
			ExpectedStatus: http.StatusOK,
			Timeout:        "30",
			Body: map[string]any{
				"workflow_id":            workflowID,
				"run_id":                 runID,
				"credential_well_knowns": output.CredentialWellKnowns,
				"presentation_results":   output.PresentationResults,
			},
		},
	}

	updateCtx := workflow.WithActivityOptions(
		ctx,
		evidenceActivityOptions(&baseAO, 2*time.Minute, 5),
	)
	var updateResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(updateCtx, internalHTTPActivity.Name(), updateReq).
		Get(updateCtx, &updateResult); err != nil {
		if temporal.IsCanceledError(err) {
			return err
		}
		appendSetupWarning(finalOutput, fmt.Sprintf("pipeline evidence storage failed: %v", err))
		logger.Warn("Pipeline evidence storage failed", "error", err)
	}

	return nil
}

func hasPipelineEvidenceStep(wfDef *pipelineinternal.WorkflowDefinition) bool {
	if wfDef == nil {
		return false
	}
	for _, step := range wfDef.Steps {
		if step.Use == "credential-offer" || step.Use == "use-case-verification-deeplink" {
			return true
		}
	}
	return false
}

func decodePipelineEvidenceOutput(
	result workflowengine.ActivityResult,
) (activities.PipelineEvidenceExtractionOutput, error) {
	var out activities.PipelineEvidenceExtractionOutput
	raw, err := json.Marshal(result.Output)
	if err != nil {
		return out, fmt.Errorf("marshal output: %w", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode output: %w", err)
	}
	return out, nil
}

func evidenceActivityOptions(
	base *workflow.ActivityOptions,
	startToClose time.Duration,
	maxAttempts int32,
) workflow.ActivityOptions {
	var opts workflow.ActivityOptions
	if base != nil {
		opts = *base
	}
	if opts.StartToCloseTimeout == 0 {
		opts.StartToCloseTimeout = startToClose
	}
	opts.RetryPolicy = &temporal.RetryPolicy{
		InitialInterval: time.Second,
		MaximumInterval: 5 * time.Second,
		MaximumAttempts: maxAttempts,
	}
	return opts
}

func appendSetupWarnings(finalOutput *map[string]any, warnings []string) {
	for _, warning := range warnings {
		appendSetupWarning(finalOutput, warning)
	}
}

func appendSetupWarning(finalOutput *map[string]any, warning string) {
	appendOutputWarning(finalOutput, setupWarningsOutputKey, warning)
}

func appendCleanupWarning(finalOutput *map[string]any, warning string) {
	appendOutputWarning(finalOutput, cleanupWarningsOutputKey, warning)
}

func appendOutputWarning(finalOutput *map[string]any, key string, warning string) {
	warning = strings.TrimSpace(warning)
	if warning == "" {
		return
	}
	if *finalOutput == nil {
		*finalOutput = map[string]any{}
	}
	existing, _ := (*finalOutput)[key].([]string)
	existing = append(existing, warning)
	(*finalOutput)[key] = existing
}

func finalOutputValue(finalOutput *map[string]any, key string) any {
	if finalOutput == nil || *finalOutput == nil {
		return ""
	}
	return (*finalOutput)[key]
}
