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
