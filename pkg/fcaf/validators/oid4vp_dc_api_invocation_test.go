// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// encodeTestKBJWT builds the Key Binding JWT that terminates a presentation.
func encodeTestKBJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	encode := func(value any) string {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return encode(map[string]any{"alg": "ES256", "typ": "kb+jwt"}) + "." +
		encode(payload) + ".c2ln"
}

func dcAPISession(protocol string, invocation map[string]any, presented bool) map[string]any {
	dcAPI := map[string]any{
		"request":         map[string]any{"protocol": protocol},
		"expected_origin": "https://verifier.example",
	}
	if invocation != nil {
		dcAPI["invocation"] = invocation
	}
	session := map[string]any{"dc_api": dcAPI}
	if presented {
		session["observed"] = map[string]any{"wallet_response": map[string]any{
			"value": map[string]any{"vp_token": map[string]any{"pid": []any{"ey.e30.sig~"}}},
		}}
	}
	return session
}

func dcAPIInvocation(outcome string, vpTokenPresent bool) map[string]any {
	return map[string]any{
		"at":                "2026-09-22T11:00:00Z",
		"outcome":           outcome,
		"response_returned": vpTokenPresent,
		"vp_token_present":  vpTokenPresent,
	}
}

func TestOID4VPDCAPIInvocationValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		params     map[string]any
		wantStatus Status
	}{
		{
			name:       "presentation returned proves invocation",
			value:      dcAPISession("openid4vp-v1-signed", nil, true),
			params:     map[string]any{"invoked": true, "protocol": "openid4vp-v1-signed"},
			wantStatus: StatusPass,
		},
		{
			name: "a refusal still proves the wallet was invoked",
			value: dcAPISession(
				"openid4vp-v1-signed",
				dcAPIInvocation("rejected", false),
				false,
			),
			params:     map[string]any{"invoked": true},
			wantStatus: StatusPass,
		},
		{
			// The environment failure this validator exists to separate: the
			// button press never reached a wallet.
			name: "an unavailable browser API is not an invocation",
			value: dcAPISession(
				"openid4vp-v1-signed",
				dcAPIInvocation("api_unavailable", false),
				false,
			),
			params:     map[string]any{"invoked": true},
			wantStatus: StatusFail,
		},
		{
			name:       "nothing recorded at all is not an invocation",
			value:      dcAPISession("openid4vp-v1-signed", nil, false),
			params:     map[string]any{"invoked": true},
			wantStatus: StatusFail,
		},
		{
			// A definition that names only the protocol asserts what the verifier
			// delivered, not what the Wallet did, so it keeps its pass on a
			// session where no Wallet ran at all.
			name:       "a request-shape-only definition keeps its pass",
			value:      dcAPIUnsignedRequest(dcAPICaptureRecord(nil)),
			params:     map[string]any{"protocol": "openid4vp-v1-unsigned"},
			wantStatus: StatusPass,
		},
		{
			name: "a request-shape-only definition keeps its pass without a browser API",
			value: dcAPIUnsignedRequest(
				dcAPICaptureRecord(dcAPIInvocation(dcAPIOutcomeAPIUnavailable, false)),
			),
			params:     map[string]any{"protocol": "openid4vp-v1-unsigned"},
			wantStatus: StatusPass,
		},
		{
			// The wallet-outcome half: an invocation was required and the
			// session records none, which the validator reports itself.
			name:  "a required invocation that was never reported fails",
			value: dcAPIUnsignedRequest(dcAPICaptureRecord(nil)),
			params: map[string]any{
				"allowed_outcomes": []any{"rejected"},
				"vp_token_present": false,
			},
			wantStatus: StatusFail,
		},
		{
			name: "a reported refusal satisfies the wallet-outcome assertion",
			value: dcAPIUnsignedRequest(
				dcAPICaptureRecord(dcAPIInvocation(dcAPIOutcomeRejected, false)),
			),
			params: map[string]any{
				"allowed_outcomes": []any{"rejected", "no_vp_token", "failed"},
				"vp_token_present": false,
			},
			wantStatus: StatusPass,
		},
		{
			name: "a browser without an API never invoked a wallet",
			value: dcAPIUnsignedRequest(
				dcAPICaptureRecord(dcAPIInvocation(dcAPIOutcomeAPIUnavailable, false)),
			),
			params:     map[string]any{"invoked": true},
			wantStatus: StatusFail,
		},
		{
			// The wallet-outcome path the guard exists for: the validator reads a
			// vp_token anywhere in the record as proof of invocation, while the
			// capture record itself reports no invocation and no captured
			// response, so the pass has no evidence behind it.
			name: "a wallet-outcome pass with no captured response is blocked",
			value: func() map[string]any {
				session := dcAPIUnsignedRequest(dcAPICaptureRecord(nil))
				session["request_evidence"] = map[string]any{
					"vp_token": map[string]any{"pid": []any{"ey.e30.sig~"}},
				}
				return session
			}(),
			params:     map[string]any{"invoked": true},
			wantStatus: StatusBlocked,
		},
		{
			name:       "a redirect session is not a DC API session",
			value:      map[string]any{"response_mode": "direct_post"},
			params:     map[string]any{"invoked": true},
			wantStatus: StatusFail,
		},
		{
			name: "a session without an expected origin cannot bind the response",
			value: map[string]any{"dc_api": map[string]any{
				"request": map[string]any{"protocol": "openid4vp-v1-signed"},
			}},
			params:     map[string]any{"invoked": true},
			wantStatus: StatusFail,
		},
		{
			name:       "the delivered protocol was the unsigned one",
			value:      dcAPISession("openid4vp-v1-unsigned", nil, true),
			params:     map[string]any{"protocol": "openid4vp-v1-signed"},
			wantStatus: StatusFail,
		},
		{
			name: "refusal outcome is one of the accepted ones",
			value: dcAPISession(
				"openid4vp-v1-signed",
				dcAPIInvocation("no_vp_token", false),
				false,
			),
			params: map[string]any{
				"allowed_outcomes": []any{"rejected", "no_vp_token", "failed"},
				"vp_token_present": false,
			},
			wantStatus: StatusPass,
		},
		{
			name: "an unavailable API is not an accepted refusal",
			value: dcAPISession(
				"openid4vp-v1-signed",
				dcAPIInvocation("api_unavailable", false),
				false,
			),
			params: map[string]any{
				"allowed_outcomes": []any{"rejected", "no_vp_token", "failed"},
			},
			wantStatus: StatusFail,
		},
		{
			name:  "the wallet answered with a vp_token after all",
			value: dcAPISession("openid4vp-v1-signed", dcAPIInvocation("no_vp_token", true), false),
			params: map[string]any{
				"allowed_outcomes": []any{"rejected", "no_vp_token", "failed"},
				"vp_token_present": false,
			},
			wantStatus: StatusFail,
		},
		{
			name:       "an outcome was required but none was reported",
			value:      dcAPISession("openid4vp-v1-signed", nil, true),
			params:     map[string]any{"allowed_outcomes": []any{"rejected"}},
			wantStatus: StatusFail,
		},
		{
			name: "an unknown outcome name is a definition error",
			value: dcAPISession(
				"openid4vp-v1-signed",
				dcAPIInvocation("rejected", false),
				false,
			),
			params:     map[string]any{"allowed_outcomes": []any{"cancelled"}},
			wantStatus: StatusError,
		},
	}

	validator := OID4VPDCAPIInvocationValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(
				context.Background(),
				Input{Value: test.value, Params: test.params},
			)
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}

func TestSDJWTKBJWTClaimStringPrefixValidator(t *testing.T) {
	presentation := func(audience any) map[string]any {
		token := testSDJWT(t, map[string]any{"given_name": "Mario"})
		if audience != nil {
			token += encodeTestKBJWT(t, map[string]any{"nonce": "n", "aud": audience})
		}
		return map[string]any{"pid": []any{token}}
	}

	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{"origin-prefixed audience", presentation("origin:https://verifier.example"), StatusPass},
		{"client identifier audience", presentation("x509_hash:abc"), StatusFail},
		{"non-string audience", presentation([]any{"origin:https://verifier.example"}), StatusFail},
		{"no key binding jwt", presentation(nil), StatusFail},
		{"evidence carries no presentation", "not-a-presentation", StatusFail},
	}

	validator := SDJWTKBJWTClaimStringPrefixValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"claim": "aud", "prefix": "origin:"},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
