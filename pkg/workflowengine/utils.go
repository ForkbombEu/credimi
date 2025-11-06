// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflowengine

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// DecodePayload decodes a JSON payload into a given type.
// It returns the decoded object and an error if the decoding fails.
// This allows for the decoding of arbitrary JSON payloads into Go types.
func DecodePayload[T any](input any) (T, error) {
	var t T
	b, err := json.Marshal(input)
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}
	if err := validate.Struct(t); err != nil {
		return t, err
	}
	return t, nil
}

func AsSliceOfMaps(val any) []map[string]any {
	if v, ok := val.([]map[string]any); ok {
		return v
	}
	if arr, ok := val.([]any); ok {
		res := make([]map[string]any, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				res = append(res, m)
			}
		}
		return res
	}
	return nil
}
func AsSliceOfStrings(val any) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		res := make([]string, len(v))
		for i, item := range v {
			res[i] = fmt.Sprint(item)
		}
		return res
	default:
		return nil
	}
}

func AsString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func AsMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func AsBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
