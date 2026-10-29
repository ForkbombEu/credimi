// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"fmt"
	"strconv"
	"strings"
)

// captureSelector locates the values a validator checks inside a Capture
// issuance exchange, so one validator serves every request, header, proof or
// attestation the sources name:
//
//	select:
//	  request: /par                               # Wallet requests ending in /par
//	  path: raw_headers[OAuth-Client-Attestation] # path inside each request
//	  require: ""                                 # path that must exist, else fail
//	  absent: not_applicable                      # status when path selects nothing
//
// Without `request` the path starts at the capture session (the input itself
// when it carries no capture_session). A path is dot-separated: `name` reads an
// object member, `name[]` every element of an array, `name[Key]` the `value`
// of every `{name, value}` element whose name is Key in any case (the shape of
// Capture's raw_headers), and a number an array index.
type captureSelector struct {
	Request string `json:"request"`
	Path    string `json:"path"`
	Require string `json:"require"`
	Absent  Status `json:"absent"`
	Reason  string `json:"reason"`
}

// selectedValue is one value a selector found, with the object its path
// started from, so a check can read a sibling value of the same request.
type selectedValue struct {
	value any
	base  map[string]any
	label string
}

type selectorParams struct {
	Select *captureSelector `json:"select"`
}

// resolveSelection returns the values params.select picks from the input, or
// the input value itself when the validator has no select param. A non-nil
// Result ends the validation: the selection failed, was redacted, or selected
// nothing, which the selector's `absent` status decides.
func resolveSelection(input Input) ([]selectedValue, *Result) {
	params, err := DecodeParams[selectorParams](input.Params)
	if err != nil {
		return nil, &Result{Status: StatusError, Message: err.Error()}
	}
	if params.Select == nil {
		return []selectedValue{{value: input.Value, label: "input"}}, nil
	}
	return params.Select.resolve(input.Value)
}

func (s captureSelector) resolve(value any) ([]selectedValue, *Result) {
	if s.Path == "" {
		return nil, &Result{Status: StatusError, Message: "select.path is required"}
	}
	switch s.Absent {
	case "", StatusFail, StatusNotApplicable, StatusInconclusive, StatusBlocked:
	default:
		return nil, &Result{
			Status:  StatusError,
			Message: "select.absent must be fail, not_applicable, inconclusive or blocked",
		}
	}
	bases, result := s.bases(value)
	if result != nil {
		return nil, result
	}
	var selected []selectedValue
	for _, base := range bases {
		if s.Require != "" && len(walkCapturePath(base.value, s.Require, base.label)) == 0 {
			return nil, missingCaptureEvidence(
				value,
				fmt.Sprintf("%s records no %s", base.label, s.Require),
			)
		}
		for _, found := range walkCapturePath(base.value, s.Path, base.label) {
			found.base = base.base
			if isRedactionMarker(found.value) {
				return nil, &Result{
					Status: StatusInconclusive,
					Message: fmt.Sprintf(
						"%s is redacted; only a session created with evidence_mode \"full\" records it",
						found.label,
					),
				}
			}
			selected = append(selected, found)
		}
	}
	if len(selected) == 0 {
		status := s.Absent
		if status == "" {
			status = StatusFail
		}
		message := fmt.Sprintf("no %s was recorded", s.describe())
		if s.Reason != "" {
			message += "; " + s.Reason
		}
		return nil, &Result{Status: status, Message: message}
	}
	return selected, nil
}

func (s captureSelector) describe() string {
	if s.Request == "" {
		return s.Path
	}
	return s.Path + " in a Wallet request ending in " + s.Request
}

// bases returns the objects the selector's path starts from: every Wallet
// request ending in `request`, or the capture session.
func (s captureSelector) bases(value any) ([]selectedValue, *Result) {
	object, ok := normalizeJSONObject(value)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %s, expected object", jsonTypeName(value)),
		}
	}
	if s.Request == "" {
		if session, ok := normalizeJSONObject(object["capture_session"]); ok {
			object = session
		}
		return []selectedValue{{value: object, base: object, label: "the capture session"}}, nil
	}
	exchange, err := decodeOID4VCIExchange(value)
	if err != nil {
		return nil, &Result{Status: StatusFail, Message: err.Error()}
	}
	requests := exchange.sessionRequestsEndingIn(s.Request)
	if len(requests) == 0 {
		return nil, missingCaptureEvidence(value, fmt.Sprintf(
			"the issuance session records no Wallet request ending in %s",
			s.Request,
		))
	}
	bases := make([]selectedValue, 0, len(requests))
	for _, request := range requests {
		label := fmt.Sprintf("the %s %s request", request.method, request.path)
		if !request.at.IsZero() {
			label += " at " + request.at.Format("2006-01-02T15:04:05.000Z07:00")
		}
		bases = append(bases, selectedValue{value: request.raw, base: request.raw, label: label})
	}
	return bases, nil
}

// missingCaptureEvidence reports evidence a selection needs and the capture
// lacks; for an issuance exchange it is blocked when the issuer stopped the
// flow before (see oid4vciExchange.missingEvidence).
func missingCaptureEvidence(value any, message string) *Result {
	exchange, err := decodeOID4VCIExchange(value)
	if err != nil {
		return &Result{Status: StatusFail, Message: message}
	}
	result := exchange.missingEvidence(message)
	return &result
}

// onSelection runs check on every value params.select picks, as the check's
// input, and returns the first result that is not a pass. Without a select
// param it runs check on the input as given. It lets a generic JSON validator
// read a member of a Capture exchange, such as raw.credential_request, with
// the selector's blocked and not-applicable outcomes.
func onSelection(input Input, check func(Input) Result) Result {
	if _, ok := input.Params["select"]; !ok {
		return check(input)
	}
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	var last Result
	for _, item := range selected {
		scoped := input
		scoped.Value = item.value
		last = check(scoped)
		if last.Status != StatusPass {
			last.Message = item.label + ": " + last.Message
			return last
		}
	}
	return last
}

// walkCapturePath returns every value the path selects in value.
func walkCapturePath(value any, path string, label string) []selectedValue {
	current := []selectedValue{{value: value, label: label}}
	for _, segment := range splitCapturePath(path) {
		next := make([]selectedValue, 0, len(current))
		for _, item := range current {
			next = append(next, stepCapturePath(item, segment)...)
		}
		current = next
	}
	return current
}

// splitCapturePath splits a path on dots outside brackets, so a header name
// in `raw_headers[...]` may hold any character but `]`.
func splitCapturePath(path string) []string {
	var segments []string
	depth := 0
	start := 0
	for index, character := range path {
		switch character {
		case '[':
			depth++
		case ']':
			depth--
		case '.':
			if depth == 0 {
				segments = append(segments, path[start:index])
				start = index + 1
			}
		}
	}
	return append(segments, path[start:])
}

func stepCapturePath(item selectedValue, segment string) []selectedValue {
	name, selector, indexed := strings.Cut(segment, "[")
	selector = strings.TrimSuffix(selector, "]")
	child := func(value any, suffix string) selectedValue {
		return selectedValue{value: value, label: item.label + " " + suffix}
	}
	var target any
	switch typed := item.value.(type) {
	case map[string]any:
		value, ok := typed[name]
		if !ok || value == nil {
			return nil
		}
		target = value
	case []any:
		index, err := strconv.Atoi(name)
		if err != nil || index < 0 || index >= len(typed) {
			return nil
		}
		target = typed[index]
	default:
		return nil
	}
	if !indexed {
		return []selectedValue{child(target, name)}
	}
	elements, ok := target.([]any)
	if !ok {
		return nil
	}
	var out []selectedValue
	for index, element := range elements {
		if selector == "" {
			out = append(out, child(element, fmt.Sprintf("%s[%d]", name, index)))
			continue
		}
		entry, ok := element.(map[string]any)
		if !ok {
			continue
		}
		if header, _ := entry["name"].(string); strings.EqualFold(header, selector) {
			if value, ok := entry["value"]; ok {
				out = append(out, child(value, fmt.Sprintf("%s[%s] #%d", name, selector, index+1)))
			}
		}
	}
	return out
}

// selectedString returns the string a selection holds, or a fail Result.
func selectedString(item selectedValue) (string, *Result) {
	text, ok := item.value.(string)
	if !ok {
		return "", &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"%s is a JSON %s, expected string",
				item.label,
				jsonTypeName(item.value),
			),
		}
	}
	return text, nil
}

// exchangeValue reads a value of the exchange by a path whose first segment
// names an exchange member, e.g. issuer_metadata.credential_issuer.
func exchangeValue(input any, path string) (any, bool) {
	object, ok := normalizeJSONObject(input)
	if !ok {
		return nil, false
	}
	found := walkCapturePath(object, path, "")
	if len(found) != 1 {
		return nil, false
	}
	return found[0].value, true
}
