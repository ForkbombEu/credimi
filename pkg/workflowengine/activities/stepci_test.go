// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func ptr[T any](value T) *T {
	return &value
}

func writeRunner(t testing.TB, dir, content string) {
	t.Helper()

	path := filepath.Join(dir, "stepci-captured-runner")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}
