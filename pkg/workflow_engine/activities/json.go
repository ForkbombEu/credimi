// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
)

type JsonActivity struct {
	StructRegistry map[string]reflect.Type // Maps type names to their reflect.Type
}

func (a *JsonActivity) Name() string {
	return "Parse a JSON and validate it against a schema"
}

func (a *JsonActivity) Execute(ctx context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
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
		return workflowengine.Fail(&workflowengine.ActivityResult{}, "Missing structType in payload")
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
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return workflowengine.Fail(&workflowengine.ActivityResult{},
			fmt.Sprintf("Invalid JSON: %v", err))
	}
	return workflowengine.ActivityResult{
		Output: reflect.ValueOf(target).Elem().Interface(),
	}, nil
}
