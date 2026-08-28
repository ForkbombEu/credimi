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
	Evidence            map[string]EvidenceBinding `json:"evidence"             yaml:"evidence"`
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
