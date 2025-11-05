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

func TestParseEudiwInput(t *testing.T) {
	defaultData := map[string]any{
		"id": `        {{

           credimi` + " ` " + `
              {
                "credimi_id": "eudiw_base",
                "field_default_value": "uuidv4",
                "field_description": "i18n_eudiw_description",
                "field_id": "testid",
                "field_label": "i18n_testid",
                "field_options": [],
                "field_type": "string"
              }
        ` + "` " + `}}`,
		"nonce": `        {{

           credimi` + " ` " + `
              {
                "credimi_id": "eudiw_nonce_base",
                "field_default_value": "uuidv4",
                "field_description": "i18n_eudiw_nonce_description",
                "field_id": "testnonce",
                "field_label": "i18n_testnonce",
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
			input: "user1+abc",
			want: map[string]any{
				"id": `{{
        
           credimi ` + " ` " + `
              {
                "credimi_id": "user1_abc_eudiw_base",
                "field_default_value": "uuidv4",
                "field_description": "i18n_eudiw_description",
                "field_id": "testid",
                "field_label": "i18n_testid",
                "field_options": [],
                "field_type": "string"
              }
         ` + "` " + `}}`,
				"nonce": `{{
        
           credimi ` + " ` " + `
              {
                "credimi_id": "user1_abc_eudiw_nonce_base",
                "field_default_value": "uuidv4",
                "field_description": "i18n_eudiw_nonce_description",
                "field_id": "testnonce",
                "field_label": "i18n_testnonce",
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
			name:      "Missing id",
			input:     "test",
			want:      nil,
			wantError: true,
		},
		{
			name:      "Missing nonce",
			input:     "test",
			want:      nil,
			wantError: true,
		},
	}

	emptyFile := writeTempYAML(t, map[string]any{})
	defer os.Remove(emptyFile)

	missingIDFile := writeTempYAML(t, map[string]any{"nonce": "foo"})
	defer os.Remove(missingIDFile)

	missingNonceFile := writeTempYAML(t, map[string]any{"id": "foo"})
	defer os.Remove(missingNonceFile)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fileToUse string
			switch tt.name {
			case "Empty YAML file":
				fileToUse = emptyFile
			case "Missing id":
				fileToUse = missingIDFile
			case "Missing nonce":
				fileToUse = missingNonceFile
			default:
				fileToUse = defaultFile
			}

			yamlBytes, err := ParseEudiwInput(tt.input, fileToUse)

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
