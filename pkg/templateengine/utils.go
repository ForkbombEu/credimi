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

func extractCredimiJSON(template string) (map[string]any, string, error) {
	re := regexp.MustCompile("(?s)credimi\\s+`(.*?)`")
	matches := re.FindStringSubmatch(template)
	if len(matches) < 2 {
		return nil, "", errors.New("could not find embedded JSON in credimi")
	}

	jsonStr := strings.TrimSpace(matches[1])

	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, "", fmt.Errorf("invalid JSON: %w", err)
	}

	reAfter := regexp.MustCompile("(?s)credimi\\s+`.*?`([\\s\\S]*?)\\}\\}")
	afterMatches := reAfter.FindStringSubmatch(template)
	afterContent := ""
	if len(afterMatches) >= 2 {
		afterContent = strings.TrimSpace(afterMatches[1])
	}

	return result, afterContent, nil
}

func generateCredimiTemplate(data map[string]any, afterContent string) (string, error) {
	var b strings.Builder

	b.WriteString("{{\n   credimi `\n")
	b.WriteString("      {\n")

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

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

	b.WriteString("      }\n")
	b.WriteString(fmt.Sprintf("   `\n%s}}", afterContent))

	return b.String(), nil
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
