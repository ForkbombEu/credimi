// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"os"
	"path/filepath"
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
echo '{"passed":false}'`)
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

func writeRunner(t testing.TB, dir, content string) {
	t.Helper()

	path := filepath.Join(dir, "stepci-captured-runner")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}
