// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package template_engine

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

type PlaceholderMetadata struct {
	FieldName      string
	CredimiID      string
	LabelKey       string
	DescriptionKey string
	Type           string
	Example        string
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

func PreprocessTemplate(content string) (string, error) {
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

func ExtractMetadata() []PlaceholderMetadata {
	var extracted []PlaceholderMetadata
	for _, meta := range metadataStore {
		extracted = append(extracted, meta)
	}
	return extracted
}

func RemoveNewlinesAndBackslashes(input string) string {
	output := strings.ReplaceAll(input, "\n", "")
	output = strings.ReplaceAll(output, "\\", "")
	output = strings.ReplaceAll(output, "\"", "'")
	return output
}

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

	processedContent, _ := PreprocessTemplate(templateContent)

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

		preprocessedContent, err := PreprocessTemplate(content)
		if err != nil {
			return nil, err
		}

		placeholders := ExtractMetadata()

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
