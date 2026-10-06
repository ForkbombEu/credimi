// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestCovWEStartCheckWorkflowRejectsInvalidSetups(t *testing.T) {
	t.Setenv("OPENIDNET_TOKEN", "test_token")
	invalidPayload := errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code
	missingConfig := errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code
	stepCIOutput := errorcodes.Codes[errorcodes.UnexpectedStepCIOutput].Code

	tests := []struct {
		name     string
		payload  any
		appURL   string
		captures map[string]any
		childErr error
		wantCode string
		wantText string
	}{
		{
			name:     "payload without check id",
			payload:  StartCheckWorkflowPayload{Suite: EWCSuite},
			wantCode: invalidPayload,
		},
		{
			name:     "empty app url",
			payload:  StartCheckWorkflowPayload{Suite: EWCSuite, CheckID: "c"},
			appURL:   "-",
			wantCode: missingConfig,
			wantText: "app_url",
		},
		{
			name:     "unsupported suite",
			payload:  StartCheckWorkflowPayload{Suite: "unknown-suite", CheckID: "c"},
			wantCode: invalidPayload,
			wantText: "unsupported suite: unknown-suite",
		},
		{
			name:     "openid suite without test name",
			payload:  StartCheckWorkflowPayload{Suite: OpenIDConformanceSuite, CheckID: "c"},
			wantCode: invalidPayload,
			wantText: "test is required",
		},
		{
			name: "openid suite without rid capture",
			payload: StartCheckWorkflowPayload{
				Suite: OpenIDConformanceSuite, CheckID: "c", TestName: "t",
			},
			captures: map[string]any{"deeplink": "openid4vp://x"},
			wantCode: stepCIOutput,
			wantText: "rid",
		},
		{
			name: "openid wallet suite without deeplink capture",
			payload: StartCheckWorkflowPayload{
				Suite: OpenIDConformanceSuite, CheckID: "c", TestName: "t",
			},
			captures: map[string]any{"rid": "rid-1"},
			wantCode: stepCIOutput,
			wantText: "deeplink",
		},
		{
			name:     "ewc suite without standard",
			payload:  StartCheckWorkflowPayload{Suite: EWCSuite, CheckID: "c"},
			captures: map[string]any{"deeplink": "x", "session_id": "s"},
			wantCode: missingConfig,
			wantText: "standard",
		},
		{
			name: "ewc suite with unsupported standard",
			payload: StartCheckWorkflowPayload{
				Suite: EWCSuite, CheckID: "c", Standard: OpenID4VCIIssuerStandard,
			},
			captures: map[string]any{"deeplink": "x", "session_id": "s"},
			wantCode: missingConfig,
			wantText: "unsupported standard",
		},
		{
			name: "wallet standard without deeplink capture",
			payload: StartCheckWorkflowPayload{
				Suite: EWCSuite, CheckID: "c", Standard: OpenID4VPWalletStandard,
			},
			captures: map[string]any{"session_id": "s"},
			wantCode: stepCIOutput,
			wantText: "deeplink",
		},
		{
			name: "webuild verifier without session id",
			payload: StartCheckWorkflowPayload{
				Suite: WebuildSuite, CheckID: "c", Standard: OpenID4VPVerifierStandard,
			},
			captures: map[string]any{"result": "started"},
			wantCode: stepCIOutput,
			wantText: "session_id",
		},
		{
			name: "webuild issuer status child fails",
			payload: StartCheckWorkflowPayload{
				Suite: WebuildSuite, CheckID: "c", Standard: OpenID4VCIIssuerStandard,
				Parameters: map[string]any{"session_id": "s"},
			},
			captures: map[string]any{"result": "started"},
			childErr: covWENonRetryable("status child failed"),
			wantCode: errorcodes.Codes[errorcodes.ChildWorkflowExecutionError].Code,
			wantText: "status child failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			stepCI := activities.NewStepCIWorkflowActivity()
			env.RegisterActivityWithOptions(
				stepCI.Execute,
				activity.RegisterOptions{Name: stepCI.Name()},
			)
			childWebuild := NewWebuildStatusWorkflow()
			env.RegisterWorkflowWithOptions(
				childWebuild.Workflow,
				workflow.RegisterOptions{Name: childWebuild.Name()},
			)
			w := NewStartCheckWorkflow()
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})
			if tc.captures != nil {
				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{"captures": tc.captures},
					}, nil)
			}
			if tc.childErr != nil {
				env.OnWorkflow(childWebuild.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.WorkflowResult{}, tc.childErr)
			}
			appURL := "https://credimi.test"
			if tc.appURL == "-" {
				appURL = ""
			}

			env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
				Payload: tc.payload,
				Config: map[string]any{
					"app_url":   appURL,
					"template":  "tpl",
					"namespace": "ns",
				},
				ActivityOptions: &DefaultActivityOptions,
			})

			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantCode)
			assert.Contains(t, err.Error(), tc.wantText)
		})
	}
}

func TestCovWEConformanceSuiteHasLogs(t *testing.T) {
	tests := []struct {
		suite string
		want  bool
	}{
		{suite: EWCSuite, want: true},
		{suite: WebuildSuite, want: true},
		{suite: OpenIDConformanceSuite, want: true},
		{suite: "eudiw", want: false},
		{suite: "", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.suite, func(t *testing.T) {
			assert.Equal(t, tc.want, ConformanceSuiteHasLogs(tc.suite))
		})
	}
}

func TestCovWEConformanceCheckStandardPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		payload StartCheckWorkflowPayload
		config  map[string]any
		want    string
	}{
		{
			name:    "payload standard wins over memo",
			payload: StartCheckWorkflowPayload{Standard: OpenID4VPWalletStandard},
			config:  map[string]any{"memo": map[string]any{"standard": OpenID4VCIIssuerStandard}},
			want:    OpenID4VPWalletStandard,
		},
		{
			name:   "memo standard is the fallback",
			config: map[string]any{"memo": map[string]any{"standard": OpenID4VCIIssuerStandard}},
			want:   OpenID4VCIIssuerStandard,
		},
		{name: "no memo yields empty standard", config: map[string]any{}, want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, conformanceCheckStandard(tc.payload, tc.config))
		})
	}
}
