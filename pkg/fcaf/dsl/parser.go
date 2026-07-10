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
	node, err := parseNode(data, "test")
	if err != nil {
		return nil, err
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

func ParsePrecondition(data []byte) (*PreconditionDefinition, error) {
	node, err := parseNode(data, "precondition")
	if err != nil {
		return nil, err
	}

	var def PreconditionDefinition
	if err := node.Decode(&def); err != nil {
		return nil, fmt.Errorf("decode fcaf precondition yaml: %w", err)
	}
	if err := ValidatePreconditionDefinition(def); err != nil {
		return nil, err
	}
	return &def, nil
}

func ParsePreconditionFile(path string) (*PreconditionDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fcaf precondition yaml %q: %w", path, err)
	}
	return ParsePrecondition(data)
}

func parseNode(data []byte, context string) (*yaml.Node, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parse fcaf %s yaml: %w", context, err)
	}
	return &node, nil
}
