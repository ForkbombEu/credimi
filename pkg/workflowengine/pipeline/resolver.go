// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// mergeConfigs merges global config with step-level config
func MergeConfigs(global, step map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range global {
		res[k] = v
	}
	for k, v := range step {
		res[k] = v
	}
	return res
}

var stepPayloadExclusions = map[string][]string{
	"rest-chain": {"yaml"},
}

// helper to check if a string is exactly a single ${{ ... }} ref
func isFullRef(s string) bool {
	matches := exprRegexp.FindStringSubmatch(s)
	return matches != nil && strings.TrimSpace(s) == matches[0]
}

// Matches expressions like ${{ ... }}
var exprRegexp = regexp.MustCompile(`\${{\s*([a-zA-Z0-9_\-\.\[\]]+)\s*}}`)

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

// resolveRef resolves a dotted ref like "user.addresses[0].city" in the given context
func resolveRef(ref string, ctx map[string]any) (any, error) {
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
			return fmt.Sprintf("%v", resolved)
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

func castType(val any, typeStr string) (any, error) {
	switch typeStr {
	case "string", "":
		return fmt.Sprintf("%v", val), nil
	case "int":
		switch v := val.(type) {
		case int:
			return v, nil
		case string:
			return strconv.Atoi(v)
		default:
			return nil, fmt.Errorf("cannot cast %T to int", val)
		}
	case "map":
		m, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot cast %T to map", val)
		}
		return m, nil
	case "[]string":
		return workflowengine.AsSliceOfStrings(val), nil
	case "[]map":
		return workflowengine.AsSliceOfMaps(val), nil
	case "bytes", "[]byte":
		switch v := val.(type) {
		case string:
			return []byte(v), nil
		case []byte:
			return v, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to []byte", val)
		}
	default:
		return val, nil
	}
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
	step StepDefinition,
	globalCfg map[string]string,
	ctx map[string]any,
) (map[string]any, map[string]string, error) {
	stepCfg := make(map[string]string)
	for k, src := range step.With.Config {
		val, err := ResolveExpressions(src, ctx)
		if err != nil {
			return nil, nil, err
		}
		stepCfg[k] = val.(string)
	}
	cfg := MergeConfigs(globalCfg, stepCfg)

	payload := make(map[string]any)
	for k, src := range step.With.Payload {
		var val any
		var err error

		if shouldSkipInString(step.Use, k, src.Value) {
			val = src.Value
		} else {
			val, err = ResolveExpressions(src.Value, ctx)
			if err != nil {
				return nil, nil, err
			}
		}

		if src.Type != "" {
			val, err = castType(val, src.Type)
			if err != nil {
				return nil, nil, err
			}
		}
		payload[k] = val
	}

	return payload, cfg, nil
}
func ResolveSubworkflowInputs(
	step StepDefinition,
	subDef WorkflowBlock,
	globalCfg map[string]string,
	ctx map[string]any, // merged context (workflow inputs + previous outputs)
) (map[string]any, error) {
	resolvedInputs := make(map[string]any)

	// Iterate declared inputs of the subworkflow
	for k := range subDef.Inputs {
		src, ok := step.With.Payload[k]
		if !ok {
			return nil, fmt.Errorf("missing payload for subworkflow input %q", k)
		}

		var val any
		var err error

		// Handle expression resolution
		if shouldSkipInString(step.Use, k, src.Value) {
			val = src.Value
		} else {
			val, err = ResolveExpressions(src.Value, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve input %q: %w", k, err)
			}
		}

		// Cast to type if defined
		if src.Type != "" {
			val, err = castType(val, src.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to cast input %q: %w", k, err)
			}
		}

		resolvedInputs[k] = val
	}

	return resolvedInputs, nil
}
