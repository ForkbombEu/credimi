// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfigSource_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yamlStr string
		want    ConfigSource
		wantErr bool
	}{
		{
			name:    "scalar string",
			yamlStr: `"hello"`,
			want:    ConfigSource{Value: "hello"},
		},
		{
			name:    "map with ref",
			yamlStr: `ref: someRef`,
			want:    ConfigSource{Ref: "someRef"},
		},
		{
			name:    "map with value",
			yamlStr: `value: directVal`,
			want:    ConfigSource{Value: "directVal"},
		},
		{
			name:    "invalid type (sequence)",
			yamlStr: `- one`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got ConfigSource
			err := yaml.Unmarshal([]byte(tc.yamlStr), &got)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}

func TestInputSource_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yamlStr string
		want    InputSource
		wantErr bool
	}{
		{
			name:    "scalar string",
			yamlStr: `"abc"`,
			want:    InputSource{Value: "abc"},
		},
		{
			name:    "scalar int",
			yamlStr: `42`,
			want:    InputSource{Value: 42},
		},
		{
			name: "mapping with ref and type",
			yamlStr: `
ref: some.path
type: int
value: 123
`,
			want: InputSource{Ref: "some.path", Type: "int", Value: 123},
		},
		{
			name: "mapping with value only",
			yamlStr: `
value: hello
`,
			want: InputSource{Value: "hello"},
		},
		{
			name: "mapping generic object",
			yamlStr: `
foo: bar
num: 99
`,
			want: InputSource{Value: map[string]any{"foo": "bar", "num": 99}},
		},
		{
			name: "sequence array",
			yamlStr: `
- a
- b
- 1
`,
			want: InputSource{Value: []any{"a", "b", 1}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got InputSource
			err := yaml.Unmarshal([]byte(tc.yamlStr), &got)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}

func TestWorkflowDefinition_UnmarshalYAML(t *testing.T) {
	yml := `
config:
  globalKey: globalVal
steps:
  - name: step1
    activity: http
    inputs:
      config:
        apiKey: "abc123"
      payload:
        url: "http://example.com"
  - name: step2
    activity: docker
    inputs:
      config:
        image: { value: "alpine:latest" }
      payload:
        args:
          - run
          - echo
`
	var wf WorkflowDefinition
	err := yaml.Unmarshal([]byte(yml), &wf)
	require.NoError(t, err)

	require.Equal(t, "globalVal", wf.Config["globalKey"])
	require.Len(t, wf.Steps, 2)

	s1 := wf.Steps[0]
	require.Equal(t, "step1", s1.Name)
	require.Equal(t, "http", s1.Activity)
	require.Equal(t, "abc123", s1.Inputs.Config["apiKey"].Value)
	require.Equal(t, "http://example.com", s1.Inputs.Payload["url"].Value)

	s2 := wf.Steps[1]
	require.Equal(t, "step2", s2.Name)
	require.Equal(t, "docker", s2.Activity)
	require.Equal(t, "alpine:latest", s2.Inputs.Config["image"].Value)
	require.Equal(t, []any{"run", "echo"}, s2.Inputs.Payload["args"].Value)
}
