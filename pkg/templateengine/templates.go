// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package templateengine provides functionality for rendering templates
package templateengine

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

type TypeOfFieldType string

const (
	TypeOfFieldTypeString  TypeOfFieldType = "string"
	TypeOfFieldTypeObject  TypeOfFieldType = "object"
	TypeOfFieldTypeOptions TypeOfFieldType = "options"
)

// PlaceholderMetadata holds metadata for a placeholder in a template.
// It matches the placeholderInput struct.
type PlaceholderMetadata struct {
	CredimiID    string          `json:"credimi_id"`
	FieldID      string          `json:"field_id"`
	FieldLabel   string          `json:"field_label"`
	FieldDesc    string          `json:"field_description"`
	FieldDefault interface{}     `json:"field_default_value"`
	FieldType    TypeOfFieldType `json:"field_type"`
	FieldOptions []string        `json:"field_options"`
}

// GetDefaultValue returns the default value of the placeholder,
// if the original value is a string return a string, if is an object return a json string
func (p *PlaceholderMetadata) GetDefaultValue() string {
	if p.FieldDefault == nil {
		return ""
	}

	switch v := p.FieldDefault.(type) {
	case string:
		return v
	case map[string]interface{}:
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes)
	default:
		return fmt.Sprintf("%v", v)
	}
}

var metadataStore = make(map[string]PlaceholderMetadata)

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
func GetPlaceholders(readers []io.Reader, names []string) (map[string]interface{}, error) {
	specificFields, allPlaceholders, err := getFields(readers, names)
	if err != nil {
		return nil, err
	}
	normalizedFields := normalizeFields(allPlaceholders)
	result := map[string]interface{}{
		"normalized_fields": normalizedFields,
		"specific_fields":   specificFields,
	}
	return result, nil
}

func getFields(
	readers []io.Reader,
	names []string,
) (map[string]interface{}, []PlaceholderMetadata, error) {
	specificFields := make(map[string]interface{})
	var allPlaceholders []PlaceholderMetadata

	for i, r := range readers {
		metadataStore = make(map[string]PlaceholderMetadata)

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			return nil, nil, err
		}
		content := buf.String()

		preprocessedContent, err := preprocessTemplate(content)
		if err != nil {
			return nil, nil, err
		}

		placeholders := extractMetadata()

		allPlaceholders = append(allPlaceholders, placeholders...)

		specificFields[names[i]] = map[string]interface{}{
			"content": preprocessedContent,
			"fields":  placeholders,
		}
	}

	return sortSpecificFields(specificFields), allPlaceholders, nil
}

func sortSpecificFields(fields map[string]interface{}) map[string]interface{} {
	for name, v := range fields {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		phs, ok := m["fields"].([]PlaceholderMetadata)
		if !ok {
			continue
		}
		stringPH := make([]PlaceholderMetadata, 0)
		otherPH := make([]PlaceholderMetadata, 0)
		for _, ph := range phs {
			if ph.FieldType == TypeOfFieldTypeString {
				stringPH = append(stringPH, ph)
			} else {
				otherPH = append(otherPH, ph)
			}
		}
		stringPH = append(stringPH, otherPH...)
		m["fields"] = stringPH
		fields[name] = m
	}
	return fields
}

func normalizeFields(fields []PlaceholderMetadata) []map[string]interface{} {
	credimiIDCount := make(map[string]int)
	normalized := make([]map[string]interface{}, 0, len(fields))
	stringFields := make([]map[string]interface{}, 0)
	otherFields := make([]map[string]interface{}, 0)
	for _, ph := range fields {
		if credimiIDCount[ph.CredimiID] == 1 {
			field := map[string]interface{}{
				"credimi_id":          ph.CredimiID,
				"field_label":         ph.FieldLabel,
				"field_description":   ph.FieldDesc,
				"field_default_value": ph.GetDefaultValue(),
				"field_type":          ph.FieldType,
				"field_options":       ph.FieldOptions,
			}

			if ph.FieldType == TypeOfFieldTypeString {
				stringFields = append(stringFields, field)
			} else {
				otherFields = append(otherFields, field)
			}
		}
		credimiIDCount[ph.CredimiID]++
	}
	normalized = append(normalized, stringFields...)
	normalized = append(normalized, otherFields...)

	return normalized
}

func credimi(jsonStr string, args ...interface{}) (string, error) {
	jsonStr = strings.TrimSpace(jsonStr)

	var input PlaceholderMetadata
	err := json.Unmarshal([]byte(jsonStr), &input)
	if err != nil {
		return "", fmt.Errorf("failed to parse placeholder JSON: %w", err)
	}

	if len(args) > 0 {
		input.FieldDefault = args[0]
	}

	metadataStore[input.FieldID] = input

	return fmt.Sprintf("{{ .%s }}", input.FieldID), nil
}

func jwk(alg string) (string, error) {
	var crv elliptic.Curve
	var crvName string

	switch alg {
	case "ES256":
		crv = elliptic.P256()
		crvName = "P-256"
	case "ES384":
		crv = elliptic.P384()
		crvName = "P-384"
	case "ES512":
		crv = elliptic.P521()
		crvName = "P-521"
	default:
		return "", fmt.Errorf(
			"unsupported algorithm: %s. Supported algorithms are: ES256, ES384, ES512",
			alg,
		)
	}

	priv, err := ecdsa.GenerateKey(crv, rand.Reader)
	if err != nil {
		return "", err
	}

	b64 := func(b []byte) string {
		return base64.RawURLEncoding.EncodeToString(b)
	}

	padBytes := func(b []byte, size int) []byte {
		if len(b) >= size {
			return b
		}
		padded := make([]byte, size)
		copy(padded[size-len(b):], b)
		return padded
	}

	size := (priv.Curve.Params().BitSize + 7) / 8

	jwkKey := map[string]interface{}{
		"kty": "EC",
		"alg": alg,
		"crv": crvName,
		"d":   b64(padBytes(priv.D.Bytes(), size)),
		"x":   b64(padBytes(priv.X.Bytes(), size)),
		"y":   b64(padBytes(priv.Y.Bytes(), size)),
	}

	jwkSet := map[string]interface{}{
		"keys": []interface{}{jwkKey},
	}

	jwkJSON, err := json.Marshal(jwkSet)
	if err != nil {
		return "", err
	}
	return string(jwkJSON), nil
}

func preprocessTemplate(content string) (string, error) {
	handler := sprout.New(
		sprout.WithGroups(all.RegistryGroup()),
	)
	funcs := handler.Build()

	funcs["credimi"] = credimi
	funcs["jwk"] = jwk

	tmpl, err := template.New("preprocess").Funcs(funcs).Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func extractMetadata() []PlaceholderMetadata {
	extracted := make([]PlaceholderMetadata, 0, len(metadataStore))
	for _, meta := range metadataStore {
		extracted = append(extracted, meta)
	}
	return extracted
}
