// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
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

func (s *StepInputs) UnmarshalYAML(value *yaml.Node) error {
	var tmp map[string]any
	if err := value.Decode(&tmp); err != nil {
		return fmt.Errorf("invalid step inputs: %w", err)
	}

	s.Config = make(map[string]any)
	payload := make(map[string]any)

	for key, val := range tmp {
		if key == "config" {
			cfgMap, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid config section: expected map, got %T", val)
			}
			s.Config = cfgMap
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

func (s *StepDefinition) DecodePayload() (any, error) {
	desc, ok := registry.Registry[s.Use]
	if !ok {
		return nil, fmt.Errorf("unknown step type: %s", s.Use)
	}
	if desc.PayloadType == nil {
		return nil, fmt.Errorf("no input type registered for %s", s.Use)
	}

	data, err := json.Marshal(s.With.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload map: %w", err)
	}

	valPtr := reflect.New(desc.PayloadType).Interface()

	if err := json.Unmarshal(data, valPtr); err != nil {
		return nil, fmt.Errorf("failed to decode payload for %s: %w", s.ID, err)
	}

	return valPtr, nil
}
