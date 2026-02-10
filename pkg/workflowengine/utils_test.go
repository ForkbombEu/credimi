// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type payloadForDecode struct {
	Name  string `json:"name"  validate:"required"`
	Count int    `json:"count"`
}

func TestDecodePayload(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		input := map[string]any{"name": "ok", "count": 3}
		got, err := DecodePayload[payloadForDecode](input)
		require.NoError(t, err)
		require.Equal(t, "ok", got.Name)
		require.Equal(t, 3, got.Count)
	})

	t.Run("invalid marshal", func(t *testing.T) {
		_, err := DecodePayload[payloadForDecode](make(chan int))
		require.Error(t, err)
	})

	t.Run("invalid unmarshal", func(t *testing.T) {
		input := map[string]any{"name": "ok", "count": "nope"}
		_, err := DecodePayload[payloadForDecode](input)
		require.Error(t, err)
	})

	t.Run("validation error", func(t *testing.T) {
		input := map[string]any{"name": "", "count": 1}
		_, err := DecodePayload[payloadForDecode](input)
		require.Error(t, err)
	})
}

func TestAsSliceOfMaps(t *testing.T) {
	maps := []map[string]any{{"a": 1}, {"b": 2}}
	require.Equal(t, maps, AsSliceOfMaps(maps))

	mixed := []any{map[string]any{"a": 1}, "skip", map[string]any{"b": 2}}
	got := AsSliceOfMaps(mixed)
	require.Len(t, got, 2)
	require.Equal(t, map[string]any{"a": 1}, got[0])
	require.Equal(t, map[string]any{"b": 2}, got[1])

	require.Nil(t, AsSliceOfMaps("nope"))
}

func TestAsSliceOfStrings(t *testing.T) {
	require.Equal(t, []string{"a", "b"}, AsSliceOfStrings([]string{"a", "b"}))

	got := AsSliceOfStrings([]any{1, "two", true})
	require.Equal(t, []string{"1", "two", "true"}, got)

	require.Nil(t, AsSliceOfStrings(123))
}

func TestAsString(t *testing.T) {
	require.Equal(t, "ok", AsString("ok"))
	require.Equal(t, "42", AsString(42))
}

func TestAsMap(t *testing.T) {
	m := map[string]any{"a": 1}
	require.Equal(t, m, AsMap(m))
	require.Nil(t, AsMap([]any{"no"}))
}

func TestAsBool(t *testing.T) {
	require.True(t, AsBool(true))
	require.False(t, AsBool("no"))
}
