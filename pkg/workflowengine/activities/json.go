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

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// JSONActivity is an activity that parses a JSON string and validates it against a registered struct type.
type JSONActivity struct {
	*workflowengine.BaseActivity
	StructRegistry map[string]reflect.Type // Maps type names to their reflect.Type
}

// JSONActivityPayload is the input payload for the JSONActivity.
type JSONActivityPayload struct {
	RawJSON    string `json:"rawJSON" yaml:"rawJSON" validate:"required"`
	StructType string `json:"struct_type" yaml:"struct_type" validate:"required"`
}

func NewJSONActivity(structRegistry map[string]reflect.Type) *JSONActivity {
	return &JSONActivity{
		BaseActivity: &workflowengine.BaseActivity{
			Name: "Parse and validate JSON against a schema",
		},
		StructRegistry: structRegistry,
	}
}

// Name returns the name of the activity.
func (a *JSONActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute parses a JSON string from the input payload and validates it against a registered struct type.
func (a *JSONActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}

	payload, err := workflowengine.DecodePayload[JSONActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	// Look up the struct type from the registry
	structType, ok := a.StructRegistry[payload.StructType]
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnregisteredStructType]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: '%s'", errCode.Description, payload.StructType),
			payload.StructType,
		)
	}

	// Create a new instance of the struct
	target := reflect.New(structType).Interface()

	decoder := json.NewDecoder(strings.NewReader(payload.RawJSON))

	if err := decoder.Decode(target); err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.RawJSON,
		)
	}
	result.Output = reflect.ValueOf(target).Elem().Interface()
	return result, nil
}
