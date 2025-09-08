// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestMergeConfigs(t *testing.T) {
	tests := []struct {
		name     string
		global   map[string]string
		step     map[string]string
		expected map[string]string
	}{
		{
			name:     "step overrides global",
			global:   map[string]string{"a": "1", "b": "2"},
			step:     map[string]string{"b": "3"},
			expected: map[string]string{"a": "1", "b": "3"},
		},
		{
			name:     "empty step",
			global:   map[string]string{"a": "1"},
			step:     map[string]string{},
			expected: map[string]string{"a": "1"},
		},
		{
			name:     "empty global",
			global:   map[string]string{},
			step:     map[string]string{"c": "x"},
			expected: map[string]string{"c": "x"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MergeConfigs(tc.global, tc.step)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestResolveRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		ctx      map[string]any
		expected any
		wantErr  bool
	}{
		{
			name:     "simple one-level",
			ref:      "foo",
			ctx:      map[string]any{"foo": "bar"},
			expected: "bar",
		},
		{
			name:     "nested map",
			ref:      "a.b",
			ctx:      map[string]any{"a": map[string]any{"b": 42}},
			expected: 42,
		},
		{
			name:    "missing key",
			ref:     "x.y",
			ctx:     map[string]any{"x": map[string]any{}},
			wantErr: true,
		},
		{
			name:    "invalid path (not a map)",
			ref:     "a.b",
			ctx:     map[string]any{"a": 123},
			wantErr: true,
		},
		{
			name: "slice access",
			ref:  "arr[1]",
			ctx: map[string]any{
				"arr": []any{"zero", "one", "two"},
			},
			expected: "one",
		},
		{
			name: "nested slice in map",
			ref:  "m.values[2]",
			ctx: map[string]any{
				"m": map[string]any{
					"values": []any{10, 20, 30},
				},
			},
			expected: 30,
		},
		{
			name:    "slice index out of bounds",
			ref:     "arr[5]",
			ctx:     map[string]any{"arr": []any{1, 2, 3}},
			wantErr: true,
		},
		{
			name:    "slice on non-slice value",
			ref:     "notarr[0]",
			ctx:     map[string]any{"notarr": 123},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveRef(tc.ref, tc.ctx)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, got)
			}
		})
	}
}
func TestResolveExpressions(t *testing.T) {
	ctx := map[string]any{
		"user": map[string]any{
			"name":    "Alice",
			"age":     30,
			"emails":  []any{"a@example.com", "b@example.com"},
			"address": map[string]any{"city": "Wonderland"},
		},
	}

	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name:     "simple string replacement",
			input:    "${{ user.name }}",
			expected: "Alice",
		},
		{
			name:     "string with multiple expressions",
			input:    "Name: ${{ user.name }}, Age: ${{ user.age }}",
			expected: "Name: Alice, Age: 30",
		},
		{
			name:    "invalid expression",
			input:   "${{ unknown.key }}",
			wantErr: true,
		},
		{
			name: "nested map",
			input: map[string]any{
				"name": "${{ user.name }}",
				"city": "${{ user.address.city }}",
			},
			expected: map[string]any{
				"name": "Alice",
				"city": "Wonderland",
			},
		},
		{
			name: "nested array",
			input: []any{
				"${{ user.emails[0] }}",
				"${{ user.emails[1] }}",
			},
			expected: []any{
				"a@example.com",
				"b@example.com",
			},
		},
		{
			name:     "non-string value returns as-is",
			input:    42,
			expected: 42,
		},
		{
			name: "map with mixed types",
			input: map[string]any{
				"int":   123,
				"email": "${{ user.emails[0] }}",
				"nested": map[string]any{
					"city": "${{ user.address.city }}",
				},
			},
			expected: map[string]any{
				"int":   123,
				"email": "a@example.com",
				"nested": map[string]any{
					"city": "Wonderland",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveExpressions(tc.input, ctx)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, got)
			}
		})
	}
}

func TestCastType(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		typeStr  string
		expected any
		wantErr  bool
	}{
		{"string from int", 42, "string", "42", false},
		{"string from string", "hello", "string", "hello", false},
		{"int from string", "123", "int", 123, false},
		{"int from int", 99, "int", 99, false},
		{"int invalid string", "abc", "int", nil, true},
		{"map success", map[string]any{"a": 1}, "map", map[string]any{"a": 1}, false},
		{"map invalid", "notmap", "map", nil, true},
		{"slice of string", []any{"a", "b"}, "[]string", []string{"a", "b"}, false},
		{"slice of map", []any{map[string]any{"x": 1}}, "[]map", []map[string]any{{"x": 1}}, false},
		{"bytes from string", "data", "[]byte", []byte("data"), false},
		{"bytes from []byte", []byte("ok"), "[]byte", []byte("ok"), false},
		{"bytes invalid", 123, "[]byte", nil, true},
		{"unknown type returns original", 42, "unknown", 42, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := castType(tc.val, tc.typeStr)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, got)
			}
		})
	}
}
func TestResolveInputs(t *testing.T) {
	type testCase struct {
		name      string
		step      StepDefinition
		globalCfg map[string]string
		ctx       map[string]any
		wantErr   bool
		expected  *workflowengine.ActivityInput
	}

	tests := []testCase{
		{
			name: "payload from value",
			step: StepDefinition{
				With: StepInputs{
					Config: map[string]string{
						"key": "value",
					},
					Payload: map[string]InputSource{
						"p": {Value: "data"},
					},
				},
			},
			globalCfg: map[string]string{"g": "G"},
			ctx:       map[string]any{},
			expected: &workflowengine.ActivityInput{
				Config:  map[string]string{"key": "value", "g": "G"},
				Payload: map[string]any{"p": "data"},
			},
		},
		{
			name: "payload int conversion from string",
			step: StepDefinition{
				With: StepInputs{
					Payload: map[string]InputSource{
						"num": {Value: "123", Type: "int"},
					},
				},
			},
			ctx: map[string]any{},
			expected: &workflowengine.ActivityInput{
				Config:  map[string]string{},
				Payload: map[string]any{"num": 123},
			},
		},
		{
			name: "payload expression resolution",
			step: StepDefinition{
				With: StepInputs{
					Payload: map[string]InputSource{
						"val": {Value: "${{ ctx.key }}"},
					},
				},
			},
			ctx: map[string]any{"ctx": map[string]any{"key": "ok"}},
			expected: &workflowengine.ActivityInput{
				Config:  map[string]string{},
				Payload: map[string]any{"val": "ok"},
			},
		},
		{
			name: "type cast failure (cannot cast map to int)",
			step: StepDefinition{
				With: StepInputs{
					Payload: map[string]InputSource{
						"num": {Value: map[string]any{}, Type: "int"},
					},
				},
			},
			ctx:     map[string]any{},
			wantErr: true,
		},
		{
			name: "nested payload map and array expressions",
			step: StepDefinition{
				With: StepInputs{
					Payload: map[string]InputSource{
						"nested": {
							Value: map[string]any{
								"level1": map[string]any{
									"level2": map[string]any{
										"value": "${{ ctx.key }}",
									},
								},
								"array": []any{
									"${{ ctx.key }}",
									"static",
								},
							},
						},
					},
				},
			},
			ctx: map[string]any{"ctx": map[string]any{"key": 99}},
			expected: &workflowengine.ActivityInput{
				Config: map[string]string{},
				Payload: map[string]any{
					"nested": map[string]any{
						"level1": map[string]any{
							"level2": map[string]any{
								"value": 99,
							},
						},
						"array": []any{99, "static"},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveInputs(tc.step, tc.globalCfg, tc.ctx)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, got)
			}
		})
	}
}
