// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExtractCredimiJSON(t *testing.T) {
	t.Run("replaces credimi blocks and preserves functions", func(t *testing.T) {
		input := `
before
{{ credimi ` + "`" + `{"field_type":"string","field_default_value":"hello"}` + "`" + ` }}
middle
{{ credimi ` + "`" + `{"field_type":"string","field_default_value":"world"}` + "`" + ` uuidv4 }}
after
`
		out, err := extractCredimiJSON(input)
		require.NoError(t, err)
		require.NotContains(t, out, "{{")
		require.Contains(t, out, `"field_default_value":"hello"`)
		require.Contains(t, out, `"field_function":"uuidv4"`)
	})

	t.Run("invalid JSON reports error and keeps original", func(t *testing.T) {
		input := `{{ credimi ` + "`" + `{invalid` + "`" + ` }}`
		out, err := extractCredimiJSON(input)
		require.Error(t, err)
		require.Contains(t, out, input)
	})
}

func TestExtractValues(t *testing.T) {
	t.Run("extracts defaults and functions", func(t *testing.T) {
		node := map[string]any{
			"name": map[string]any{
				"field_type":          "string",
				"field_default_value": "Ada",
			},
			"payload": map[string]any{
				"field_type":          "object",
				"field_default_value": `{"x":1}`,
			},
			"uuid": map[string]any{
				"field_type":     "string",
				"field_function": "uuidv4",
			},
		}

		out := extractValues(node).(map[string]any)
		require.Equal(t, "Ada", out["name"])
		require.Equal(t, map[string]any{"x": float64(1)}, out["payload"])

		uuidStr, ok := out["uuid"].(string)
		require.True(t, ok)
		parsed, err := uuid.Parse(uuidStr)
		require.NoError(t, err)
		require.Equal(t, strings.ToLower(uuidStr), parsed.String())
	})

	t.Run("stringified field type is extracted", func(t *testing.T) {
		node := map[string]any{
			"value": `{"field_type":"string","field_default_value":"inline"}`,
		}
		out := extractValues(node).(map[string]any)
		require.Equal(t, "inline", out["value"])
	})

	t.Run("unknown function falls back to default value", func(t *testing.T) {
		node := map[string]any{
			"value": map[string]any{
				"field_type":          "string",
				"field_default_value": "fallback",
				"field_function":      "unknown",
			},
		}
		out := extractValues(node).(map[string]any)
		require.Equal(t, "fallback", out["value"])
	})
}
