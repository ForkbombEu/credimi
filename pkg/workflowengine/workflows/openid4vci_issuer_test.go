// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func Test_OpenID4VCIIssuerWorkflow(t *testing.T) {
	os.Setenv("OPENIDNET_TOKEN", "test_token")

	baseConfig := map[string]any{
		"app_url":   "https://test-app.com",
		"app_name":  "Credimi",
		"app_logo":  "https://logo.png",
		"user_name": "Test User",
		"namespace": "test-namespace",
		"template":  "steps: []",
	}

	testCases := []struct {
		name          string
		payload       OpenID4VCIIssuerWorkflowPayload
		config        map[string]any
		mockActivity  func(env *testsuite.TestWorkflowEnvironment)
		expectErr     bool
		errorContains string
		errorCode     string
		checkResult   func(t *testing.T, result workflowengine.WorkflowResult)
	}{
		{
			name: "succeeds when StepCI returns passing result",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "openid-credential-offer://...",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config: baseConfig,
			mockActivity: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{
								"result": []any{"PASSED"},
								"logs":   "all tests passed",
							},
						},
					}, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				t.Helper()
				require.Equal(t, "Check completed successfully", result.Message)
				require.Equal(t, "all tests passed", result.Log)
			},
		},
		{
			name: "succeeds when result capture is absent (StepCI handles failure internally)",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "openid-credential-offer://...",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config: baseConfig,
			mockActivity: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{},
						},
					}, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				t.Helper()
				require.Equal(t, "Check completed successfully", result.Message)
			},
		},
		{
			name: "returns OpenID4VCIIssuerCheckFailed error when result is FAILED",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "openid-credential-offer://...",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config: baseConfig,
			mockActivity: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{
								"result": []any{"FAILED"},
								"logs":   "assertion failed at step 3",
							},
						},
					}, nil)
			},
			expectErr:     true,
			errorCode:     errorcodes.Codes[errorcodes.OpenID4VCIIssuerCheckFailed].Code,
			errorContains: errorcodes.Codes[errorcodes.OpenID4VCIIssuerCheckFailed].Description,
		},
		{
			name: "returns error when StepCI activity fails",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "openid-credential-offer://...",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config: baseConfig,
			mockActivity: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, errors.New("stepci connection timeout"))
			},
			expectErr:     true,
			errorContains: "stepci connection timeout",
		},
		{
			name: "returns error when StepCI output is missing captures",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "openid-credential-offer://...",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config: baseConfig,
			mockActivity: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{"unexpected": "value"},
					}, nil)
			},
			expectErr:     true,
			errorContains: "StepCI unexpected output",
		},
		{
			name: "returns error when credential_offer is missing",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config:        baseConfig,
			mockActivity:  func(_ *testsuite.TestWorkflowEnvironment) {},
			expectErr:     true,
			errorContains: "CredentialOffer",
		},
		{
			name: "returns error when template config is missing",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "openid-credential-offer://...",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config: map[string]any{
				"app_url":   "https://test-app.com",
				"app_name":  "Credimi",
				"app_logo":  "https://logo.png",
				"user_name": "Test User",
				"namespace": "test-namespace",
				// no "template"
			},
			mockActivity:  func(_ *testsuite.TestWorkflowEnvironment) {},
			expectErr:     true,
			errorContains: "template",
		},
		{
			name: "returns error when app_url config is empty",
			payload: OpenID4VCIIssuerWorkflowPayload{
				CredentialOffer: "openid-credential-offer://...",
				UserMail:        "tester@example.org",
				TestName:        "happy-flow",
			},
			config: map[string]any{
				"app_url":   "", // empty triggers the guard in ExecuteWorkflow
				"app_name":  "Credimi",
				"app_logo":  "https://logo.png",
				"user_name": "Test User",
				"namespace": "test-namespace",
				"template":  "steps: []",
			},
			mockActivity:  func(_ *testsuite.TestWorkflowEnvironment) {},
			expectErr:     true,
			errorContains: "app_url",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			w := NewOpenID4VCIIssuerWorkflow()
			tc.mockActivity(env)

			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Payload:         tc.payload,
				Config:          tc.config,
				ActivityOptions: &DefaultActivityOptions,
			})

			require.True(t, env.IsWorkflowCompleted())

			if tc.expectErr {
				err := env.GetWorkflowError()
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
				if tc.errorCode != "" {
					var appErr *temporal.ApplicationError
					if errors.As(err, &appErr) {
						require.Equal(t, tc.errorCode, appErr.Type())
					} else {
						require.Contains(t, err.Error(), tc.errorCode)
					}
				}
			} else {
				require.NoError(t, env.GetWorkflowError())
				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result))
				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}
		})
	}
}

func TestOpenID4VCIIssuerWorkflowStart(t *testing.T) {
	origStart := openID4VCIIssuerStartWorkflowWithOptions
	t.Cleanup(func() {
		openID4VCIIssuerStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string

	openID4VCIIssuerStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		_ workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}

	w := NewOpenID4VCIIssuerWorkflow()
	input := workflowengine.WorkflowInput{Config: map[string]any{"namespace": "ns-issuer"}}
	result, err := w.Start(input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-issuer", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, OpenID4VCIIssuerTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "OpenID4VCIIssuerCheckWorkflow"))
}
