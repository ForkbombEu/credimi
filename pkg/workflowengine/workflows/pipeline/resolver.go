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

// Merge global config with step-level config (step overrides global)
func mergeConfigs(global, step map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range global {
		res[k] = v
	}
	for k, v := range step {
		res[k] = v
	}
	return res
}

// Resolve a dotted ref string in a nested context
var sliceIndexRegexp = regexp.MustCompile(`^([a-zA-Z0-9_]+)(?:\[(\d+)\])?$`)

func resolveRef(ref string, ctx map[string]any) (any, error) {
	parts := strings.Split(ref, ".")
	var cur any = ctx

	for _, part := range parts {
		matches := sliceIndexRegexp.FindStringSubmatch(part)
		if matches == nil {
			return nil, fmt.Errorf("invalid ref part: %s", part)
		}
		key := matches[1]
		idxStr := matches[2]

		// map access
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map at %s", part)
		}
		v, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("ref not found: %s", ref)
		}
		cur = v

		// slice access if [index] present
		if idxStr != "" {
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("expected slice at %s in ref %s", part, ref)
			}
			idx, _ := strconv.Atoi(idxStr)
			if idx < 0 || idx >= len(arr) {
				return nil, fmt.Errorf("slice index out of bounds at %s in ref %s", part, ref)
			}
			cur = arr[idx]
		}
	}

	return cur, nil
}
func castType(val any, typeStr string) (any, error) {
	switch typeStr {
	case "string":
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

func resolveValue(val any, ctx map[string]any) (any, error) {
	switch v := val.(type) {
	case map[string]any:

		if ref, ok := v["ref"].(string); ok {
			res, err := resolveRef(ref, ctx)
			if err != nil {
				return nil, err
			}
			// recursively resolve in case the resolved value is another ref
			resolvedVal, err := resolveValue(res, ctx)
			if err != nil {
				return nil, err
			}
			// Apply type conversion if specified
			if typeStr, ok := v["type"].(string); ok {
				return castType(resolvedVal, typeStr)
			}
			return resolvedVal, nil
		}
		res := make(map[string]any)
		for key, val := range v {
			r, err := resolveValue(val, ctx)
			if err != nil {
				return nil, err
			}
			res[key] = r
		}
		return res, nil
	case []any:
		arr := make([]any, len(v))
		for i, val := range v {
			r, err := resolveValue(val, ctx)
			if err != nil {
				return nil, err
			}
			arr[i] = r
		}
		return arr, nil
	default:
		return val, nil
	}
}

// Resolve step inputs and merge configs
func ResolveInputs(
	step StepDefinition,
	globalCfg map[string]string,
	ctx map[string]any,
) (*workflowengine.ActivityInput, error) {
	// Resolve step-level config
	stepCfg := make(map[string]string)
	for k, src := range step.Inputs.Config {
		var val string
		if src.Ref != "" {
			v, err := resolveRef(src.Ref, ctx)
			if err != nil {
				return nil, fmt.Errorf("resolving config ref %s: %w", src.Ref, err)
			}
			val = fmt.Sprintf("%v", v)
		} else {
			val = src.Value
		}
		stepCfg[k] = val
	}

	// Merge global + step configs
	cfg := mergeConfigs(globalCfg, stepCfg)

	// Resolve payload
	payload := make(map[string]any)
	for k, src := range step.Inputs.Payload {
		var val any
		var err error

		if src.Ref != "" {
			val, err = resolveRef(src.Ref, ctx)
			if err != nil {
				return nil, fmt.Errorf("resolving payload ref %s: %w", src.Ref, err)
			}
		} else {
			val, err = resolveValue(src.Value, ctx)
			if err != nil {
				return nil, fmt.Errorf("resolving payload value for %s: %w", k, err)
			}
		}

		if src.Type != "" {
			val, err = castType(val, src.Type)
			if err != nil {
				return nil, err
			}
		}

		payload[k] = val
	}

	return &workflowengine.ActivityInput{
		Payload: payload,
		Config:  cfg,
	}, nil
}
