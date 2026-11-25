// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"
)

// unresolvedIntentMarker is how Android's activity manager reports an intent
// no installed app handles (`am start` prints `Error: Activity not started,
// unable to resolve Intent { act=android.intent.action.VIEW dat=<link> ... }`).
// Maestro's openLink raises it as AndroidOperationFailedException and stops the
// flow even when the command is optional, so the flow output is the only trace.
const unresolvedIntentMarker = "unable to resolve Intent"

// EvidenceCustomSchemeInvokedValidator proves that opening a link under a
// custom URL scheme invoked the Wallet. Like evidence.screenshot_named it
// passes on the screenshot the flow takes only while the Wallet shows the
// request opened from the link. When that screenshot is missing because
// Android reported that no app handles the scheme, the Wallet cannot be invoked
// through it, which is a failure of the Wallet rather than a missing piece of
// evidence. Every other outcome is the screenshot_named verdict, so a step
// that stored nothing for another reason still reports blocked.
//
// The evidence is an object pairing the mobile step's screenshots with its
// Maestro console output:
//
//	wallet_screenshots: the step's flow_output.maestro_screenshot_urls
//	maestro_output:     the step's flow_output.output (Maestro's stdout)
//	maestro_stderr:     the step's flow_output.stderr (Maestro's stderr)
type EvidenceCustomSchemeInvokedValidator struct{}

// ID returns the validator identifier.
func (EvidenceCustomSchemeInvokedValidator) ID() string {
	return "evidence.custom_scheme_invoked"
}

type evidenceCustomSchemeInvokedParams struct {
	Name   string `json:"name"`
	Scheme string `json:"scheme"`
}

// Validate requires the named screenshot, and fails instead of blocking when
// Android could not resolve the intent of a link under params.scheme.
func (EvidenceCustomSchemeInvokedValidator) Validate(ctx context.Context, input Input) Result {
	params, err := DecodeParams[evidenceCustomSchemeInvokedParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	scheme := strings.TrimSuffix(strings.TrimSpace(params.Scheme), "//")
	if scheme == "" || !strings.HasSuffix(scheme, ":") {
		return Result{
			Status:  StatusError,
			Message: "scheme param is required, as `<scheme>:` or `<scheme>://`",
		}
	}
	evidence, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("evidence value is %T, expected an object", input.Value),
		}
	}

	shown := EvidenceScreenshotNamedValidator{}.Validate(ctx, Input{
		Value:  evidence["wallet_screenshots"],
		Params: map[string]any{"name": params.Name},
	})
	if shown.Status == StatusPass || shown.Status == StatusError {
		return shown
	}
	for _, key := range []string{"maestro_output", "maestro_stderr"} {
		text, _ := evidence[key].(string)
		if line, found := unresolvedSchemeIntent(text, scheme); found {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"the wallet registers no handler for %s//: Android could not resolve the intent (%s)",
					scheme,
					line,
				),
			}
		}
	}
	return shown
}

// unresolvedSchemeIntent returns the line of text where Android reports that
// it could not resolve an intent for a link under scheme. The scheme must be
// on the same line as the report, so a failure to open another link the flow
// followed is not attributed to it.
func unresolvedSchemeIntent(text, scheme string) (string, bool) {
	scheme = strings.ToLower(scheme)
	for line := range strings.Lines(text) {
		if strings.Contains(line, unresolvedIntentMarker) &&
			strings.Contains(strings.ToLower(line), scheme) {
			return strings.TrimSpace(line), true
		}
	}
	return "", false
}
