// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseWorkflow parses YAML string into Workflow struct
func ParseWorkflow(yamlStr string) (*WorkflowDefinition, error) {
	var wf WorkflowDefinition
	if err := yaml.Unmarshal([]byte(yamlStr), &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow yaml: %w", err)
	}
	return &wf, nil
}

func (f *FinallyDefinition) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var steps []FinallyStepDefinition
		if err := value.Decode(&steps); err != nil {
			return fmt.Errorf("invalid finally steps: %w", err)
		}
		f.Always = steps
		f.OnSuccess = nil
		f.OnFailure = nil
		return nil
	case yaml.MappingNode:
		type finallyDefinition FinallyDefinition
		var tmp finallyDefinition
		if err := value.Decode(&tmp); err != nil {
			return fmt.Errorf("invalid finally definition: %w", err)
		}
		*f = FinallyDefinition(tmp)
		return nil
	case yaml.ScalarNode:
		if value.Tag == "!!null" {
			return nil
		}
	}

	return fmt.Errorf(
		"invalid finally definition: expected list or map, got YAML node kind %d",
		value.Kind,
	)
}

func (f *FinallyDefinition) UnmarshalJSON(data []byte) error {
	var steps []FinallyStepDefinition
	if err := json.Unmarshal(data, &steps); err == nil {
		f.Always = steps
		f.OnSuccess = nil
		f.OnFailure = nil
		return nil
	}

	type finallyDefinition FinallyDefinition
	var tmp finallyDefinition
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("invalid finally definition: %w", err)
	}
	*f = FinallyDefinition(tmp)
	return nil
}

// copyNode returns a deep copy of n whose Alias pointers are rewired to the
// copies, so the result is a self-contained graph. Step inputs are decoded from
// a copy because retagging timestamps mutates nodes: an anchored scalar inside
// `with` may also be aliased from elsewhere in the document, where it must keep
// yaml.v3's own resolution.
func copyNode(n *yaml.Node, copies map[*yaml.Node]*yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	if copied, ok := copies[n]; ok {
		return copied
	}
	copied := *n
	copies[n] = &copied
	copied.Alias = copyNode(n.Alias, copies)
	if n.Content != nil {
		copied.Content = make([]*yaml.Node, len(n.Content))
		for i, child := range n.Content {
			copied.Content[i] = copyNode(child, copies)
		}
	}
	return &copied
}

// keepTimestampScalarsAsStrings retags YAML `!!timestamp` scalars as strings so
// their literal text survives decoding. Step inputs are forwarded to activities
// as JSON, where a date-like scalar has no timestamp type: decoding it to
// time.Time would silently rewrite the author's `2012-03-04` into
// `2012-03-04T00:00:00Z` the moment the body is marshalled. It mutates the
// nodes it is given, so callers must pass a copy (see copyNode).
func keepTimestampScalarsAsStrings(node *yaml.Node) {
	seen := map[*yaml.Node]struct{}{}
	var walk func(*yaml.Node)
	walk = func(n *yaml.Node) {
		if n == nil {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		if n.Kind == yaml.ScalarNode && n.Tag == "!!timestamp" {
			n.Tag = "!!str"
		}
		walk(n.Alias)
		for _, child := range n.Content {
			walk(child)
		}
	}
	walk(node)
}

func (s *StepInputs) UnmarshalYAML(value *yaml.Node) error {
	// The retag mutates the tree, so decode a copy: anchors declared inside
	// `with` may be aliased elsewhere in the document.
	inputs := copyNode(value, map[*yaml.Node]*yaml.Node{})
	keepTimestampScalarsAsStrings(inputs)

	var tmp map[string]any
	if err := inputs.Decode(&tmp); err != nil {
		return fmt.Errorf("invalid step inputs: %w", err)
	}

	s.Config = make(map[string]any)
	payload := make(map[string]any)

	if val, ok := tmp["config"]; ok {
		cfgMap, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid config section: expected map, got %T", val)
		}
		s.Config = cfgMap
	}

	if val, ok := tmp["payload"]; ok {
		if val != nil {
			payloadMap, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid payload section: expected map, got %T", val)
			}
			payload = payloadMap
		}
	}

	for key, val := range tmp {
		if key == "config" || key == "payload" {
			continue
		}
		payload[key] = val
	}

	s.Payload = payload
	return nil
}

func (s *StepInputs) UnmarshalJSON(data []byte) error {
	var tmp map[string]any
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("invalid step inputs: %w", err)
	}
	if tmp == nil {
		tmp = map[string]any{}
	}

	s.Config = map[string]any{}
	s.Payload = map[string]any{}

	if val, ok := tmp["config"]; ok {
		if val != nil {
			cfgMap, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid config section: expected map, got %T", val)
			}
			s.Config = cfgMap
		}
		delete(tmp, "config")
	}

	if val, ok := tmp["payload"]; ok {
		if val != nil {
			payloadMap, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid payload section: expected map, got %T", val)
			}
			s.Payload = payloadMap
		}
		delete(tmp, "payload")
	}

	for key, val := range tmp {
		s.Payload[key] = val
	}

	return nil
}
