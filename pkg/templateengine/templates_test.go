// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package templateengine

import (
	"fmt"
	"io"
	"strings"
	"testing"
)


func TestCredimi_ValidJSON(t *testing.T) {
	jsonInput := `
	{
		"credimi_id": "credimi_123",
		"field_id": "field_abc",
		"field_label": "label_abc",
		"field_description": "desc_abc",
		"field_default_value": "default_val",
		"field_type": "string",
		"field_options": ["opt1", "opt2"]
	}`

	result, err := credimi(jsonInput)
	if err != nil {
		t.Errorf("credimi() returned error: %v", err)
	}
	expected := "{{ .field_abc }}"
	if result != expected {
		t.Errorf("credimi() = %v, want %v", result, expected)
	}

	meta, ok := metadataStore["field_abc"]
	if !ok {
		t.Errorf("metadataStore does not contain field_abc")
	}
	if meta.CredimiID != "credimi_123" ||
		meta.FieldID != "field_abc" ||
		meta.FieldLabel != "label_abc" ||
		meta.FieldDefault != "desc_abc" ||
		meta.FieldType != "string" ||
		meta.FieldDefault != "default_val" ||
		len(meta.FieldOptions) != 2 ||
		meta.FieldOptions[0] != "opt1" ||
		meta.FieldOptions[1] != "opt2" {
		t.Errorf("metadataStore entry incorrect: %+v", meta)
	}
}

func TestCredimi_InvalidJSON(t *testing.T) {
	invalidJSON := `{"credimi_id": "id", "field_id": "field"`
	_, err := credimi(invalidJSON)
	if err == nil {
		t.Errorf("Expected error for invalid JSON, got nil")
	}
}

func TestCredimi_WhitespaceTrim(t *testing.T) {
	jsonInput := `
	{
		"credimi_id": "id_trim",
		"field_id": "field_trim",
		"field_label": "label_trim",
		"field_description": "desc_trim",
		"field_default_value": "val_trim",
		"field_type": "string",
		"field_options": []
	}
	`
	result, err := credimi("\n  "+jsonInput+"  \n")
	if err != nil {
		t.Errorf("credimi() returned error: %v", err)
	}
	expected := "{{ .field_trim }}"
	if result != expected {
		t.Errorf("credimi() = %v, want %v", result, expected)
	}
}

func TestCredimi_BackslashCleanup(t *testing.T) {
	jsonInput := `
	{
		"credimi_id": "id_bs",
		"field_id": "field_bs",
		"field_label": "label_bs",
		"field_description": "desc_bs",
		"field_default_value": "foo\\\\\\\\bar",
		"field_type": "string",
		"field_options": []
	}
	`
	_, err := credimi(jsonInput)
	if err != nil {
		t.Errorf("credimi() returned error: %v", err)
	}
	meta, ok := metadataStore["field_bs"]
	if !ok {
		t.Errorf("metadataStore does not contain field_bs")
	}
	if meta.FieldDefault != "foobar" {
		t.Errorf("Expected Example to be 'foobar', got '%s'", meta.FieldDefault)
	}
}
func TestPreprocessTemplate_BasicPlaceholder(t *testing.T) {
	input := `variant:
  credential_format: iso_mdl
  client_id_scheme: did
  request_method: request_uri_signed
  response_mode: direct_post.jwt
form:
  alias: |>
    {{
      credimi ` + "`" + `
        {
          "credimi_id": "iso_mdl_did_request_uri_signed_direct_post_jwt_oid_alias",
          "field_id": "testalias",
          "field_label": "i18n_testalias",
          "field_description": "i18n_testalias_description",
          "field_default_value": "uuidv4",
          "field_type": "string",
          "field_options": []
        }
    ` + "`" + `}}
  server:
    authorization_endpoint: openid-vc://`

	fmt.Printf("Input: %s\n", input)
	
	expected := `variant:
  credential_format: iso_mdl
  client_id_scheme: did
  request_method: request_uri_signed
  response_mode: direct_post.jwt
form:
  alias: |>
    {{ .testalias }}
  server:
    authorization_endpoint: openid-vc://`
	result, err := preprocessTemplate(input)
	if err != nil {
		t.Fatalf("preprocessTemplate returned error: %v", err)
	}
	if result != expected {
		t.Errorf("preprocessTemplate() = %q, want %q\n--- Got ---\n%s\n--- Want ---\n%s", result, expected, result, expected)
	}
}

func TestPreprocessTemplate_InvalidJSON(t *testing.T) {
	input := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_description\":\"desc1\",\"field_default_value\":\"example1\",\"field_type\":\"string\",\"field_options\":[]}" }}`
	expected := "{{ .field1 }}"
	result, err := preprocessTemplate(input)
	if err != nil {
		t.Fatalf("preprocessTemplate returned error: %v", err)
	}
	if result != expected {
		t.Errorf("preprocessTemplate() = %q, want %q", result, expected)
	}
}

func TestPreprocessTemplate_MultiplePlaceholders(t *testing.T) {
	input := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_description\":\"desc1\",\"field_default_value\":\"example1\",\"field_type\":\"string\",\"field_options\":[]}" }} {{ credimi "{\"credimi_id\":\"id2\",\"field_id\":\"field2\",\"field_label\":\"label2\",\"field_description\":\"desc2\",\"field_default_value\":\"example2\",\"field_type\":\"string\",\"field_options\":[]}" }}`
	expected := "{{ .field1 }} {{ .field2 }}"
	result, err := preprocessTemplate(input)
	if err != nil {
		t.Fatalf("preprocessTemplate returned error: %v", err)
	}
	if result != expected {
		t.Errorf("preprocessTemplate() = %q, want %q", result, expected)
	}
}

func TestPreprocessTemplate_WhitespaceInput(t *testing.T) {
	input := `   {{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_description\":\"desc1\",\"field_default_value\":\"example1\",\"field_type\":\"string\",\"field_options\":[]}" }}   `
	expected := "   {{ .field1 }}   "
	result, err := preprocessTemplate(input)
	if err != nil {
		t.Fatalf("preprocessTemplate returned error: %v", err)
	}
	if result != expected {
		t.Errorf("preprocessTemplate() = %q, want %q", result, expected)
	}
}


func TestPreprocessTemplate_EmptyInput(t *testing.T) {
	input := ""
	expected := ""
	result, err := preprocessTemplate(input)
	if err != nil {
		t.Fatalf("preprocessTemplate returned error: %v", err)
	}
	if result != expected {
		t.Errorf("preprocessTemplate() = %q, want %q", result, expected)
	}
}
func TestGetPlaceholders_SingleTemplateSinglePlaceholder(t *testing.T) {
	templateStr := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_description\":\"desc1\",\"field_default_value\":\"example1\",\"field_type\":\"string\",\"field_options\":[\"a\",\"b\"]}" }}`
	reader := strings.NewReader(templateStr)
	names := []string{"template1"}

	result, err := GetPlaceholders([]io.Reader{reader}, names)
	if err != nil {
		t.Fatalf("GetPlaceholders returned error: %v", err)
	}

	// Check normalized_fields is empty (no shared credimi_id)
	norm, ok := result["normalized_fields"].([]map[string]interface{})
	if !ok {
		t.Fatalf("normalized_fields missing or wrong type")
	}
	if len(norm) != 0 {
		t.Errorf("Expected 0 normalized_fields, got %d", len(norm))
	}

	// Check specific_fields
	spec, ok := result["specific_fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("specific_fields missing or wrong type")
	}
	entry, ok := spec["template1"].(map[string]interface{})
	if !ok {
		t.Fatalf("specific_fields[template1] missing or wrong type")
	}
	fields, ok := entry["fields"].([]PlaceholderMetadata)
	if !ok {
		t.Fatalf("fields missing or wrong type")
	}
	if len(fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(fields))
	}
	ph := fields[0]
	if ph.CredimiID != "id1" || ph.FieldID != "field1" || ph.FieldLabel != "label1" ||
		ph.FieldDesc != "desc1" || ph.FieldType != "string" || ph.FieldDefault != "example1" ||
		len(ph.FieldOptions) != 2 || ph.FieldOptions[0] != "a" || ph.FieldOptions[1] != "b" {
		t.Errorf("Unexpected placeholder metadata: %+v", ph)
	}
}

func TestGetPlaceholders_MultipleTemplatesWithSharedCredimiID(t *testing.T) {
	// Both templates use the same credimi_id
	template1 := `{{ credimi "{\"credimi_id\":\"shared\",\"field_id\":\"fieldA\",\"field_label\":\"labelA\",\"field_description\":\"descA\",\"field_default_value\":\"exA\",\"field_type\":\"string\",\"field_options\":[]}" }}`
	template2 := `{{ credimi "{\"credimi_id\":\"shared\",\"field_id\":\"fieldB\",\"field_label\":\"labelB\",\"field_description\":\"descB\",\"field_default_value\":\"exB\",\"field_type\":\"string\",\"field_options\":[]}" }}`
	readers := []io.Reader{strings.NewReader(template1), strings.NewReader(template2)}
	names := []string{"tmpl1", "tmpl2"}

	result, err := GetPlaceholders(readers, names)
	if err != nil {
		t.Fatalf("GetPlaceholders returned error: %v", err)
	}

	// Should have one normalized field for the shared credimi_id
	norm, ok := result["normalized_fields"].([]map[string]interface{})
	if !ok {
		t.Fatalf("normalized_fields missing or wrong type")
	}
	if len(norm) != 1 {
		t.Errorf("Expected 1 normalized_field, got %d", len(norm))
	}
	nf := norm[0]
	if nf["CredimiID"] != "shared" {
		t.Errorf("Expected CredimiID 'shared', got %v", nf["CredimiID"])
	}

	// Each template should have its own field
	spec, ok := result["specific_fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("specific_fields missing or wrong type")
	}
	for _, name := range names {
		entry, ok := spec[name].(map[string]interface{})
		if !ok {
			t.Fatalf("specific_fields[%s] missing or wrong type", name)
		}
		fields, ok := entry["fields"].([]PlaceholderMetadata)
		if !ok {
			t.Fatalf("fields missing or wrong type for %s", name)
		}
		if len(fields) != 1 {
			t.Fatalf("Expected 1 field for %s, got %d", name, len(fields))
		}
	}
}

func TestGetPlaceholders_MultipleTemplatesNoSharedCredimiID(t *testing.T) {
	template1 := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_description\":\"desc1\",\"field_default_value\":\"ex1\",\"field_type\":\"string\",\"field_options\":[]}" }}`
	template2 := `{{ credimi "{\"credimi_id\":\"id2\",\"field_id\":\"field2\",\"field_label\":\"label2\",\"field_description\":\"desc2\",\"field_default_value\":\"ex2\",\"field_type\":\"string\",\"field_options\":[]}" }}`
	readers := []io.Reader{strings.NewReader(template1), strings.NewReader(template2)}
	names := []string{"tmpl1", "tmpl2"}

	result, err := GetPlaceholders(readers, names)
	if err != nil {
		t.Fatalf("GetPlaceholders returned error: %v", err)
	}

	norm, ok := result["normalized_fields"].([]map[string]interface{})
	if !ok {
		t.Fatalf("normalized_fields missing or wrong type")
	}
	if len(norm) != 0 {
		t.Errorf("Expected 0 normalized_fields, got %d", len(norm))
	}

	spec, ok := result["specific_fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("specific_fields missing or wrong type")
	}
	for _, name := range names {
		entry, ok := spec[name].(map[string]interface{})
		if !ok {
			t.Fatalf("specific_fields[%s] missing or wrong type", name)
		}
		fields, ok := entry["fields"].([]PlaceholderMetadata)
		if !ok {
			t.Fatalf("fields missing or wrong type for %s", name)
		}
		if len(fields) != 1 {
			t.Fatalf("Expected 1 field for %s, got %d", name, len(fields))
		}
	}
}

func TestGetPlaceholders_ErrorOnInvalidTemplate(t *testing.T) {
	// Invalid JSON in placeholder
	templateStr := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\"" }}`
	reader := strings.NewReader(templateStr)
	names := []string{"badtemplate"}

	_, err := GetPlaceholders([]io.Reader{reader}, names)
	if err == nil {
		t.Fatalf("Expected error for invalid template, got nil")
	}
}

func TestGetPlaceholders_EmptyInput(t *testing.T) {
	readers := []io.Reader{}
	names := []string{}
	result, err := GetPlaceholders(readers, names)
	if err != nil {
		t.Fatalf("GetPlaceholders returned error: %v", err)
	}
	norm, ok := result["normalized_fields"].([]map[string]interface{})
	if !ok {
		t.Fatalf("normalized_fields missing or wrong type")
	}
	if len(norm) != 0 {
		t.Errorf("Expected 0 normalized_fields, got %d", len(norm))
	}
	spec, ok := result["specific_fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("specific_fields missing or wrong type")
	}
	if len(spec) != 0 {
		t.Errorf("Expected 0 specific_fields, got %d", len(spec))
	}
}
