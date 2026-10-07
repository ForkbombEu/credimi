// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/forkbombeu/credimi-conformance-assessment/pkg/conformance"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipelinehistory"
	"github.com/pocketbase/pocketbase/core"
)

const PipelineReportGenerationActivityName = "Generate pipeline conformance report"

type PipelineReportGenerationActivity struct {
	workflowengine.BaseActivity
	app        core.App
	outputKind pipelinehistory.OutputKindFunc
}

// PipelineReportGenerationInput identifies the run whose history holds the definition and
// step outputs the report is built from.
type PipelineReportGenerationInput struct {
	Namespace  string `json:"namespace"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	// PipelineOutputMeta holds the pipeline output keys that are not step outputs, such as
	// warnings, which exist only in the workflow's memory.
	PipelineOutputMeta map[string]any                   `json:"pipeline_output_meta,omitempty"`
	Evidence           PipelineEvidenceExtractionOutput `json:"evidence"`
}

type PipelineReportGenerationOutput struct {
	MarkdownSHA256 string   `json:"markdown_sha256"`
	Filename       string   `json:"filename"`
	Fixture        string   `json:"fixture"`
	Slug           string   `json:"slug"`
	PassedCount    int      `json:"passed_count"`
	Warnings       []string `json:"warnings,omitempty"`
}

func NewPipelineReportGenerationActivity(
	app core.App,
	outputKind pipelinehistory.OutputKindFunc,
) *PipelineReportGenerationActivity {
	return &PipelineReportGenerationActivity{
		BaseActivity: workflowengine.BaseActivity{Name: PipelineReportGenerationActivityName},
		app:          app,
		outputKind:   outputKind,
	}
}

func (a *PipelineReportGenerationActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute builds the conformance report from the definition and step outputs recorded in
// the run's history and stores its markdown on the pipeline result. A storage failure is
// reported as a warning.
func (a *PipelineReportGenerationActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[PipelineReportGenerationInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	if strings.TrimSpace(payload.WorkflowID) == "" || strings.TrimSpace(payload.RunID) == "" {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
				Message: "workflow_id and run_id are required",
			},
		)
	}

	run, err := a.loadRun(ctx, payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
				Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
				Message: fmt.Sprintf("load pipeline history: %v", err),
			},
		)
	}
	pipelineOutputValue := maps.Clone(payload.PipelineOutputMeta)
	if pipelineOutputValue == nil {
		pipelineOutputValue = map[string]any{}
	}
	maps.Copy(pipelineOutputValue, run.DataContext())

	pipelineInput, err := marshalRaw(map[string]any{"workflow_definition": run.Definition})
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
				Message: fmt.Sprintf("marshal pipeline input: %v", err),
			},
		)
	}
	pipelineOutput, err := marshalRaw(pipelineOutputValue)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
				Message: fmt.Sprintf("marshal pipeline output: %v", err),
			},
		)
	}
	evidence, err := marshalRaw(payload.Evidence)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
				Message: fmt.Sprintf("marshal evidence: %v", err),
			},
		)
	}

	fixture := strings.TrimSpace(payload.WorkflowID)

	reportResult, err := conformance.Generate(
		conformance.ReportInput{
			Fixture:        fixture,
			PipelineInput:  pipelineInput,
			PipelineOutput: pipelineOutput,
			Evidence:       evidence,
		},
		conformance.ReportOptions{},
	)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
				Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
				Message: fmt.Sprintf("generate conformance report: %v", err),
			},
		)
	}
	if len(reportResult.Reports) == 0 {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
				Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
				Message: "generate conformance report: no report returned",
			},
		)
	}

	report := reportResult.Reports[0]
	sum := sha256.Sum256([]byte(report.Markdown))
	output := PipelineReportGenerationOutput{
		MarkdownSHA256: hex.EncodeToString(sum[:]),
		Filename:       pipelineresults.SanitizeReportFilename(fixture),
		Fixture:        report.Fixture,
		Slug:           report.Slug,
		PassedCount:    report.PassedCount,
	}
	if strings.TrimSpace(report.Markdown) == "" {
		output.Warnings = append(output.Warnings, "generated conformance report markdown is empty")
		return workflowengine.ActivityResult{Output: output}, nil
	}
	if err := pipelineresults.StoreReport(
		a.app,
		payload.WorkflowID,
		payload.RunID,
		output.Filename,
		report.Markdown,
	); err != nil {
		output.Warnings = append(
			output.Warnings,
			fmt.Sprintf("pipeline report storage failed: %v", err),
		)
	}

	return workflowengine.ActivityResult{Output: output}, nil
}

func (a *PipelineReportGenerationActivity) loadRun(
	ctx context.Context,
	payload PipelineReportGenerationInput,
) (*pipelinehistory.Run, error) {
	c, err := temporalclient.GetTemporalClientWithNamespace(payload.Namespace)
	if err != nil {
		return nil, err
	}
	return pipelinehistory.Load(ctx, c, payload.WorkflowID, payload.RunID, a.outputKind)
}

func marshalRaw(value any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage(`{}`), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}
