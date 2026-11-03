// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase/core"
)

const planKeyLabel = "credimi/action-author/plan/v1"

// PlanGrant is a human's confirmation of the plan of one ActionAuthor
// workflow, signed by Credimi.
type PlanGrant struct {
	AuthorWorkflowID   string `json:"author_workflow_id"`
	Plan               Spec   `json:"plan"`
	ApproverID         string `json:"approver_id"`
	ApproverCollection string `json:"approver_collection"`
	ConfirmedAt        int64  `json:"confirmed_at"`
}

// SignPlan returns the plan token, in the format of SignApproval.
func SignPlan(app core.App, g PlanGrant) (string, error) {
	return signToken(app, planKeyLabel, g)
}

// VerifyPlan accepts only a plan token Credimi signed for this workflow;
// every other token is ErrForbidden. Plans do not expire: activities accept
// them only while their session runs.
func VerifyPlan(app core.App, token, authorWorkflowID string) (PlanGrant, error) {
	var grant PlanGrant
	if err := verifyToken(app, planKeyLabel, token, &grant); err != nil {
		if errors.Is(err, errUnsigned) {
			return PlanGrant{}, fmt.Errorf("%w: plan not signed by Credimi", ErrForbidden)
		}
		return PlanGrant{}, err
	}
	if grant.AuthorWorkflowID != authorWorkflowID {
		return PlanGrant{}, fmt.Errorf("%w: plan for another session", ErrForbidden)
	}
	return grant, nil
}

// PlanSummary describes a normalized plan in lines a human confirms: every
// parameter source in full, so the card shows what each run will call.
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
	if len(prerequisites) == 0 {
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

// summaryBodyLimit caps the characters of an http body on the plan card.
const summaryBodyLimit = 500

// summarySource describes one parameter source in full for the plan card:
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
