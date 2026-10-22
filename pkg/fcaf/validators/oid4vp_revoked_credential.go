// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// OID4VCIRevocationPrecedesRequestValidator proves the precondition of a
// revoked-credential test: the Status List service revoked every reference of
// the issuance, and the Wallet received the Verifier's request long enough
// after the revocation that any Status List Token it fetched before it had
// expired.
//
// A Wallet may rely on a Status List Token until its exp (Token Status List,
// Section 13.7), so a request that arrives earlier says nothing about whether
// the Wallet checks revocation. That case is blocked, not a Wallet failure.
//
// Input: { revocation: <POST /sessions/{id}/status response>,
// presentation_session: <Capture verifier session record> }.
type OID4VCIRevocationPrecedesRequestValidator struct{}

// ID returns the validator identifier.
func (OID4VCIRevocationPrecedesRequestValidator) ID() string {
	return "oid4vci.revocation_precedes_request"
}

// Validate requires a fully successful revocation and a request retrieval at
// least params.min_delay_seconds after it.
func (OID4VCIRevocationPrecedesRequestValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		MinDelaySeconds int `json:"min_delay_seconds"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.MinDelaySeconds <= 0 {
		return Result{Status: StatusError, Message: "min_delay_seconds param must be positive"}
	}
	evidence, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "revocation evidence is not an object"}
	}
	revokedAt, result := successfulRevocationTime(evidence["revocation"])
	if result != nil {
		return *result
	}
	session, ok := normalizeJSONObject(evidence["presentation_session"])
	if !ok {
		return Result{Status: StatusFail, Message: "presentation session is not an object"}
	}
	retrievedAt, found, err := firstSessionEventTime(session, "vp_request_retrieved")
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	if !found {
		return Result{
			Status:  StatusBlocked,
			Message: "the Wallet never retrieved the request, so the revoked credential was never requested",
		}
	}
	delay := retrievedAt.Sub(revokedAt)
	minimum := time.Duration(params.MinDelaySeconds) * time.Second
	if delay < minimum {
		return Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				"the Wallet retrieved the request %s after the revocation, less than %s, "+
					"so a Status List Token fetched before the revocation may not have expired",
				delay.Round(time.Second),
				minimum,
			),
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"every status list reference was revoked, and the Wallet retrieved the request %s later",
			delay.Round(time.Second),
		),
	}
}

// successfulRevocationTime reads Capture's revocation response: the requested
// status must be revoked and every allocated reference must have been set to 1
// by the Status List service. A partial or failed revocation leaves the
// precondition unbuilt, which blocks the test.
func successfulRevocationTime(value any) (time.Time, *Result) {
	revocation, ok := normalizeJSONObject(value)
	if !ok {
		return time.Time{}, &Result{
			Status:  StatusFail,
			Message: "revocation response is not an object",
		}
	}
	if revocation["requested_status"] != "revoked" {
		return time.Time{}, &Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				"revocation requested_status is %v, expected revoked",
				revocation["requested_status"],
			),
		}
	}
	results, _ := revocation["results"].([]any)
	if len(results) == 0 {
		return time.Time{}, &Result{
			Status:  StatusBlocked,
			Message: "revocation response lists no status list reference",
		}
	}
	for index, raw := range results {
		entry, _ := normalizeJSONObject(raw)
		status, _ := claimPathIndex(entry["status"])
		if entry["ok"] != true || status != 1 {
			return time.Time{}, &Result{
				Status: StatusBlocked,
				Message: fmt.Sprintf(
					"status list reference %d was not revoked (ok %v, status %v)",
					index,
					entry["ok"],
					entry["status"],
				),
			}
		}
	}
	at, err := parseCaptureTime(revocation["at"])
	if err != nil {
		return time.Time{}, &Result{Status: StatusFail, Message: "revocation at: " + err.Error()}
	}
	return at, nil
}

// firstSessionEventTime returns the time of the first event of eventType in a
// Capture session record.
func firstSessionEventTime(session map[string]any, eventType string) (time.Time, bool, error) {
	events, _ := session["events"].([]any)
	for _, raw := range events {
		event, ok := normalizeJSONObject(raw)
		if !ok || event["type"] != eventType {
			continue
		}
		at, err := parseCaptureTime(event["at"])
		if err != nil {
			return time.Time{}, false, fmt.Errorf("%s event at: %w", eventType, err)
		}
		return at, true, nil
	}
	return time.Time{}, false, nil
}

func parseCaptureTime(value any) (time.Time, error) {
	text, ok := value.(string)
	if !ok || text == "" {
		return time.Time{}, fmt.Errorf("timestamp is missing")
	}
	at, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", text, err)
	}
	return at, nil
}

// OID4VPWalletErrorConcealsReasonValidator requires the Wallet to answer with
// an error and no presentation, without disclosing why: none of params.terms
// may appear in the error, error_description or error_uri it sent.
//
// FCAF WS_RP_SM_IssuerIntegrity__012 asks for an error "without revealing
// reason (revocation of MSO)". Disclosure can only be checked against named
// terms, so this is an approximation: it catches a Wallet that says the
// credential is revoked, not one that paraphrases it.
type OID4VPWalletErrorConcealsReasonValidator struct{}

// ID returns the validator identifier.
func (OID4VPWalletErrorConcealsReasonValidator) ID() string {
	return "oid4vp.wallet_error_conceals_reason"
}

// walletErrorTextMembers are the Authorization Error Response members a Wallet
// can explain itself in (RFC 6749 Section 4.1.2.1).
var walletErrorTextMembers = []string{"error", "error_description", "error_uri"}

// Validate requires an error, no vp_token, and no params.terms in the error.
func (OID4VPWalletErrorConcealsReasonValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Terms []string `json:"terms"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.Terms) == 0 {
		return Result{Status: StatusError, Message: "terms param is required"}
	}
	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: sessionNotObjectMessage}
	}
	if responseValue, _ := findObjectKey(session, "vp_token"); !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned vp_token, expected an error"}
	}
	code := normalizeString(findWalletError(session))
	if code == "" {
		return Result{Status: StatusFail, Message: "wallet returned no error to the verifier"}
	}
	for _, member := range walletErrorTextMembers {
		raw, _ := findObjectKeySkipping(session, member, verifierReplyKeys)
		text := strings.ToLower(normalizeString(raw))
		for _, term := range params.Terms {
			if term != "" && strings.Contains(text, strings.ToLower(term)) {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"wallet error %s reveals the reason: it contains %q",
						member,
						term,
					),
				}
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"wallet returned error %s without a presentation or a stated reason",
			code,
		),
	}
}
