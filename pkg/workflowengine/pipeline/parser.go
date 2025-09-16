// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
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

func (s *StepInputs) UnmarshalYAML(value *yaml.Node) error {
	var tmp map[string]any
	if err := value.Decode(&tmp); err != nil {
		return err
	}

	s.Payload = make(map[string]InputSource)
	s.Config = make(map[string]string)

	for k, v := range tmp {
		if k == "config" {
			cfgBytes, err := yaml.Marshal(v)
			if err != nil {
				return err
			}
			if err := yaml.Unmarshal(cfgBytes, &s.Config); err != nil {
				return err
			}
		} else {
			// everything else goes into Payload
			switch val := v.(type) {
			case map[string]any:
				if _, ok := val["type"]; ok {
					if _, ok := val["value"]; ok {
						var src InputSource
						nodeBytes, err := yaml.Marshal(val)
						if err != nil {
							return err
						}
						if err := yaml.Unmarshal(nodeBytes, &src); err != nil {
							return err
						}
						s.Payload[k] = src
						continue
					}
				}
				// otherwise store whole map as Value
				s.Payload[k] = InputSource{Value: val}
			default:
				s.Payload[k] = InputSource{Value: val}
			}
		}
	}
	return nil
}
