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

	return fmt.Errorf("invalid finally definition: expected list or map, got YAML node kind %d", value.Kind)
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

func (s *StepInputs) UnmarshalYAML(value *yaml.Node) error {
	var tmp map[string]any
	if err := value.Decode(&tmp); err != nil {
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
