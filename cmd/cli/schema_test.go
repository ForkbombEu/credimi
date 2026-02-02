// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratePipelineSchemaIncludesActivityOptions(t *testing.T) {
	schema, err := generatePipelineSchema()
	require.NoError(t, err)

	defs, ok := schema["$defs"].(map[string]any)
	require.True(t, ok)

	_, ok = defs["ActivityOptions"]
	require.True(t, ok)
}

func TestGeneratePipelineSchemaIncludesRegistryAndSpecialSteps(t *testing.T) {
	schema, err := generatePipelineSchema()
	require.NoError(t, err)

	oneOf := schemaStepOneOf(t, schema)
	uses := extractStepUses(t, oneOf)

	registryKeys := sortedRegistryKeys()
	require.Len(t, uses, len(registryKeys)+2)
	require.Equal(t, registryKeys, uses[:len(registryKeys)])
	require.Equal(t, []string{"debug", "child-pipeline"}, uses[len(registryKeys):])
}

func schemaStepOneOf(t *testing.T, schema map[string]any) []any {
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	steps, ok := properties["steps"].(map[string]any)
	require.True(t, ok)

	items, ok := steps["items"].(map[string]any)
	require.True(t, ok)

	oneOf, ok := items["oneOf"].([]any)
	require.True(t, ok)

	return oneOf
}

func extractStepUses(t *testing.T, oneOf []any) []string {
	uses := make([]string, 0, len(oneOf))

	for _, variant := range oneOf {
		variantMap, ok := variant.(map[string]any)
		require.True(t, ok)

		properties, ok := variantMap["properties"].(map[string]any)
		require.True(t, ok)

		useSchema, ok := properties["use"].(map[string]any)
		require.True(t, ok)

		useConst, ok := useSchema["const"].(string)
		require.True(t, ok)
		uses = append(uses, useConst)
	}

	return uses
}
