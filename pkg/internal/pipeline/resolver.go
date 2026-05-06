// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// mergeConfigs merges global config with step-level config
func MergeConfigs(global, step map[string]any) map[string]any {
	res := make(map[string]any)
	for k, v := range global {
		res[k] = v
	}
	for k, v := range step {
		res[k] = v
	}
	return res
}

var stepPayloadExclusions = map[string][]string{
	"rest-chain":        {"yaml"},
	"conformance-check": {"config", "template"},
}

// helper to check if a string is exactly a single ${{ ... }} ref
func isFullRef(s string) bool {
	matches := exprRegexp.FindStringSubmatch(s)
	return matches != nil && strings.TrimSpace(s) == matches[0]
}

// Matches expressions like ${{ ... }}
var exprRegexp = regexp.MustCompile(`\${{\s*([^{}]+?)\s*}}`)

func parsePart(part string) (key string, idxs []int, err error) {
	bracket := strings.Index(part, "[")
	if bracket == -1 {
		return part, nil, nil
	}

	key = part[:bracket]
	rest := part[bracket:]

	for len(rest) > 0 {
		if rest[0] != '[' {
			return "", nil, fmt.Errorf("invalid syntax in part: %s", part)
		}
		end := strings.Index(rest, "]")
		if end == -1 {
			return "", nil, fmt.Errorf("missing ] in part: %s", part)
		}
		numStr := rest[1:end]
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return "", nil, fmt.Errorf("invalid index %q in part: %s", numStr, part)
		}
		idxs = append(idxs, n)
		rest = rest[end+1:]
	}

	return key, idxs, nil
}

func ParsePipeline(expr string) (initialValue string, functions []string, err error) {
	parts := strings.Split(expr, "|")
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("empty pipeline expression")
	}

	initialValue = strings.TrimSpace(parts[0])
	if initialValue == "" {
		return "", nil, fmt.Errorf("empty initial value in pipeline")
	}

	for i := 1; i < len(parts); i++ {
		funcName := strings.TrimSpace(parts[i])
		if funcName == "" {
			return "", nil, fmt.Errorf("empty function name at position %d", i)
		}
		functions = append(functions, funcName)
	}

	return initialValue, functions, nil
}

func ApplyFunction(value any, funcName string) (any, error) {
	re := regexp.MustCompile(`^(\w+)\(([^)]*)\)$`)
	matches := re.FindStringSubmatch(funcName)

	if len(matches) == 0 {
		switch funcName {
		case "upper":
			return toUpper(value), nil
		case "lower":
			return toLower(value), nil
		case "url_encode":
			return urlEncode(value), nil
		case "url_decode":
			return urlDecode(value)
		default:
			return nil, fmt.Errorf("unknown function: %s", funcName)
		}
	}
	baseFunc := matches[1]
	paramsStr := matches[2]

	var params []*int
	if paramsStr != "" {
		parts := strings.Split(paramsStr, ":")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				params = append(params, nil)
				continue
			}
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid parameter %q in function %s: %v", part, baseFunc, err)
			}
			params = append(params, &num)
		}
	}

	switch baseFunc {
	case "slice":
		return slice(value, params)
	default:
		return nil, fmt.Errorf("unknown function: %s", baseFunc)
	}

}

func ResolvePipeline(expr string, ctx map[string]any) (any, error) {
	initialValue, functions, err := ParsePipeline(expr)
	if err != nil {
		return nil, err
	}

	current, err := resolveRef(initialValue, ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving initial value %q: %w", initialValue, err)
	}

	for _, funcName := range functions {
		current, err = ApplyFunction(current, funcName)
		if err != nil {
			return nil, fmt.Errorf("applying function %q: %w", funcName, err)
		}
	}

	return current, nil
}

// resolveRef resolves a dotted ref like "user.addresses[0].city" in the given context
func resolveRef(ref string, ctx map[string]any) (any, error) {
	if strings.Contains(ref, "|") {
		return ResolvePipeline(ref, ctx)
	}
	parts := strings.Split(ref, ".")
	var cur any = ctx

	for _, part := range parts {
		key, idxs, err := parsePart(part)
		if err != nil {
			return nil, err
		}

		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map at %s", part)
		}

		v, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("ref not found: %s", ref)
		}
		cur = v

		for _, idx := range idxs {
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("expected slice at %s in ref %s", part, ref)
			}
			if idx < 0 || idx >= len(arr) {
				return nil, fmt.Errorf("slice index out of bounds at %s in ref %s", part, ref)
			}
			cur = arr[idx]
		}
	}

	return cur, nil
}

// resolveExpressions recursively replaces ${{ ... }} expressions in a value
func ResolveExpressions(val any, ctx map[string]any) (any, error) {
	switch v := val.(type) {
	case string:
		matches := exprRegexp.FindStringSubmatch(v)
		if len(matches) == 2 && matches[0] == v {
			inner := matches[1]
			return resolveRef(inner, ctx)
		}
		return exprRegexp.ReplaceAllStringFunc(v, func(expr string) string {
			matches := exprRegexp.FindStringSubmatch(expr)
			if len(matches) < 2 {
				return fmt.Sprintf("ERR(invalid expression: %s)", expr)
			}
			inner := matches[1]
			resolved, err := resolveRef(inner, ctx)
			if err != nil {
				return fmt.Sprintf("ERR(%s)", err.Error())
			}
			return stringifyResolvedValue(resolved)
		}), nil

	case map[string]any:
		res := make(map[string]any)
		for k, vv := range v {
			rv, err := ResolveExpressions(vv, ctx)
			if err != nil {
				return nil, err
			}
			res[k] = rv
		}
		return res, nil

	case []any:
		arr := make([]any, len(v))
		for i, vv := range v {
			rv, err := ResolveExpressions(vv, ctx)
			if err != nil {
				return nil, err
			}
			arr[i] = rv
		}
		return arr, nil

	default:
		return v, nil
	}
}

func stringifyResolvedValue(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case map[string]any, []any:
		jsonBytes, err := json.MarshalIndent(value, "", "  ")
		if err == nil {
			return string(jsonBytes)
		}
	}

	return fmt.Sprintf("%v", v)
}

func shouldSkipInString(stepRun, key string, val any) bool {
	if strVal, ok := val.(string); ok {
		if !isFullRef(strVal) {
			if keys, exists := stepPayloadExclusions[stepRun]; exists {
				for _, exKey := range keys {
					if exKey == key {
						return true
					}
				}
			}
		}
	}
	return false
}

// ResolveInputs builds activity input for a step
func ResolveInputs(
	step *StepDefinition,
	globalCfg map[string]any,
	ctx map[string]any,
) error {
	stepCfg := make(map[string]any)
	var val any
	var err error
	for k, src := range step.With.Config {
		if shouldSkipInString(step.Use, k, src) {
			val = src
		} else {
			val, err = ResolveExpressions(src, ctx)
			if err != nil {
				return err
			}
		}
		stepCfg[k] = val
	}
	step.With.Config = MergeConfigs(globalCfg, stepCfg)

	for k, v := range step.With.Payload {
		if shouldSkipInString(step.Use, k, v) {
			continue
		}
		rv, err := ResolveExpressions(v, ctx)
		if err != nil {
			return fmt.Errorf("resolving payload key %q: %w", k, err)
		}
		step.With.Payload[k] = rv
	}

	return nil
}

type caseTransformer struct {
	transformString func(string) string
	transformKey    func(string) string
}

var (
	upperTransformer = caseTransformer{
		transformString: strings.ToUpper,
		transformKey:    strings.ToUpper,
	}
	lowerTransformer = caseTransformer{
		transformString: strings.ToLower,
		transformKey:    strings.ToLower,
	}
)

type urlTransformer struct {
	transform func(string) (string, error)
}

var (
	urlEncoder = urlTransformer{
		transform: func(s string) (string, error) {
			return url.PathEscape(s), nil
		},
	}
	urlDecoder = urlTransformer{
		transform: func(s string) (string, error) {
			decoded, err := url.PathUnescape(s)
			if err != nil {
				return "", fmt.Errorf("failed to decode URL string %q: %w", s, err)
			}
			return decoded, nil
		},
	}
)

func transform(value any, t caseTransformer) any {
	switch v := value.(type) {
	case string:
		return t.transformString(v)
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v
	case []any:
		if len(v) == 0 {
			return v
		}
		result := make([]any, len(v))
		for i, elem := range v {
			result[i] = transform(elem, t)
		}
		return result
	case nil:
		return nil
	case map[string]any:
		if len(v) == 0 {
			return make(map[string]any)
		}
		result := make(map[string]any)
		for k, val := range v {
			result[t.transformKey(k)] = transform(val, t)
		}
		return result
	default:
		bytes, err := json.Marshal(v)
		if err == nil {
			var asMap map[string]any
			if err := json.Unmarshal(bytes, &asMap); err == nil {
				return transform(asMap, t)
			}
			return t.transformString(string(bytes))
		}
		return t.transformString(fmt.Sprintf("%v", v))
	}
}

func toUpper(value any) any {
	return transform(value, upperTransformer)
}

func toLower(value any) any {
	return transform(value, lowerTransformer)
}

func transformURL(value any, t urlTransformer) (any, error) {
	switch v := value.(type) {
	case string:
		return t.transform(v)
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v, nil
	case []any:
		if len(v) == 0 {
			return v, nil
		}
		result := make([]any, len(v))
		for i, elem := range v {
			transformed, err := transformURL(elem, t)
			if err != nil {
				return nil, err
			}
			result[i] = transformed
		}
		return result, nil
	case nil:
		return nil, nil
	case map[string]any:
		if len(v) == 0 {
			return make(map[string]any), nil
		}
		result := make(map[string]any)
		for k, val := range v {
			transformed, err := transformURL(val, t)
			if err != nil {
				return nil, err
			}
			result[k] = transformed
		}
		return result, nil
	default:
		bytes, err := json.Marshal(v)
		if err == nil {
			var asMap map[string]any
			if err := json.Unmarshal(bytes, &asMap); err == nil {
				return transformURL(asMap, t)
			}
			return t.transform(string(bytes))
		}
		return t.transform(fmt.Sprintf("%v", v))
	}
}

func urlEncode(value any) any {
	result, _ := transformURL(value, urlEncoder)
	return result
}

func urlDecode(value any) (any, error) {
	return transformURL(value, urlDecoder)
}

// slice extracts a substring or array subset based on 0-based indices.
// Syntax: slice() - returns the whole value
//
//	slice(start) - returns element at start index (supports negative)
//	slice(start:end) - returns elements from start to end-1 (end omitted = to end)
//	slice(:end) - returns first end elements
//
// For strings: operates on Unicode characters
// For arrays: operates on elements
// For other types (objects, numbers, etc.): converts to JSON string then slices
func slice(value any, params []*int) (any, error) {
	switch v := value.(type) {
	case []any:
		return sliceArray(v, params)
	case string:
		return sliceString(v, params)
	default:
		var str string
		switch val := value.(type) {
		case map[string]any:
			bytes, err := json.Marshal(val)
			if err != nil {
				str = fmt.Sprintf("%v", val)
			} else {
				str = string(bytes)
			}
		default:
			str = fmt.Sprintf("%v", val)
		}
		return sliceString(str, params)
	}
}

func sliceString(s string, params []*int) (any, error) {
	runes := []rune(s)
	length := len(runes)

	if len(params) == 0 {
		return s, nil
	}

	if len(params) == 1 && params[0] != nil {
		idx := *params[0]
		if idx < 0 {
			idx = length + idx
		}
		if idx < 0 || idx >= length {
			return "", nil
		}
		return string(runes[idx]), nil
	}

	if len(params) == 2 {
		start := 0
		end := length

		if params[0] != nil {
			start = *params[0]
			if start < 0 {
				start = length + start
			}
		}

		if params[1] != nil {
			end = *params[1]
			if end < 0 {
				end = length + end
			}
		}

		if start > end {
			return "", nil
		}

		return string(runes[start:end]), nil
	}

	return nil, fmt.Errorf("slice accepts 0, 1, or 2 parameters, got %d", len(params))
}

func sliceArray(arr []any, params []*int) (any, error) {
	length := len(arr)

	if len(params) == 0 {
		return arr, nil
	}

	if len(params) == 1 && params[0] != nil {
		idx := *params[0]
		if idx < 0 {
			idx = length + idx
		}
		if idx < 0 || idx >= length {
			return nil, nil
		}
		return arr[idx], nil
	}

	if len(params) == 2 {
		start := 0
		end := length

		if params[0] != nil {
			start = *params[0]
			if start < 0 {
				start = length + start
			}
		}

		if params[1] != nil {
			end = *params[1]
			if end < 0 {
				end = length + end
			}
		}

		if start > end {
			return []any{}, nil
		}

		return arr[start:end], nil
	}

	return nil, fmt.Errorf("slice accepts 0, 1, or 2 parameters, got %d", len(params))
}
