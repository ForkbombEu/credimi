// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package dsl

type TestDefinition struct {
	ID                  string                     `json:"id"                   yaml:"id"`
	Title               string                     `json:"title,omitempty"      yaml:"title,omitempty"`
	Source              Source                     `json:"source,omitempty"     yaml:"source,omitempty"`
	Suite               Suite                      `json:"suite"                yaml:"suite"`
	Applicability       map[string]any             `json:"applicability"        yaml:"applicability"`
	NormativeReferences []NormativeReference       `json:"normative_references" yaml:"normative_references"`
	Preconditions       []PreconditionRef          `json:"preconditions"        yaml:"preconditions"`
	Evidence            map[string]EvidenceBinding `json:"evidence,omitempty"   yaml:"evidence,omitempty"`
	Assertions          []AssertionDefinition      `json:"assertions"           yaml:"assertions"`
	Verdict             VerdictPolicy              `json:"verdict"              yaml:"verdict"`
}

type Source struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

type Suite struct {
	SUT     string `json:"sut"               yaml:"sut"`
	Role    string `json:"role"              yaml:"role"`
	Section string `json:"section,omitempty" yaml:"section,omitempty"`
}

type NormativeReference struct {
	Title   string `json:"title"             yaml:"title"`
	URL     string `json:"url,omitempty"     yaml:"url,omitempty"`
	Section string `json:"section,omitempty" yaml:"section,omitempty"`
}

type PreconditionRef struct {
	Ref string `json:"ref" yaml:"ref"`
}

type EvidenceBinding struct {
	From string `json:"from" yaml:"from"`
}

type AssertionDefinition struct {
	ID        string         `json:"id"               yaml:"id"`
	Validator string         `json:"validator"        yaml:"validator"`
	Input     string         `json:"input"            yaml:"input"`
	Params    map[string]any `json:"params,omitempty" yaml:"params,omitempty"`
}

type VerdictPolicy struct {
	PassWhen string `json:"pass_when" yaml:"pass_when"`
}

type PreconditionDefinition struct {
	ID            string                      `json:"id"                       yaml:"id"`
	Kind          string                      `json:"kind"                     yaml:"kind"`
	Description   string                      `json:"description,omitempty"    yaml:"description,omitempty"`
	PipelineID    string                      `json:"pipeline_id,omitempty"    yaml:"pipeline_id,omitempty"`
	RequiredSteps []string                    `json:"required_steps,omitempty" yaml:"required_steps,omitempty"`
	DependsOn     []string                    `json:"depends_on,omitempty"     yaml:"depends_on,omitempty"`
	Input         *InputBinding               `json:"input,omitempty"          yaml:"input,omitempty"`
	Validator     string                      `json:"validator,omitempty"      yaml:"validator,omitempty"`
	Params        map[string]any              `json:"params,omitempty"         yaml:"params,omitempty"`
	Outputs       map[string]OutputDefinition `json:"outputs,omitempty"        yaml:"outputs,omitempty"`
	TestID        string                      `json:"test_id,omitempty"        yaml:"test_id,omitempty"`
	FailurePolicy string                      `json:"failure_policy,omitempty" yaml:"failure_policy,omitempty"`
}

type InputBinding struct {
	From string `json:"from" yaml:"from"`
}

type OutputDefinition struct {
	Path    string `json:"path"              yaml:"path"`
	Decoder string `json:"decoder,omitempty" yaml:"decoder,omitempty"`
}
