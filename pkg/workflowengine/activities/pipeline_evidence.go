// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/credoffer"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/discovery"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/presentation"
)

const PipelineEvidenceExtractionActivityName = "Extract pipeline conformance evidence"

type PipelineEvidenceExtractionActivity struct {
	workflowengine.BaseActivity
}

type PipelineEvidenceExtractionInput struct {
	WorkflowDefinition *pipelineinternal.WorkflowDefinition `json:"workflow_definition"`
	CredimiBaseURL     string                               `json:"credimi_base_url"`
}

type PipelineEvidenceExtractionOutput struct {
	CredentialWellKnowns []map[string]any `json:"credential_well_knowns"`
	PresentationResults  []map[string]any `json:"presentation_results"`
	Warnings             []string         `json:"warnings,omitempty"`
}

func NewPipelineEvidenceExtractionActivity() *PipelineEvidenceExtractionActivity {
	return &PipelineEvidenceExtractionActivity{
		BaseActivity: workflowengine.BaseActivity{Name: PipelineEvidenceExtractionActivityName},
	}
}

func (a *PipelineEvidenceExtractionActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *PipelineEvidenceExtractionActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[PipelineEvidenceExtractionInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	if payload.WorkflowDefinition == nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			"workflow_definition is required",
		)
	}
	if strings.TrimSpace(payload.CredimiBaseURL) == "" {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
			"credimi_base_url is required",
		)
	}

	timeout := 30 * time.Second
	client := &http.Client{Timeout: timeout}
	discovered, err := discoverWorkflowDefinition(payload.WorkflowDefinition)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			err.Error(),
		)
	}

	out := PipelineEvidenceExtractionOutput{}
	out.CredentialWellKnowns = extractCredentialWellKnowns(
		ctx,
		client,
		discovered.CredentialOfferSteps,
		payload,
		&out.Warnings,
	)
	out.PresentationResults = extractPresentationResults(
		ctx,
		client,
		discovered.PresentationRequestSteps,
		payload,
		timeout,
		&out.Warnings,
	)
	if len(out.CredentialWellKnowns) == 0 && len(out.PresentationResults) == 0 {
		out.Warnings = append(out.Warnings, "no credential well-knowns or presentation results were extracted")
	}

	return workflowengine.ActivityResult{Output: out}, nil
}

func discoverWorkflowDefinition(def *pipelineinternal.WorkflowDefinition) (*discovery.Result, error) {
	raw, err := json.Marshal(map[string]any{"workflow_definition": def})
	if err != nil {
		return nil, fmt.Errorf("marshal workflow definition: %w", err)
	}
	discovered, err := discovery.Discover(raw)
	if err != nil {
		return nil, fmt.Errorf("discover evidence steps: %w", err)
	}
	return discovered, nil
}

func extractCredentialWellKnowns(
	ctx context.Context,
	client *http.Client,
	steps []discovery.Step,
	payload PipelineEvidenceExtractionInput,
	warnings *[]string,
) []map[string]any {
	results := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		if ctx.Err() != nil {
			*warnings = append(*warnings, ctx.Err().Error())
			return results
		}
		res := credoffer.Resolve(
			client,
			payload.CredimiBaseURL,
			step.CredentialID,
			"auto",
			5,
		)
		res.StepID = step.StepID
		if res.Status != "ok" || res.CredentialOffer == nil {
			appendCredentialWarning(warnings, step, res)
			continue
		}
		wellKnown, fetch, err := credoffer.FetchIssuerMetadata(client, res.CredentialOffer)
		if err != nil {
			*warnings = append(
				*warnings,
				fmt.Sprintf("failed to fetch issuer metadata for step %s: %v", step.StepID, err),
			)
			continue
		}
		results = append(results, map[string]any{
			"step_id":       step.StepID,
			"credential_id": step.CredentialID,
			"well_known":    decodeRawJSON(wellKnown),
			"fetch":         fetch,
		})
	}
	return results
}

func extractPresentationResults(
	ctx context.Context,
	client *http.Client,
	steps []discovery.Step,
	payload PipelineEvidenceExtractionInput,
	timeout time.Duration,
	warnings *[]string,
) []map[string]any {
	results := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		if ctx.Err() != nil {
			*warnings = append(*warnings, ctx.Err().Error())
			return results
		}
		res := presentation.Resolve(
			client,
			payload.CredimiBaseURL,
			step.UseCaseID,
			"auto",
			"auto",
			timeout,
		)
		res.StepID = step.StepID
		if res.Status != "ok" {
			appendPresentationWarning(warnings, step, res)
			continue
		}
		results = append(results, map[string]any{
			"step_id":     step.StepID,
			"use_case_id": step.UseCaseID,
			"result":      buildPresentationResult(res),
		})
	}
	return results
}

func buildPresentationResult(res *presentation.Result) map[string]any {
	out := map[string]any{
		"deeplink_uri":           res.DeeplinkURI,
		"source_request_uri":     res.RequestURI,
		"request_uri_method":     res.RequestURIMethod,
		"post_strategy_selected": res.PostStrategy,
		"raw":                    res.RequestURIRaw,
		"fetch":                  res.RequestURIFetch,
	}
	if res.RequestObject != nil {
		out["format"] = "jwt"
		out["header"] = decodeRawJSON(res.RequestObject.Header)
		out["payload"] = decodeRawJSON(res.RequestObject.Payload)
		out["signature"] = res.RequestObject.Signature
		out["signature_present"] = res.RequestObject.SignaturePresent
	}
	return out
}

func appendCredentialWarning(warnings *[]string, step discovery.Step, res *credoffer.Result) {
	if res != nil && res.Error != nil {
		*warnings = append(
			*warnings,
			fmt.Sprintf("failed to extract credential evidence for step %s: %s", step.StepID, res.Error.Error.Message),
		)
		return
	}
	*warnings = append(*warnings, fmt.Sprintf("failed to extract credential evidence for step %s", step.StepID))
}

func appendPresentationWarning(warnings *[]string, step discovery.Step, res *presentation.Result) {
	if res != nil && res.Error != nil {
		*warnings = append(
			*warnings,
			fmt.Sprintf("failed to extract presentation evidence for step %s: %s", step.StepID, res.Error.Error.Message),
		)
		return
	}
	*warnings = append(*warnings, fmt.Sprintf("failed to extract presentation evidence for step %s", step.StepID))
}

func decodeRawJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return string(raw)
	}
	return decoded
}
