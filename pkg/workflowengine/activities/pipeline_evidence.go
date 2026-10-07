// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/credoffer"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/discovery"
	"github.com/forkbombeu/eudi-conformance-evidence/pkg/presentation"
	"github.com/pocketbase/pocketbase/core"
)

const PipelineEvidenceExtractionActivityName = "Extract pipeline conformance evidence"

type PipelineEvidenceExtractionActivity struct {
	workflowengine.BaseActivity
	app core.App
	// storeRetrySleep waits d between evidence storage attempts and returns
	// ctx.Err() when ctx ends first.
	storeRetrySleep func(ctx context.Context, d time.Duration) error
}

// PipelineEvidenceExtractionInput carries the pipeline's evidence steps, the
// deeplink the pipeline resolved in-process for each step (keyed by step ID, or
// the resolution error), and the run whose pipeline result stores the evidence.
type PipelineEvidenceExtractionInput struct {
	WorkflowDefinition *pipelineinternal.WorkflowDefinition `json:"workflow_definition"`
	Deeplinks          map[string]string                    `json:"deeplinks,omitempty"`
	DeeplinkErrors     map[string]string                    `json:"deeplink_errors,omitempty"`
	WorkflowID         string                               `json:"workflow_id"`
	RunID              string                               `json:"run_id"`
}

type PipelineEvidenceExtractionOutput struct {
	CredentialOffers     []map[string]any `json:"credential_offers"`
	CredentialWellKnowns []map[string]any `json:"credential_well_knowns"`
	PresentationResults  []map[string]any `json:"presentation_results"`
	Warnings             []string         `json:"warnings,omitempty"`
}

func NewPipelineEvidenceExtractionActivity(app core.App) *PipelineEvidenceExtractionActivity {
	return &PipelineEvidenceExtractionActivity{
		BaseActivity:    workflowengine.BaseActivity{Name: PipelineEvidenceExtractionActivityName},
		app:             app,
		storeRetrySleep: sleepContext,
	}
}

func (a *PipelineEvidenceExtractionActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute resolves the credential offers and presentation results of the
// pipeline's evidence steps from the deeplinks the pipeline resolved in-process and stores the
// extracted evidence on the run's pipeline result. Storage is retried while
// the pipeline result is not yet created or the database fails; a final
// storage failure is reported as a warning.
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
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
				Message: "workflow_definition is required",
			},
		)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	discovered, err := discoverWorkflowDefinition(payload.WorkflowDefinition)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
				Message: err.Error(),
			},
		)
	}

	out := PipelineEvidenceExtractionOutput{}
	var evidenceErr error
	out.CredentialOffers, out.CredentialWellKnowns, evidenceErr = extractCredentialEvidence(
		ctx,
		client,
		discovered.CredentialOfferSteps,
		payload,
		&out.Warnings,
	)
	if evidenceErr != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed].Code,
			Summary: errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed].Description,
			Message: evidenceErr.Error(),
		})
	}
	out.PresentationResults = extractPresentationResults(
		ctx,
		client,
		discovered.PresentationRequestSteps,
		payload,
		&out.Warnings,
	)
	if len(out.CredentialWellKnowns) == 0 && len(out.PresentationResults) == 0 {
		out.Warnings = append(
			out.Warnings,
			"no credential well-knowns or presentation results were extracted",
		)
		return workflowengine.ActivityResult{Output: out}, nil
	}
	a.storeEvidence(ctx, payload, &out)

	return workflowengine.ActivityResult{Output: out}, nil
}

// evidenceStoreRetryWaits are the default waits between evidence storage
// attempts. The pipeline result may be created after the evidence setup hook
// starts, so a missing result or a database failure is retried.
var evidenceStoreRetryWaits = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	5 * time.Second,
}

func (a *PipelineEvidenceExtractionActivity) storeEvidence(
	ctx context.Context,
	payload PipelineEvidenceExtractionInput,
	out *PipelineEvidenceExtractionOutput,
) {
	if strings.TrimSpace(payload.WorkflowID) == "" || strings.TrimSpace(payload.RunID) == "" {
		out.Warnings = append(
			out.Warnings,
			"pipeline evidence storage skipped: missing workflow_id or run_id",
		)
		return
	}
	if err := a.storeEvidenceWithRetry(ctx, payload, out); err != nil {
		out.Warnings = append(
			out.Warnings,
			fmt.Sprintf("pipeline evidence storage failed: %v", err),
		)
	}
}

func (a *PipelineEvidenceExtractionActivity) storeEvidenceWithRetry(
	ctx context.Context,
	payload PipelineEvidenceExtractionInput,
	out *PipelineEvidenceExtractionOutput,
) error {
	attempts := len(evidenceStoreRetryWaits) + 1
	for attempt := 1; ; attempt++ {
		err := pipelineresults.StoreEvidence(
			a.app,
			payload.WorkflowID,
			payload.RunID,
			out.CredentialWellKnowns,
			out.PresentationResults,
		)
		if err == nil || errors.Is(err, pipelineresults.ErrInvalidInput) {
			return err
		}
		if attempt == attempts {
			return fmt.Errorf("after %d attempts: %w", attempts, err)
		}
		if sleepErr := a.storeRetrySleep(ctx, evidenceStoreRetryWaits[attempt-1]); sleepErr != nil {
			return fmt.Errorf("%w (retry canceled: %w)", err, sleepErr)
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func discoverWorkflowDefinition(
	def *pipelineinternal.WorkflowDefinition,
) (*discovery.Result, error) {
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

func extractCredentialEvidence(
	ctx context.Context,
	client *http.Client,
	steps []discovery.Step,
	payload PipelineEvidenceExtractionInput,
	warnings *[]string,
) ([]map[string]any, []map[string]any, error) {
	offers := make([]map[string]any, 0, len(steps))
	wellKnowns := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		if ctx.Err() != nil {
			*warnings = append(*warnings, ctx.Err().Error())
			return offers, wellKnowns, ctx.Err()
		}
		deeplink, ok := stepDeeplink(payload, step, "credential", warnings)
		if !ok {
			continue
		}
		res := credoffer.ResolveDeeplink(client, step.CredentialID, deeplink, 5)
		res.StepID = step.StepID
		if res.Status != "ok" || res.CredentialOffer == nil {
			appendCredentialWarning(warnings, step, res)
			continue
		}
		offers = append(offers, map[string]any{
			"step_id":          step.StepID,
			"credential_id":    step.CredentialID,
			"credential_offer": res.CredentialOffer,
		})
		wellKnown, fetch, err := fetchIssuerMetadataWithRetry(
			ctx,
			client,
			res.CredentialOffer,
		)
		if err != nil {
			if isRetryableMetadataFetchError(err) {
				return offers, wellKnowns, fmt.Errorf(
					"fetch issuer metadata for step %s: %w",
					step.StepID,
					err,
				)
			}
			*warnings = append(
				*warnings,
				fmt.Sprintf("failed to fetch issuer metadata for step %s: %v", step.StepID, err),
			)
			continue
		}
		wellKnowns = append(wellKnowns, map[string]any{
			"step_id":       step.StepID,
			"credential_id": step.CredentialID,
			"well_known":    decodeRawJSON(wellKnown),
			"fetch":         fetch,
		})
	}
	return offers, wellKnowns, nil
}

func isRetryableMetadataFetchError(err error) bool {
	var networkErr net.Error
	return errors.As(err, &networkErr)
}

func fetchIssuerMetadataWithRetry(
	ctx context.Context,
	client *http.Client,
	offer json.RawMessage,
) (json.RawMessage, *credoffer.MetadataFetch, error) {
	const maxAttempts = 3

	var wellKnown json.RawMessage
	var fetch *credoffer.MetadataFetch
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return wellKnown, fetch, ctx.Err()
		}
		wellKnown, fetch, err = credoffer.FetchIssuerMetadata(client, offer)
		if err == nil {
			return wellKnown, fetch, nil
		}
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return wellKnown, fetch, ctx.Err()
			case <-time.After(time.Duration(1<<(attempt-1)) * time.Second):
			}
		}
	}
	return wellKnown, fetch, fmt.Errorf(
		"fetch issuer metadata after %d attempts: %w",
		maxAttempts,
		err,
	)
}

func extractPresentationResults(
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
		deeplink, ok := stepDeeplink(payload, step, "verification", warnings)
		if !ok {
			continue
		}
		res := presentation.ResolveDeeplink(client, step.UseCaseID, deeplink, "auto")
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

// stepDeeplink returns the deeplink the pipeline resolved for step, or records
// why it is missing as a warning.
func stepDeeplink(
	payload PipelineEvidenceExtractionInput,
	step discovery.Step,
	kind string,
	warnings *[]string,
) (string, bool) {
	if reason, failed := payload.DeeplinkErrors[step.StepID]; failed {
		*warnings = append(*warnings, fmt.Sprintf(
			"failed to resolve %s deeplink for step %s: %s", kind, step.StepID, reason,
		))
		return "", false
	}
	deeplink := strings.TrimSpace(payload.Deeplinks[step.StepID])
	if deeplink == "" {
		*warnings = append(*warnings, fmt.Sprintf(
			"no %s deeplink was resolved for step %s", kind, step.StepID,
		))
		return "", false
	}
	return deeplink, true
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
			fmt.Sprintf(
				"failed to extract credential evidence for step %s: %s",
				step.StepID,
				res.Error.Error.Message,
			),
		)
		return
	}
	*warnings = append(
		*warnings,
		fmt.Sprintf("failed to extract credential evidence for step %s", step.StepID),
	)
}

func appendPresentationWarning(warnings *[]string, step discovery.Step, res *presentation.Result) {
	if res != nil && res.Error != nil {
		*warnings = append(
			*warnings,
			fmt.Sprintf(
				"failed to extract presentation evidence for step %s: %s",
				step.StepID,
				res.Error.Error.Message,
			),
		)
		return
	}
	*warnings = append(
		*warnings,
		fmt.Sprintf("failed to extract presentation evidence for step %s", step.StepID),
	)
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
