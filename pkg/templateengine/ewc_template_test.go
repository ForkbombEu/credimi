// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package templateengine

import (
	"encoding/json"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseEwcInput(t *testing.T) {
	defaultData := map[string]any{
		"sessionId": `        {{

           credimi` + " ` " + `
              {
                "credimi_id": "ewc_base",
                "field_default_value": "uuidv4",
                "field_description": "i18n_ewc_description",
                "field_id": "testewc",
                "field_label": "i18n_testewc",
                "field_options": [],
                "field_type": "string"
              }
        ` + "` " + `}}`,
	}

	defaultFile := writeTempYAML(t, defaultData)
	defer os.Remove(defaultFile)

	tests := []struct {
		name      string
		input     string
		want      map[string]any
		wantError bool
	}{
		{
			name:  "Valid input",
			input: "test1+test2+test3",
			want: map[string]any{
				"sessionId": `{{
        
           credimi ` + " ` " + `
              {
                "credimi_id": "test1_test2_test3_ewc_base",
                "field_default_value": "uuidv4",
                "field_description": "i18n_ewc_description",
                "field_id": "testewc",
                "field_label": "i18n_testewc",
                "field_options": [],
                "field_type": "string"
              }
         ` + "` " + `}}`,
			},
			wantError: false,
		},
		{
			name:      "Empty YAML file",
			input:     "test",
			want:      nil,
			wantError: true,
		},
		{
			name:      "Missing sessionId",
			input:     "test",
			want:      nil,
			wantError: true,
		},
	}

	emptyFile := writeTempYAML(t, map[string]any{})
	defer os.Remove(emptyFile)
	missingSessionFile := writeTempYAML(t, map[string]any{"foo": "bar"})
	defer os.Remove(missingSessionFile)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fileToUse string
			switch tt.name {
			case "Empty YAML file":
				fileToUse = emptyFile
			case "Missing sessionId":
				fileToUse = missingSessionFile
			default:
				fileToUse = defaultFile
			}

			yamlBytes, err := ParseEwcInput(tt.input, fileToUse)

			if (err != nil) != tt.wantError {
				t.Fatalf("expected error: %v, got: %v", tt.wantError, err)
			}

			if tt.wantError {
				return
			}

			var parsed map[string]any
			if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
				t.Fatalf("failed to unmarshal output: %v", err)
			}

			normalizedWant := normalizeStrings(tt.want)
			normalizedGot := normalizeStrings(parsed)

			wantJSON, _ := json.Marshal(normalizedWant)
			gotJSON, _ := json.Marshal(normalizedGot)

			if string(wantJSON) != string(gotJSON) {
				t.Errorf("YAML output mismatch:\nexpected:\n%s\ngot:\n%s", wantJSON, gotJSON)
			}
		})
	}
}
