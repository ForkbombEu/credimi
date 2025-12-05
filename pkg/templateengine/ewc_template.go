// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package templateengine

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseEwcInput(input, defaultFile string) ([]byte, error) {
	var rootNode yaml.Node
	if err := LoadYAML(defaultFile, &rootNode); err != nil {
		return nil, err
	}
	if len(rootNode.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}

	sessionIDNode := findMapKey(rootNode.Content[0], "sessionId")
	if sessionIDNode == nil {
		return nil, fmt.Errorf("missing 'sessionId' key in default YAML")
	}

	sessionIDJSON, afterContent, err := extractCredimiJSON(sessionIDNode.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Credimi JSON: %w", err)
	}

	prefix := strings.ReplaceAll(input, "-", "_")
	sessionIDJSON["credimi_id"] = fmt.Sprintf("%s_%s", prefix, sessionIDJSON["credimi_id"])

	updatedTemplate, err := generateCredimiTemplate(sessionIDJSON, afterContent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template: %w", err)
	}

	sessionIDUpdatedNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: updatedTemplate,
		Style: yaml.FoldedStyle,
	}
	setMapKey(rootNode.Content[0], "sessionId", sessionIDUpdatedNode)

	yamlBytes, err := yaml.Marshal(&rootNode)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated YAML: %w", err)
	}
	return yamlBytes, nil
}
