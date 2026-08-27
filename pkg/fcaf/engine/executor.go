// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
)

type Engine struct {
	registry *validators.Registry
}

func New(registry *validators.Registry) (*Engine, error) {
	if registry == nil {
		var err error
		registry, err = validators.DefaultRegistry()
		if err != nil {
			return nil, err
		}
	}
	return &Engine{registry: registry}, nil
}

func (e *Engine) ExecuteCatalog(
	ctx context.Context,
	cat *catalog.Catalog,
	testIDs []string,
	suite string,
	runtime map[string]any,
	bundle evidence.Bundle,
) (Report, error) {
	if cat == nil {
		return Report{}, fmt.Errorf("catalog is required")
	}

	selected, err := cat.ResolveSelectedTests(testIDs, suite, runtime)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		Suite:           suite,
		SelectedTestIDs: selected,
		Tests:           make([]TestResult, 0, len(selected)),
	}
	for _, testID := range selected {
		result := e.evaluateTest(ctx, cat.Tests[testID], bundle, runtime)
		report.Tests = append(report.Tests, result)
		addSummary(&report.Summary, result.Status)
	}
	return report, nil
}

func (e *Engine) evaluateTest(
	ctx context.Context,
	test dsl.TestDefinition,
	bundle evidence.Bundle,
	runtime map[string]any,
) TestResult {
	evidenceResults := make([]EvidenceResult, 0, len(test.Evidence))
	resolvedEvidence := make(map[string]any, len(test.Evidence))
	status := validators.StatusPass

	names := make([]string, 0, len(test.Evidence))
	for name := range test.Evidence {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		binding := test.Evidence[name]
		value, sourceNode, found, err := resolvePipelineOutput(binding.From, bundle.PipelineOutputs)
		if err != nil {
			status = validators.StatusError
			continue
		}
		if !found {
			if status != validators.StatusError {
				status = validators.StatusBlocked
			}
			continue
		}
		resolvedEvidence[name] = value
		evidenceResults = append(evidenceResults, EvidenceResult{
			Name:       name,
			SourceNode: sourceNode,
			From:       binding.From,
			Value:      value,
		})
	}

	assertions := make([]AssertionResult, 0, len(test.Assertions))
	for _, assertion := range test.Assertions {
		result := e.executeAssertion(
			ctx,
			test,
			assertion,
			bundle,
			runtime,
			resolvedEvidence,
		)
		assertions = append(assertions, result)
		status = mergeStatus(status, result.Status)
	}
	if status == validators.StatusPass {
		status = AggregateVerdict(assertions)
	}

	return TestResult{
		ID:                  test.ID,
		Title:               test.Title,
		Status:              status,
		Suite:               test.Suite,
		Assertions:          assertions,
		NormativeReferences: test.NormativeReferences,
		Evidence:            evidenceResults,
		Message:             verdictMessage(status),
	}
}

func resolvePipelineOutput(
	binding string,
	pipelineOutputs map[string]any,
) (any, string, bool, error) {
	sourceNode, outputName := splitPipelineOutputBinding(binding)
	if sourceNode == "" || outputName == "" {
		return nil, sourceNode, false, fmt.Errorf("invalid pipeline output binding %q", binding)
	}

	raw, found := pipelineOutputs[sourceNode]
	if !found {
		return nil, sourceNode, false, nil
	}

	result, err := evidence.DecodePipelineExecutionResult(raw)
	if err != nil {
		return nil, sourceNode, false, err
	}
	output, ok := result.Output.(map[string]any)
	if !ok {
		return nil, sourceNode, false, fmt.Errorf(
			"aggregate pipeline output is %T, expected object",
			result.Output,
		)
	}
	value, found := output[outputName]
	return value, sourceNode, found, nil
}

func splitPipelineOutputBinding(binding string) (string, string) {
	const marker = ".outputs."
	index := strings.Index(binding, marker)
	if index < 0 {
		return "", ""
	}
	return binding[:index], binding[index+len(marker):]
}

func (e *Engine) executeAssertion(
	ctx context.Context,
	test dsl.TestDefinition,
	assertion dsl.AssertionDefinition,
	bundle evidence.Bundle,
	runtime map[string]any,
	resolvedEvidence map[string]any,
) AssertionResult {
	value, ok := resolveAssertionInput(assertion.Input, bundle, runtime, resolvedEvidence)
	if !ok {
		return AssertionResult{
			ID:           assertion.ID,
			Validator:    assertion.Validator,
			Input:        assertion.Input,
			Status:       validators.StatusBlocked,
			Message:      fmt.Sprintf("input %q is missing", assertion.Input),
			EvidenceKeys: evidenceKeysFromAssertionInput(assertion.Input),
		}
	}

	validator, ok := e.registry.Get(assertion.Validator)
	if !ok {
		return AssertionResult{
			ID:           assertion.ID,
			Validator:    assertion.Validator,
			Input:        assertion.Input,
			Status:       validators.StatusError,
			Message:      fmt.Sprintf("validator %q is not registered", assertion.Validator),
			EvidenceKeys: evidenceKeysFromAssertionInput(assertion.Input),
		}
	}
	result := validator.Validate(ctx, validators.Input{
		Value:   value,
		Bundle:  bundle,
		Params:  assertion.Params,
		Runtime: runtime,
		Suite:   test.Suite,
	})
	return AssertionResult{
		ID:           assertion.ID,
		Validator:    assertion.Validator,
		Input:        assertion.Input,
		Status:       result.Status,
		Message:      result.Message,
		Details:      result.Details,
		EvidenceKeys: evidenceKeysFromAssertionInput(assertion.Input),
	}
}

func resolveAssertionInput(
	input string,
	bundle evidence.Bundle,
	runtime map[string]any,
	resolvedEvidence map[string]any,
) (any, bool) {
	if strings.HasPrefix(input, "evidence.") {
		name := strings.TrimPrefix(input, "evidence.")
		if value, ok := resolvedEvidence[name]; ok {
			return value, true
		}
		result := evidence.Lookup(bundle, input)
		return result.Value, result.Found
	}
	if strings.HasPrefix(input, "runtime.") {
		return lookupRuntime(runtime, strings.TrimPrefix(input, "runtime."))
	}
	return nil, false
}

func lookupRuntime(runtime map[string]any, path string) (any, bool) {
	current := any(runtime)
	for _, part := range strings.Split(path, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = obj[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func evidenceKeysFromAssertionInput(input string) []string {
	const prefix = "evidence."
	if !strings.HasPrefix(input, prefix) {
		return nil
	}
	key := strings.TrimPrefix(input, prefix)
	if key == "" || strings.Contains(key, ".") {
		return nil
	}
	return []string{key}
}

func mergeStatus(current validators.Status, next validators.Status) validators.Status {
	switch {
	case next == validators.StatusError || current == validators.StatusError:
		return validators.StatusError
	case current == validators.StatusFail || next == validators.StatusFail:
		return validators.StatusFail
	case current == validators.StatusBlocked || next == validators.StatusBlocked:
		return validators.StatusBlocked
	case current == validators.StatusInconclusive || next == validators.StatusInconclusive:
		return validators.StatusInconclusive
	default:
		return current
	}
}

func addSummary(summary *Summary, status validators.Status) {
	switch status {
	case validators.StatusPass:
		summary.Pass++
	case validators.StatusFail:
		summary.Fail++
	case validators.StatusBlocked:
		summary.Blocked++
	case validators.StatusSkipped:
		summary.Skipped++
	case validators.StatusInconclusive:
		summary.Inconclusive++
	case validators.StatusNotApplicable:
		summary.NotApplicable++
	default:
		summary.Error++
	}
}
