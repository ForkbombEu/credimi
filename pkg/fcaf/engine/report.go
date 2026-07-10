// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
)

type Report struct {
	Suite           string         `json:"suite,omitempty"`
	SelectedTestIDs []string       `json:"selected_test_ids,omitempty"`
	Status          string         `json:"status,omitempty"`
	Tests           []TestResult   `json:"tests,omitempty"`
	ExecutedTests   []ExecutedTest `json:"executed_tests,omitempty"`
	Evidence        EvidenceMap    `json:"evidence,omitempty"`
	ExpandedNodes   []NodeResult   `json:"expanded_nodes,omitempty"`
	Summary         Summary        `json:"summary"`
	Failures        []TestFailure  `json:"failures,omitempty"`
}

type TestResult struct {
	ID                  string                   `json:"id"`
	Title               string                   `json:"title,omitempty"`
	Status              validators.Status        `json:"status"`
	Suite               dsl.Suite                `json:"suite"`
	Assertions          []AssertionResult        `json:"assertions"`
	NormativeReferences []dsl.NormativeReference `json:"normative_references"`
	Preconditions       []NodeResult             `json:"preconditions"`
	Evidence            []EvidenceResult         `json:"evidence,omitempty"`
	Message             string                   `json:"message,omitempty"`
}

type NodeResult struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Status      validators.Status `json:"status"`
	Message     string            `json:"message,omitempty"`
	WorkflowID  string            `json:"workflow_id,omitempty"`
	RunID       string            `json:"run_id,omitempty"`
	PipelineURL string            `json:"pipeline_url,omitempty"`
	Validator   string            `json:"validator,omitempty"`
	Params      map[string]any    `json:"params,omitempty"`
	Outputs     map[string]any    `json:"outputs,omitempty"`
	Evidence    []EvidenceResult  `json:"evidence,omitempty"`
}

type EvidenceResult struct {
	Name       string `json:"name,omitempty"`
	SourceNode string `json:"source_node,omitempty"`
	Path       string `json:"path,omitempty"`
	From       string `json:"from,omitempty"`
	Value      any    `json:"-"`
	Type       string `json:"type,omitempty"`
}

type AssertionResult struct {
	ID           string            `json:"id"`
	Validator    string            `json:"validator"`
	Input        string            `json:"input"`
	Status       validators.Status `json:"status"`
	Message      string            `json:"message,omitempty"`
	Details      map[string]any    `json:"details,omitempty"`
	EvidenceKeys []string          `json:"evidence_keys,omitempty"`
}

type Summary struct {
	Pass          int `json:"pass"`
	Fail          int `json:"fail"`
	Blocked       int `json:"blocked"`
	Skipped       int `json:"skipped"`
	Inconclusive  int `json:"inconclusive"`
	NotApplicable int `json:"not_applicable"`
	Error         int `json:"error"`
}

type TestFailure struct {
	TestID  string            `json:"test_id"`
	Title   string            `json:"title,omitempty"`
	Status  validators.Status `json:"status"`
	Reasons []FailureReason   `json:"reasons"`
	Message string            `json:"message,omitempty"`
}

type FailureReason struct {
	Scope   string            `json:"scope"`
	ID      string            `json:"id"`
	Status  validators.Status `json:"status"`
	Message string            `json:"message,omitempty"`
}

type ExecutedTest struct {
	TestID        string          `json:"test_id"`
	Title         string          `json:"title,omitempty"`
	Status        string          `json:"status"`
	Preconditions []ExecutedCheck `json:"preconditions,omitempty"`
	Assertions    []ExecutedCheck `json:"assertions,omitempty"`
	Outcome       TestOutcome     `json:"outcome"`
}

type ExecutedCheck struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Message      string   `json:"message,omitempty"`
	EvidenceKeys []string `json:"evidence_keys,omitempty"`
}

type TestOutcome struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type EvidenceMap map[string]EvidenceRecord

type EvidenceRecord struct {
	Type       string `json:"type,omitempty"`
	SourceNode string `json:"source_node,omitempty"`
	Path       string `json:"path,omitempty"`
	From       string `json:"from,omitempty"`
	Value      any    `json:"value,omitempty"`
}

func (r Report) HasNonPassingTests() bool {
	for _, test := range r.Tests {
		if test.Status != validators.StatusPass {
			return true
		}
	}
	return false
}

func (r *Report) PopulateFailures() {
	if r == nil {
		return
	}
	r.Failures = r.Failures[:0]
	for _, test := range r.Tests {
		if test.Status == validators.StatusPass {
			continue
		}
		failure := TestFailure{
			TestID:  test.ID,
			Title:   test.Title,
			Status:  test.Status,
			Message: test.Message,
		}
		for _, precondition := range test.Preconditions {
			if precondition.Status == validators.StatusPass {
				continue
			}
			failure.Reasons = append(failure.Reasons, FailureReason{
				Scope:   "precondition",
				ID:      precondition.ID,
				Status:  precondition.Status,
				Message: precondition.Message,
			})
		}
		for _, assertion := range test.Assertions {
			if assertion.Status == validators.StatusPass {
				continue
			}
			failure.Reasons = append(failure.Reasons, FailureReason{
				Scope:   "assertion",
				ID:      assertion.ID,
				Status:  assertion.Status,
				Message: assertion.Message,
			})
		}
		if len(failure.Reasons) == 0 {
			failure.Reasons = append(failure.Reasons, FailureReason{
				Scope:   "test",
				ID:      test.ID,
				Status:  test.Status,
				Message: test.Message,
			})
		}
		r.Failures = append(r.Failures, failure)
	}
}

func (r *Report) PopulateExecutedTests() {
	if r == nil {
		return
	}
	r.ExecutedTests = r.ExecutedTests[:0]
	for _, test := range r.Tests {
		outcome := buildTestOutcome(test)
		executed := ExecutedTest{
			Status:  outcome.Status,
			TestID:  test.ID,
			Title:   test.Title,
			Outcome: outcome,
		}
		for _, precondition := range test.Preconditions {
			evidenceKeys := evidenceKeysFromResults(precondition.Evidence)
			if runEvidenceKey := pipelineRunEvidenceKey(precondition); runEvidenceKey != "" {
				evidenceKeys = prependEvidenceKey(evidenceKeys, runEvidenceKey)
			}
			executed.Preconditions = append(executed.Preconditions, ExecutedCheck{
				ID:           precondition.ID,
				Kind:         precondition.Kind,
				Status:       normalizeExecutionStatus(precondition.Status),
				Message:      precondition.Message,
				EvidenceKeys: evidenceKeys,
			})
		}
		for _, assertion := range test.Assertions {
			executed.Assertions = append(executed.Assertions, ExecutedCheck{
				ID:           assertion.ID,
				Kind:         "assertion",
				Status:       normalizeExecutionStatus(assertion.Status),
				Message:      assertion.Message,
				EvidenceKeys: append([]string(nil), assertion.EvidenceKeys...),
			})
		}
		r.ExecutedTests = append(r.ExecutedTests, executed)
	}
}

func (r *Report) PopulateEvidence() {
	if r == nil {
		return
	}
	evidenceMap := EvidenceMap{}
	for _, test := range r.Tests {
		for _, item := range test.Evidence {
			item = hydrateTestEvidenceValue(test, item)
			addEvidenceRecord(evidenceMap, item)
		}
		for _, precondition := range test.Preconditions {
			addPipelineRunEvidenceRecord(evidenceMap, precondition)
			for _, item := range precondition.Evidence {
				item = hydratePreconditionEvidenceValue(precondition, item)
				addEvidenceRecord(evidenceMap, item)
			}
		}
	}
	if len(evidenceMap) == 0 {
		r.Evidence = nil
		return
	}
	r.Evidence = evidenceMap
}

func (r *Report) PopulateDerivedViews() {
	if r == nil {
		return
	}
	r.Status = reportStatus(r)
	r.PopulateFailures()
	r.PopulateEvidence()
	r.PopulateExecutedTests()
}

func (r Report) PublicReport() Report {
	r.Tests = nil
	r.ExpandedNodes = nil
	r.Failures = nil
	return r
}

func reportStatus(r *Report) string {
	if r.HasNonPassingTests() {
		return "failed"
	}
	return "passed"
}

func addEvidenceRecord(out EvidenceMap, item EvidenceResult) {
	if item.Name == "" {
		return
	}
	out[item.Name] = EvidenceRecord{
		Type:       firstNonEmpty(item.Type, evidenceType(item.Value)),
		SourceNode: item.SourceNode,
		Path:       item.Path,
		From:       item.From,
		Value:      evidenceValue(item.Value),
	}
}

func addPipelineRunEvidenceRecord(out EvidenceMap, precondition NodeResult) {
	key := pipelineRunEvidenceKey(precondition)
	if key == "" {
		return
	}
	out[key] = EvidenceRecord{
		Type:       "pipeline.run",
		SourceNode: precondition.ID,
		Value: map[string]any{
			"workflow_id":  precondition.WorkflowID,
			"run_id":       precondition.RunID,
			"pipeline_url": precondition.PipelineURL,
		},
	}
}

func hydrateTestEvidenceValue(test TestResult, item EvidenceResult) EvidenceResult {
	if item.Value != nil {
		return item
	}
	sourceNode, sourcePath := splitEvidenceSource(item.From)
	if sourceNode == "" {
		sourceNode = item.SourceNode
	}
	if sourcePath == "" {
		sourcePath = item.Name
	}
	for _, precondition := range test.Preconditions {
		if precondition.ID != sourceNode {
			continue
		}
		if value, ok := precondition.Outputs[sourcePath]; ok {
			item.Value = value
		}
		return item
	}
	return item
}

func hydratePreconditionEvidenceValue(precondition NodeResult, item EvidenceResult) EvidenceResult {
	if item.Value != nil {
		return item
	}
	if value, ok := precondition.Outputs[item.Name]; ok {
		item.Value = value
	}
	return item
}

func splitEvidenceSource(binding string) (string, string) {
	const marker = ".outputs."
	for i := 0; i+len(marker) <= len(binding); i++ {
		if binding[i:i+len(marker)] == marker {
			return binding[:i], binding[i+len(marker):]
		}
	}
	return binding, ""
}

func evidenceKeysFromResults(items []EvidenceResult) []string {
	keys := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if item.Name == "" {
			continue
		}
		if _, ok := seen[item.Name]; ok {
			continue
		}
		seen[item.Name] = struct{}{}
		keys = append(keys, item.Name)
	}
	return keys
}

func prependEvidenceKey(keys []string, key string) []string {
	if key == "" {
		return keys
	}
	for _, existing := range keys {
		if existing == key {
			return keys
		}
	}
	out := make([]string, 0, len(keys)+1)
	out = append(out, key)
	out = append(out, keys...)
	return out
}

func pipelineRunEvidenceKey(precondition NodeResult) string {
	if precondition.Kind != "pipeline" {
		return ""
	}
	if precondition.WorkflowID == "" && precondition.RunID == "" && precondition.PipelineURL == "" {
		return ""
	}
	return precondition.ID + ".run"
}

func evidenceValue(value any) any {
	switch typed := value.(type) {
	case *evidence.SDJWTPresentation:
		return map[string]any{
			"raw":               typed.Raw,
			"claims":            typed.Claims,
			"protected_headers": typed.ProtectedHeaders,
			"issuer_payload":    typed.IssuerPayload,
			"key_binding":       typed.KeyBinding,
		}
	case evidence.SDJWTPresentation:
		return map[string]any{
			"raw":               typed.Raw,
			"claims":            typed.Claims,
			"protected_headers": typed.ProtectedHeaders,
			"issuer_payload":    typed.IssuerPayload,
			"key_binding":       typed.KeyBinding,
		}
	case *evidence.MDocPresentation:
		return typed
	case evidence.MDocPresentation:
		return typed
	default:
		return value
	}
}

func evidenceType(value any) string {
	switch value.(type) {
	case *evidence.SDJWTPresentation, evidence.SDJWTPresentation:
		return "sdjwt.presentation"
	case *evidence.MDocPresentation, evidence.MDocPresentation:
		return "mdoc.presentation"
	case map[string]any:
		return "json.object"
	case []any:
		return "json.array"
	case string:
		return "string"
	default:
		if value == nil {
			return ""
		}
		return "value"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func buildTestOutcome(test TestResult) TestOutcome {
	preconditionFailure := firstNonPassingNode(test.Preconditions)
	assertionFailure := firstNonPassingAssertion(test.Assertions)

	if preconditionFailure != nil &&
		(assertionFailure == nil || assertionFailure.Status == validators.StatusBlocked) {
		return TestOutcome{
			Status: "skipped",
			Reason: preconditionFailure.Message,
		}
	}

	switch test.Status {
	case validators.StatusPass:
		return TestOutcome{Status: "passed", Reason: test.Message}
	case validators.StatusFail,
		validators.StatusError,
		validators.StatusBlocked,
		validators.StatusInconclusive:
		reason := test.Message
		if assertionFailure != nil && assertionFailure.Message != "" {
			reason = assertionFailure.Message
		} else if preconditionFailure != nil && preconditionFailure.Message != "" {
			reason = preconditionFailure.Message
		}
		return TestOutcome{Status: "failed", Reason: reason}
	case validators.StatusSkipped:
		return TestOutcome{Status: "skipped", Reason: test.Message}
	case validators.StatusNotApplicable:
		return TestOutcome{Status: "skipped", Reason: test.Message}
	default:
		return TestOutcome{Status: "failed", Reason: test.Message}
	}
}

func firstNonPassingNode(nodes []NodeResult) *NodeResult {
	for i := range nodes {
		if nodes[i].Status != validators.StatusPass {
			return &nodes[i]
		}
	}
	return nil
}

func firstNonPassingAssertion(assertions []AssertionResult) *AssertionResult {
	for i := range assertions {
		if assertions[i].Status != validators.StatusPass {
			return &assertions[i]
		}
	}
	return nil
}

func normalizeExecutionStatus(status validators.Status) string {
	switch status {
	case validators.StatusPass:
		return "passed"
	case validators.StatusFail, validators.StatusError:
		return "failed"
	case validators.StatusBlocked:
		return "blocked"
	case validators.StatusSkipped, validators.StatusNotApplicable:
		return "skipped"
	case validators.StatusInconclusive:
		return "inconclusive"
	default:
		if status == "" {
			return ""
		}
		return string(status)
	}
}
