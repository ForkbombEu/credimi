// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/forkbombeu/credimi/pkg/utils"
)

type Engine struct {
	registry  *validators.Registry
	extractor func(root any, path string, decoder string) (any, error)
	nodeCache *NodeResultCache
}

type executionState struct {
	catalog *catalog.Catalog
	bundle  evidence.Bundle
	runtime map[string]any
	cache   map[string]NodeResult
}

func New(registry *validators.Registry) (*Engine, error) {
	if registry == nil {
		var err error
		registry, err = validators.DefaultRegistry()
		if err != nil {
			return nil, err
		}
	}
	return NewWithExtractor(registry, evidence.Extract)
}

func NewWithExtractor(
	registry *validators.Registry,
	extractor func(root any, path string, decoder string) (any, error),
) (*Engine, error) {
	return NewWithCaches(registry, extractor, nil)
}

func NewWithCaches(
	registry *validators.Registry,
	extractor func(root any, path string, decoder string) (any, error),
	nodeCache *NodeResultCache,
) (*Engine, error) {
	if registry == nil {
		var err error
		registry, err = validators.DefaultRegistry()
		if err != nil {
			return nil, err
		}
	}
	if extractor == nil {
		extractor = evidence.Extract
	}
	return &Engine{registry: registry, extractor: extractor, nodeCache: nodeCache}, nil
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

	state := &executionState{
		catalog: cat,
		bundle:  bundle,
		runtime: runtime,
		cache:   map[string]NodeResult{},
	}

	report := Report{
		Suite:           suite,
		SelectedTestIDs: selected,
		Tests:           make([]TestResult, 0, len(selected)),
	}

	for _, testID := range selected {
		test := cat.Tests[testID]
		result, err := e.evaluateTest(ctx, state, test)
		if err != nil {
			return Report{}, err
		}
		report.Tests = append(report.Tests, result)
		addSummary(&report.Summary, result.Status)
	}

	return report, nil
}

func (e *Engine) evaluateTest(
	ctx context.Context,
	state *executionState,
	test dsl.TestDefinition,
) (TestResult, error) {
	preconditions := make([]NodeResult, 0, len(test.Preconditions))
	evidenceResults := make([]EvidenceResult, 0, len(test.Evidence))
	resolvedEvidence := map[string]any{}
	status := validators.StatusPass

	for _, ref := range test.Preconditions {
		node, err := e.evaluateReference(ctx, state, ref.Ref)
		if err != nil {
			return TestResult{}, err
		}
		preconditions = append(preconditions, node)
		if node.Status == validators.StatusFail {
			status = validators.StatusFail
		}
		if node.Status == validators.StatusBlocked && status != validators.StatusFail {
			status = validators.StatusBlocked
		}
	}

	for name, binding := range test.Evidence {
		sourceNode, sourcePath := splitNodeBinding(binding.From)
		node, err := e.evaluateReference(ctx, state, sourceNode)
		if err != nil {
			return TestResult{}, err
		}
		value, ok := lookupNodeOutput(node, sourcePath)
		if !ok {
			if status != validators.StatusFail {
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
			state.bundle,
			state.runtime,
			resolvedEvidence,
		)
		assertions = append(assertions, result)
		status = mergeStatus(status, result.Status)
	}
	if status == validators.StatusPass {
		status = AggregateVerdict(assertions)
	}

	state.cache["test."+test.ID] = NodeResult{
		ID:      "test." + test.ID,
		Kind:    "test",
		Status:  status,
		Message: verdictMessage(status),
	}

	return TestResult{
		ID:                  test.ID,
		Title:               test.Title,
		Status:              status,
		Suite:               test.Suite,
		Assertions:          assertions,
		NormativeReferences: test.NormativeReferences,
		Preconditions:       preconditions,
		Evidence:            evidenceResults,
		Message:             verdictMessage(status),
	}, nil
}

func (e *Engine) evaluateReference(
	ctx context.Context,
	state *executionState,
	ref string,
) (NodeResult, error) {
	if node, ok := state.cache[ref]; ok {
		return node, nil
	}

	switch {
	case strings.HasPrefix(ref, "pipeline."), strings.HasPrefix(ref, "assertion."):
		precondition, ok := state.catalog.Preconditions[ref]
		if !ok {
			return NodeResult{}, fmt.Errorf("precondition %q not found", ref)
		}
		cacheKey := reusablePreconditionCacheKey(precondition, state.bundle)
		if node, found := e.nodeCache.Get(cacheKey); found {
			state.cache[ref] = node
			return node, nil
		}
		node, err := e.evaluatePrecondition(ctx, state, precondition)
		if err != nil {
			return NodeResult{}, err
		}
		state.cache[ref] = node
		e.nodeCache.Put(cacheKey, node)
		return node, nil
	case strings.HasPrefix(ref, "test."):
		testID := strings.TrimPrefix(ref, "test.")
		test, ok := state.catalog.Tests[testID]
		if !ok {
			return NodeResult{}, fmt.Errorf("test %q not found", testID)
		}
		result, err := e.evaluateTest(ctx, state, test)
		if err != nil {
			return NodeResult{}, err
		}
		node := NodeResult{ID: ref, Kind: "test", Status: result.Status, Message: result.Message}
		state.cache[ref] = node
		return node, nil
	default:
		return NodeResult{}, fmt.Errorf("unsupported reference %q", ref)
	}
}

func reusablePreconditionCacheKey(
	precondition dsl.PreconditionDefinition,
	bundle evidence.Bundle,
) string {
	executions := make([]string, 0)
	seen := map[string]struct{}{}
	for _, raw := range bundle.PipelineOutputs {
		result, err := evidence.DecodePipelineExecutionResult(raw)
		if err != nil || result.WorkflowID == "" {
			continue
		}
		execution := result.WorkflowID + "\x00" + result.WorkflowRunID
		if _, exists := seen[execution]; exists {
			continue
		}
		seen[execution] = struct{}{}
		executions = append(executions, execution)
	}
	if len(executions) == 0 {
		return ""
	}
	sort.Strings(executions)
	definition, err := json.Marshal(precondition)
	if err != nil {
		return ""
	}
	scope := lookupString(bundle.Runtime, "namespace") + "\x00" +
		lookupString(bundle.Runtime, "app_url") + "\x00" +
		lookupString(bundle.Runtime, "fixture") + "\x00" +
		strings.Join(executions, "\x01")
	sum := sha256.Sum256(append(definition, []byte(scope)...))
	return fmt.Sprintf("%x", sum)
}

func (e *Engine) evaluatePrecondition(
	ctx context.Context,
	state *executionState,
	precondition dsl.PreconditionDefinition,
) (NodeResult, error) {
	node := NodeResult{
		ID:      precondition.ID,
		Kind:    precondition.Kind,
		Params:  precondition.Params,
		Outputs: map[string]any{},
	}

	switch precondition.Kind {
	case "pipeline":
		pipelineID := precondition.PipelineID
		if fixture := lookupString(state.runtime, "fixture"); fixture != "" {
			if selected, ok := precondition.Fixtures[fixture]; ok {
				pipelineID = selected
			}
		}
		raw, ok := state.bundle.PipelineOutputs[precondition.ID]
		if !ok {
			raw, ok = state.bundle.PipelineOutputs[strings.TrimPrefix(precondition.ID, "pipeline.")]
		}
		if !ok && strings.TrimSpace(pipelineID) != "" {
			raw, ok = state.bundle.PipelineOutputs[pipelineID]
		}
		if !ok {
			node.Status = validators.StatusBlocked
			node.Message = "pipeline output was not provided"
			return node, nil
		}
		pipelineResult, err := evidence.DecodePipelineExecutionResult(raw)
		if err != nil {
			node.Status = validators.StatusError
			node.Message = err.Error()
			return node, nil //nolint:nilerr // Decode failures are represented in the FCAF report.
		}
		rawEnvelope := pipelineResult.LegacyMap()
		node.WorkflowID = pipelineResult.WorkflowID
		node.RunID = pipelineResult.WorkflowRunID
		node.PipelineURL = pipelineRunURL(
			state.runtime,
			pipelineResult.WorkflowID,
			pipelineResult.WorkflowRunID,
		)
		if failure := validatePipelineRequiredSteps(precondition, pipelineResult); failure != "" {
			node.Status = validators.StatusFail
			node.Message = failure
			return node, nil
		}
		for name, output := range precondition.Outputs {
			value, extractMessage := e.extractPipelineOutput(rawEnvelope, output)
			if extractMessage != "" {
				node.Status = validators.StatusFail
				node.Message = extractMessage
				return node, nil //nolint:nilerr // Decode failures are represented in the FCAF report.
			}
			state.bundle.Preconditions = ensureMap(state.bundle.Preconditions)
			state.bundle.Preconditions[precondition.ID+".outputs."+name] = value
			node.Outputs[name] = value
			node.Evidence = append(node.Evidence, EvidenceResult{
				Name:       name,
				SourceNode: precondition.ID,
				Path:       output.Path,
				Value:      value,
			})
		}
		node.Status = validators.StatusPass
		if len(precondition.RequiredSteps) > 0 {
			node.Message = fmt.Sprintf(
				"pipeline required steps passed: %s",
				strings.Join(precondition.RequiredSteps, ", "),
			)
		} else {
			node.Message = "pipeline outputs extracted"
		}
		return node, nil
	case "assertion":
		for _, dependency := range precondition.DependsOn {
			dependencyNode, err := e.evaluateReference(ctx, state, dependency)
			if err != nil {
				return NodeResult{}, err
			}
			if dependencyNode.Status != validators.StatusPass {
				node.Status = validators.StatusBlocked
				node.Message = "dependency did not pass"
				return node, nil
			}
		}
		value, ok := resolveBindingValue(precondition.Input.From, state.bundle, state.runtime)
		if !ok {
			node.Status = validators.StatusBlocked
			node.Message = "precondition input is missing"
			return node, nil
		}
		validator, ok := e.registry.Get(precondition.Validator)
		if !ok {
			node.Status = validators.StatusError
			node.Message = fmt.Sprintf("validator %q is not registered", precondition.Validator)
			return node, nil
		}
		result := validator.Validate(ctx, validators.Input{
			Value:   value,
			Bundle:  state.bundle,
			Params:  precondition.Params,
			Runtime: state.runtime,
		})
		node.Status = result.Status
		node.Message = result.Message
		node.Validator = precondition.Validator
		if evidenceKey := evidenceKeyFromBinding(precondition.Input.From); evidenceKey != "" {
			node.Evidence = append(node.Evidence, EvidenceResult{
				Name:       evidenceKey,
				SourceNode: precondition.ID,
				From:       precondition.Input.From,
				Value:      value,
			})
		}
		return node, nil
	case "test":
		dependency, err := e.evaluateReference(ctx, state, "test."+precondition.TestID)
		if err != nil {
			return NodeResult{}, err
		}
		node.Status = dependency.Status
		node.Message = dependency.Message
		return node, nil
	default:
		return NodeResult{}, fmt.Errorf("unsupported precondition kind %q", precondition.Kind)
	}
}

func validatePipelineRequiredSteps(
	precondition dsl.PreconditionDefinition,
	result evidence.PipelineExecutionResult,
) string {
	failures := make(map[string]evidence.PipelineStepFailure, len(result.StepFailures))
	for _, failure := range result.StepFailures {
		if failure.StepID != "" {
			failures[failure.StepID] = failure
		}
	}

	if len(precondition.RequiredSteps) == 0 {
		if len(result.StepFailures) == 0 {
			return ""
		}
		return pipelineStepFailureMessage(result.StepFailures[0])
	}

	output, ok := result.Output.(map[string]any)
	if !ok {
		return fmt.Sprintf(
			"pipeline output is %T, expected object containing step results",
			result.Output,
		)
	}
	for _, stepID := range precondition.RequiredSteps {
		if failure, failed := failures[stepID]; failed {
			return pipelineStepFailureMessage(failure)
		}
		if _, executed := output[stepID]; !executed {
			return fmt.Sprintf(
				"required pipeline step %q was not executed or produced no result",
				stepID,
			)
		}
	}
	return ""
}

func pipelineStepFailureMessage(failure evidence.PipelineStepFailure) string {
	reason := strings.TrimSpace(failure.Message)
	if reason == "" {
		reason = strings.TrimSpace(failure.Summary)
	}
	if reason == "" {
		reason = "step failed"
	}
	if failure.Code != "" {
		reason = failure.Code + " " + reason
	}
	if failure.StepID == "" {
		return "pipeline failed: " + reason
	}
	return fmt.Sprintf("required pipeline step %q failed: %s", failure.StepID, reason)
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
	return resolveBindingValue(input, bundle, runtime)
}

func resolveBindingValue(
	binding string,
	bundle evidence.Bundle,
	runtime map[string]any,
) (any, bool) {
	switch {
	case strings.HasPrefix(binding, "runtime."):
		return lookupRuntime(runtime, strings.TrimPrefix(binding, "runtime."))
	case strings.HasPrefix(binding, "evidence."):
		result := evidence.Lookup(bundle, binding)
		return result.Value, result.Found
	default:
		return lookupPreconditionValue(bundle.Preconditions, binding)
	}
}

func splitNodeBinding(binding string) (string, string) {
	idx := strings.Index(binding, ".outputs.")
	if idx < 0 {
		return binding, ""
	}
	return binding[:idx], strings.TrimPrefix(binding[idx:], ".outputs.")
}

func lookupNodeOutput(node NodeResult, output string) (any, bool) {
	value, ok := node.Outputs[output]
	return value, ok
}

func lookupPreconditionValue(preconditions map[string]any, key string) (any, bool) {
	value, ok := preconditions[key]
	return value, ok
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
	key := evidenceKeyFromBinding(input)
	if key == "" {
		return nil
	}
	return []string{key}
}

func evidenceKeyFromBinding(binding string) string {
	const prefix = "evidence."
	if !strings.HasPrefix(binding, prefix) {
		return ""
	}
	key := strings.TrimPrefix(binding, prefix)
	if key == "" || strings.Contains(key, ".") {
		return ""
	}
	return key
}

func pipelineRunURL(runtime map[string]any, workflowID string, runID string) string {
	if strings.TrimSpace(workflowID) == "" || strings.TrimSpace(runID) == "" {
		return ""
	}
	appURL := lookupString(runtime, "app_url")
	if appURL == "" {
		return ""
	}
	return utils.JoinURL(appURL, "my", "tests", "runs", workflowID, runID)
}

func lookupString(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func ensureMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return in
}

func (e *Engine) extractPipelineOutput(raw any, output dsl.OutputDefinition) (any, string) {
	value, err := e.extractor(raw, output.Path, output.Decoder)
	if err != nil {
		return nil, err.Error()
	}
	return value, ""
}

func mergeStatus(current validators.Status, next validators.Status) validators.Status {
	switch {
	case next == validators.StatusError:
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
