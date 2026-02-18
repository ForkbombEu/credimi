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

func writeChecksFile(t *testing.T, path string, checks []string) {
	t.Helper()
	data, err := json.Marshal(Checks{Checks: checks})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
}

func TestRunTemplateOutputDirErrors(t *testing.T) {
	err := runTemplate("missing.json", "default", "config", "missing-dir")
	require.Error(t, err)

	temp := t.TempDir()
	filePath := filepath.Join(temp, "file")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0600))
	err = runTemplate("missing.json", "default", "config", filePath)
	require.Error(t, err)
}

func TestRunTemplateInputErrors(t *testing.T) {
	temp := t.TempDir()
	outDir := filepath.Join(temp, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	err := runTemplate(filepath.Join(temp, "missing.json"), "default", "config", outDir)
	require.Error(t, err)

	invalidPath := filepath.Join(temp, "invalid.json")
	require.NoError(t, os.WriteFile(invalidPath, []byte("not-json"), 0600))
	err = runTemplate(invalidPath, "default", "config", outDir)
	require.Error(t, err)
}

func TestRunTemplateSkipsInvalidPath(t *testing.T) {
	temp := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(cwd))
	})
	require.NoError(t, os.Chdir(temp))

	outDir := filepath.Join(temp, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	inputPath := "input.json"
	writeChecksFile(t, inputPath, []string{"check-1"})

	err = runTemplate(inputPath, "default", "config", outDir)
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(outDir, "check-1.yaml"))
	require.Error(t, statErr)
}

func TestRunTemplateOpenidnetWritesFile(t *testing.T) {
	temp := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(cwd))
	})
	require.NoError(t, os.Chdir(temp))

	origParse := parseOpenidnetInput
	t.Cleanup(func() {
		parseOpenidnetInput = origParse
	})

	parseOpenidnetInput = func(_ string, _ string, _ string) ([]byte, error) {
		return []byte("output"), nil
	}

	inputDir := filepath.Join("a", "b", "openidnet")
	require.NoError(t, os.MkdirAll(inputDir, 0755))
	inputPath := filepath.Join(inputDir, "input.json")
	writeChecksFile(t, inputPath, []string{"check-1"})

	outDir := filepath.Join(temp, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	err = runTemplate(inputPath, "default", "config", outDir)
	require.NoError(t, err)
	data, readErr := os.ReadFile(filepath.Join(outDir, "check-1.yaml"))
	require.NoError(t, readErr)
	require.Equal(t, "output", string(data))
}

func TestRunTemplateEwcWritesFile(t *testing.T) {
	temp := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(cwd))
	})
	require.NoError(t, os.Chdir(temp))

	origParse := parseEwcInput
	t.Cleanup(func() {
		parseEwcInput = origParse
	})

	parseEwcInput = func(_ string, _ string) ([]byte, error) {
		return []byte("ewc-output"), nil
	}

	inputDir := filepath.Join("a", "b", "ewc")
	require.NoError(t, os.MkdirAll(inputDir, 0755))
	inputPath := filepath.Join(inputDir, "input.json")
	writeChecksFile(t, inputPath, []string{"check-2"})

	outDir := filepath.Join(temp, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	err = runTemplate(inputPath, "default", "config", outDir)
	require.NoError(t, err)
	data, readErr := os.ReadFile(filepath.Join(outDir, "check-2.yaml"))
	require.NoError(t, readErr)
	require.Equal(t, "ewc-output", string(data))
}

func TestRunTemplateEudiwWritesFile(t *testing.T) {
	temp := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(cwd))
	})
	require.NoError(t, os.Chdir(temp))

	origParse := parseEudiwInput
	t.Cleanup(func() {
		parseEudiwInput = origParse
	})

	parseEudiwInput = func(_ string, _ string) ([]byte, error) {
		return []byte("eudiw-output"), nil
	}

	inputDir := filepath.Join("a", "b", "eudiw")
	require.NoError(t, os.MkdirAll(inputDir, 0755))
	inputPath := filepath.Join(inputDir, "input.json")
	writeChecksFile(t, inputPath, []string{"check-3"})

	outDir := filepath.Join(temp, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	err = runTemplate(inputPath, "default", "config", outDir)
	require.NoError(t, err)
	data, readErr := os.ReadFile(filepath.Join(outDir, "check-3.yaml"))
	require.NoError(t, readErr)
	require.Equal(t, "eudiw-output", string(data))
}
