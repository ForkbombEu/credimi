// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateStepIDs(t *testing.T) {
	main := func(id string, onError ...string) StepDefinition {
		step := StepDefinition{StepSpec: StepSpec{ID: id, Use: "http-request"}}
		for _, hookID := range onError {
			step.OnError = append(step.OnError, &OnErrorStepDefinition{
				StepSpec: StepSpec{ID: hookID, Use: "http-request"},
			})
		}
		return step
	}
	linked := func(id, linkedTo string) StepDefinition {
		step := main(id)
		step.LinkedTo = linkedTo
		return step
	}

	tests := []struct {
		name    string
		def     *WorkflowDefinition
		wantErr string
	}{
		{
			name: "unique main ids",
			def:  &WorkflowDefinition{Steps: []StepDefinition{main("a"), main("b")}},
		},
		{
			name:    "duplicate main ids",
			def:     &WorkflowDefinition{Steps: []StepDefinition{main("a"), main("b"), main("a")}},
			wantErr: `duplicate step id "a"`,
		},
		{
			name: "empty main ids are ignored",
			def:  &WorkflowDefinition{Steps: []StepDefinition{main(""), main("")}},
		},
		{
			name: "hook ids may repeat among themselves",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				main("a", "notify"),
				main("b", "notify"),
			}},
		},
		{
			name:    "on_error id collides with a main id",
			def:     &WorkflowDefinition{Steps: []StepDefinition{main("a", "b"), main("b")}},
			wantErr: `hook step id "b" collides with a main step id`,
		},
		{
			name: "on_success id collides with a main id",
			def: &WorkflowDefinition{Steps: []StepDefinition{{
				StepSpec: StepSpec{ID: "a"},
				OnSuccess: []*OnSuccessStepDefinition{
					{StepSpec: StepSpec{ID: "a"}},
				},
			}}},
			wantErr: `hook step id "a" collides with a main step id`,
		},
		{
			name: "finally id collides with a main id",
			def: &WorkflowDefinition{
				Steps: []StepDefinition{main("a")},
				Finally: FinallyDefinition{
					OnFailure: []FinallyStepDefinition{{StepSpec: StepSpec{ID: "a"}}},
				},
			},
			wantErr: `hook step id "a" collides with a main step id`,
		},
		{
			name: "valid linked chain",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				main("a"), linked("b", "a"), linked("c", "b"), linked("d", "c"),
			}},
		},
		{
			name: "adjacent blocks split by an unlinked step",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				main("a"), linked("b", "a"), main("c"), linked("d", "c"),
			}},
		},
		{
			name:    "linked_to on the first step",
			def:     &WorkflowDefinition{Steps: []StepDefinition{linked("a", "x")}},
			wantErr: `step "a": linked_to cannot be set on the first step`,
		},
		{
			name: "previous step without id",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				{StepSpec: StepSpec{Use: "debug"}}, linked("b", "a"),
			}},
			wantErr: `step "b": linked_to needs the previous step to have an id`,
		},
		{
			name:    "linked_to names another step",
			def:     &WorkflowDefinition{Steps: []StepDefinition{main("a"), linked("b", "wrong")}},
			wantErr: `step "b": linked_to "wrong" must name the previous step "a"`,
		},
		{
			name: "unlinkable step type",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				main("a"),
				{StepSpec: StepSpec{ID: "b", Use: "child-pipeline"}, LinkedTo: "a"},
			}},
			wantErr: `step "b": child-pipeline steps cannot be linked`,
		},
		{
			name: "linked step with hooks",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				main("a", "notify"), linked("b", "a"),
			}},
			wantErr: `step "a": linked steps cannot have on_error or on_success steps`,
		},
		{
			name: "retry policy on a non-last linked step",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				{StepSpec: StepSpec{
					ID:  "a",
					Use: "http-request",
					ActivityOptions: &ActivityOptionsConfig{
						RetryPolicy: RetryPolicy{MaximumAttempts: 3},
					},
				}},
				linked("b", "a"),
			}},
			wantErr: `step "a": only the last linked step "b" can set activity_options.retry_policy`,
		},
		{
			name: "linked steps with different continue_on_error",
			def: &WorkflowDefinition{Steps: []StepDefinition{
				main("a"),
				{
					StepSpec:        StepSpec{ID: "b", Use: "http-request"},
					LinkedTo:        "a",
					ContinueOnError: true,
				},
			}},
			wantErr: `step "b": linked steps must share the same continue_on_error`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStepIDs(tc.def)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestLinkedStepBlocks(t *testing.T) {
	step := func(linkedTo string) StepDefinition {
		return StepDefinition{LinkedTo: linkedTo}
	}
	tests := []struct {
		name  string
		steps []StepDefinition
		want  [][2]int
	}{
		{name: "none", steps: []StepDefinition{step(""), step("")}},
		{
			name:  "one block",
			steps: []StepDefinition{step(""), step("a"), step("b")},
			want:  [][2]int{{0, 2}},
		},
		{
			name:  "two blocks",
			steps: []StepDefinition{step(""), step("a"), step(""), step(""), step("c")},
			want:  [][2]int{{0, 1}, {3, 4}},
		},
		{
			name:  "trailing single step",
			steps: []StepDefinition{step(""), step("a"), step("")},
			want:  [][2]int{{0, 1}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, LinkedStepBlocks(tc.steps))
		})
	}
}
