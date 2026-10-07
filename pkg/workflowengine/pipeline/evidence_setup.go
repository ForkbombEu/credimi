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
	discoveryDef := evidenceDiscoveryDefinition(wfDef)

	deeplinks, deeplinkErrors, err := resolveEvidenceDeeplinks(
		ctx,
		discoveryDef,
		config,
		runData,
		baseAO,
	)
	if err != nil {
		return err
	}

	workflowID, runID := pipelineWorkflowIDs(ctx, finalOutput)
	extractionReq := workflowengine.ActivityInput{
		Payload: activities.PipelineEvidenceExtractionInput{
			WorkflowDefinition: discoveryDef,
			Deeplinks:          deeplinks,
			DeeplinkErrors:     deeplinkErrors,
			WorkflowID:         workflowID,
			RunID:              runID,
		},
	}

	extractionCtx := workflow.WithActivityOptions(
		ctx,
		evidenceActivityOptions(&baseAO, 5*time.Minute, 1),
	)
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

// evidenceDeeplinkStepPrefix keeps the child workflow that resolves an evidence
// deeplink apart from the child workflow of the pipeline step itself.
const evidenceDeeplinkStepPrefix = "evidence-"

// resolveEvidenceDeeplinks resolves the deeplink of every evidence step through
// the same registry child workflow the step runs (credential offer or use case
// verification deeplink), so evidence extraction never calls Credimi over HTTP.
// A step that fails to resolve is reported by step ID instead of failing the
// pipeline; only cancellation is returned.
func resolveEvidenceDeeplinks(
	ctx workflow.Context,
	def *pipelineinternal.WorkflowDefinition,
	config map[string]any,
	runData *map[string]any,
	ao workflow.ActivityOptions,
) (map[string]string, map[string]string, error) {
	dataCtx := map[string]any{}
	if runData != nil && *runData != nil {
		dataCtx = *runData
	}
	deeplinks := map[string]string{}
	deeplinkErrors := map[string]string{}
	for _, step := range def.Steps {
		// ResolveInputs writes resolved values back into the maps it receives;
		// clone them so the pipeline step itself still resolves its own inputs.
		with := pipelineinternal.StepInputs{
			Config:  maps.Clone(step.With.Config),
			Payload: maps.Clone(step.With.Payload),
		}
		output, err := ExecuteStep(
			evidenceDeeplinkStepPrefix+step.ID,
			step.Use,
			with,
			step.ActivityOptions,
			ctx,
			config,
			dataCtx,
			ao,
		)
		if err != nil {
			if temporal.IsCanceledError(err) {
				return nil, nil, err
			}
			deeplinkErrors[step.ID] = err.Error()
			continue
		}
		deeplink, ok := output.(string)
		if !ok || strings.TrimSpace(deeplink) == "" {
			deeplinkErrors[step.ID] = fmt.Sprintf("%s returned no deeplink", step.Use)
			continue
		}
		deeplinks[step.ID] = deeplink
	}
	return deeplinks, deeplinkErrors, nil
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

// evidenceDiscoveryDefinition keeps only the steps evidence discovery reads, so the
// extraction activity input does not copy the whole definition.
func evidenceDiscoveryDefinition(
	wfDef *pipelineinternal.WorkflowDefinition,
) *pipelineinternal.WorkflowDefinition {
	discovery := &pipelineinternal.WorkflowDefinition{Name: wfDef.Name}
	for _, step := range wfDef.Steps {
		if step.Use == "credential-offer" || step.Use == "use-case-verification-deeplink" {
			discovery.Steps = append(discovery.Steps, step)
		}
	}
	return discovery
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
