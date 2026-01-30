// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunTemplateRequiresOutputDir(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "checks.json")
	payload, err := json.Marshal(Checks{Checks: []string{}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(inputPath, payload, 0600))

	err = runTemplate(inputPath, "", "", filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestRunTemplateWithEmptyChecks(t *testing.T) {
	outputDir := t.TempDir()
	inputPath := filepath.Join(t.TempDir(), "checks.json")
	payload, err := json.Marshal(Checks{Checks: []string{}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(inputPath, payload, 0600))

	err = runTemplate(inputPath, "", "", outputDir)
	require.NoError(t, err)

	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}
