// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package activities is a package that provides activities for the workflow engine.
// This file contains the JSONActivity struct and its methods.
package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// JSONActivity is an activity that parses a JSON string and validates it against a registered struct type.
type JSONActivity struct {
	StructRegistry map[string]reflect.Type // Maps type names to their reflect.Type
}

// Name returns the name of the activity.
func (a *JSONActivity) Name() string {
	return "Parse a JSON and validate it against a schema"
}

// Execute parses a JSON string from the input payload and validates it against a registered struct type.
func (a *JSONActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	// Get rawJSON
	raw, ok := input.Payload["rawJSON"]
	if !ok {
		return workflowengine.Fail(&workflowengine.ActivityResult{}, "Missing rawJSON in payload")
	}
	rawStr, ok := raw.(string)
	if !ok {
		return workflowengine.Fail(&workflowengine.ActivityResult{}, "rawJSON must be a string")
	}

	// Get struct type name
	structTypeName, ok := input.Payload["structType"].(string)
	if !ok {
		return workflowengine.Fail(
			&workflowengine.ActivityResult{},
			"Missing structType in payload",
		)
	}

	// Look up the struct type from the registry
	structType, ok := a.StructRegistry[structTypeName]
	if !ok {
		return workflowengine.Fail(&workflowengine.ActivityResult{},
			fmt.Sprintf("Unregistered struct type: %s", structTypeName))
	}

	// Create a new instance of the struct
	target := reflect.New(structType).Interface()
	// add additional extra properties
	decoder := json.NewDecoder(strings.NewReader(rawStr))

	if err := decoder.Decode(target); err != nil {
		return workflowengine.Fail(&workflowengine.ActivityResult{},
			fmt.Sprintf("Invalid JSON: %v", err))
	}
	return workflowengine.ActivityResult{
		Output: reflect.ValueOf(target).Elem().Interface(),
	}, nil
}
