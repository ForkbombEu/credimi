// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompleteFCAFPipelineResolvesValidationAfterScenarioFailures(t *testing.T) {
	path := filepath.Join(
		"../../../config_templates/fcaf/wallet_solution/relying_party/pipelines",
		"fcaf-wallet-solution-relying-party-complete-validation.yaml",
	)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	definition, err := ParseWorkflow(string(data))
	require.NoError(t, err)
	require.NotEmpty(t, definition.Steps)

	context := make(map[string]any, len(definition.Steps)-1)
	for _, step := range definition.Steps[:len(definition.Steps)-1] {
		context[step.ID] = map[string]any{"outputs": nil}
	}
	validation := definition.Steps[len(definition.Steps)-1]
	require.Equal(t, "fcaf-validation", validation.Use)
	require.NoError(t, ResolveInputs(&validation, nil, context))
	require.Len(t, validation.With.Payload["pipeline_outputs"], 145)
}

func TestFCAFPipelineTemplatesParse(t *testing.T) {
	root := "../../../config_templates/fcaf/wallet_solution/relying_party"
	pipelinePaths, err := filepath.Glob(filepath.Join(root, "pipelines", "*.yaml"))
	require.NoError(t, err)
	scenarioPaths, err := filepath.Glob(filepath.Join(root, "scenarios", "*.yaml"))
	require.NoError(t, err)
	paths := append([]string{}, pipelinePaths...)
	paths = append(paths, scenarioPaths...)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			_, parseErr := ParseWorkflow(string(data))
			require.NoError(t, parseErr)
		})
	}
}
