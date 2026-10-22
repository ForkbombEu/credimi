// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// isRedactionMarker reports whether value is the {redacted, present, length}
// object Capture writes in place of a sensitive value. The marker keeps only
// the value's presence, so its JSON type and content are not evidence.
func isRedactionMarker(value any) bool {
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	redacted, _ := object["redacted"].(bool)
	return redacted
}

// memberPresent reports whether an object member carries a value: the key
// exists with a non-null value, and a redaction marker records it as present.
func memberPresent(object map[string]any, key string) bool {
	value, ok := object[key]
	if !ok || value == nil {
		return false
	}
	if isRedactionMarker(value) {
		present, _ := value.(map[string]any)["present"].(bool)
		return present
	}
	return true
}

// jsonTypeName names the JSON type of a decoded JSON value.
func jsonTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
	}
}

var jsonTypeNames = []string{"string", "number", "boolean", "array", "object", "null"}

// JSONFieldTypeValidator requires an object member to hold a value of one JSON type.
type JSONFieldTypeValidator struct{}

func (JSONFieldTypeValidator) ID() string { return "json.field_type" }

// Validate passes when input[field] exists and has JSON type params.type. A
// Capture redaction marker in its place is inconclusive: it hides the type.
func (JSONFieldTypeValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field string `json:"field"`
		Type  string `json:"type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: fieldParamRequired}
	}
	if !slices.Contains(jsonTypeNames, params.Type) {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("type param must be one of %v", jsonTypeNames),
		}
	}
	object, ok := input.Value.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %s, expected object", jsonTypeName(input.Value)),
		}
	}
	value, exists := object[params.Field]
	if !exists {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("field %q is missing", params.Field),
		}
	}
	if isRedactionMarker(value) {
		return Result{
			Status: StatusInconclusive,
			Message: fmt.Sprintf(
				"field %q is redacted in the evidence, so its JSON type is not recorded",
				params.Field,
			),
		}
	}
	if actual := jsonTypeName(value); actual != params.Type {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"field %q is a JSON %s, expected %s",
				params.Field,
				actual,
				params.Type,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("field %q is a JSON %s", params.Field, params.Type),
	}
}

// JSONObjectExactlyOneMemberValidator requires an object member to be an object
// carrying exactly one of the listed members.
type JSONObjectExactlyOneMemberValidator struct{}

func (JSONObjectExactlyOneMemberValidator) ID() string { return "json.object_exactly_one_member" }

// Validate passes when input[field] is an object in which exactly one of
// params.members is present (see memberPresent).
func (JSONObjectExactlyOneMemberValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Field   string   `json:"field"`
		Members []string `json:"members"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Field == "" {
		return Result{Status: StatusError, Message: fieldParamRequired}
	}
	if len(params.Members) < 2 {
		return Result{Status: StatusError, Message: "members param needs at least two members"}
	}
	object, ok := input.Value.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %s, expected object", jsonTypeName(input.Value)),
		}
	}
	nested, ok := object[params.Field].(map[string]any)
	if !ok || isRedactionMarker(nested) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"field %q is %s, expected an object",
				params.Field,
				jsonTypeName(object[params.Field]),
			),
		}
	}
	present := make([]string, 0, len(params.Members))
	for _, member := range params.Members {
		if memberPresent(nested, member) {
			present = append(present, member)
		}
	}
	if len(present) != 1 {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"field %q carries %d of %s (%s), expected exactly one",
				params.Field,
				len(present),
				strings.Join(params.Members, ", "),
				strings.Join(present, ", "),
			),
			Details: map[string]any{"present": present},
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("field %q carries only %q", params.Field, present[0]),
		Details: map[string]any{"present": present},
	}
}
