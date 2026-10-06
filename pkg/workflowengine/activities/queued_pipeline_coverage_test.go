// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

type covWEFailingStarter struct{ err error }

func (s covWEFailingStarter) ExecuteWorkflow(
	context.Context,
	client.StartWorkflowOptions,
	interface{},
	...interface{},
) (client.WorkflowRun, error) {
	return nil, s.err
}

func TestCovWEStartQueuedPipelineActivityFailures(t *testing.T) {
	validYAML := "name: test\nsteps: []\n"
	tests := []struct {
		name       string
		yaml       string
		factoryErr error
		startErr   error
		wantCode   string
		wantText   string
	}{
		{
			name:     "malformed yaml",
			yaml:     "name: [",
			wantCode: errorcodes.PipelineParsingError,
			wantText: "parse workflow definition",
		},
		{
			name: "step output reference without step ids",
			yaml: `name: test
steps:
  - use: credential-offer
  - use: mobile-automation
    with:
      payload:
        deeplink: ${{ issuer.outputs.deeplink }}
`,
			wantCode: errorcodes.PipelineParsingError,
			wantText: "steps are missing id",
		},
		{
			name:       "temporal client unavailable",
			yaml:       validYAML,
			factoryErr: errors.New("temporal dial failed"),
			wantCode:   errorcodes.PipelineExecutionError,
			wantText:   "temporal dial failed",
		},
		{
			name:     "workflow start rejected",
			yaml:     validYAML,
			startErr: errors.New("workflow already started"),
			wantCode: errorcodes.PipelineExecutionError,
			wantText: "workflow already started",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			act := NewStartQueuedPipelineActivity(newCredimiTestApp(t))
			factoryCalls := 0
			act.temporalClientFactory = func(string) (temporalWorkflowStarter, error) {
				factoryCalls++
				if tc.factoryErr != nil {
					return nil, tc.factoryErr
				}
				return covWEFailingStarter{err: tc.startErr}, nil
			}

			_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
				Payload: StartQueuedPipelineActivityInput{
					TicketID:           "ticket-1",
					OwnerNamespace:     testOrgANamespace,
					PipelineIdentifier: testOrgANamespace + "/p",
					YAML:               tc.yaml,
				},
			})

			requireActivityError(t, err, tc.wantCode, false)
			assert.Contains(t, err.Error(), tc.wantText)
			if tc.wantCode == errorcodes.PipelineParsingError {
				assert.Zero(t, factoryCalls, "invalid pipelines must not reach Temporal")
			}
		})
	}
}

func TestCovWEWorkflowDefinitionUsesStepOutputReferences(t *testing.T) {
	ref := "${{ step-a.outputs.value }}"
	spec := func(payload map[string]any) pipeline.StepSpec {
		return pipeline.StepSpec{With: pipeline.StepInputs{Payload: payload}}
	}
	tests := []struct {
		name string
		def  *pipeline.WorkflowDefinition
		want bool
	}{
		{name: "nil definition", def: nil, want: false},
		{
			name: "reference inside a list in on_error step",
			def: &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{{
				OnError: []*pipeline.OnErrorStepDefinition{
					nil,
					{StepSpec: spec(map[string]any{"items": []any{"static", ref}})},
				},
			}}},
			want: true,
		},
		{
			name: "reference in on_success nested map",
			def: &pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{{
				OnSuccess: []*pipeline.OnSuccessStepDefinition{
					{StepSpec: spec(map[string]any{"nested": map[string]any{"v": ref}})},
				},
			}}},
			want: true,
		},
		{
			name: "reference in finally step metadata",
			def: &pipeline.WorkflowDefinition{Finally: pipeline.FinallyDefinition{
				OnFailure: []pipeline.FinallyStepDefinition{{
					StepSpec: pipeline.StepSpec{Metadata: map[string]any{"note": ref}},
				}},
			}},
			want: true,
		},
		{
			name: "non-output expressions and other types are ignored",
			def: &pipeline.WorkflowDefinition{
				Steps: []pipeline.StepDefinition{{
					StepSpec: spec(map[string]any{
						"env":    "${{ config.namespace }}",
						"count":  3,
						"list":   []any{1, "plain"},
						"nested": map[string]any{"x": false},
					}),
					OnError:   []*pipeline.OnErrorStepDefinition{{StepSpec: spec(nil)}},
					OnSuccess: []*pipeline.OnSuccessStepDefinition{nil},
				}},
				Finally: pipeline.FinallyDefinition{
					Always: []pipeline.FinallyStepDefinition{
						{StepSpec: spec(map[string]any{"a": "b"})},
					},
				},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, workflowDefinitionUsesStepOutputReferences(tc.def))
		})
	}
}

func TestCovWEWorkflowDefinitionMissingStepIDs(t *testing.T) {
	assert.Nil(t, workflowDefinitionMissingStepIDs(nil))

	def := &pipeline.WorkflowDefinition{
		Steps: []pipeline.StepDefinition{
			{
				StepSpec: pipeline.StepSpec{ID: "first"},
				OnError: []*pipeline.OnErrorStepDefinition{
					nil,
					{StepSpec: pipeline.StepSpec{ID: " "}},
					{StepSpec: pipeline.StepSpec{ID: "recover"}},
				},
				OnSuccess: []*pipeline.OnSuccessStepDefinition{
					{StepSpec: pipeline.StepSpec{}},
				},
			},
			{StepSpec: pipeline.StepSpec{}},
		},
		Finally: pipeline.FinallyDefinition{
			Always:    []pipeline.FinallyStepDefinition{{StepSpec: pipeline.StepSpec{}}},
			OnSuccess: []pipeline.FinallyStepDefinition{{StepSpec: pipeline.StepSpec{ID: "ok"}}},
			OnFailure: []pipeline.FinallyStepDefinition{
				{StepSpec: pipeline.StepSpec{ID: "fail"}},
				{StepSpec: pipeline.StepSpec{}},
			},
		},
	}

	assert.Equal(t, []string{
		"steps[0].on_error[1]",
		"steps[0].on_success[0]",
		"steps[1]",
		"finally.always[0]",
		"finally.on_failure[1]",
	}, workflowDefinitionMissingStepIDs(def))

	def.Finally.OnSuccess[0].ID = ""
	assert.Contains(t, workflowDefinitionMissingStepIDs(def), "finally.on_success[0]")
}

func TestCovWEPipelineRunTypeFromMemo(t *testing.T) {
	tests := []struct {
		name string
		memo map[string]any
		want string
	}{
		{name: "nil memo defaults to manual", memo: nil, want: pipeline.RunTypeManual},
		{
			name: "valid run type is kept",
			memo: map[string]any{pipeline.RunTypeMemoKey: pipeline.RunTypeScheduled},
			want: pipeline.RunTypeScheduled,
		},
		{
			name: "unknown run type defaults to manual",
			memo: map[string]any{pipeline.RunTypeMemoKey: "cosmic"},
			want: pipeline.RunTypeManual,
		},
		{
			name: "non string run type defaults to manual",
			memo: map[string]any{pipeline.RunTypeMemoKey: 42},
			want: pipeline.RunTypeManual,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, pipelineRunTypeFromMemo(tc.memo))
		})
	}
}
