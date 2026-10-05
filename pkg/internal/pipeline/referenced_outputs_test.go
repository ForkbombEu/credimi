// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferencedStepOutputs(t *testing.T) {
	withRefs := func(texts ...string) *WorkflowDefinition {
		def := &WorkflowDefinition{
			Steps: []StepDefinition{{StepSpec: StepSpec{ID: "child", Use: "child-pipeline"}}},
		}
		for _, text := range texts {
			def.Steps = append(def.Steps, StepDefinition{StepSpec: StepSpec{
				ID:   "consumer",
				Use:  "http-request",
				With: StepInputs{Payload: map[string]any{"body": text}},
			}})
		}
		return def
	}

	tests := []struct {
		name string
		def  *WorkflowDefinition
		want []string
	}{
		{
			name: "nested output key",
			def:  withRefs("${{ child.outputs.inner.outputs.v }}"),
			want: []string{"inner"},
		},
		{
			name: "keys are unique and sorted, indexes and functions stripped",
			def: withRefs(
				"${{ child.outputs.b[0].outputs | optional }}",
				"x ${{ child.outputs.a.outputs }} y ${{ child.outputs.b }}",
			),
			want: []string{"a", "b"},
		},
		{
			name: "whole outputs",
			def:  withRefs("${{ child.outputs }}"),
			want: []string{AllStepOutputs},
		},
		{
			name: "whole step entry",
			def:  withRefs("${{ child | optional }}"),
			want: []string{AllStepOutputs},
		},
		{
			name: "aggregated pipeline output",
			def:  withRefs("${{ pipeline_output.outputs }}"),
			want: []string{AllStepOutputs},
		},
		{
			name: "other steps only",
			def:  withRefs("${{ children.outputs.inner }}", "${{ inputs.child }}"),
			want: nil,
		},
		{
			name: "no references",
			def:  withRefs(),
			want: nil,
		},
		{
			name: "reference in a hook",
			def: &WorkflowDefinition{Steps: []StepDefinition{{
				StepSpec: StepSpec{ID: "child", Use: "child-pipeline"},
				OnSuccess: []*OnSuccessStepDefinition{{StepSpec: StepSpec{
					ID:   "notify",
					With: StepInputs{Payload: map[string]any{"text": "${{ child.outputs.x }}"}},
				}}},
			}}},
			want: []string{"x"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ReferencedStepOutputs(tc.def, "child"))
		})
	}
}

func TestRequiredStepIDs(t *testing.T) {
	tests := []struct {
		name   string
		inputs StepInputs
		want   []string
	}{
		{
			name: "full and embedded references in payload and config",
			inputs: StepInputs{
				Config: map[string]any{"token": "${{ login.outputs.token }}"},
				Payload: map[string]any{
					"deeplink": "${{ offer.outputs }}",
					"nested":   []any{map[string]any{"url": "x ${{ offer.outputs.url | upper }}"}},
				},
			},
			want: []string{"login", "offer"},
		},
		{
			name: "optional references do not count",
			inputs: StepInputs{Payload: map[string]any{
				"a": "${{ offer.outputs | optional }}",
				"b": "${{ offer.outputs.x | optional | upper }}",
			}},
			want: nil,
		},
		{
			name: "pipeline_output references count for the step",
			inputs: StepInputs{Payload: map[string]any{
				"a": "${{ pipeline_output.offer.outputs.x }}",
				"b": "${{ items[0].outputs }}",
			}},
			want: []string{"items", "offer"},
		},
		{
			name:   "no references",
			inputs: StepInputs{Payload: map[string]any{"a": "plain"}},
			want:   nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, RequiredStepIDs(tc.inputs))
		})
	}
}
