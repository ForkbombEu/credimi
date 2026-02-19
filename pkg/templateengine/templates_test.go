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
	result, err := credimi("\n  " + jsonInput + "  \n")
	if err != nil {
		t.Errorf("credimi() returned error: %v", err)
	}
	expected := "{{ .field_trim }}"
	if result != expected {
		t.Errorf("credimi() = %v, want %v", result, expected)
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
		t.Errorf(
			"preprocessTemplate() = %q, want %q\n--- Got ---\n%s\n--- Want ---\n%s",
			result,
			expected,
			result,
			expected,
		)
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

func TestPlaceholderMetadataGetDefaultValue(t *testing.T) {
	meta := PlaceholderMetadata{FieldDefault: "foo"}
	if got := meta.GetDefaultValue(); got != "foo" {
		t.Errorf("GetDefaultValue() = %q, want %q", got, "foo")
	}

	meta = PlaceholderMetadata{FieldDefault: map[string]any{"a": "b"}}
	if got := meta.GetDefaultValue(); got == "" || got == "{}" {
		t.Errorf("GetDefaultValue() = %q, want non-empty JSON", got)
	}

	meta = PlaceholderMetadata{FieldDefault: 123}
	if got := meta.GetDefaultValue(); got != "123" {
		t.Errorf("GetDefaultValue() = %q, want %q", got, "123")
	}
}

func TestRenderTemplateBasic(t *testing.T) {
	templateStr := `{"name":"{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"name\",\"field_label\":\"label\",\"field_description\":\"desc\",\"field_default_value\":\"\",\"field_type\":\"string\",\"field_options\":[]}" }}"}`
	out, err := RenderTemplate(strings.NewReader(templateStr), map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	expected := `{"name":"Alice"}`
	if out != expected {
		t.Errorf("RenderTemplate() = %q, want %q", out, expected)
	}
}

func TestNormalizeFieldsDedupes(t *testing.T) {
	fields := []PlaceholderMetadata{
		{CredimiID: "id-1", FieldLabel: "A", FieldType: TypeOfFieldTypeString},
		{CredimiID: "id-1", FieldLabel: "B", FieldType: TypeOfFieldTypeObject},
		{CredimiID: "id-2", FieldLabel: "C", FieldType: TypeOfFieldTypeString},
	}
	normalized := normalizeFields(fields)
	if len(normalized) != 1 {
		t.Fatalf("normalizeFields() len = %d, want 1", len(normalized))
	}
	if normalized[0]["credimi_id"] != "id-1" {
		t.Fatalf("normalizeFields() credimi_id = %v, want id-1", normalized[0]["credimi_id"])
	}
}

func TestSortSpecificFieldsStringFirst(t *testing.T) {
	fields := map[string]interface{}{
		"tmpl": map[string]interface{}{
			"fields": []PlaceholderMetadata{
				{FieldType: TypeOfFieldTypeObject, FieldID: "b"},
				{FieldType: TypeOfFieldTypeString, FieldID: "a"},
			},
		},
	}
	sorted := sortSpecificFields(fields)
	entry := sorted["tmpl"].(map[string]interface{})
	phs := entry["fields"].([]PlaceholderMetadata)
	if len(phs) != 2 || phs[0].FieldType != TypeOfFieldTypeString {
		t.Fatalf("sortSpecificFields() did not move string fields first")
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
	if nf["credimi_id"] != "shared" {
		t.Errorf("Expected CredimiID 'shared', got %v", nf["credimi_id"])
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
func TestNormalizeFields_EmptyInput(t *testing.T) {
	fields := []PlaceholderMetadata{}
	result := normalizeFields(fields)
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestNormalizeFields_NoSharedCredimiID(t *testing.T) {
	fields := []PlaceholderMetadata{
		{
			CredimiID:    "id1",
			FieldID:      "field1",
			FieldLabel:   "label1",
			FieldDesc:    "desc1",
			FieldDefault: "def1",
			FieldType:    TypeOfFieldTypeString,
			FieldOptions: []string{"a"},
		},
		{
			CredimiID:    "id2",
			FieldID:      "field2",
			FieldLabel:   "label2",
			FieldDesc:    "desc2",
			FieldDefault: "def2",
			FieldType:    TypeOfFieldTypeOptions,
			FieldOptions: []string{"b"},
		},
	}
	result := normalizeFields(fields)
	if len(result) != 0 {
		t.Errorf("Expected 0 normalized fields, got %d", len(result))
	}
}

func TestNormalizeFields_WithSharedCredimiID(t *testing.T) {
	fields := []PlaceholderMetadata{
		{
			CredimiID:    "shared",
			FieldID:      "field1",
			FieldLabel:   "label1",
			FieldDesc:    "desc1",
			FieldDefault: "def1",
			FieldType:    TypeOfFieldTypeString,
			FieldOptions: []string{"a"},
		},
		{
			CredimiID:    "shared",
			FieldID:      "field2",
			FieldLabel:   "label2",
			FieldDesc:    "desc2",
			FieldDefault: "def2",
			FieldType:    TypeOfFieldTypeString,
			FieldOptions: []string{"b"},
		},
		{
			CredimiID:    "unique",
			FieldID:      "field3",
			FieldLabel:   "label3",
			FieldDesc:    "desc3",
			FieldDefault: "def3",
			FieldType:    TypeOfFieldTypeString,
			FieldOptions: []string{"c"},
		},
	}
	result := normalizeFields(fields)
	if len(result) != 1 {
		t.Fatalf("Expected 2 normalized fields, got %d", len(result))
	}
}

func TestNormalizeFields_MultipleSharedCredimiIDs(t *testing.T) {
	fields := []PlaceholderMetadata{
		{
			CredimiID:    "idA",
			FieldID:      "fieldA1",
			FieldLabel:   "labelA1",
			FieldDesc:    "descA1",
			FieldDefault: "defA1",
			FieldType:    TypeOfFieldTypeString,
			FieldOptions: []string{},
		},
		{
			CredimiID:    "idA",
			FieldID:      "fieldA2",
			FieldLabel:   "labelA2",
			FieldDesc:    "descA2",
			FieldDefault: "defA2",
			FieldType:    TypeOfFieldTypeOptions,
			FieldOptions: []string{},
		},
		{
			CredimiID:    "idB",
			FieldID:      "fieldB1",
			FieldLabel:   "labelB1",
			FieldDesc:    "descB1",
			FieldDefault: "defB1",
			FieldType:    TypeOfFieldTypeString,
			FieldOptions: []string{},
		},
		{
			CredimiID:    "idB",
			FieldID:      "fieldB2",
			FieldLabel:   "labelB2",
			FieldDesc:    "descB2",
			FieldDefault: "defB2",
			FieldType:    TypeOfFieldTypeObject,
			FieldOptions: []string{},
		},
	}
	result := normalizeFields(fields)
	if len(result) != 2 {
		t.Fatalf("Expected 2 normalized fields, got %d", len(result))
	}
	// Should have one string and one other for each shared credimi_id
	countA, countB := 0, 0
	for _, field := range result {
		switch field["credimi_id"] {
		case "idA":
			countA++
		case "idB":
			countB++
		default:
			t.Errorf("Unexpected credimi_id: %v", field["credimi_id"])
		}
	}
	if countA != 1 || countB != 1 {
		t.Errorf("Expected 2 fields for each shared credimi_id, got idA=%d, idB=%d", countA, countB)
	}
}
func TestGetFields_SingleTemplateSinglePlaceholder(t *testing.T) {
	templateStr := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_description\":\"desc1\",\"field_default_value\":\"def1\",\"field_type\":\"string\",\"field_options\":[\"a\",\"b\"]}" }}`
	names := []string{"template_0"}
	reader := strings.NewReader(templateStr)

	specific, all, err := getFields([]io.Reader{reader}, names)
	if err != nil {
		t.Fatalf("getFields returned error: %v", err)
	}

	// Check specificFields
	entry, ok := specific["template_0"].(map[string]interface{})
	if !ok {
		t.Fatalf("specificFields[template_0] missing or wrong type")
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
		ph.FieldDesc != "desc1" || ph.FieldType != "string" || ph.FieldDefault != "def1" ||
		len(ph.FieldOptions) != 2 || ph.FieldOptions[0] != "a" || ph.FieldOptions[1] != "b" {
		t.Errorf("Unexpected placeholder metadata: %+v", ph)
	}

	// Check allPlaceholders
	if len(all) != 1 {
		t.Errorf("Expected 1 allPlaceholder, got %d", len(all))
	}
}

func TestGetFields_MultipleTemplatesMultiplePlaceholders(t *testing.T) {
	template1 := `{{ credimi "{\"credimi_id\":\"id1\",\"field_id\":\"field1\",\"field_label\":\"label1\",\"field_description\":\"desc1\",\"field_default_value\":\"def1\",\"field_type\":\"string\",\"field_options\":[]}" }}`
	template2 := `{{ credimi "{\"credimi_id\":\"id2\",\"field_id\":\"field2\",\"field_label\":\"label2\",\"field_description\":\"desc2\",\"field_default_value\":\"def2\",\"field_type\":\"options\",\"field_options\":[\"x\"]}" }}`
	names := []string{"template_0", "template_1"}
	readers := []io.Reader{strings.NewReader(template1), strings.NewReader(template2)}
	specific, all, err := getFields(readers, names)
	if err != nil {
		t.Fatalf("getFields returned error: %v", err)
	}

	// Check specificFields
	for i := 0; i < 2; i++ {
		entry, ok := specific[fmt.Sprintf("template_%d", i)].(map[string]interface{})
		if !ok {
			t.Fatalf("specificFields[template_%d] missing or wrong type", i)
		}
		fields, ok := entry["fields"].([]PlaceholderMetadata)
		if !ok {
			t.Fatalf("fields missing or wrong type for template_%d", i)
		}
		if len(fields) != 1 {
			t.Fatalf("Expected 1 field for template_%d, got %d", i, len(fields))
		}
	}
	if len(all) != 2 {
		t.Errorf("Expected 2 allPlaceholders, got %d", len(all))
	}
}

func TestGetFields_TemplateWithNoPlaceholders(t *testing.T) {
	templateStr := `This template has no placeholders.`
	names := []string{"template_0"}
	reader := strings.NewReader(templateStr)

	specific, all, err := getFields([]io.Reader{reader}, names)
	if err != nil {
		t.Fatalf("getFields returned error: %v", err)
	}

	entry, ok := specific["template_0"].(map[string]interface{})
	if !ok {
		t.Fatalf("specificFields[template_0] missing or wrong type")
	}
	fields, ok := entry["fields"].([]PlaceholderMetadata)
	if !ok {
		t.Fatalf("fields missing or wrong type")
	}
	if len(fields) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(fields))
	}
	if len(all) != 0 {
		t.Errorf("Expected 0 allPlaceholders, got %d", len(all))
	}
}

func TestGetFields_EmptyInput(t *testing.T) {
	specific, all, err := getFields([]io.Reader{}, nil)
	if err != nil {
		t.Fatalf("getFields returned error: %v", err)
	}
	if len(specific) != 0 {
		t.Errorf("Expected 0 specificFields, got %d", len(specific))
	}
	if len(all) != 0 {
		t.Errorf("Expected 0 allPlaceholders, got %d", len(all))
	}
}
