// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package templateengine

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VariantOrder   []string                        `json:"variant_order"   yaml:"variant_order"`
	VariantKeys    map[string][]string             `json:"variant_keys"    yaml:"variant_keys"`
	OptionalFields map[string]map[string]FieldRule `json:"optional_fields" yaml:"optional_fields"`
}

type FieldRule struct {
	Values   map[string][]string `json:"values"   yaml:"values"`
	Template string              `json:"template" yaml:"template"`
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

func ParseOpenidnetInput(input, defaultFile, configFile string) ([]byte, error) {
	var config Config
	if err := LoadYAML(configFile, &config); err != nil {
		return nil, err
	}

	expectedKeys := make([]string, 0, len(config.VariantKeys))
	for key := range config.VariantKeys {
		expectedKeys = append(expectedKeys, key)
	}

	parts := strings.Split(input, "+")
	if len(parts) != len(expectedKeys) {
		expectedFormat := strings.Join(expectedKeys, "+")
		return nil, fmt.Errorf(
			"expected %d parts in variant input (format: %s), got %d",
			len(expectedKeys),
			expectedFormat,
			len(parts),
		)
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

	testNode := findMapKey(rootNode.Content[0], "test")
	if testNode == nil {
		return nil, fmt.Errorf("missing 'test' key in default YAML")
	}

	aliasNode := findMapKey(formNode, "alias")
	if aliasNode == nil {
		return nil, fmt.Errorf("missing 'alias' key in form")
	}
	aliasJSON, afterContent, err := extractCredimiJSON(aliasNode.Value)
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
	updatedTemplate, err := generateCredimiTemplate(aliasJSON, afterContent)
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
							// Always override existing field if config specifies it
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
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "test"},
			testNode,
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
