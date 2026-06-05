// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	internalpipeline "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/stretchr/testify/require"
)

func TestBuildPipelineFailureSummary(t *testing.T) {
	summary := buildPipelineFailureSummary([]pipelineStepFailure{
		{
			StepID:  "get-credential",
			Message: "CRE302: workflow engine Get a credential offer: stepci run failed",
		},
		{
			StepID:  "mobile",
			Message: "CRE228: Failed to resolve pipeline inputs: error decoding payload for step mobile",
		},
	})

	require.Equal(
		t,
		"Pipeline failed: 2 steps failed (get-credential, mobile)",
		summary,
	)
}

func TestBuildPipelineFailureSummaryLimitsStepIDs(t *testing.T) {
	summary := buildPipelineFailureSummary([]pipelineStepFailure{
		{StepID: "step-1", Message: "err-1"},
		{StepID: "step-2", Message: "err-2"},
		{StepID: "step-3", Message: "err-3"},
		{StepID: "step-4", Message: "err-4"},
	})

	require.Equal(
		t,
		"Pipeline failed: 4 steps failed (step-1, step-2, step-3, +1 more)",
		summary,
	)
}

func TestDecodePayloadReportsInvalidParameterValueType(t *testing.T) {
	step := &internalpipeline.StepDefinition{
		StepSpec: internalpipeline.StepSpec{
			ID:  "mobile-step",
			Use: "mobile-automation",
			With: internalpipeline.StepInputs{
				Payload: map[string]any{
					"runner_id": "ios-runner",
					"parameters": map[string]any{
						"offer": map[string]any{"url": "https://example.test"},
					},
				},
			},
		},
	}

	_, err := DecodePayload(step)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode payload for mobile-step")
	require.Contains(t, err.Error(), "invalid payload for mobile-automation")
	require.Contains(t, err.Error(), "with.payload.parameters must contain only string values")
	require.Contains(t, err.Error(), `parameter "offer" resolved to object`)
}
