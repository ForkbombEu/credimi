// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi-conformance-assessment/pkg/conformance"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

const PipelineReportGenerationActivityName = "Generate pipeline conformance report"

type PipelineReportGenerationActivity struct {
	workflowengine.BaseActivity
}

type PipelineReportGenerationInput struct {
	WorkflowDefinition *pipelineinternal.WorkflowDefinition `json:"workflow_definition"`
	PipelineOutput     map[string]any                       `json:"pipeline_output"`
	Evidence           PipelineEvidenceExtractionOutput     `json:"evidence"`
	WorkflowID         string                               `json:"workflow_id"`
	RunID              string                               `json:"run_id"`
}

type PipelineReportGenerationOutput struct {
	Markdown    string   `json:"markdown"`
	Filename    string   `json:"filename"`
	Fixture     string   `json:"fixture"`
	Slug        string   `json:"slug"`
	PassedCount int      `json:"passed_count"`
	Warnings    []string `json:"warnings,omitempty"`
}

func NewPipelineReportGenerationActivity() *PipelineReportGenerationActivity {
	return &PipelineReportGenerationActivity{
		BaseActivity: workflowengine.BaseActivity{Name: PipelineReportGenerationActivityName},
	}
}

func (a *PipelineReportGenerationActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *PipelineReportGenerationActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[PipelineReportGenerationInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	if payload.WorkflowDefinition == nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			"workflow_definition is required",
		)
	}

	pipelineInput, err := marshalRaw(
		map[string]any{"workflow_definition": payload.WorkflowDefinition},
	)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			fmt.Sprintf("marshal pipeline input: %v", err),
		)
	}
	pipelineOutput, err := marshalRaw(payload.PipelineOutput)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			fmt.Sprintf("marshal pipeline output: %v", err),
		)
	}
	evidence, err := marshalRaw(payload.Evidence)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			fmt.Sprintf("marshal evidence: %v", err),
		)
	}

	fixture := strings.TrimSpace(payload.WorkflowID)
	if fixture == "" {
		fixture = strings.TrimSpace(payload.WorkflowDefinition.Name)
	}
	if fixture == "" {
		fixture = "pipeline-report"
	}

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
			errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			fmt.Sprintf("generate conformance report: %v", err),
		)
	}
	if len(reportResult.Reports) == 0 {
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			"generate conformance report: no report returned",
		)
	}

	report := reportResult.Reports[0]
	output := PipelineReportGenerationOutput{
		Markdown:    report.Markdown,
		Filename:    sanitizeReportFilename(fixture) + ".md",
		Fixture:     report.Fixture,
		Slug:        report.Slug,
		PassedCount: report.PassedCount,
	}
	if strings.TrimSpace(output.Markdown) == "" {
		output.Warnings = append(output.Warnings, "generated conformance report markdown is empty")
	}

	return workflowengine.ActivityResult{Output: output}, nil
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

var unsafeReportFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeReportFilename(name string) string {
	name = strings.TrimSpace(name)
	name = unsafeReportFilenameChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, ".-_")
	if name == "" {
		return "pipeline-report"
	}
	return name
}
