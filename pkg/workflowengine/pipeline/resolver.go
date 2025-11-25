// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"fmt"
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
func ResolveSubworkflowInputs(
	step *StepDefinition,
	subDef WorkflowBlock,
	globalCfg map[string]any,
	ctx map[string]any, // merged context (workflow inputs + previous outputs)
) error {
	// Iterate declared inputs of the subworkflow
	for inputName := range subDef.Inputs {
		val, ok := step.With.Payload[inputName]
		if !ok {
			return fmt.Errorf("missing payload for subworkflow input %q", inputName)
		}

		if shouldSkipInString(step.Use, inputName, val) {
			continue
		}

		rv, err := ResolveExpressions(val, ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve subworkflow input %q: %w", inputName, err)
		}
		step.With.Payload[inputName] = rv
	}
	return nil
}
