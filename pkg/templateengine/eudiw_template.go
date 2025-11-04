// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package templateengine

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseEudiwInput parses a YAML template file and updates both "id" and "nonce"
// fields using the provided input as a prefix for the credimi_id values.
// It mirrors the logic of ParseEwcInput but operates on two fields.
func ParseEudiwInput(input, defaultFile string) ([]byte, error) {
	var rootNode yaml.Node
	if err := LoadYAML(defaultFile, &rootNode); err != nil {
		return nil, err
	}
	if len(rootNode.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}

	idNode := findMapKey(rootNode.Content[0], "id")
	if idNode == nil {
		return nil, fmt.Errorf("missing 'id' key in default YAML")
	}

	nonceNode := findMapKey(rootNode.Content[0], "nonce")
	if nonceNode == nil {
		return nil, fmt.Errorf("missing 'nonce' key in default YAML")
	}

	idJSON, afterIdContent, err := extractCredimiJSON(idNode.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Credimi JSON from 'id': %w", err)
	}

	nonceJSON, afterNonceContent, err := extractCredimiJSON(nonceNode.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Credimi JSON from 'nonce': %w", err)
	}

	prefix := strings.ReplaceAll(input, "+", "_")
	prefix = strings.ReplaceAll(prefix, "-", "_")

	idJSON["credimi_id"] = fmt.Sprintf("%s_%s", prefix, idJSON["credimi_id"])
	nonceJSON["credimi_id"] = fmt.Sprintf("%s_%s", prefix, nonceJSON["credimi_id"])

	updatedIdTemplate, err := generateCredimiTemplate(idJSON, afterIdContent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate 'id' template: %w", err)
	}

	updatedNonceTemplate, err := generateCredimiTemplate(nonceJSON, afterNonceContent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate 'nonce' template: %w", err)
	}

	idUpdatedNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: updatedIdTemplate,
		Style: yaml.FoldedStyle,
	}
	setMapKey(rootNode.Content[0], "id", idUpdatedNode)

	nonceUpdatedNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: updatedNonceTemplate,
		Style: yaml.FoldedStyle,
	}
	setMapKey(rootNode.Content[0], "nonce", nonceUpdatedNode)

	yamlBytes, err := yaml.Marshal(&rootNode)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	return yamlBytes, nil
}
