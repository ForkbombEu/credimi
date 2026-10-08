// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"maps"
	"strings"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
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

// evidenceExtractionTimeout bounds the extraction activity, which generates a
// deeplink (running StepCI for dynamic records) and contacts the issuer or
// verifier for every evidence step of the pipeline.
const evidenceExtractionTimeout = 30 * time.Minute

// PipelineEvidenceSetupHook extracts issuer and verifier evidence for the
// pipeline's credential-offer and use-case-verification-deeplink steps in a
// single activity, before the steps run. The activity generates its own
// deeplinks: a verifier serves a request object only once, so evidence cannot
// reuse the deeplink the wallet receives.
func PipelineEvidenceSetupHook(
	ctx workflow.Context,
	wfDef *pipelineinternal.WorkflowDefinition,
	_ map[string]any,
	runData *map[string]any,
	finalOutput *map[string]any,
	logger log.Logger,
) error {
	if !hasPipelineEvidenceStep(wfDef) {
		return nil
	}
	baseAO := PrepareWorkflowOptions(wfDef.Runtime).ActivityOptions
	discoveryDef, inputErrors := resolveEvidenceInputs(wfDef, runData)

	workflowID, runID := pipelineWorkflowIDs(ctx, finalOutput)
	extractionReq := workflowengine.ActivityInput{
		Payload: activities.PipelineEvidenceExtractionInput{
			WorkflowDefinition: discoveryDef,
			InputErrors:        inputErrors,
			WorkflowID:         workflowID,
			RunID:              runID,
		},
	}

	extractionAO := evidenceActivityOptions(&baseAO, evidenceExtractionTimeout, 1)
	extractionAO.StartToCloseTimeout = evidenceExtractionTimeout
	extractionAO.ScheduleToCloseTimeout = 0
	extractionCtx := workflow.WithActivityOptions(ctx, extractionAO)
	var extractionResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		extractionCtx,
		activities.PipelineEvidenceExtractionActivityName,
		extractionReq,
	).Get(extractionCtx, &extractionResult); err != nil {
		if temporal.IsCanceledError(err) {
			return err
		}
		appendSetupWarning(finalOutput, fmt.Sprintf("pipeline evidence extraction failed: %v", err))
		logger.Warn("Pipeline evidence extraction failed", "error", err)
		return nil
	}

	output, err := workflowengine.DecodeOutput[activities.PipelineEvidenceExtractionOutput](
		extractionResult.Output,
	)
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
	return nil
}

// resolveEvidenceInputs returns the pipeline's evidence steps with their inputs
// resolved against the run data available at setup, so the extraction activity
// input does not copy the whole definition. A step whose inputs cannot be
// resolved yet, such as one that reads an earlier step's output, keeps its raw
// inputs and is reported by step ID instead of failing the pipeline.
func resolveEvidenceInputs(
	wfDef *pipelineinternal.WorkflowDefinition,
	runData *map[string]any,
) (*pipelineinternal.WorkflowDefinition, map[string]string) {
	dataCtx := map[string]any{}
	if runData != nil && *runData != nil {
		dataCtx = *runData
	}
	def := &pipelineinternal.WorkflowDefinition{Name: wfDef.Name}
	inputErrors := map[string]string{}
	for _, step := range wfDef.Steps {
		if !isPipelineEvidenceStep(step.Use) {
			continue
		}
		resolved := step
		// ResolveInputs writes resolved values back into the maps it receives;
		// clone them so the pipeline step itself still resolves its own inputs.
		resolved.With = pipelineinternal.StepInputs{
			Config:  maps.Clone(step.With.Config),
			Payload: maps.Clone(step.With.Payload),
		}
		if err := pipelineinternal.ResolveInputs(&resolved, nil, dataCtx); err != nil {
			inputErrors[step.ID] = err.Error()
			resolved.With = step.With
		}
		def.Steps = append(def.Steps, resolved)
	}
	return def, inputErrors
}

func isPipelineEvidenceStep(use string) bool {
	return use == "credential-offer" || use == "use-case-verification-deeplink"
}

func hasPipelineEvidenceStep(wfDef *pipelineinternal.WorkflowDefinition) bool {
	if wfDef == nil {
		return false
	}
	for _, step := range wfDef.Steps {
		if isPipelineEvidenceStep(step.Use) {
			return true
		}
	}
	return false
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
