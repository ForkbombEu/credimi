// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"slices"
)

const (
	staticDiscoveryAudience = "https://self-issued.me/v2"
	staticDiscovery         = "static"
	dynamicDiscovery        = "dynamic"
	audienceAccepted        = "accepted"
	audienceRejected        = "rejected"
)

// OID4VPRequestAudienceValidator evaluates the Wallet against the aud claim of
// the Request Object it was served (OpenID4VP Section 5.8). A Wallet using
// dynamic discovery POSTs wallet_metadata carrying issuer, and aud must equal
// it; otherwise the Wallet relies on static discovery, and aud must be
// https://self-issued.me/v2. The test applies only to the discovery mode named
// in params; the delivered aud must match (outcome accepted) or differ from
// (outcome rejected) the value that mode requires, and the Wallet must then
// present or answer with an error and no presentation. Its input is the
// captured presentation session.
type OID4VPRequestAudienceValidator struct{}

func (OID4VPRequestAudienceValidator) ID() string { return "oid4vp.request_audience" }

func (OID4VPRequestAudienceValidator) Validate(ctx context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Discovery string `json:"discovery"`
		Outcome   string `json:"outcome"`
		Code      string `json:"code"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Discovery != staticDiscovery && params.Discovery != dynamicDiscovery {
		return Result{Status: StatusError, Message: "discovery param must be static or dynamic"}
	}
	if params.Outcome != audienceAccepted && params.Outcome != audienceRejected {
		return Result{Status: StatusError, Message: "outcome param must be accepted or rejected"}
	}

	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: sessionNotObjectMessage}
	}
	discovery, expected, result := walletDiscovery(session)
	if result != nil {
		return *result
	}
	if discovery != params.Discovery {
		return Result{
			Status: StatusNotApplicable,
			Message: fmt.Sprintf(
				"Wallet uses %s discovery, the test requires %s discovery",
				discovery,
				params.Discovery,
			),
		}
	}

	raw, _ := normalizeJSONObject(session["raw"])
	payload, err := compactJWTPart(raw["authorization_request_jwt"], 1)
	if err != nil {
		return Result{Status: StatusFail, Message: "served Request Object: " + err.Error()}
	}
	matches, delivered := audienceMatches(payload["aud"], expected)
	wantMatch := params.Outcome == audienceAccepted
	if matches != wantMatch {
		return Result{
			Status: StatusInconclusive,
			Message: fmt.Sprintf(
				"served Request Object aud %v does not set up the %s case for expected %q",
				delivered,
				params.Outcome,
				expected,
			),
		}
	}

	if wantMatch {
		if presentation, _ := findObjectKey(session, "vp_token"); isEmptyDCQLValue(presentation) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"aud %q matches, but the Wallet returned no vp_token",
					expected,
				),
			}
		}
		return Result{
			Status: StatusPass,
			Message: fmt.Sprintf(
				"Wallet presented for aud %q under %s discovery",
				expected,
				discovery,
			),
		}
	}

	if params.Code != "" {
		return OID4VPErrorResponseRequiredValidator{}.Validate(ctx, Input{
			Value:  session,
			Params: map[string]any{"code": params.Code},
		})
	}
	if presentation, _ := findObjectKey(session, "vp_token"); !isEmptyDCQLValue(presentation) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"Wallet returned vp_token for aud %v, expected %q",
				delivered,
				expected,
			),
		}
	}
	errorValue, _ := findObjectKey(session, "error")
	code := normalizeString(errorValue)
	if code == "" {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"Wallet returned no error for aud %v, expected %q",
				delivered,
				expected,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("Wallet answered aud %v with error %q", delivered, code),
	}
}

// walletDiscovery returns the discovery mode the Wallet used and the aud it
// requires: the issuer of POSTed wallet_metadata, or the static identifier.
func walletDiscovery(session map[string]any) (string, string, *Result) {
	observed, _ := normalizeJSONObject(session["observed"])
	requestURIPayload, _ := normalizeJSONObject(observed["request_uri_payload"])
	payload, ok := normalizeJSONObject(requestURIPayload["value"])
	if !ok || payload["wallet_metadata"] == nil {
		return staticDiscovery, staticDiscoveryAudience, nil
	}
	metadata, err := decodeWalletMetadata(payload["wallet_metadata"])
	if err != nil {
		return "", "", &Result{Status: StatusFail, Message: err.Error()}
	}
	issuer := normalizeString(metadata["issuer"])
	if issuer == "" {
		return staticDiscovery, staticDiscoveryAudience, nil
	}
	return dynamicDiscovery, issuer, nil
}

func audienceMatches(value any, expected string) (bool, any) {
	switch typed := value.(type) {
	case string:
		return typed == expected, typed
	case []any:
		return slices.Contains(typed, any(expected)), typed
	default:
		return false, value
	}
}
