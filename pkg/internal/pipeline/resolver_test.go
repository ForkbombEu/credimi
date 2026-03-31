// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeConfigs(t *testing.T) {
	tests := []struct {
		name     string
		global   map[string]any
		step     map[string]any
		expected map[string]any
	}{
		{
			name:     "step overrides global",
			global:   map[string]any{"a": "1", "b": "2"},
			step:     map[string]any{"b": "3"},
			expected: map[string]any{"a": "1", "b": "3"},
		},
		{
			name:     "empty step",
			global:   map[string]any{"a": "1"},
			step:     map[string]any{},
			expected: map[string]any{"a": "1"},
		},
		{
			name:     "empty global",
			global:   map[string]any{},
			step:     map[string]any{"c": "x"},
			expected: map[string]any{"c": "x"},
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

func TestResolveInputs(t *testing.T) {
	tests := []struct {
		name            string
		step            StepDefinition
		globalCfg       map[string]any
		ctx             map[string]any
		wantErr         bool
		expectedPayload map[string]any
		expectedConfig  map[string]any
	}{
		{
			name: "payload from scalar value",
			step: StepDefinition{
				StepSpec: StepSpec{
					With: StepInputs{
						Config:  map[string]any{"key": "value"},
						Payload: map[string]any{"p": "data"},
					},
				},
			},
			globalCfg: map[string]any{"g": "G"},
			ctx:       map[string]any{},
			expectedPayload: map[string]any{
				"p": "data",
			},
			expectedConfig: map[string]any{
				"key": "value",
				"g":   "G",
			},
		},
		{
			name: "payload int",
			step: StepDefinition{
				StepSpec: StepSpec{
					With: StepInputs{
						Payload: map[string]any{"num": 123},
					},
				},
			},
			ctx: map[string]any{},
			expectedPayload: map[string]any{
				"num": 123,
			},
			expectedConfig: map[string]any{},
		},
		{
			name: "payload expression resolution",
			step: StepDefinition{
				StepSpec: StepSpec{
					With: StepInputs{
						Payload: map[string]any{"val": "${{ ctx.key }}"},
					},
				},
			},
			ctx: map[string]any{"ctx": map[string]any{"key": "ok"}},
			expectedPayload: map[string]any{
				"val": "ok",
			},
			expectedConfig: map[string]any{},
		},
		{
			name: "nested payload map and array expressions",
			step: StepDefinition{
				StepSpec: StepSpec{
					With: StepInputs{
						Payload: map[string]any{
							"nested": map[string]any{
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
			expectedPayload: map[string]any{
				"nested": map[string]any{
					"level1": map[string]any{
						"level2": map[string]any{
							"value": 99,
						},
					},
					"array": []any{99, "static"},
				},
			},
			expectedConfig: map[string]any{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			step := tc.step

			err := ResolveInputs(&step, tc.globalCfg, tc.ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tc.expectedPayload, step.With.Payload)
			require.Equal(t, tc.expectedConfig, step.With.Config)
		})
	}
}
