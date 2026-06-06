// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestPipelineReportGenerationActivityExecute(t *testing.T) {
	act := NewPipelineReportGenerationActivity()
	res, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{
			Payload: PipelineReportGenerationInput{
				WorkflowDefinition: &pipelineinternal.WorkflowDefinition{
					Name: "report-pipeline",
					Steps: []pipelineinternal.StepDefinition{
						{
							StepSpec: pipelineinternal.StepSpec{
								ID:  "credential-step",
								Use: "credential-offer",
							},
						},
					},
				},
				PipelineOutput: map[string]any{
					"workflow_id":     "workflow-1",
					"run_id":          "run-1",
					"credential-step": map[string]any{"outputs": map[string]any{"status": "ok"}},
				},
				Evidence: PipelineEvidenceExtractionOutput{
					CredentialOffers: []map[string]any{
						{
							"step_id":       "credential-step",
							"credential_id": "tenant/credential",
							"credential_offer": map[string]any{
								"credential_issuer":            "https://issuer.example",
								"credential_configuration_ids": []string{"pid_sd_jwt"},
								"grants": map[string]any{
									"urn:ietf:params:oauth:grant-type:pre-authorized_code": map[string]any{},
								},
							},
						},
					},
					CredentialWellKnowns: []map[string]any{
						{
							"step_id":       "credential-step",
							"credential_id": "tenant/credential",
							"well_known": map[string]any{
								"credential_configurations_supported": map[string]any{
									"pid_sd_jwt": map[string]any{
										"format": "vc+sd-jwt",
										"proof_types_supported": map[string]any{
											"jwt": map[string]any{
												"proof_signing_alg_values_supported": []string{
													"ES256",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				WorkflowID: "workflow-1",
				RunID:      "run-1",
			},
		},
	)
	require.NoError(t, err)

	out, ok := res.Output.(PipelineReportGenerationOutput)
	require.True(t, ok)
	require.Equal(t, "workflow-1.md", out.Filename)
	require.Equal(t, "workflow-1", out.Fixture)
	require.NotEmpty(t, out.Markdown)
	require.Contains(t, out.Markdown, "Credimi Conformance Assessment")
}

func TestSanitizeReportFilename(t *testing.T) {
	require.Equal(t, "workflow-1.md", sanitizeReportFilename(" workflow/1 ")+".md")
	require.Equal(t, "pipeline-report", sanitizeReportFilename("///"))
}

func TestPipelineReportGenerationActivityValidation(t *testing.T) {
	act := NewPipelineReportGenerationActivity()
	_, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{Payload: PipelineReportGenerationInput{}},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "workflow_definition is required")

	raw, err := marshalRaw(nil)
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(raw))
}

func TestPipelineReportGenerationActivityMarshalOutputError(t *testing.T) {
	act := NewPipelineReportGenerationActivity()
	_, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{
			Payload: PipelineReportGenerationInput{
				WorkflowDefinition: &pipelineinternal.WorkflowDefinition{Name: "bad-output"},
				PipelineOutput:     map[string]any{"bad": make(chan int)},
			},
		},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported type: chan int")
}
