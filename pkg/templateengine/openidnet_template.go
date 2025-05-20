// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package templateengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VariantOrder   []string                        `json:"variant_order" yaml:"variant_order"`
	VariantKeys    map[string][]string             `json:"variant_keys" yaml:"variant_keys"`
	OptionalFields map[string]map[string]FieldRule `json:"optional_fields" yaml:"optional_fields"`
}

type FieldRule struct {
	Values   map[string][]string `json:"values" yaml:"values"`
	Template string              `json:"template" yaml:"template"`
}

func LoadYAML(filename string, v any) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("failed to decode %s: %w", filename, err)
	}
	return nil
}

func ValidateVariant(variant map[string]string, config Config) error {
	for key, allowedValues := range config.VariantKeys {
		value, exists := variant[key]
		if !exists {
			return fmt.Errorf("missing key '%s' in variant", key)
		}
		valid := false
		for _, allowed := range allowedValues {
			if value == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value '%s' for key '%s'", value, key)
		}
	}
	return nil
}

func ParseInput(input, defaultFile, configFile string) ([]byte, error) {
	var config Config
	if err := LoadYAML(configFile, &config); err != nil {
		return nil, err
	}

	expectedKeys := make([]string, 0, len(config.VariantKeys))
	for key := range config.VariantKeys {
		expectedKeys = append(expectedKeys, key)
	}

	parts := strings.Split(input, ":")
	if len(parts) != len(expectedKeys) {
		return nil, fmt.Errorf("expected %d parts in variant input, got %d", len(expectedKeys), len(parts))
	}

	variant := make(map[string]string)
	for i, key := range config.VariantOrder {
		variant[key] = parts[i]
	}

	var rootNode yaml.Node
	if err := LoadYAML(defaultFile, &rootNode); err != nil {
		return nil, err
	}
	if len(rootNode.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}

	if err := ValidateVariant(variant, config); err != nil {
		return nil, err
	}

	formNode := findMapKey(rootNode.Content[0], "form")
	if formNode == nil {
		return nil, fmt.Errorf("missing 'form' key in default YAML")
	}

	aliasNode := findMapKey(formNode, "alias")
	if aliasNode == nil {
		return nil, fmt.Errorf("missing 'alias' key in form")
	}
	aliasJSON, err := extractCredimiJSON(aliasNode.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template: %w", err)
	}
	variantParts := make([]string, len(config.VariantOrder))
	for i, key := range config.VariantOrder {
		variantParts[i] = strings.ReplaceAll(variant[key], ".", "_")
	}
	variantPrefix := strings.Join(variantParts, "_")
	variantPrefix = strings.ReplaceAll(variantPrefix, ".", "_")
	aliasJSON["credimi_id"] = fmt.Sprintf("%s_%s", variantPrefix, aliasJSON["credimi_id"])
	updatedTemplate, err := generateCredimiTemplate(aliasJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template: %w", err)
	}
	aliasUpdatedNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: updatedTemplate,
		Style: yaml.FoldedStyle,
	}
	setMapKey(formNode, "alias", aliasUpdatedNode)

	sectionNames := make([]string, 0, len(config.OptionalFields))
	for name := range config.OptionalFields {
		sectionNames = append(sectionNames, name)
	}
	sort.Strings(sectionNames)

	for _, sectionName := range sectionNames {
		fields := config.OptionalFields[sectionName]
		sectionNode := findMapKey(formNode, sectionName)
		if sectionNode == nil {

			sectionNode = &yaml.Node{
				Kind:    yaml.MappingNode,
				Tag:     "!!map",
				Content: []*yaml.Node{},
			}
			setMapKey(formNode, sectionName, sectionNode)

		}
		fieldKeys := make([]string, 0, len(fields))
		for k := range fields {
			fieldKeys = append(fieldKeys, k)
		}
		sort.Strings(fieldKeys)

		for _, field := range fieldKeys {
			rule := fields[field]
			for param, allowedValues := range rule.Values {
				if value, exists := variant[param]; exists {
					for _, allowed := range allowedValues {
						if value == allowed {
							templateNode := &yaml.Node{
								Kind:  yaml.ScalarNode,
								Tag:   "!!str",
								Value: rule.Template,
								Style: yaml.FoldedStyle,
							}
							setMapKey(sectionNode, field, templateNode)
							break
						}
					}
				}
			}
		}
	}

	variantNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{},
	}
	for _, k := range config.VariantOrder {
		v := variant[k]
		variantNode.Content = append(variantNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v},
		)
	}

	topLevel := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "variant"},
			variantNode,
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "form"},
			formNode,
		},
	}

	finalDoc := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{topLevel},
	}

	yamlBytes, err := yaml.Marshal(finalDoc)
	if err != nil {
		return nil, err
	}
	return yamlBytes, nil
}

func findMapKey(mapNode *yaml.Node, key string) *yaml.Node {
	if mapNode.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(mapNode.Content); i += 2 {
		k := mapNode.Content[i]
		v := mapNode.Content[i+1]
		if k.Value == key {
			return v
		}
	}
	return nil
}

func setMapKey(mapNode *yaml.Node, key string, valueNode *yaml.Node) {
	if mapNode.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(mapNode.Content); i += 2 {
		k := mapNode.Content[i]
		if k.Value == key {
			mapNode.Content[i+1] = valueNode
			return
		}
	}

	mapNode.Content = append(mapNode.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: key,
	}, valueNode)
}

func extractCredimiJSON(template string) (map[string]any, error) {
	re := regexp.MustCompile(`(?s)credimi\s+\\\"(.*?)\\\"`)
	matches := re.FindStringSubmatch(template)
	if len(matches) < 2 {
		fmt.Println("matches", matches)
		return nil, errors.New("could not find embedded escaped JSON")

	}
	// Unmarshal
	var result map[string]any
	if err := json.Unmarshal([]byte(matches[1]), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return result, nil
}

func generateCredimiTemplate(data map[string]any) (string, error) {
	var b strings.Builder

	// Write opening lines
	b.WriteString("{{\n   credimi \\\"\n")
	b.WriteString("      {\n")

	// Collect and sort keys for stable output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write each field with exact indentation (6 spaces)
	for i, key := range keys {
		valueBytes, err := json.Marshal(data[key])
		if err != nil {
			return "", err
		}
		b.WriteString(fmt.Sprintf("        \"%s\": %s", key, string(valueBytes)))
		if i != len(keys)-1 {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}

	// Closing lines
	b.WriteString("      }\n")
	b.WriteString("\\\"}}\n")

	return b.String(), nil
}
