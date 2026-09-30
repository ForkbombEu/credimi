// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestStepCIConfigureAndRender(t *testing.T) {
	activity := NewStepCIWorkflowActivity()

	input := &workflowengine.ActivityInput{
		Config: map[string]string{
			"template": "Hello [[ .name ]]",
		},
		Payload: StepCIWorkflowActivityPayload{
			Data: map[string]any{"name": "Ada"},
		},
	}

	require.NoError(t, activity.Configure(input))
	payload, ok := input.Payload.(StepCIWorkflowActivityPayload)
	require.True(t, ok)
	require.Equal(t, "Hello Ada", payload.Yaml)
}

func TestStepCIConfigureMissingTemplate(t *testing.T) {
	activity := NewStepCIWorkflowActivity()
	input := &workflowengine.ActivityInput{
		Config: map[string]string{},
		Payload: StepCIWorkflowActivityPayload{
			Data: map[string]any{"name": "Ada"},
		},
	}
	err := activity.Configure(input)
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.True(t, temporal.IsApplicationError(err))
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code, appErr.Type())
}

func TestStepCIExecuteOutputs(t *testing.T) {
	t.Run("success JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeRunner(t, tmpDir, `#!/bin/sh
echo '{"passed":true}'`)
		t.Setenv("BIN", tmpDir)

		activity := NewStepCIWorkflowActivity()
		result, err := activity.Execute(
			context.Background(),
			workflowengine.ActivityInput{
				Payload: StepCIWorkflowActivityPayload{Yaml: "test"},
			},
		)
		require.NoError(t, err)
		out, ok := result.Output.(StepCICliReturns)
		require.True(t, ok)
		require.True(t, out.Passed)
	})

	t.Run("failed JSON maps to StepCIRunFailed", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeRunner(t, tmpDir, `#!/bin/sh
echo '{"passed":false,"messages":["Workflow failed"],"captures":{"secret":"value"},"tests":[{"id":"suite","passed":false,"steps":[{"testId":"suite","name":"bad step","passed":false,"errored":true,"errorMessage":"boom","skipped":false}]}]}'`)
		t.Setenv("BIN", tmpDir)

		activity := NewStepCIWorkflowActivity()
		_, err := activity.Execute(
			context.Background(),
			workflowengine.ActivityInput{
				Payload: StepCIWorkflowActivityPayload{Yaml: "test"},
			},
		)
		require.Error(t, err)
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, errorcodes.Codes[errorcodes.StepCIRunFailed].Code, appErr.Type())
		var failure workflowengine.ActivityError
		require.NoError(t, appErr.Details(&failure))
		require.Equal(t, "StepCI checks failed", failure.Summary)
		require.Equal(t, "Workflow failed\n\nStep errors:\n  - bad step: boom", failure.Message)
		require.Equal(t, "StepCI checks failed: "+failure.Message, appErr.Message())
		require.Equal(t, "test_failure", failure.Category)
		require.Contains(t, failure.Details, "result")
		summary, ok := failure.Details["summary"].(StepCIFailureSummary)
		require.True(t, ok)
		require.Equal(t, []string{"Workflow failed"}, summary.Messages)
		require.Len(t, summary.FailedSteps, 1)
		require.Equal(t, "bad step", summary.FailedSteps[0].Name)
		result, ok := failure.Details["result"].(StepCICliReturns)
		require.True(t, ok)
		require.Equal(t, "value", result.Captures["secret"])
	})

	t.Run("large failed JSON keeps summary and omits full result", func(t *testing.T) {
		output := StepCICliReturns{
			Passed:   false,
			Messages: []string{"Workflow failed"},
			Captures: map[string]any{
				"large": strings.Repeat("x", maxStepCIResultErrorBytes),
			},
			Tests: []TestResult{
				{
					ID:     "suite",
					Passed: false,
					Steps: []StepResult{
						{
							TestID:       "suite",
							Name:         ptr("bad step"),
							Passed:       false,
							Errored:      true,
							ErrorMessage: ptr("boom"),
						},
					},
				},
			},
		}

		details := stepCIFailureDetails(output)

		require.NotContains(t, details, "result")
		require.Equal(t, true, details["result_omitted"])
		require.Contains(t, details, "result_size_bytes")
		summary, ok := details["summary"].(StepCIFailureSummary)
		require.True(t, ok)
		require.Len(t, summary.FailedSteps, 1)
	})

	t.Run("command failure includes command details", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeRunner(t, tmpDir, `#!/bin/sh
echo 'network unavailable' >&2
exit 1`)
		t.Setenv("BIN", tmpDir)

		activity := NewStepCIWorkflowActivity()
		_, err := activity.Execute(
			context.Background(),
			workflowengine.ActivityInput{
				Payload: StepCIWorkflowActivityPayload{Yaml: "test"},
			},
		)
		require.Error(t, err)
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, errorcodes.Codes[errorcodes.CommandExecutionFailed].Code, appErr.Type())
		var failure workflowengine.ActivityError
		require.NoError(t, appErr.Details(&failure))
		require.Equal(t, "StepCI command failed", failure.Summary)
		require.Equal(t, "external_command", failure.Category)
		require.Contains(t, failure.Details["stderr"], "network unavailable")
	})

	t.Run("invalid JSON returns raw output", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeRunner(t, tmpDir, `#!/bin/sh
echo 'not-json'`)
		t.Setenv("BIN", tmpDir)

		activity := NewStepCIWorkflowActivity()
		result, err := activity.Execute(
			context.Background(),
			workflowengine.ActivityInput{
				Payload: StepCIWorkflowActivityPayload{Yaml: "test"},
			},
		)
		require.NoError(t, err)
		require.Equal(t, "not-json\n", result.Output)
	})
}

func TestFormatStepCIFailureReport(t *testing.T) {
	checksReport := []string{
		"Workflow failed. Details:",
		"Step Failed: get offer",
		"  URL: https://issuer.example/offer",
		"  Method: GET",
		"  Failed Checks:\n    - status:\n        Expected: 200\n        Got:      404",
		"",
		"Step Failed: next",
		"",
	}

	// Runner JSON shape observed from stepci-captured-runner: the request is resolved
	// (secrets substituted) and headers keep the order they were sent in.
	requestStdout := `{"passed":false,
"messages":["Workflow failed. Details:","Step Failed: token","  URL: https://issuer.example/token","  Method: POST","  Failed Checks:\n    - status:\n        Expected: 200\n        Got:      401","","Step Failed: offer",""],
"tests":[{"id":"t1","steps":[
 {"testId":"t1","name":"ok","passed":true,"request":{"protocol":"HTTP/1.1","url":"https://issuer.example/ok","method":"GET"}},
 {"testId":"t1","name":"token","passed":false,"request":{"protocol":"HTTP/1.1","url":"https://issuer.example/token","method":"POST",
  "headers":{"Authorization":"Bearer s3cr3t-token","X-Trace":["a","b"],"Content-Type":"application/json"},
  "body":"{\"client_secret\":\"s3cr3t-token\",\"pin\":\"123\"}"}},
 {"testId":"t1","name":"offer","passed":false,"skipped":true}
]}]}`

	tests := []struct {
		name    string
		output  StepCICliReturns
		stdout  string
		secrets map[string]any
		want    string
	}{
		{
			name:    "failed step shows its request like StepCI, with secrets masked",
			stdout:  requestStdout,
			secrets: map[string]any{"token": "s3cr3t-token", "pin": "123", "n": 42},
			want: "Workflow failed. Details:\nStep Failed: token\n" +
				"  URL: https://issuer.example/token\n  Method: POST\n" +
				"  Request:\n" +
				"    POST https://issuer.example/token HTTP/1.1\n" +
				"    Authorization: Bearer ***\n" +
				"    X-Trace: [\"a\",\"b\"]\n" +
				"    Content-Type: application/json\n" +
				"\n" +
				"    {\"client_secret\":\"***\",\"pin\":\"123\"}\n" +
				"  Failed Checks:\n    - status:\n        Expected: 200\n        Got:      401\n\n" +
				"Step Failed: offer",
		},
		{
			name: "request without a matching message block is appended",
			stdout: `{"passed":false,"messages":["Workflow failed. Details:"],
"tests":[{"id":"t1","steps":[{"testId":"t1","name":"token","passed":false,
"request":{"protocol":"HTTP/1.1","url":"https://issuer.example/token","method":"GET"}}]}]}`,
			want: "Workflow failed. Details:\n\n" +
				"Request of token:\n    GET https://issuer.example/token HTTP/1.1",
		},
		{
			name:   "failed checks keep the StepCI report",
			output: StepCICliReturns{Messages: checksReport},
			want: "Workflow failed. Details:\nStep Failed: get offer\n" +
				"  URL: https://issuer.example/offer\n  Method: GET\n" +
				"  Failed Checks:\n    - status:\n        Expected: 200\n        Got:      404\n\n" +
				"Step Failed: next",
		},
		{
			name: "errored step appends its error, skipped steps do not",
			output: StepCICliReturns{
				Messages: []string{"Workflow failed. Details:", "Step Failed: bad host", ""},
				Tests: []TestResult{{
					ID: "t1",
					Steps: []StepResult{
						{
							TestID:  "t1",
							Name:    ptr("bad host"),
							Errored: true,
							ErrorMessage: ptr(
								"Unable to connect. Is the computer able to access the url?",
							),
						},
						{
							TestID:       "t1",
							Name:         ptr("after"),
							Skipped:      true,
							ErrorMessage: ptr("Step was skipped because previous one failed"),
						},
						{TestID: "t1", Errored: true, ErrorMessage: ptr("unnamed failure")},
					},
				}},
			},
			want: "Workflow failed. Details:\nStep Failed: bad host\n\n" +
				"Step errors:\n" +
				"  - bad host: Unable to connect. Is the computer able to access the url?\n" +
				"  - t1: unnamed failure",
		},
		{
			name: "runner errors without messages",
			output: StepCICliReturns{
				Errors: []CliError{
					{Message: "unexpected end of the stream within a flow collection (2:1)"},
				},
			},
			want: "Errors:\n  - unexpected end of the stream within a flow collection (2:1)",
		},
		{
			name: "no usable details falls back to generic message",
			output: StepCICliReturns{
				Messages: []string{"", " "},
				Errors:   []CliError{{Message: " "}},
			},
			want: "One or more StepCI assertions failed.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.output
			var requests stepCIRequests
			if tc.stdout != "" {
				require.NoError(t, json.Unmarshal([]byte(tc.stdout), &output))
				requests = parseStepCIRequests([]byte(tc.stdout))
			}
			require.Equal(t, tc.want, formatStepCIFailureReport(output, requests, tc.secrets))
		})
	}

	t.Run("oversized report is truncated on a rune boundary", func(t *testing.T) {
		got := formatStepCIFailureReport(StepCICliReturns{
			Messages: []string{strings.Repeat("é", maxStepCIFailureReportSize)},
		}, nil, nil)

		require.LessOrEqual(t, len(got), maxStepCIFailureReportSize)
		require.True(t, strings.HasSuffix(got, stepCIReportTruncatedNote))
		require.True(t, utf8.ValidString(got))
	})
}

func ptr[T any](value T) *T {
	return &value
}

func writeRunner(t testing.TB, dir, content string) {
	t.Helper()

	path := filepath.Join(dir, "stepci-captured-runner")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}
