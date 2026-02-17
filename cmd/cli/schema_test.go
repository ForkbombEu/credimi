// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaCmdOutputsJSON(t *testing.T) {
	prevOutputPath := outputPath
	outputPath = ""
	t.Cleanup(func() {
		outputPath = prevOutputPath
	})

	cmd := NewSchemaCmd()
	output := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	require.True(t, strings.HasPrefix(strings.TrimSpace(output), "{"))
	require.Contains(t, output, "\"$defs\"")
}

func TestNewSchemaCmdOutputsYAMLFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "schema.yaml")

	prevOutputPath := outputPath
	outputPath = ""
	t.Cleanup(func() {
		outputPath = prevOutputPath
	})

	cmd := NewSchemaCmd()
	cmd.SetArgs([]string{"--output", path})
	output := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	require.Contains(t, output, "saved to")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.Contains(t, string(content), "steps:")
}

func TestGeneratePipelineSchemaIncludesDefs(t *testing.T) {
	schema, err := generatePipelineSchema()
	require.NoError(t, err)

	defs, ok := schema["$defs"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, defs, "ActivityOptions")

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, properties, "steps")
}

func TestGenerateSingleStepSchemaMobileAutomation(t *testing.T) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := generateSingleStepSchema(&reflector, "mobile-automation")
	require.NotNil(t, schema)

	properties := schema["properties"].(map[string]any)
	with := properties["with"].(map[string]any)
	require.Contains(t, with, "oneOf")
}

func TestExtractXOneOfGroups(t *testing.T) {
	type sample struct {
		A string `json:"a" xoneof:"group1"`
		B string `json:"b" xoneof:"group1"`
		C string `json:"-" xoneof:"group2"`
	}

	groups := extractXOneOfGroups(reflect.TypeOf(sample{}))
	require.Equal(t, []string{"a", "b"}, groups["group1"])
	_, ok := groups["group2"]
	require.False(t, ok)
}
