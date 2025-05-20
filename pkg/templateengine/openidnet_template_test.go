// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package templateengine

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadYAML(t *testing.T) {
	// Create a valid JSON file
	validData := map[string]string{"test": "value"}
	validFile := writeTempYAML(t, validData)
	defer os.Remove(validFile)

	// Create an invalid JSON file
	invalidFile, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(invalidFile.Name())
	if _, err := invalidFile.WriteString("{invalid_yaml"); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}
	invalidFile.Close()

	tests := []struct {
		name      string
		filename  string
		wantError bool
		wantData  map[string]string
	}{
		{"Valid YAML file", validFile, false, validData},
		{"Invalid YAML file", invalidFile.Name(), true, nil},
		{"Non-existent file", "nonexistent.yaml", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data map[string]string
			err := LoadYAML(tt.filename, &data)

			if (err != nil) != tt.wantError {
				t.Errorf("expected error: %v, got: %v", tt.wantError, err)
			}

			if !tt.wantError && !reflect.DeepEqual(data, tt.wantData) {
				t.Errorf("expected data: %v, got: %v", tt.wantData, data)
			}
		})
	}
}

func TestValidateVariant(t *testing.T) {
	validConfig := Config{
		VariantKeys: map[string][]string{
			"key1": {"test1", "test2"},
			"key2": {"test"},
			"key3": {"test1", "test2"},
			"key4": {"test"},
		},
	}

	tests := []struct {
		name      string
		variant   map[string]string
		config    Config
		wantError bool
	}{
		{
			"Valid variant",
			map[string]string{"key1": "test1", "key2": "test", "key3": "test2", "key4": "test"},
			validConfig,
			false,
		},
		{
			"Invalid key 1 value",
			map[string]string{"key1": "invalid", "key2": "test", "key3": "test2", "key4": "test"},
			validConfig,
			true,
		},
		{
			"Invalid key 3 value",
			map[string]string{"key1": "test2", "key2": "test", "key3": "invalid", "key4": "test"},
			validConfig,
			true,
		},
		{
			"Missing key",
			map[string]string{"key2": "test", "key3": "invalid", "key4": "test"},
			validConfig,
			true,
		},
		{
			"Not valid key",
			map[string]string{"notvalid": "test1", "key2": "test", "key3": "invalid", "key4": "test"},
			validConfig,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariant(tt.variant, tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error: %v, got: %v", tt.wantError, err)
			}
		})
	}
}

func TestParseInput(t *testing.T) {
	// Create default YAML file
	defaultData := map[string]any{
		"form": map[string]any{
			"alias": `        {{

           credimi \"
              {
                "credimi_id": "oid_alias",
                "field_default_value": "uuidv4",
                "field_description": "i18n_test_alias_description",
                "field_id": "testalias",
                "field_label": "i18n_testalias",
                "field_options": [],
                "field_type": "string"
              }
        \"}}`,
			"client": map[string]any{
				"default_field": "default_value",
			},
		},
	}
	defaultFile := writeTempYAML(t, defaultData)
	defer os.Remove(defaultFile)

	configData := map[string]any{
		"variant_order": []string{"credential_format", "client_id_scheme", "request_method", "response_mode"},
		"variant_keys": map[string][]string{
			"credential_format": {"test1"},
			"client_id_scheme":  {"test2"},
			"request_method":    {"test3"},
			"response_mode":     {"test4"},
		},
		"optional_fields": map[string]any{
			"test_field": map[string]any{
				"values": map[string][]string{
					"credential_format": {"test1"},
				},
				"template": "test_value",
			},
		},
	}
	configFile := writeTempYAML(t, configData)
	defer os.Remove(configFile)

	tests := []struct {
		name      string
		input     string
		wantForm  map[string]any
		wantError bool
	}{
		{
			name:  "Valid input",
			input: "test1:test2:test3:test4",
			wantForm: map[string]any{
				"alias": `{{
        
           credimi \"
              {
                "credimi_id": "test1_test2_test3_test4_oid_alias",
                "field_default_value": "uuidv4",
                "field_description": "i18n_test_alias_description",
                "field_id": "testalias",
                "field_label": "i18n_testalias",
                "field_options": [],
                "field_type": "string"
              }
        \"}}`,
				"client": map[string]any{
					"default_field": "default_value",
					"test_field":    "test_value",
				},
			},
			wantError: false,
		},
		{
			name:      "Invalid input format",
			input:     "invalid_format",
			wantForm:  nil,
			wantError: true,
		},
		{
			name:      "Not allowed variant value",
			input:     "tes1:invalid:test3:test4",
			wantForm:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlBytes, err := ParseInput(tt.input, defaultFile, configFile)

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

			form, ok := parsed["form"]
			if !ok {
				t.Fatalf("missing 'form' in output YAML")
			}

			normalizedWant := normalizeStrings(tt.wantForm)
			normalizedGot := normalizeStrings(form)

			wantJSON, _ := json.Marshal(normalizedWant)
			gotJSON, _ := json.Marshal(normalizedGot)

			if string(wantJSON) != string(gotJSON) {
				t.Errorf("form mismatch:\nexpected:\n%s\ngot:\n%s", wantJSON, gotJSON)
			}

		})
	}
}

// writeTempYAML creates a temporary YAML file with the provided content.
func writeTempYAML(t *testing.T, content interface{}) string {
	t.Helper()
	file, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	data, err := yaml.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal YAML: %v", err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	return file.Name()
}

func normalizeStrings(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, v2 := range val {
			out[k] = normalizeStrings(v2)
		}
		return out
	case []any:
		for i := range val {
			val[i] = normalizeStrings(val[i])
		}
		return val
	case string:
		return strings.Join(strings.Fields(val), " ")
	default:
		return val
	}
}
