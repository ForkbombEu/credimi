// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"sort"
	"strings"
)

// Messages the guard reports when a pass rests on a Capture verifier session
// record that shows no Wallet ever engaged with the Authorization Request.
const (
	walletNeverRetrievedMessage = "wallet never retrieved the request, so its rejection is not evidenced"
	walletNeverInvokedMessage   = "the wallet was never invoked through the Digital Credentials API, " +
		"so its rejection is not evidenced"
	walletNeverInvokedAPIMissingMessage = "the browser exposed no Digital Credentials API, " +
		"so the wallet was never invoked and its rejection is not evidenced"
)

// Capture session members and event types the guard reads. The names are the
// ones Capture Wallet writes (src/state.ts, src/server.ts): a verifier session
// record always carries session_id, status and request_delivery; the status
// moves off created only when the Wallet retrieves the request or answers it.
const (
	captureSessionStatusCreated = "created"
	captureRequestDeliveryRef   = "by_reference"
	eventRequestRetrieved       = "vp_request_retrieved"
	eventPresentationReceived   = "vp_presentation_response_received"
)

// engagementState is what one Capture verifier session record proves about the
// Wallet's participation in the presentation exchange.
type engagementState int

const (
	// engagementUnknown: the evidence carries no Capture verifier session
	// record, so nothing can be concluded about the Wallet from it.
	engagementUnknown engagementState = iota
	// engagementNotEstablished: the record shows the Wallet never received the
	// request, so no Wallet response to it can exist.
	engagementNotEstablished
	// engagementEstablished: the record shows the Wallet retrieved the request
	// or was invoked with it.
	engagementEstablished
)

// walletEngagement is the conclusion drawn from the evidence a validator is
// about to pass on.
type walletEngagement struct {
	state     engagementState
	sessionID string
	message   string
}

// guardWalletEngagement turns a pass into StatusBlocked when the evidence is a
// Capture verifier session record that shows the Wallet never engaged with the
// Authorization Request.
//
// A validator whose pass states what the Wallet did with a request — that it
// rejected it, presented nothing, or matched no credential — has no evidence at
// all once the Wallet never received the request: the request never left the
// verifier, and a Wallet that was never shown it cannot have refused it. Every
// other result is returned unchanged: fails, errors and blocks, and passes
// whose evidence is not a session record (a delivered request object, a
// decrypted response, a bare status) or whose session shows the Wallet did
// engage. With by_value or plain delivery Capture records no retrieval, so the
// guard stays silent there and the gap is documented instead of guessed.
func guardWalletEngagement(value any, result Result) Result {
	if result.Status != StatusPass {
		return result
	}
	engagement := walletEngagementFromEvidence(value)
	if engagement.state != engagementNotEstablished {
		return result
	}
	details := map[string]any{"wallet_engagement": "not_established"}
	if engagement.sessionID != "" {
		details["session_id"] = engagement.sessionID
	}
	return Result{
		Status:  StatusBlocked,
		Message: engagement.message,
		Details: details,
	}
}

// walletEngagementFromEvidence finds the Capture verifier session record the
// evidence carries and classifies what it proves about the Wallet.
func walletEngagementFromEvidence(value any) walletEngagement {
	session, found := captureVerifierSession(value)
	if !found {
		return walletEngagement{state: engagementUnknown}
	}
	identifier, _ := session["session_id"].(string)
	if dcAPI, ok := normalizeJSONObject(session["dc_api"]); ok {
		return dcAPIEngagement(session, dcAPI, identifier)
	}
	return redirectEngagement(session, identifier)
}

// captureVerifierSession returns the Capture OpenID4VP presentation session
// record the evidence carries. A test binds either the record itself or a value
// that wraps it, so nested objects are searched in a stable order; a session
// object also appears inside another session's members, and the first record
// found is the one the evidence is about.
func captureVerifierSession(value any) (map[string]any, bool) {
	if text, ok := value.(string); ok && !strings.HasPrefix(strings.TrimSpace(text), "{") {
		// A compact JWS or JWE is never a session record, and parsing it as
		// JSON would be wasted work on every guarded assertion.
		return nil, false
	}
	body, ok := normalizeJSONObject(value)
	if !ok {
		return nil, false
	}
	if isCaptureVerifierSession(body) {
		return body, true
	}
	names := make([]string, 0, len(body))
	for name := range body {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if nested, found := captureVerifierSession(body[name]); found {
			return nested, true
		}
		items, ok := body[name].([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			if nested, found := captureVerifierSession(item); found {
				return nested, true
			}
		}
	}
	return nil, false
}

// isCaptureVerifierSession reports whether an object is a Capture OpenID4VP
// presentation session record. session_id, status and request_delivery are the
// three members Capture always sets on such a record and never on an OpenID4VCI
// issuance session, so the guard cannot mistake one for the other.
func isCaptureVerifierSession(object map[string]any) bool {
	identifier, ok := object["session_id"].(string)
	if !ok || identifier == "" {
		return false
	}
	if _, ok := object["status"].(string); !ok {
		return false
	}
	delivery, ok := object["request_delivery"].(string)
	return ok && delivery != ""
}

// captureSessionRecordedAResponse reports whether a Capture verifier session
// records a Wallet Authorization Response. Capture writes the response and the
// judgement on it together (server.ts captureVpResponse writes
// observed.wallet_response, the raw.presentation_response* members and the
// vp_token they carry), so a session that carries one of them while its checks
// still hold Capture's initial values (state.ts) is a session whose response was
// never judged.
func captureSessionRecordedAResponse(session map[string]any) bool {
	if _, found := findObjectKey(session, "vp_token"); found {
		return true
	}
	if observed, ok := normalizeJSONObject(session["observed"]); ok {
		if response, ok := normalizeJSONObject(observed["wallet_response"]); ok &&
			!isEmptyDCQLValue(response["value"]) {
			return true
		}
	}
	raw, ok := normalizeJSONObject(session["raw"])
	if !ok {
		return false
	}
	for _, key := range []string{
		"presentation_response",
		"presentation_response_decrypted",
		"presentation_response_raw",
	} {
		if !isEmptyDCQLValue(raw[key]) {
			return true
		}
	}
	return false
}

// redirectEngagement classifies a session Capture reaches over request_uri
// retrieval. Only by-reference delivery is observable: with by_value or plain
// delivery the Wallet is handed the request inside the deeplink or the DC API
// call itself, so Capture records no retrieval for a Wallet that never opened
// it and the guard reports nothing rather than guessing.
func redirectEngagement(session map[string]any, identifier string) walletEngagement {
	if normalizeString(session["request_delivery"]) != captureRequestDeliveryRef {
		return walletEngagement{state: engagementUnknown}
	}
	if sessionRetrievedRequest(session) || sessionWalletResponseCaptured(session) {
		return walletEngagement{state: engagementEstablished}
	}
	switch normalizeString(session["status"]) {
	case captureSessionStatusCreated:
		return walletEngagement{
			state:     engagementNotEstablished,
			sessionID: identifier,
			message:   walletNeverRetrievedMessage,
		}
	case "":
		// Without a status the guard cannot tell whether the record was trimmed.
		return walletEngagement{state: engagementUnknown}
	default:
		// Capture moves the status off created only once it has served the
		// Request Object or captured an Authorization Response, so any other
		// value is itself a record that the Wallet engaged.
		return walletEngagement{state: engagementEstablished}
	}
}

// dcAPIEngagement classifies a Digital Credentials API session, where the User
// Agent mediates the invocation and Capture learns about it only from the
// presentation page's report. An outcome of api_unavailable says the browser
// exposed no Digital Credentials API, so no Wallet ran at all.
func dcAPIEngagement(
	session, dcAPI map[string]any,
	identifier string,
) walletEngagement {
	if sessionWalletResponseCaptured(session) {
		return walletEngagement{state: engagementEstablished}
	}
	invocation, reported := normalizeJSONObject(dcAPI["invocation"])
	if !reported {
		return walletEngagement{
			state:     engagementNotEstablished,
			sessionID: identifier,
			message:   walletNeverInvokedMessage,
		}
	}
	if normalizeString(invocation["outcome"]) == dcAPIOutcomeAPIUnavailable {
		return walletEngagement{
			state:     engagementNotEstablished,
			sessionID: identifier,
			message:   walletNeverInvokedAPIMissingMessage,
		}
	}
	return walletEngagement{state: engagementEstablished}
}

// sessionRetrievedRequest reports whether the session records the Wallet
// fetching the Request Object: the retrieval event, the retrieval HTTP envelope,
// or the POST payload Capture receives with it.
func sessionRetrievedRequest(session map[string]any) bool {
	if sessionEventRecorded(session, eventRequestRetrieved) {
		return true
	}
	if raw, ok := normalizeJSONObject(session["raw"]); ok {
		if _, present := raw["request_uri_http"]; present {
			return true
		}
	}
	observed, ok := normalizeJSONObject(session["observed"])
	if !ok {
		return false
	}
	payload, ok := normalizeJSONObject(observed["request_uri_payload"])
	return ok && !isEmptyDCQLValue(payload["value"])
}

// sessionWalletResponseCaptured reports whether the session records an
// Authorization Response from the Wallet, whatever its verdict: a posted
// presentation, a Wallet error, or an encrypted response that was captured.
func sessionWalletResponseCaptured(session map[string]any) bool {
	if sessionEventRecorded(session, eventPresentationReceived) {
		return true
	}
	if _, present := session["decoded_presentations"]; present {
		return true
	}
	if raw, ok := normalizeJSONObject(session["raw"]); ok {
		for _, name := range []string{
			"presentation_response_http",
			"presentation_response_decrypted",
		} {
			if _, present := raw[name]; present {
				return true
			}
		}
	}
	observed, ok := normalizeJSONObject(session["observed"])
	if !ok {
		return false
	}
	response, ok := normalizeJSONObject(observed["wallet_response"])
	return ok && !isEmptyDCQLValue(response["value"])
}

// sessionEventRecorded reports whether the session logged one event type.
func sessionEventRecorded(session map[string]any, eventType string) bool {
	events, ok := session["events"].([]any)
	if !ok {
		return false
	}
	for _, rawEvent := range events {
		event, ok := normalizeJSONObject(rawEvent)
		if !ok {
			continue
		}
		if normalizeString(event["type"]) == eventType {
			return true
		}
	}
	return false
}
