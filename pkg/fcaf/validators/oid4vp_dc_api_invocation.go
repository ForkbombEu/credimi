// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"
)

// Outcomes the presentation page reports when an invocation produced no
// Authorization Response.
const (
	dcAPIOutcomeAPIUnavailable = "api_unavailable"
	dcAPIOutcomeRejected       = "rejected"
	dcAPIOutcomeNoVPToken      = "no_vp_token"
	dcAPIOutcomeFailed         = "failed"
)

// OID4VPDCAPIInvocationValidator reads the browser-side record of a Digital
// Credentials API invocation.
//
// A DC API presentation is mediated by the User Agent, so the verifier learns
// nothing from the wire when the Wallet refuses: no Authorization Response is
// ever posted. The capture page therefore reports the outcome separately, and
// that report is the only protocol-grade evidence that the Wallet was invoked
// at all. Distinguishing api_unavailable from rejected matters: the first says
// the browser never offered the API, which is an environment failure rather
// than a Wallet verdict.
type OID4VPDCAPIInvocationValidator struct{}

// ID returns the validator identifier.
func (OID4VPDCAPIInvocationValidator) ID() string {
	return "oid4vp.dc_api_invocation"
}

// Validate requires the session to record a DC API request and, optionally, a
// specific invocation outcome.
func (OID4VPDCAPIInvocationValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Protocol        string   `json:"protocol"`
		Invoked         bool     `json:"invoked"`
		AllowedOutcomes []string `json:"allowed_outcomes"`
		VPTokenPresent  *bool    `json:"vp_token_present"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	for _, outcome := range params.AllowedOutcomes {
		switch outcome {
		case dcAPIOutcomeAPIUnavailable,
			dcAPIOutcomeRejected,
			dcAPIOutcomeNoVPToken,
			dcAPIOutcomeFailed:
		default:
			return Result{
				Status: StatusError,
				Message: fmt.Sprintf(
					"allowed_outcomes entry %q is not a reported DC API outcome",
					outcome,
				),
			}
		}
	}

	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "captured presentation session is not an object"}
	}
	dcAPI, ok := normalizeJSONObject(session["dc_api"])
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "captured session has no dc_api record, so it was not a DC API session",
		}
	}
	request, ok := normalizeJSONObject(dcAPI["request"])
	if !ok {
		return Result{Status: StatusFail, Message: "captured session has no dc_api.request"}
	}
	if params.Protocol != "" {
		protocol := normalizeString(request["protocol"])
		if protocol != params.Protocol {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"DC API request protocol is %q, expected %q",
					protocol,
					params.Protocol,
				),
			}
		}
	}
	if normalizeString(dcAPI["expected_origin"]) == "" {
		return Result{
			Status:  StatusFail,
			Message: "captured session records no expected origin for the DC API request",
		}
	}

	invocation, reported := normalizeJSONObject(dcAPI["invocation"])
	_, presented := findObjectKey(session, "vp_token")
	if params.Invoked {
		// A completed presentation is the strongest proof of invocation: the
		// page reports an outcome only when no Authorization Response came
		// back.
		switch {
		case presented:
		case !reported:
			return Result{
				Status: StatusFail,
				Message: "captured session records neither an Authorization Response " +
					"nor a DC API invocation outcome, so the wallet was never invoked",
			}
		case normalizeString(invocation["outcome"]) == dcAPIOutcomeAPIUnavailable:
			// The button press never reached a wallet. That is an environment
			// failure, not a Wallet verdict, and must never read as success.
			return Result{
				Status: StatusFail,
				Message: "the browser exposed no Digital Credentials API, " +
					"so no wallet was invoked",
			}
		}
	}
	if len(params.AllowedOutcomes) == 0 && params.VPTokenPresent == nil {
		if params.Invoked {
			return Result{
				Status:  StatusPass,
				Message: "wallet was invoked through the Digital Credentials API",
			}
		}
		return Result{Status: StatusPass, Message: "DC API request evidence is present"}
	}
	if !reported {
		return Result{
			Status:  StatusFail,
			Message: "captured session records no DC API invocation outcome",
		}
	}

	outcome := normalizeString(invocation["outcome"])
	if len(params.AllowedOutcomes) > 0 && !containsString(params.AllowedOutcomes, outcome) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"DC API invocation outcome is %q, expected one of %s",
				outcome,
				strings.Join(params.AllowedOutcomes, ", "),
			),
		}
	}
	if params.VPTokenPresent != nil {
		present, ok := invocation["vp_token_present"].(bool)
		if !ok || present != *params.VPTokenPresent {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"DC API invocation reports vp_token_present %v, expected %v",
					invocation["vp_token_present"],
					*params.VPTokenPresent,
				),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("DC API invocation reported outcome %q", outcome),
	}
}
