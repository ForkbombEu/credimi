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
	Schema    string         `json:"schema"              yaml:"schema"              validate:"required"`
	Data      map[string]any `json:"data"                yaml:"data"                validate:"required"`
	SubSchema any            `json:"subschema,omitempty" yaml:"subschema,omitempty"`
}

type SchemaValidationErrorDetails struct {
	Issues []SchemaValidationIssue `json:"issues"`
}

type SchemaValidationIssue struct {
	Scope        string   `json:"scope"`
	CredentialID string   `json:"credential_id,omitempty"`
	Field        string   `json:"field,omitempty"`
	Path         []string `json:"path,omitempty"`
	Message      string   `json:"message"`
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
						workflowengine.ActivityError{
							Code:    errCode.Code,
							Summary: errCode.Description,
							Message: "'subschema' must be a string or list of strings",
						},
					)
				}
			}
		default:
			return result, a.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: "'subschema' must be a string or list of strings",
				},
			)
		}
	}

	errCodeUnMarshal := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
	errCodeInvalidSchema := errorcodes.Codes[errorcodes.InvalidSchema]
	mainSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(payload.Schema))
	if err != nil {
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCodeUnMarshal.Code,
				Summary: errCodeUnMarshal.Description,
				Message: err.Error(),
				Details: map[string]any{"schema": payload.Schema},
			},
		)
	}
	for i, sub := range subSchemaStrs {
		id := fmt.Sprintf("/subschema%d.json", i+1)
		subSchema, err := jsonschema.UnmarshalJSON(strings.NewReader(sub))
		if err != nil {
			return result, a.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCodeUnMarshal.Code,
					Summary: errCodeUnMarshal.Description,
					Message: err.Error(),
					Details: map[string]any{"subschema": sub},
				},
			)
		}
		if err := compiler.AddResource(id, subSchema); err != nil {
			return result, a.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCodeInvalidSchema.Code,
					Summary: errCodeInvalidSchema.Description,
					Message: err.Error(),
					Details: map[string]any{"subschema": sub},
				},
			)
		}
	}

	if err := compiler.AddResource(schemaID, mainSchema); err != nil {
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCodeInvalidSchema.Code,
				Summary: errCodeInvalidSchema.Description,
				Message: err.Error(),
				Details: map[string]any{"schema_id": schemaID},
			},
		)
	}
	schema, err := compiler.Compile(schemaID)
	if err != nil {
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCodeInvalidSchema.Code,
				Summary: errCodeInvalidSchema.Description,
				Message: err.Error(),
				Details: map[string]any{"schema_id": schemaID},
			},
		)
	}

	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
	}
	var decoded any
	if err := json.Unmarshal(rawBytes, &decoded); err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
	}

	// Validate data
	if err := schema.Validate(decoded); err != nil {
		errCode := errorcodes.Codes[errorcodes.SchemaValidationFailed]
		ve := &jsonschema.ValidationError{}
		if errors.As(err, &ve) {
			details := SchemaValidationErrorDetails{
				Issues: schemaValidationIssues(ve),
			}
			return result, a.NewNonRetryableActivityError(
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: "schema validation failed",
					Details: map[string]any{
						"issues":           details.Issues,
						"validation_error": ve.GoString(),
					},
				},
			)
		}
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{"data": payload.Data},
			},
		)
	}

	// Output valid data
	result.Output = map[string]any{
		"status":  "success",
		"message": "JSON is valid against the schema",
	}
	return result, nil
}

type schemaValidationLeafIssue struct {
	SchemaValidationIssue
	got     any
	want    any
	missing []string
}

func schemaValidationIssues(ve *jsonschema.ValidationError) []SchemaValidationIssue {
	leaves := schemaValidationLeafIssues(ve)
	issues := make([]SchemaValidationIssue, 0, len(leaves))
	seen := map[string]bool{}
	for _, leaf := range leaves {
		key := strings.Join(
			[]string{leaf.Field, leaf.Message},
			"\x00",
		)
		if seen[key] {
			continue
		}
		seen[key] = true
		issues = append(issues, leaf.SchemaValidationIssue)
	}

	return issues
}

func schemaValidationLeafIssues(ve *jsonschema.ValidationError) []schemaValidationLeafIssue {
	if ve == nil {
		return nil
	}
	if len(ve.Causes) == 0 {
		return []schemaValidationLeafIssue{schemaValidationLeafIssueFromError(ve)}
	}

	var issues []schemaValidationLeafIssue
	for _, cause := range ve.Causes {
		issues = append(issues, schemaValidationLeafIssues(cause)...)
	}
	return issues
}

func schemaValidationLeafIssueFromError(ve *jsonschema.ValidationError) schemaValidationLeafIssue {
	kind := schemaValidationErrorKindMap(ve.ErrorKind)
	missing := workflowengine.AsSliceOfStrings(kind["Missing"])
	path := append([]string(nil), ve.InstanceLocation...)
	if len(missing) > 0 {
		path = append(path, missing[0])
	}
	field := schemaValidationField(path)

	issue := schemaValidationLeafIssue{
		SchemaValidationIssue: SchemaValidationIssue{
			Field: field,
			Path:  path,
		},
		got:     kind["Got"],
		want:    kind["Want"],
		missing: missing,
	}
	issue.Message = schemaValidationMessage(issue)

	return issue
}

func schemaValidationErrorKindMap(kind jsonschema.ErrorKind) map[string]any {
	if kind == nil {
		return nil
	}

	raw, err := json.Marshal(kind)
	if err != nil {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

func schemaValidationField(location []string) string {
	if len(location) == 0 {
		return ""
	}
	return strings.Join(location, ".")
}

func schemaValidationMessage(issue schemaValidationLeafIssue) string {
	field := issue.Field
	if field == "" {
		field = "metadata"
	}

	switch {
	case len(issue.missing) > 0:
		return fmt.Sprintf("%s is missing", field)
	case issue.got != nil && issue.want != nil:
		return fmt.Sprintf(
			"%s got %v, expected %s",
			field,
			issue.got,
			schemaValidationExpectedValue(issue.want),
		)
	case issue.want != nil:
		return fmt.Sprintf("%s must match %s", field, schemaValidationExpectedValue(issue.want))
	default:
		return fmt.Sprintf("%s is not valid", field)
	}
}

func schemaValidationExpectedValue(value any) string {
	values := workflowengine.AsSliceOfStrings(value)
	if len(values) == 1 {
		return values[0]
	}
	if len(values) > 1 {
		return strings.Join(values, ", ")
	}
	return fmt.Sprint(value)
}
