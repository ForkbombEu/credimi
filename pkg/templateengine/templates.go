// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package templateengine provides functionality for rendering templates
package templateengine

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

// PlaceholderMetadata holds metadata for a placeholder in a template.
// It includes the field name, Credimi ID, label key, description key,
// type, and example value.
type PlaceholderMetadata struct {
	FieldName      string
	CredimiID      string
	LabelKey       string
	DescriptionKey string
	Type           string
	Example        string
}

// func RemoveNewlinesAndBackslashes(input string) string {
// 	output := strings.ReplaceAll(input, "\n", "")
// 	output = strings.ReplaceAll(output, "\\", "")
// 	output = strings.ReplaceAll(output, "\"", "'")
// 	return output
// }

// RenderTemplate takes a reader containing a template and a data map,
// renders the template using the provided data, and returns the resulting string.
// It also preprocesses the template to handle custom placeholders and
// metadata extraction.
func RenderTemplate(reader io.Reader, data map[string]interface{}) (string, error) {
	handler := sprout.New(
		sprout.WithGroups(all.RegistryGroup()),
	)
	funcs := handler.Build()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return "", err
	}

	templateContent := buf.String()

	processedContent, _ := preprocessTemplate(templateContent)

	tmpl, err := template.New("tmpl").Funcs(funcs).Parse(processedContent)
	if err != nil {
		return "", err
	}

	buf.Reset()
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	result := buf.String()

	result = strings.ReplaceAll(result, "\"{", "{")
	result = strings.ReplaceAll(result, "}\"", "}")

	return result, nil
}

// GetPlaceholders processes a list of template readers and their corresponding names
// to extract placeholder metadata and organize it into normalized and specific fields.
//
// Parameters:
//   - readers: A slice of io.Reader objects, each representing a template to be processed.
//   - names: A slice of strings representing the names corresponding to each reader.
//
// Returns:
//   - A map containing two keys:
//   - "normalized_fields": A slice of maps, each representing a placeholder that appears
//     in multiple templates. Each map contains the following keys:
//   - "CredimiID": The unique identifier of the placeholder.
//   - "Type": The type of the placeholder.
//   - "DescriptionKey": The description key associated with the placeholder.
//   - "LabelKey": The label key associated with the placeholder.
//   - "Example": An example value for the placeholder.
//   - "specific_fields": A map where each key is a name from the `names` slice, and the
//     value is a map containing:
//   - "content": The preprocessed content of the corresponding template.
//   - "fields": A slice of placeholder metadata extracted from the template.
//   - An error if any issues occur during processing.
//
// Notes:
//   - The function ensures that placeholders with the same CredimiID across multiple templates
//     are grouped together in the "normalized_fields".
//   - The `metadataStore` is cleared for each reader to avoid mixing placeholder metadata
//     between templates.
func GetPlaceholders(readers []io.Reader, names []string) (map[string]interface{}, error) {
	var allPlaceholders []PlaceholderMetadata
	specificFields := make(map[string]interface{})
	credimiIDCount := make(map[string]int)

	for i, r := range readers {
		// Clear metadataStore for each reader to avoid mixing placeholders
		metadataStore = make(map[string]PlaceholderMetadata)

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			return nil, err
		}
		content := buf.String()

		preprocessedContent, err := preprocessTemplate(content)
		if err != nil {
			return nil, err
		}

		placeholders := extractMetadata()

		for _, ph := range placeholders {
			credimiIDCount[ph.CredimiID]++
			allPlaceholders = append(allPlaceholders, ph)
		}

		specificFields[names[i]] = map[string]interface{}{
			"content": preprocessedContent,
			"fields":  placeholders,
		}
	}

	normalizedFields := make([]map[string]interface{}, 0)
	seenCredimiIDs := make(map[string]bool)
	for _, ph := range allPlaceholders {
		if credimiIDCount[ph.CredimiID] > 1 && !seenCredimiIDs[ph.CredimiID] {
			seenCredimiIDs[ph.CredimiID] = true
			field := map[string]interface{}{
				"CredimiID":      ph.CredimiID,
				"Type":           ph.Type,
				"DescriptionKey": ph.DescriptionKey,
				"LabelKey":       ph.LabelKey,
				"Example":        ph.Example,
			}
			normalizedFields = append(normalizedFields, field)
		}
	}

	result := map[string]interface{}{
		"normalized_fields": normalizedFields,
		"specific_fields":   specificFields,
	}

	return result, nil
}

var metadataStore = make(map[string]PlaceholderMetadata)

func credimiPlaceholder(fieldName, credimiID, labelKey, descriptionKey, fieldType, example string) (string, error) {
	metadataStore[fieldName] = PlaceholderMetadata{
		FieldName:      fieldName,
		CredimiID:      credimiID,
		LabelKey:       labelKey,
		DescriptionKey: descriptionKey,
		Type:           fieldType,
		Example:        strings.ReplaceAll(example, "\\\\\\\\", ""),
	}
	return fmt.Sprintf("{{ .%s }}", fieldName), nil
}

func preprocessTemplate(content string) (string, error) {
	handler := sprout.New(
		sprout.WithGroups(all.RegistryGroup()),
	)
	funcs := handler.Build()

	funcs["credimiPlaceholder"] = credimiPlaceholder

	tmpl, err := template.New("preprocess").Funcs(funcs).Parse(content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func extractMetadata() []PlaceholderMetadata {
	var extracted []PlaceholderMetadata
	for _, meta := range metadataStore {
		extracted = append(extracted, meta)
	}
	return extracted
}
