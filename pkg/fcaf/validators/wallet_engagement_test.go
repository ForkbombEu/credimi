// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A Capture verifier session as the tests below need it. The members are the
// ones src/state.ts always writes on a VpSessionCapture: session_id, status,
// request_delivery, response_mode, authorization_request, deeplink, observed,
// checks, events and raw; a DC API session carries dc_api instead of the
// request_uri/response_uri members.
const captureSessionID = "9f6cac61-0b43-452d-afdf-8546ce5e7d68"

func captureEvents(types ...string) []any {
	events := make([]any, 0, len(types))
	for _, eventType := range types {
		events = append(events, map[string]any{
			"at":     "2026-10-13T19:49:37.614Z",
			"type":   eventType,
			"detail": map[string]any{},
		})
	}
	return events
}

func capturePresentationRecord(status string, events ...string) map[string]any {
	return map[string]any{
		"session_id":       captureSessionID,
		"status":           status,
		"request_delivery": captureRequestDeliveryRef,
		"response_mode":    "direct_post.jwt",
		"request_uri":      "https://capture-wallet.example/openid4vp/sessions/" + captureSessionID + "/request",
		"deeplink":         "haip-vp://?client_id=x509_hash%3Aabc&request_uri=https%3A%2F%2Fcapture",
		"authorization_request": map[string]any{
			"response_mode": "direct_post.jwt",
		},
		"observed": map[string]any{
			"request_uri_payload": map[string]any{
				"value":        nil,
				"source":       nil,
				"also_seen_in": []any{},
			},
			"wallet_response": map[string]any{
				"value":        nil,
				"source":       nil,
				"also_seen_in": []any{},
			},
		},
		"checks": map[string]any{"presentation_valid": nil},
		"events": captureEvents(events...),
		"raw": map[string]any{
			"authorization_request": map[string]any{"response_mode": "direct_post.jwt"},
		},
	}
}

// sessionWithEvidence nests arbitrary validator evidence inside a session
// record, which is what a scenario output holding "the session carrying this
// query" looks like to a validator: findObjectKey still reaches the query.
func sessionWithEvidence(session, evidence map[string]any) map[string]any {
	if evidence != nil {
		session["request_evidence"] = evidence
	}
	return session
}

// neverRetrievedSession is a by-reference session the Wallet never fetched: the
// status is still created and the only events are the session and its deeplink.
func neverRetrievedSession(evidence map[string]any) map[string]any {
	return sessionWithEvidence(
		capturePresentationRecord("created", "vp_session_created", "vp_deeplink_generated"),
		evidence,
	)
}

// retrievedSessionWithoutResponse is a by-reference session the Wallet fetched
// and then never answered: Capture records the retrieval and stops there.
func retrievedSessionWithoutResponse(evidence map[string]any) map[string]any {
	session := capturePresentationRecord(
		"request_retrieved",
		"vp_session_created",
		"vp_deeplink_generated",
		eventRequestRetrieved,
	)
	session["raw"].(map[string]any)["request_uri_http"] = map[string]any{"method": "POST"}
	session["observed"].(map[string]any)["request_uri_payload"] = map[string]any{
		"value":        map[string]any{"wallet_nonce": "nonce"},
		"source":       "request_uri.post",
		"also_seen_in": []any{},
	}
	return sessionWithEvidence(session, evidence)
}

// retrievedSessionWithPresentation is the same session after the Wallet posted a
// presentation, which Capture records decrypted. The presentation defaults to an
// unusable token, which is all the modes that only count entries need.
func retrievedSessionWithPresentation(
	evidence map[string]any,
	credentialID string,
	presentation ...string,
) map[string]any {
	token := "ey.e30.sig~"
	if len(presentation) > 0 {
		token = presentation[0]
	}
	presented := map[string]any{}
	for key, value := range evidence {
		presented[key] = value
	}
	presented["vp_token"] = map[string]any{credentialID: []any{token}}
	session := retrievedSessionWithoutResponse(presented)
	session["status"] = "presentation_invalid"
	session["raw"].(map[string]any)["presentation_response_decrypted"] = map[string]any{
		"state":    "state",
		"vp_token": map[string]any{credentialID: []any{token}},
	}
	return session
}

// dcAPICaptureRecord is a DC API session as Capture writes it, with the
// invocation report the presentation page posts (or none).
func dcAPICaptureRecord(invocation map[string]any) map[string]any {
	dcAPI := map[string]any{
		"request":         map[string]any{"protocol": "openid4vp-v1-unsigned"},
		"expected_origin": "https://capture-wallet.example",
		"expires_at":      "2026-10-13T20:00:00.000Z",
	}
	if invocation != nil {
		dcAPI["invocation"] = invocation
	}
	session := capturePresentationRecord("created", "vp_session_created", "vp_deeplink_generated")
	session["request_delivery"] = "by_value"
	session["dc_api"] = dcAPI
	delete(session, "request_uri")
	delete(session, "raw")
	session["raw"] = map[string]any{}
	return session
}

// dcAPIUnsignedRequest sets the delivered DC API request to the unsigned
// protocol, which is what a request-shape-only definition asserts.
func dcAPIUnsignedRequest(session map[string]any) map[string]any {
	session["dc_api"].(map[string]any)["request"] = map[string]any{
		"protocol": "openid4vp-v1-unsigned",
	}
	return session
}

func TestCaptureVerifierSessionRecognition(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "by-reference session", value: capturePresentationRecord("created"), want: true},
		{name: "dc api session", value: dcAPICaptureRecord(nil), want: true},
		{
			name:  "session nested in another evidence value",
			value: map[string]any{"session": capturePresentationRecord("created")},
			want:  true,
		},
		{
			name: "OpenID4VCI issuance session",
			value: map[string]any{
				"session_id": captureSessionID, "status": "created",
				"flow": "authorization_code", "credential_offer_mode": "credential_offer",
			},
			want: false,
		},
		{
			name:  "delivered request object",
			value: map[string]any{"client_id": "x509_hash:abc", "response_mode": "direct_post.jwt"},
			want:  false,
		},
		{name: "status only", value: map[string]any{"status": "created"}, want: false},
		{name: "nil evidence", value: nil, want: false},
		{name: "compact JWS string", value: "ey.e30.sig", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, found := captureVerifierSession(test.value)
			require.Equal(t, test.want, found)
		})
	}
}

func TestWalletEngagementFromEvidence(t *testing.T) {
	errorSession := capturePresentationRecord(
		"request_retrieved",
		"vp_session_created",
		eventRequestRetrieved,
		eventPresentationReceived,
	)
	errorSession["raw"].(map[string]any)["presentation_response_decrypted"] = map[string]any{
		"error": "invalid_request",
	}
	byValueSession := capturePresentationRecord("created", "vp_session_created")
	byValueSession["request_delivery"] = "by_value"
	plainSession := capturePresentationRecord("created", "vp_session_created")
	plainSession["request_delivery"] = "plain"
	postRetrievalSession := capturePresentationRecord("request_retrieved", "vp_session_created")
	postRetrievalSession["raw"].(map[string]any)["request_uri_http"] = map[string]any{
		"method": "POST",
	}
	postRetrievalSession["observed"].(map[string]any)["request_uri_payload"] = map[string]any{
		"value": map[string]any{
			"wallet_nonce": "nonce",
		},
		"source":       "request_uri.post",
		"also_seen_in": []any{},
	}
	trimmedStatusSession := capturePresentationRecord("request_retrieved", "vp_session_created")

	tests := []struct {
		name  string
		value any
		want  engagementState
	}{
		{
			name:  "never retrieved",
			value: neverRetrievedSession(nil),
			want:  engagementNotEstablished,
		},
		{
			name: "retrieval event",
			value: capturePresentationRecord(
				"request_retrieved", "vp_session_created", eventRequestRetrieved,
			),
			want: engagementEstablished,
		},
		{name: "retrieval HTTP envelope", value: postRetrievalSession, want: engagementEstablished},
		{
			name:  "presentation captured with a wallet error",
			value: errorSession,
			want:  engagementEstablished,
		},
		{
			name:  "trimmed record with a progressed status",
			value: trimmedStatusSession,
			want:  engagementEstablished,
		},
		{
			name: "decoded presentation without a status change",
			value: func() map[string]any {
				session := retrievedSessionWithoutResponse(nil)
				session["decoded_presentations"] = map[string]any{}
				return session
			}(),
			want: engagementEstablished,
		},
		{
			name:  "by_value delivery cannot be observed",
			value: byValueSession,
			want:  engagementUnknown,
		},
		{name: "plain delivery cannot be observed", value: plainSession, want: engagementUnknown},
		{
			name:  "dc api with no invocation report",
			value: dcAPICaptureRecord(nil),
			want:  engagementNotEstablished,
		},
		{
			name: "dc api with an unavailable browser API",
			value: dcAPICaptureRecord(map[string]any{
				"outcome":           dcAPIOutcomeAPIUnavailable,
				"response_returned": false,
				"vp_token_present":  false,
			}),
			want: engagementNotEstablished,
		},
		{
			name: "dc api with a reported refusal",
			value: dcAPICaptureRecord(map[string]any{
				"outcome":           dcAPIOutcomeRejected,
				"response_returned": false,
				"vp_token_present":  false,
			}),
			want: engagementEstablished,
		},
		{
			name: "dc api with a captured response",
			value: func() map[string]any {
				session := dcAPICaptureRecord(nil)
				session["raw"] = map[string]any{
					"presentation_response_http": map[string]any{"method": "POST"},
				}
				return session
			}(),
			want: engagementEstablished,
		},
		{
			name:  "not a session record",
			value: map[string]any{"dcql_query": map[string]any{}, "vp_token": map[string]any{}},
			want:  engagementUnknown,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, walletEngagementFromEvidence(test.value).state)
		})
	}
}

// TestGuardWalletEngagementOnlyConvertsPasses pins the guard's contract: it
// speaks for a pass and for nothing else, so a verdict the validator reached on
// other grounds survives untouched.
func TestGuardWalletEngagementOnlyConvertsPasses(t *testing.T) {
	never := neverRetrievedSession(nil)
	engaged := retrievedSessionWithoutResponse(nil)

	tests := []struct {
		name   string
		value  any
		result Result
		want   Result
	}{
		{
			name:   "pass without an engaged wallet is blocked",
			value:  never,
			result: Result{Status: StatusPass, Message: "wallet returned no presentation"},
			want: Result{
				Status:  StatusBlocked,
				Message: walletNeverRetrievedMessage,
				Details: map[string]any{
					"wallet_engagement": "not_established",
					"session_id":        captureSessionID,
				},
			},
		},
		{
			name:   "pass on an engaged wallet is kept",
			value:  engaged,
			result: Result{Status: StatusPass, Message: "wallet returned no presentation"},
			want:   Result{Status: StatusPass, Message: "wallet returned no presentation"},
		},
		{
			name:   "fail is kept",
			value:  never,
			result: Result{Status: StatusFail, Message: "wallet presented a credential"},
			want:   Result{Status: StatusFail, Message: "wallet presented a credential"},
		},
		{
			name:   "error is kept",
			value:  never,
			result: Result{Status: StatusError, Message: "mode param is required"},
			want:   Result{Status: StatusError, Message: "mode param is required"},
		},
		{
			name:  "not_applicable is kept",
			value: never,
			result: Result{
				Status:  StatusNotApplicable,
				Message: "wallet supports the hash function",
			},
			want: Result{
				Status:  StatusNotApplicable,
				Message: "wallet supports the hash function",
			},
		},
		{
			name:   "blocked is kept",
			value:  never,
			result: Result{Status: StatusBlocked, Message: "already blocked"},
			want:   Result{Status: StatusBlocked, Message: "already blocked"},
		},
		{
			name:   "pass without a session record is kept",
			value:  map[string]any{"status": "created"},
			result: Result{Status: StatusPass, Message: "wallet returned no credential"},
			want:   Result{Status: StatusPass, Message: "wallet returned no credential"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, guardWalletEngagement(test.value, test.result))
		})
	}
}
