// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package dsl

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Parse(data []byte) (*TestDefinition, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parse fcaf test yaml: %w", err)
	}

	var def TestDefinition
	if err := node.Decode(&def); err != nil {
		return nil, fmt.Errorf("decode fcaf test yaml: %w", err)
	}
	if err := ValidateTestDefinition(def); err != nil {
		return nil, err
	}
	return &def, nil
}

func ParseFile(path string) (*TestDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fcaf test yaml %q: %w", path, err)
	}
	return Parse(data)
}
