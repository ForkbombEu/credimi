// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/stretchr/testify/require"
)

func TestFCAFManualPipelineTemplatesParse(t *testing.T) {
	root := filepath.Join(
		"..",
		"..",
		"..",
		"config_templates",
		"fcaf",
		"wallet_solution",
		"relying_party",
		"pipelines",
	)

	files, err := filepath.Glob(filepath.Join(root, "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			raw, err := os.ReadFile(file)
			require.NoError(t, err)

			def, err := pipelineinternal.ParseWorkflow(string(raw))
			require.NoError(t, err)
			require.NotEmpty(t, def.Steps)
		})
	}
}
