// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaValidationActivity validates a JSON map against a JSON Schema.
type SchemaValidationActivity struct {
	*workflowengine.BaseActivity
}

// SchemaValidationActivityPayload is the input payload for the SchemaValidationActivity.
type SchemaValidationActivityPayload struct {
	Schema    string         `json:"schema" yaml:"schema" validate:"required"`
	Data      map[string]any `json:"data" yaml:"data" validate:"required"`
	SubSchema any            `json:"subschema,omitempty" yaml:"subschema,omitempty"`
}

func NewSchemaValidationActivity() *SchemaValidationActivity {
	return &SchemaValidationActivity{
		BaseActivity: &workflowengine.BaseActivity{
			Name: "Validate map against JSON Schema",
		},
	}
}

func (a *SchemaValidationActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *SchemaValidationActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	compiler := jsonschema.NewCompiler()
	schemaID := "/schema.json"

	payload, err := workflowengine.DecodePayload[SchemaValidationActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	var subSchemaStrs []string
	if payload.SubSchema != nil {
		switch subs := payload.SubSchema.(type) {
		case string:
			subSchemaStrs = []string{subs}
		case []interface{}:
			for _, raw := range subs {
				if s, ok := raw.(string); ok {
					subSchemaStrs = append(subSchemaStrs, s)
				} else {
					return result, a.NewActivityError(
						errCode.Code,
						fmt.Sprintf("%s:  'subschema' must be a string or list of strings", errCode.Description),
					)
				}
			}
		default:
			return result, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s:  'subschema' must be a string or list of strings", errCode.Description),
			)
		}
	}

	errCodeUnMarshal := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
	errCodeInvalidSchema := errorcodes.Codes[errorcodes.InvalidSchema]
	mainSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(payload.Schema))
	if err != nil {
		return result, a.NewActivityError(
			errCodeUnMarshal.Code,
			fmt.Sprintf("%s: %v", errCodeUnMarshal.Description, err),
			payload.Schema,
		)
	}
	for i, sub := range subSchemaStrs {
		id := fmt.Sprintf("/subschema%d.json", i+1)
		subSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(sub))
		if err != nil {
			return result, a.NewActivityError(
				errCodeUnMarshal.Code,
				fmt.Sprintf("%s: %v", errCodeUnMarshal.Description, err),
				sub,
			)
		}
		if err := compiler.AddResource(id, subSchema); err != nil {
			return result, a.NewActivityError(
				errCodeInvalidSchema.Code,
				fmt.Sprintf("%s: %v", errCodeInvalidSchema.Description, err),
				sub,
			)
		}
	}

	if err := compiler.AddResource(schemaID, mainSchema); err != nil {
		return result, a.NewActivityError(
			errCodeInvalidSchema.Code,
			fmt.Sprintf("%s: %v", errCodeInvalidSchema.Description, err),
			schemaID,
		)
	}
	schema, err := compiler.Compile(schemaID)
	if err != nil {
		return result, a.NewActivityError(
			errCodeInvalidSchema.Code,
			fmt.Sprintf("%s: %v", errCodeInvalidSchema.Description, err),
			schemaID,
		)
	}

	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}
	var decoded any
	if err := json.Unmarshal(rawBytes, &decoded); err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}

	// Validate data
	if err := schema.Validate(decoded); err != nil {
		errCode := errorcodes.Codes[errorcodes.SchemaValidationFailed]
		ve := &jsonschema.ValidationError{}
		if errors.As(err, &ve) {
			return result, a.NewNonRetryableActivityError(
				errCode.Code,
				ve.GoString(),
				ve,
			)
		}
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.Data,
		)
	}

	// Output valid data
	result.Output = map[string]any{
		"status":  "success",
		"message": "JSON is valid against the schema",
	}
	return result, nil
}
