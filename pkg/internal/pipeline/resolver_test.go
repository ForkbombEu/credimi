// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type StrangeStruct struct {
	StringField  string
	NilField     interface{}
	MapField     map[string]any
	SliceField   []string
	PrivateField string
}

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
			name:     "string with object expression renders json",
			input:    "User:\n${{ user.address }}",
			expected: "User:\n{\n  \"city\": \"Wonderland\"\n}",
		},
		{
			name:     "string with array expression renders json",
			input:    "Emails:\n${{ user.emails }}",
			expected: "Emails:\n[\n  \"a@example.com\",\n  \"b@example.com\"\n]",
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

func TestPipelineFunctionsUpperLower(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		ctx      map[string]any
		expected any
		wantErr  bool
	}{
		{
			name: "simple upper",
			expr: "${{ result | upper }}",
			ctx: map[string]any{
				"result": "  hello world  ",
			},
			expected: "  HELLO WORLD  ",
		},
		{
			name: "simple lower",
			expr: "${{ name | lower }}",
			ctx: map[string]any{
				"name": "  JOHN DOE  ",
			},
			expected: "  john doe  ",
		},
		{
			name: "number upper",
			expr: "${{ number | upper }}",
			ctx: map[string]any{
				"number": 123,
			},
			expected: 123,
		},
		{
			name: "pipeline output upper",
			expr: "${{ pipeline_output | upper }}",
			ctx: map[string]any{
				"pipeline_output": "test output",
			},
			expected: "TEST OUTPUT",
		},
		{
			name: "unknown function",
			expr: "${{ result | unknown }}",
			ctx: map[string]any{
				"result": "  hello world  ",
			},
			wantErr: true,
		},
		{
			name: "invalid pipeline - empty function",
			expr: "${{ result | }}",
			ctx: map[string]any{
				"result": "  hello world  ",
			},
			wantErr: true,
		},
		{
			name: "invalid pipeline - empty initial value",
			expr: "${{ | upper }}",
			ctx: map[string]any{
				"result": "  hello world  ",
			},
			wantErr: true,
		},
		{
			name: "complex json object with upper",
			expr: "${{ complexObject | upper }}",
			ctx: map[string]any{
				"complexObject": map[string]any{
					"hello": map[string]any{
						"world": "heLLo",
					},
					"message": " hEllO: { WorlD: {heLLo}} ",
				},
			},
			expected: map[string]any{
				"HELLO": map[string]any{
					"WORLD": "HELLO",
				},
				"MESSAGE": " HELLO: { WORLD: {HELLO}} ",
			},
		},
		{
			name: "strange struct with upper",
			expr: "${{ strange | upper }}",
			ctx: map[string]any{
				"strange": StrangeStruct{
					StringField: "hello",
					NilField:    nil,
					MapField:    map[string]any{"key": "value"},
					SliceField:  []string{},
				},
			},
			expected: map[string]any{
				"STRINGFIELD": "HELLO",
				"NILFIELD":    nil,
				"MAPFIELD": map[string]any{
					"KEY": "VALUE",
				},
				"SLICEFIELD":   []any{},
				"PRIVATEFIELD": "",
			},
		},
		{
			name: "number lower",
			expr: "${{ number | lower }}",
			ctx: map[string]any{
				"number": 123,
			},
			expected: 123,
		},
		{
			name: "invalid pipeline - empty initial value",
			expr: "${{ | lower }}",
			ctx: map[string]any{
				"result": "  hello world  ",
			},
			wantErr: true,
		},
		{
			name: "complex json object with lower",
			expr: "${{ complexObject | lower }}",
			ctx: map[string]any{
				"complexObject": map[string]any{
					"hello": map[string]any{
						"world": "heLLo",
					},
					"message": " hEllO: { WorlD: {heLLo}} ",
				},
			},
			expected: map[string]any{
				"hello": map[string]any{
					"world": "hello",
				},
				"message": " hello: { world: {hello}} ",
			},
		},
		{
			name: "array with lower",
			expr: "${{ items | lower }}",
			ctx: map[string]any{
				"items": []any{"APPLE", "BANANA"},
			},
			expected: []any{"apple", "banana"},
		},
		{
			name: "complex json object with lower",
			expr: "${{ complexObject | lower }}",
			ctx: map[string]any{
				"complexObject": map[string]any{
					"hello": map[string]any{
						"world": "heLLo",
					},
					"message": "{}",
				},
			},
			expected: map[string]any{
				"hello": map[string]any{
					"world": "hello",
				},
				"message": "{}",
			},
		},
		{
			name: "empty map to upper",
			expr: "${{ empty | upper }}",
			ctx: map[string]any{
				"empty": map[string]any{},
			},
			expected: map[string]any{},
		},
		{
			name: "strange struct with lower",
			expr: "${{ strange | lower }}",
			ctx: map[string]any{
				"strange": StrangeStruct{
					StringField: "HELLO",
					NilField:    nil,
					MapField:    map[string]any{"KEY": "VALue"},
					SliceField:  []string{},
				},
			},
			expected: map[string]any{
				"stringfield": "hello",
				"nilfield":    nil,
				"mapfield": map[string]any{
					"key": "value",
				},
				"slicefield":   []any{},
				"privatefield": "",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveExpressions(tc.expr, tc.ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestPipelineFunctionsSlice(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		ctx      map[string]any
		expected any
		wantErr  bool
	}{
		{
			name: "simple slice",
			expr: "${{ result | slice(:3) }}",
			ctx: map[string]any{
				"result": "hello world",
			},
			expected: "hel",
		},
		{
			name: "simple slice",
			expr: "${{ result | slice(7:) }}",
			ctx: map[string]any{
				"result": "hello world",
			},
			expected: "orld",
		},
		{
			name: "simple slice",
			expr: "${{ result | slice(3:7) }}",
			ctx: map[string]any{
				"result": "hello world",
			},
			expected: "lo w",
		},
		{
			name: "complex slice",
			expr: "${{ complexObject | slice(8:12) }}",
			ctx: map[string]any{
				"complexObject": map[string]any{
					"hello": map[string]any{
						"world": "heLLo",
					},
					"message": " hEllO: { WorlD: {heLLo}} ",
				},
			},
			expected: `:{"w`,
		},
		{
			name: "upper and slice",
			expr: "${{ result | upper | slice(7:) }}",
			ctx: map[string]any{
				"result": "hello world",
			},
			expected: "ORLD",
		},
		{
			name: "array slice first 2 elements",
			expr: "${{ items | slice(:2) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry", "date"},
			},
			expected: []any{"apple", "banana"},
		},
		{
			name: "array slice from index 2 to end",
			expr: "${{ items | slice(2:) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry", "date"},
			},
			expected: []any{"cherry", "date"},
		},
		{
			name: "array slice from index 1 to 3",
			expr: "${{ items | slice(1:3) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry", "date"},
			},
			expected: []any{"banana", "cherry"},
		},
		{
			name: "slice element at index 2",
			expr: "${{ result | slice(2) }}",
			ctx: map[string]any{
				"result": "hello",
			},
			expected: "l",
		},
		{
			name: "slice element at index -3",
			expr: "${{ result | slice(-3) }}",
			ctx: map[string]any{
				"result": "hello",
			},
			expected: "slice: index -3 out of bounds [0:5]",
			wantErr:  true,
		},
		{
			name: "slice element",
			expr: "${{ result | slice(-3:-1) }}",
			ctx: map[string]any{
				"result": "hello",
			},
			expected: "slice: invalid range [-3:-1] for length 5",
			wantErr:  true,
		},
		{
			name: "array slice single element at index 2",
			expr: "${{ items | slice(2) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry", "date"},
			},
			expected: "cherry",
		},
		{
			name: "array slice with negative index",
			expr: "${{ items | slice(-2) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry", "date"},
			},
			expected: "slice: index -2 out of bounds [0:4]",
			wantErr:  true,
		},
		{
			name: "array slice with negative range",
			expr: "${{ items | slice(-3:-1) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry", "date"},
			},
			expected: "slice: invalid range [-3:-1] for length 4",
			wantErr:  true,
		},
		{
			name: "array slice with upper then slice",
			expr: "${{ items | upper | slice(:2) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry"},
			},
			expected: []any{"APPLE", "BANANA"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveExpressions(tc.expr, tc.ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestPipelineFunctionsURL(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		ctx      map[string]any
		expected any
		wantErr  bool
	}{
		{
			name: "simple url encode",
			expr: "${{ result | url_encode }}",
			ctx: map[string]any{
				"result": "  hello world  ",
			},
			expected: "%20%20hello%20world%20%20",
		},
		{
			name: "simple url decode",
			expr: "${{ name | url_decode }}",
			ctx: map[string]any{
				"name": "%20%20hello%20world%20%20",
			},
			expected: "  hello world  ",
		},
		{
			name: "number url encode",
			expr: "${{ number | url_encode }}",
			ctx: map[string]any{
				"number": 123,
			},
			expected: 123,
		},
		{
			name: "pipeline output url encode",
			expr: "${{ pipeline_output | url_encode }}",
			ctx: map[string]any{
				"pipeline_output": "test output",
			},
			expected: "test%20output",
		},
		{
			name: "complex json object with url encode",
			expr: "${{ complexObject | url_encode }}",
			ctx: map[string]any{
				"complexObject": map[string]any{
					"hello": map[string]any{
						"world": "heLLo",
					},
					"message": " hEllO: { WorlD: {heLLo}} ",
				},
			},
			expected: map[string]any{
				"hello": map[string]any{
					"world": "heLLo",
				},
				"message": "%20hEllO%3A%20%7B%20WorlD%3A%20%7BheLLo%7D%7D%20",
			},
		},
		{
			name: "strange struct url encode",
			expr: "${{ strange | url_encode }}",
			ctx: map[string]any{
				"strange": StrangeStruct{
					StringField: "hello",
					NilField:    nil,
					MapField:    map[string]any{"key": "value"},
					SliceField:  []string{},
				},
			},
			expected: map[string]any{
				"StringField": "hello",
				"NilField":    nil,
				"MapField": map[string]any{
					"key": "value",
				},
				"SliceField":   []any{},
				"PrivateField": "",
			},
		},
		{
			name: "number url decode",
			expr: "${{ number | url_decode}}",
			ctx: map[string]any{
				"number": 123,
			},
			expected: 123,
		},
		{
			name: "complex json object url decode",
			expr: "${{ complexObject | url_decode }}",
			ctx: map[string]any{
				"complexObject": map[string]any{
					"hello": map[string]any{
						"world": "heLLo",
					},
					"message": "%20hEllO:%20%7B%20WorlD:%20%7BheLLo%7D%7D%20",
				},
			},
			expected: map[string]any{
				"hello": map[string]any{
					"world": "heLLo",
				},
				"message": " hEllO: { WorlD: {heLLo}} ",
			},
		},
		{
			name: "array url encode",
			expr: "${{ items | url_encode}}",
			ctx: map[string]any{
				"items": []any{"APPLE ", "BANANA"},
			},
			expected: []any{"APPLE%20", "BANANA"},
		},
		{
			name: "strange struct with url decode",
			expr: "${{ strange | url_decode }}",
			ctx: map[string]any{
				"strange": StrangeStruct{
					StringField: "HELLO",
					NilField:    nil,
					MapField:    map[string]any{"KEY": "V%C3%A0lue"},
					SliceField:  []string{},
				},
			},
			expected: map[string]any{
				"StringField": "HELLO",
				"NilField":    nil,
				"MapField": map[string]any{
					"KEY": "Vàlue",
				},
				"SliceField":   []any{},
				"PrivateField": "",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveExpressions(tc.expr, tc.ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestPipelineFunctionsReplace(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		ctx      map[string]any
		expected any
		wantErr  bool
	}{
		{
			name:     "replace number keeps number",
			expr:     "${{ num | replace(1,9) }}",
			ctx:      map[string]any{"num": 123},
			expected: 923,
		},
		{
			name: "replace map keys",
			expr: "${{ data | replace(old,new) }}",
			ctx: map[string]any{
				"data": map[string]any{
					"old_name": "old_value",
					"other":    "test",
				},
			},
			expected: map[string]any{
				"new_name": "new_value",
				"other":    "test",
			},
		},
		{
			name: "array replace",
			expr: "${{ items | replace(apple,orange) }}",
			ctx: map[string]any{
				"items": []any{"apple", "banana", "cherry"},
			},
			expected: []any{"orange", "banana", "cherry"},
		},
		{
			name: "strange struct with replace",
			expr: "${{ strange | replace(HELLO,world) | replace(KEY,my_key) }}",
			ctx: map[string]any{
				"strange": StrangeStruct{
					StringField: "HELLO",
					NilField:    nil,
					MapField:    map[string]any{"KEY": "Value"},
					SliceField:  []string{},
				},
			},
			expected: map[string]any{
				"StringField": "world",
				"NilField":    nil,
				"MapField": map[string]any{
					"my_key": "Value",
				},
				"SliceField":   []any{},
				"PrivateField": "",
			},
		},
		{
			name:     "replace float keeps float",
			expr:     "${{ num | replace(1,9) }}",
			ctx:      map[string]any{"num": 123.14},
			expected: 923.94,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveExpressions(tc.expr, tc.ctx)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}
