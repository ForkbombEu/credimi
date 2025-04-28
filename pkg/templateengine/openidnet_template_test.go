// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package templateengine

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestLoadJSON(t *testing.T) {
	// Create a valid JSON file
	validData := map[string]string{"test": "value"}
	validFile := writeTempJSON(t, validData)
	defer os.Remove(validFile)

	// Create an invalid JSON file
	invalidFile, err := os.CreateTemp("", "*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(invalidFile.Name())
	if _, err := invalidFile.WriteString("{invalid_json"); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}
	invalidFile.Close()

	tests := []struct {
		name      string
		filename  string
		wantError bool
		wantData  map[string]string
	}{
		{"Valid JSON file", validFile, false, validData},
		{"Invalid JSON file", invalidFile.Name(), true, nil},
		{"Non-existent file", "nonexistent.json", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data map[string]string
			err := LoadJSON(tt.filename, &data)

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
			"credential_format": {"test1", "test2"},
			"client_id_scheme":  {"test"},
			"request_method":    {"test1", "test2"},
			"response_mode":     {"test"},
		},
	}

	tests := []struct {
		name      string
		variant   Variant
		config    Config
		wantError bool
	}{
		{"Valid variant", Variant{"test1", "test", "test2", "test"}, validConfig, false},
		{"Invalid credential_format", Variant{"invalid", "test", "test1", "test"}, validConfig, true},
		{"Invalid request_method", Variant{"test2", "test", "invalid", "test"}, validConfig, true},
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
	// Create default JSON file
	defaultData := map[string]any{"form": map[string]any{"client": map[string]any{"default_field": "default_value"}}}
	defaultFile := writeTempJSON(t, defaultData)
	defer os.Remove(defaultFile)

	// Create config JSON file
	configData := map[string]any{
		"variant_keys": map[string][]string{
			"credential_format": {"test1"},
			"client_id_scheme":  {"test2"},
			"request_method":    {"test3"},
			"response_mode":     {"test4"},
		},
		"optional_fields": map[string]struct {
			Values   map[string][]string `json:"values"`
			Template string              `json:"template"`
		}{
			"test_field": {
				Values:   map[string][]string{"credential_format": {"test1"}},
				Template: "test_value",
			},
		},
	}
	configFile := writeTempJSON(t, configData)
	defer os.Remove(configFile)

	tests := []struct {
		name      string
		input     string
		wantForm  map[string]any
		wantError bool
	}{
		{
			"Valid input",
			"test1:test2:test3:test4",
			map[string]any{"client": map[string]any{"default_field": "default_value", "test_field": "test_value"}},
			false,
		},
		{
			"Invalid input format",
			"invalid_format",
			nil,
			true,
		},
		{
			"Not allowed variant value",
			"tes1:invalid:test3:test4",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseInput(tt.input, defaultFile, configFile)

			if (err != nil) != tt.wantError {
				t.Errorf("expected error: %v, got: %v", tt.wantError, err)
			}

			if !tt.wantError && !reflect.DeepEqual(result.Form, tt.wantForm) {
				t.Errorf("expected form: %+v, got: %+v", tt.wantForm, result.Form)
			}
		})
	}
}

// writeTempJSON creates a temporary JSON file with the provided content.
func writeTempJSON(t *testing.T, content interface{}) string {
	t.Helper()
	file, err := os.CreateTemp("", "*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	data, err := json.MarshalIndent(content, "", "    ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	return file.Name()
}
