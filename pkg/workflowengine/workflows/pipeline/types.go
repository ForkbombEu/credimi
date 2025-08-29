// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"gopkg.in/yaml.v3"
)

type WorkflowDefinition struct {
	Version string                           `yaml:"version"`
	Name    string                           `yaml:"name"`
	Entry   string                           `yaml:"entry"`
	Env     map[string]string                `yaml:"env"`
	Runtime map[string]any                   `yaml:"runtime"`
	Checks  map[string]SubWorkflowDefinition `yaml:"custom_checks"`
	Config  map[string]string                `yaml:"config"`
	Steps   []StepDefinition                 `yaml:"steps"`
}

type SubWorkflowDefinition struct {
	Description string            `yaml:"description"`
	Inputs      map[string]string `yaml:"inputs"`
	Outputs     map[string]string `yaml:"outputs"`
	Config      map[string]string `yaml:"config"`
	Steps       []StepDefinition  `yaml:"steps"`
}

type StepDefinition struct {
	ID       string                 `yaml:"id"`
	Run      string                 `yaml:"run"`
	With     StepInputs             `yaml:"with"`
	Retry    map[string]any         `yaml:"retry,omitempty"`
	Timeout  string                 `yaml:"timeout,omitempty"`
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

type StepInputs struct {
	Config  map[string]string      `yaml:"config"`
	Payload map[string]InputSource `yaml:"payload"`
}

// A single input source (always a string unless type is set)
type InputSource struct {
	Type  string `yaml:"type,omitempty"`
	Value any    `yaml:"value,omitempty"`
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
