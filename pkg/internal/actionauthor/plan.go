// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// PlanSummary describes a normalized plan in lines for the agent: every
// parameter source in full, so it can tell the user what each run will call.
// Indented lines detail the line above them.
func PlanSummary(s Spec, prerequisites []ResolvedPrerequisite) []string {
	lines := []string{"Category: " + SlotCategory(s.Slot)}
	switch s.Outcome {
	case OutcomeRejection:
		lines = append(lines, "Outcome: the wallet rejects the flow")
	case OutcomeWait:
		lines = append(lines, fmt.Sprintf("Outcome: waits %d s", s.WaitSeconds))
	default:
		lines = append(lines, "Outcome: success")
	}
	switch {
	case s.Slot == SlotInstallApp:
		lines = append(lines, "Starts from: the wallet is not installed")
	case len(prerequisites) == 0:
		lines = append(lines, "Starts from: a freshly installed app")
	}
	for i, prerequisite := range prerequisites {
		lines = append(
			lines,
			"Starts from: "+prerequisite.Name+" ("+prerequisite.Action+")",
		)
		if i >= len(s.Prerequisites) || s.Prerequisites[i].Action != prerequisite.Action {
			continue
		}
		for _, parameter := range s.Prerequisites[i].Parameters {
			for _, line := range summarySource(parameter.Name, parameter.Source) {
				lines = append(lines, "  "+line)
			}
		}
	}
	for _, parameter := range s.Parameters {
		source := parameter.Source
		if source.Status != nil {
			// Only the list of the plan's outcome is checked.
			status := *source.Status
			if s.Outcome == OutcomeRejection {
				status.Success = nil
			} else {
				status.Rejection = nil
			}
			source.Status = &status
		}
		lines = append(lines, summarySource(parameter.Name, source)...)
	}
	return lines
}

// summaryBodyLimit caps the characters of an http body in the plan summary.
const summaryBodyLimit = 500

// summarySource describes one parameter source in full for the plan summary:
// a first line, then indented details of an http call.
func summarySource(name string, src SourceSpec) []string {
	head := "${" + name + "}: "
	switch src.Kind {
	case SourceCredentialOffer:
		return []string{head + "a new credential offer from " + src.Identifier + " on every run"}
	case SourceUseCaseVerification:
		return []string{
			head + "a new presentation request from " + src.Identifier + " on every run",
		}
	case SourceLiteral:
		return []string{head + "the fixed value " + strconv.Quote(src.Value)}
	}
	request := src.Request
	if request == nil {
		request = &HTTPSourceRequest{}
	}
	lines := []string{
		head + request.Method + " " + request.URL + ", value from response field " + src.ValuePath,
	}
	for _, header := range slices.Sorted(maps.Keys(request.Headers)) {
		lines = append(lines, "  headers: "+header+": "+request.Headers[header])
	}
	if request.Body != nil {
		lines = append(lines, "  body: "+summaryBody(request.Body))
	}
	for _, capture := range slices.Sorted(maps.Keys(src.Captures)) {
		lines = append(lines, "  captures: "+capture+" from "+src.Captures[capture])
	}
	if status := src.Status; status != nil {
		expected := status.Success
		if len(expected) == 0 {
			expected = status.Rejection
		}
		lines = append(lines, "  status: GET "+status.URL+", field "+status.Path+
			", expected "+strings.Join(expected, ", "))
	}
	return lines
}

// summaryBody returns the compact JSON of an http body, cut to
// summaryBodyLimit characters with an ellipsis.
func summaryBody(body any) string {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(body); err != nil {
		return fmt.Sprintf("%v", body)
	}
	text := strings.TrimSuffix(encoded.String(), "\n")
	if utf8.RuneCountInString(text) <= summaryBodyLimit {
		return text
	}
	return string([]rune(text)[:summaryBodyLimit]) + "…"
}
