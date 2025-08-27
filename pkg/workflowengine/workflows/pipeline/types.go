// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type WorkflowDefinition struct {
	Config map[string]string `yaml:"config"`
	Steps  []StepDefinition  `yaml:"steps"`
}

type StepDefinition struct {
	Name     string     `yaml:"name"`
	Activity string     `yaml:"activity"`
	Inputs   StepInputs `yaml:"inputs"`
}

type StepInputs struct {
	Config  map[string]ConfigSource `yaml:"config"`
	Payload map[string]InputSource  `yaml:"payload"`
}

// This struct holds either a ref or a literal value
type InputSource struct {
	Type  string `yaml:"type,omitempty"`
	Ref   string `yaml:"ref,omitempty"`
	Value any    `yaml:"value,omitempty"`
}
type ConfigSource struct {
	Ref   string `yaml:"ref,omitempty"`
	Value string `yaml:"value,omitempty"`
}

func (c *ConfigSource) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		// simple string literal
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		c.Value = s
		return nil
	case yaml.MappingNode:
		tmp := map[string]string{}
		if err := value.Decode(&tmp); err != nil {
			return err
		}
		if ref, ok := tmp["ref"]; ok {
			c.Ref = ref
		}
		if val, ok := tmp["value"]; ok {
			c.Value = val
		}
		return nil
	default:
		return fmt.Errorf("config must be string or map of strings, got: %v", value.Kind)
	}
}
func (i *InputSource) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var raw any
		if err := value.Decode(&raw); err != nil {
			return err
		}
		i.Value = raw
		return nil

	case yaml.MappingNode:
		tmp := map[string]any{}
		if err := value.Decode(&tmp); err != nil {
			return err
		}
		if ref, ok := tmp["ref"].(string); ok {
			i.Ref = ref
			if typ, ok := tmp["type"].(string); ok {
				i.Type = typ
			}
			if val, ok := tmp["value"]; ok {
				i.Value = val
			}
			return nil
		}
		if val, ok := tmp["value"]; ok {
			i.Value = val
			return nil
		}
		i.Value = tmp
		return nil

	case yaml.SequenceNode:
		var arr []any
		if err := value.Decode(&arr); err != nil {
			return err
		}
		i.Value = arr
		return nil
	}

	return fmt.Errorf("unsupported YAML node kind: %v", value.Kind)
}
