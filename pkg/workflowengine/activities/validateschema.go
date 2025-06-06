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

	schemaStr, ok := input.Payload["schema"].(string)
	if !ok {
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'schema' must be a string", errCode.Description),
		)
	}

	data, ok := input.Payload["data"].(map[string]any)
	if !ok {
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'data'", errCode.Description),
		)
	}
	var subSchemaStrs []string
	if v, ok := input.Payload["subschema"]; ok {
		switch subs := v.(type) {
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
	mainSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaStr))
	if err != nil {
		return result, a.NewActivityError(
			errCodeUnMarshal.Code,
			fmt.Sprintf("%s: %v", errCodeUnMarshal.Description, err),
			schemaStr,
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

	rawBytes, err := json.Marshal(data)
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
			data,
		)
	}

	// Output valid data
	result.Output = map[string]any{
		"status":  "success",
		"message": "JSON is valid against the schema",
	}
	return result, nil
}
