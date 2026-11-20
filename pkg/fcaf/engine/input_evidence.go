// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
)

// screenshotEvidenceMissingMessage is the verdict for an assertion whose
// evidence is a mobile step's screenshot list that stored nothing. Every flow
// that takes screenshots reports them as `maestro_screenshot_urls`, so an empty
// or absent list means the step did not run or its evidence was lost before the
// assertion could judge it: that is a harness gap, not Wallet behaviour. The
// validators package owns the message because evidence.screenshot_named applies
// it too when it is called directly.
const screenshotEvidenceMissingMessage = validators.ScreenshotEvidenceMissingMessage

// evidenceInputName returns the evidence name an assertion reads, which is the
// part after the `evidence.` prefix. Only a direct evidence binding can be
// classified here; runtime paths and evidence paths with a pointer into the
// value are left to the validator.
func evidenceInputName(input string) (string, bool) {
	const prefix = "evidence."
	if !strings.HasPrefix(input, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(input, prefix)
	if name == "" {
		return "", false
	}
	return name, true
}

// screenshotEvidenceInput reports whether the assertion reads a mobile step's
// screenshot list. `evidence.screenshot_named` only ever reads one, and every
// evidence name the tests bind to a screenshot list carries the pipeline's
// `visual_evidence` or `screenshot(s)` naming (for example `visual_evidence`,
// `zero_visual_evidence`, `numeric_issuance_screenshots`).
func screenshotEvidenceInput(validatorID, evidenceName string) bool {
	if validatorID == "evidence.screenshot_named" {
		return true
	}
	name := strings.ToLower(evidenceName)
	return strings.Contains(name, "visual_evidence") || strings.Contains(name, "screenshot")
}

// emptyScreenshotList reports whether a screenshot evidence value holds no
// screenshot. A flow output that was never captured is decoded as nil, and an
// empty `[]` is what a step that ran no screenshot command stores.
func emptyScreenshotList(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case []any:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	case string:
		return strings.TrimSpace(typed) == ""
	default:
		return false
	}
}

// nilIsTheWalletsAnswer reports whether a nil value for this evidence is itself
// the fact the assertion tests. The Wallet's Authorization Response members are
// the only evidence whose absence states that the Wallet sent nothing: the
// producing step writes that evidence only when the Wallet answers, so nil is a
// real failure of an assertion that requires a response, not a harness gap.
// Every other evidence value is produced by the verifier or the harness (the
// request object, the session record, the request_uri retrieval, a signature),
// so nil there means the step never captured it.
func nilIsTheWalletsAnswer(evidenceName string) bool {
	name := evidenceName
	if index := strings.LastIndexByte(name, '.'); index >= 0 {
		name = name[index+1:]
	}
	name = strings.ToLower(name)
	return name == "wallet_response" || strings.HasSuffix(name, "_wallet_response")
}

// missingInputEvidence turns an assertion whose evidence is unusable into a
// blocked verdict instead of letting the validator read a nil value as Wallet
// misbehaviour. It returns nil when the validator should judge the value.
//
// Two cases are harness gaps and report blocked:
//   - a screenshot evidence value that holds no screenshot, because the mobile
//     step never stored one;
//   - any other nil evidence value, because the producing step did not capture
//     it, except the Wallet's own response members where nil is the absence the
//     assertion exists to test.
func missingInputEvidence(
	assertion dsl.AssertionDefinition,
	value any,
) *validators.Result {
	// A skip or a not_applicable statement reports a property of the test's
	// setup from its params and never reads the evidence value, so a nil there
	// must not be turned into a block.
	if assertion.Validator == "fcaf.skipped" ||
		assertion.Validator == "fcaf.not_applicable" {
		return nil
	}
	name, ok := evidenceInputName(assertion.Input)
	if !ok {
		return nil
	}
	if screenshotEvidenceInput(assertion.Validator, name) && emptyScreenshotList(value) {
		return &validators.Result{
			Status:  validators.StatusBlocked,
			Message: screenshotEvidenceMissingMessage,
		}
	}
	if value == nil && !nilIsTheWalletsAnswer(name) {
		return &validators.Result{
			Status:  validators.StatusBlocked,
			Message: fmt.Sprintf("evidence %s was not captured", name),
		}
	}
	return nil
}
