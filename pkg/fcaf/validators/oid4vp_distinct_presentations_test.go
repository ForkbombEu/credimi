// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

const testPIDVCT = "urn:eudi:pid:1"

// testSDJWT builds an SD-JWT presentation that discloses exactly the given
// claims, so a validator reading Claims sees them and nothing else.
func testSDJWT(t *testing.T, vct string, claims map[string]any) string {
	t.Helper()
	encode := func(value any) string {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	names := make([]string, 0, len(claims))
	for name := range claims {
		names = append(names, name)
	}
	sort.Strings(names)
	disclosures := make([]string, 0, len(names))
	digests := make([]any, 0, len(names))
	for _, name := range names {
		disclosure := encode([]any{"c2FsdA", name, claims[name]})
		digest := sha256.Sum256([]byte(disclosure))
		disclosures = append(disclosures, disclosure)
		digests = append(digests, base64.RawURLEncoding.EncodeToString(digest[:]))
	}
	token := encode(map[string]any{"alg": "ES256", "typ": "dc+sd-jwt"}) + "." +
		encode(map[string]any{"vct": vct, "_sd": digests, "_sd_alg": "sha-256"}) + ".c2ln"
	for _, disclosure := range disclosures {
		token += "~" + disclosure
	}
	return token + "~"
}

func testPIDExchange(vct string, tokens ...string) map[string]any {
	entries := make([]any, 0, len(tokens))
	for _, token := range tokens {
		entries = append(entries, token)
	}
	return map[string]any{
		"authorization_request": map[string]any{
			"dcql_query": map[string]any{"credentials": []any{map[string]any{
				"id":     "pid",
				"format": "dc+sd-jwt",
				"meta":   map[string]any{"vct_values": []any{vct}},
			}}},
		},
		"observed": map[string]any{"wallet_response": map[string]any{
			"value": map[string]any{"vp_token": map[string]any{"pid": entries}},
		}},
	}
}

func TestOID4VPDistinctPresentationsValidator(t *testing.T) {
	first := testSDJWT(t, testPIDVCT, map[string]any{"document_number": "AA-1"})
	second := testSDJWT(t, testPIDVCT, map[string]any{"document_number": "BB-2"})
	foreign := testSDJWT(t, "urn:credimi:degree:1", map[string]any{"document_number": "CC-3"})
	withoutDiscriminator := testSDJWT(t, testPIDVCT, map[string]any{"given_name": "Mario"})

	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "two exchanges returning distinct credentials",
			value: []any{
				testPIDExchange(testPIDVCT, first),
				testPIDExchange(testPIDVCT, second),
			},
			wantStatus: StatusPass,
		},
		{
			name:       "one exchange returning two distinct credentials",
			value:      testPIDExchange(testPIDVCT, first, second),
			wantStatus: StatusPass,
		},
		{
			name: "the same credential presented twice",
			value: []any{
				testPIDExchange(testPIDVCT, first),
				testPIDExchange(testPIDVCT, first),
			},
			wantStatus: StatusFail,
		},
		{
			name:       "only one credential available",
			value:      testPIDExchange(testPIDVCT, first),
			wantStatus: StatusFail,
		},
		{
			name: "a presentation of another credential type",
			value: []any{
				testPIDExchange(testPIDVCT, first),
				testPIDExchange(testPIDVCT, foreign),
			},
			wantStatus: StatusFail,
		},
		{
			name: "the discriminating claim is not disclosed",
			value: []any{
				testPIDExchange(testPIDVCT, first),
				testPIDExchange(testPIDVCT, withoutDiscriminator),
			},
			wantStatus: StatusFail,
		},
		{
			name:       "the query does not select the requested credential type",
			value:      testPIDExchange("urn:credimi:degree:1", first),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPDistinctPresentationsValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value: test.value,
				Params: map[string]any{
					"vct":     testPIDVCT,
					"claim":   "document_number",
					"minimum": 2,
				},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}

// TestSDJWTClaimResolvesDecodedVPToken pins the evidence shape the capture
// pipeline binds: a decoded vp_token object rather than a compact token or a
// bare claims map.
func TestSDJWTClaimResolvesDecodedVPToken(t *testing.T) {
	token := testSDJWT(t, testPIDVCT, map[string]any{"given_name": "Mario"})
	vpToken := map[string]any{"pid": []any{token}}

	result := SDJWTClaimPresentValidator{}.Validate(context.Background(), Input{
		Value:  vpToken,
		Params: map[string]any{"claim": "given_name"},
	})
	require.Equal(t, StatusPass, result.Status, result.Message)

	missing := SDJWTClaimPresentValidator{}.Validate(context.Background(), Input{
		Value:  vpToken,
		Params: map[string]any{"claim": "family_name"},
	})
	require.Equal(t, StatusFail, missing.Status, missing.Message)
}

func TestDCQLMultipleFalseMode(t *testing.T) {
	token := testSDJWT(t, testPIDVCT, map[string]any{"document_number": "AA-1"})
	second := testSDJWT(t, testPIDVCT, map[string]any{"document_number": "BB-2"})

	exchange := func(multiple any, tokens ...string) map[string]any {
		record := testPIDExchange(testPIDVCT, tokens...)
		request, _ := record["authorization_request"].(map[string]any)
		query, _ := request["dcql_query"].(map[string]any)
		credentials, _ := query["credentials"].([]any)
		credential, _ := credentials[0].(map[string]any)
		credential["claims"] = []any{map[string]any{"path": []any{"document_number"}}}
		if multiple != nil {
			credential["multiple"] = multiple
		}
		return record
	}

	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name:       "explicit false with one presentation",
			value:      exchange(false, token),
			wantStatus: StatusPass,
		},
		{
			name:       "explicit false with two presentations",
			value:      exchange(false, token, second),
			wantStatus: StatusFail,
		},
		{
			name:       "multiple omitted",
			value:      exchange(nil, token),
			wantStatus: StatusFail,
		},
		{
			name:       "multiple set to true",
			value:      exchange(true, token),
			wantStatus: StatusFail,
		},
	}

	validator := DCQLResponseConstraintsValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"mode": "multiple_false"},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
