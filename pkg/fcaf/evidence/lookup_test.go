// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupEvidenceSlot(t *testing.T) {
	bundle := Bundle{
		DecodedSDJWT: map[string]any{
			"email": "person@example.test",
		},
	}

	got := Lookup(bundle, "evidence.decoded_sdjwt.email")

	require.True(t, got.Found)
	require.Equal(t, "person@example.test", got.Value)
}

func TestExtractOutputPathAndDecodeSDJWT(t *testing.T) {
	root := map[string]any{
		"output": map[string]any{
			"http-get-verifier-backend.eudiw.dev-0007": map[string]any{
				"outputs": map[string]any{
					"body": map[string]any{
						"vp_token": map[string]any{
							"query_0": []any{
								"eyJhbGciOiJub25lIn0.eyJfc2QiOlsiTmRUemVld0RjZVRJOXNQVGdRdjBRUG1oU1JZaVQ5cnJwOTB3OE5TY2ZCYyJdLCJ2Y3QiOiJ1cm46ZXVkaTpwaWQ6MSIsImlzcyI6Imh0dHBzOi8vaXNzdWVyLmV4YW1wbGUifQ~WyJzYWx0IiwiZW1haWwiLCJwZXJzb25AZXhhbXBsZS50ZXN0Il0~",
							},
						},
					},
				},
			},
		},
	}

	value, err := Extract(
		root,
		"$.output.http-get-verifier-backend.eudiw.dev-0007.outputs.body.vp_token.query_0[0]",
		"sdjwt.presentation",
	)

	require.NoError(t, err)
	presentation, ok := value.(*SDJWTPresentation)
	require.True(t, ok)
	require.Equal(t, "person@example.test", presentation.Claims["email"])
}

func TestExtractAndDecodeAllSDJWTPresentations(t *testing.T) {
	token := "eyJhbGciOiJub25lIn0.eyJfc2QiOlsiTmRUemVld0RjZVRJOXNQVGdRdjBRUG1oU1JZaVQ5cnJwOTB3OE5TY2ZCYyJdLCJ2Y3QiOiJ1cm46ZXVkaTpwaWQ6MSIsImlzcyI6Imh0dHBzOi8vaXNzdWVyLmV4YW1wbGUifQ~WyJzYWx0IiwiZW1haWwiLCJwZXJzb25AZXhhbXBsZS50ZXN0Il0~"
	root := map[string]any{"query_0": []any{token, token}}

	value, err := Extract(root, "$.query_0", "sdjwt.presentations")

	require.NoError(t, err)
	presentations, ok := value.([]*SDJWTPresentation)
	require.True(t, ok)
	require.Len(t, presentations, 2)
	require.Equal(t, "person@example.test", presentations[0].Claims["email"])
	require.Equal(t, "person@example.test", presentations[1].Claims["email"])
}

func TestExtractSDJWTPresentationsRejectsInvalidMembers(t *testing.T) {
	_, err := Extract(
		map[string]any{"query_0": []any{"invalid"}},
		"$.query_0",
		"sdjwt.presentations",
	)
	require.ErrorContains(t, err, "sdjwt.presentations[0]")

	_, err = Extract(map[string]any{"query_0": []any{}}, "$.query_0", "sdjwt.presentations")
	require.ErrorContains(t, err, "input is empty")

	_, err = Extract(map[string]any{"query_0": []any{42}}, "$.query_0", "sdjwt.presentations")
	require.ErrorContains(t, err, "must be a string")
}

func TestExtractVPTokenJSONAndDecodeSDJWT(t *testing.T) {
	root := map[string]any{
		"output": map[string]any{
			"http-get-verifier-backend.eudiw.dev-0006": map[string]any{
				"outputs": map[string]any{
					"body": map[string]any{
						"observed": map[string]any{
							"wallet_response": map[string]any{
								"value": map[string]any{
									"vp_token": `{"query_0":["eyJhbGciOiJub25lIn0.eyJfc2QiOlsiTmRUemVld0RjZVRJOXNQVGdRdjBRUG1oU1JZaVQ5cnJwOTB3OE5TY2ZCYyJdLCJ2Y3QiOiJ1cm46ZXVkaTpwaWQ6MSIsImlzcyI6Imh0dHBzOi8vaXNzdWVyLmV4YW1wbGUifQ~WyJzYWx0IiwiZW1haWwiLCJwZXJzb25AZXhhbXBsZS50ZXN0Il0~"]}`,
								},
							},
						},
					},
				},
			},
		},
	}

	value, err := Extract(
		root,
		"$.output.http-get-verifier-backend.eudiw.dev-0006.outputs.body.observed.wallet_response.value.vp_token",
		"sdjwt.vp_token_json",
	)

	require.NoError(t, err)
	presentation, ok := value.(*SDJWTPresentation)
	require.True(t, ok)
	require.Equal(t, "person@example.test", presentation.Claims["email"])
}

func TestExtractPresentationTokenFromVPTokenJSONAcceptsSingleCredentialKey(t *testing.T) {
	token, err := extractPresentationTokenFromVPTokenJSON(
		`{"urn_eu_europa_ec_eudi_pid_1_mdoc_jwt":["o2d2ZXJzaW9uYzEuMA"]}`,
		"",
	)

	require.NoError(t, err)
	require.Equal(t, "o2d2ZXJzaW9uYzEuMA", token)
}

func TestExtractPresentationTokenFromVPTokenJSONRejectsAmbiguousCredentialKeys(t *testing.T) {
	_, err := extractPresentationTokenFromVPTokenJSON(
		`{"credential_a":["one"],"credential_b":["two"]}`,
		"",
	)

	require.ErrorContains(t, err, "exactly one credential entry")
}

func TestExtractRejectsMissingKey(t *testing.T) {
	_, err := Extract(map[string]any{"output": map[string]any{}}, "$.output.missing", "raw")

	require.ErrorContains(t, err, "missing key")
}

func TestParseSDJWTPresentationReconstructsNestedDisclosures(t *testing.T) {
	country := testDisclosure(t, []any{"country-salt", "country", "IT"})
	address := testDisclosure(t, []any{
		"address-salt",
		"address",
		map[string]any{"_sd": []any{sha256Base64URL(country)}},
	})
	payload := testJWTPart(t, map[string]any{
		"_sd_alg": "sha-256",
		"_sd":     []any{sha256Base64URL(address)},
		"vct":     "urn:eudi:pid:1",
	})
	token := testJWTPart(t, map[string]any{"alg": "none"}) + "." + payload + ".signature~" +
		address + "~" + country + "~"

	presentation, err := ParseSDJWTPresentation(token)

	require.NoError(t, err)
	value, found := presentation.Claim("address.country")
	require.True(t, found)
	require.Equal(t, "IT", value)
	require.Equal(t, 2, presentation.DisclosureCount)
}

func TestParseSDJWTPresentationRejectsUnreferencedDisclosure(t *testing.T) {
	disclosure := testDisclosure(t, []any{"salt", "email", "person@example.test"})
	payload := testJWTPart(t, map[string]any{"_sd": []any{"different-digest"}})
	token := testJWTPart(t, map[string]any{"alg": "none"}) + "." + payload + ".signature~" +
		disclosure + "~"

	_, err := ParseSDJWTPresentation(token)

	require.ErrorContains(t, err, "not referenced")
}

func TestParseSDJWTPresentationRejectsUnsupportedDigestAlgorithm(t *testing.T) {
	payload := testJWTPart(t, map[string]any{"_sd_alg": "sha-512"})
	token := testJWTPart(t, map[string]any{"alg": "none"}) + "." + payload + ".signature~"

	_, err := ParseSDJWTPresentation(token)

	require.ErrorContains(t, err, "unsupported SD-JWT disclosure digest algorithm")
}

func TestLookupResolvesEvidenceSlotsAndRuntime(t *testing.T) {
	slot := func(value string) map[string]any { return map[string]any{"value": value} }
	bundle := Bundle{
		RawRequestObject:            slot("raw-request"),
		DecodedRequestObject:        slot("decoded-request"),
		RawPresentationResponse:     slot("raw-response"),
		DecodedPresentationResponse: slot("decoded-response"),
		VPToken:                     map[string]any{"query_0": []any{"token-a", "token-b"}},
		PresentationSubmission:      slot("submission"),
		DecodedSDJWT:                slot("sdjwt"),
		MDoc:                        slot("mdoc"),
		IssuerMetadata:              slot("issuer"),
		VerifierMetadata:            slot("verifier"),
		AuthorizationServerMetadata: slot("authorization-server"),
		JWKS:                        slot("jwks"),
		Certificates:                slot("certificates"),
		Runner:                      slot("runner"),
		Artifacts:                   slot("artifacts"),
		Runtime:                     map[string]any{"device": map[string]any{"os": "android"}},
		Extra: map[string]any{
			"value":       "extra",
			"custom_slot": map[string]any{"enabled": false, "count": 0},
		},
	}

	tests := []struct {
		path      string
		wantValue any
		wantType  string
	}{
		{path: "evidence.raw_request_object.value", wantValue: "raw-request", wantType: "string"},
		{
			path:      "evidence.decoded_request_object.value",
			wantValue: "decoded-request",
			wantType:  "string",
		},
		{
			path:      "evidence.raw_presentation_response.value",
			wantValue: "raw-response",
			wantType:  "string",
		},
		{
			path:      "evidence.decoded_presentation_response.value",
			wantValue: "decoded-response",
			wantType:  "string",
		},
		{path: "evidence.vp_token.query_0.1", wantValue: "token-b", wantType: "string"},
		{
			path:      "evidence.presentation_submission.value",
			wantValue: "submission",
			wantType:  "string",
		},
		{path: "evidence.decoded_sdjwt.value", wantValue: "sdjwt", wantType: "string"},
		{path: "evidence.mdoc.value", wantValue: "mdoc", wantType: "string"},
		{path: "evidence.issuer_metadata.value", wantValue: "issuer", wantType: "string"},
		{path: "evidence.verifier_metadata.value", wantValue: "verifier", wantType: "string"},
		{
			path:      "evidence.authorization_server_metadata.value",
			wantValue: "authorization-server",
			wantType:  "string",
		},
		{path: "evidence.jwks.value", wantValue: "jwks", wantType: "string"},
		{path: "evidence.certificates.value", wantValue: "certificates", wantType: "string"},
		{path: "evidence.runner.value", wantValue: "runner", wantType: "string"},
		{path: "evidence.artifacts.value", wantValue: "artifacts", wantType: "string"},
		{path: "evidence.extra.value", wantValue: "extra", wantType: "string"},
		{path: "evidence.custom_slot.enabled", wantValue: false, wantType: "bool"},
		{path: "evidence.custom_slot.count", wantValue: 0, wantType: "int"},
		{
			path:      "evidence.vp_token.query_0",
			wantValue: []any{"token-a", "token-b"},
			wantType:  "[]interface {}",
		},
		{path: "runtime.device.os", wantValue: "android", wantType: "string"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := Lookup(bundle, tc.path)
			require.Equal(t, LookupResult{
				Path:  tc.path,
				Found: true,
				Value: tc.wantValue,
				Type:  tc.wantType,
			}, got)
		})
	}
}

func TestLookupReportsMissingValues(t *testing.T) {
	bundle := Bundle{
		VPToken:      map[string]any{"query_0": []any{"token-a", nil}},
		DecodedSDJWT: map[string]any{"email": "person@example.test", "address": nil},
		Extra:        map[string]any{"nil_collection": map[string]any(nil)},
	}

	tests := []struct {
		name string
		path string
	}{
		{name: "path without slot", path: "evidence"},
		{name: "unknown root", path: "pipeline.decoded_sdjwt.email"},
		{name: "unknown slot not in extra", path: "evidence.unknown.email"},
		{name: "missing map key", path: "evidence.decoded_sdjwt.family_name"},
		{name: "json null value", path: "evidence.decoded_sdjwt.address"},
		{name: "descend into json null", path: "evidence.decoded_sdjwt.address.country"},
		{name: "descend into scalar", path: "evidence.decoded_sdjwt.email.domain"},
		{name: "non numeric array index", path: "evidence.vp_token.query_0.first"},
		{name: "negative array index", path: "evidence.vp_token.query_0.-1"},
		{name: "array index out of range", path: "evidence.vp_token.query_0.2"},
		{name: "null array element", path: "evidence.vp_token.query_0.1"},
		{name: "unset slot", path: "evidence.mdoc"},
		{name: "unset slot member", path: "evidence.mdoc.version"},
		{name: "typed nil map in extra", path: "evidence.nil_collection"},
		{name: "unset runtime", path: "runtime.device"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Lookup(bundle, tc.path)
			require.Equal(t, LookupResult{Path: tc.path, Found: false}, got)
		})
	}

	got := Lookup(Bundle{}, "evidence.custom_slot.value")
	require.False(t, got.Found, "slots outside the bundle require an Extra map")
}

func TestExtractResolvesPointerPaths(t *testing.T) {
	root := map[string]any{
		"output": map[string]any{
			"step.with.dots": map[string]any{"status": "dotted"},
			"step": map[string]any{
				"with": map[string]any{"dots": map[string]any{"status": "nested"}},
			},
			"matrix": []any{[]any{"a0", "a1"}, []any{"b0", map[string]any{"cell": "b1"}}},
			"out":    "prefix-only",
		},
	}

	tests := []struct {
		name string
		path string
		want any
	}{
		{name: "longest dotted key wins", path: "$.output.step.with.dots.status", want: "dotted"},
		{name: "consecutive indexes", path: "$.output.matrix[0][1]", want: "a1"},
		{name: "index then key", path: "$.output.matrix[1][1].cell", want: "b1"},
		{name: "key that is a prefix of another key", path: "$.output.out", want: "prefix-only"},
		{name: "root object", path: "$.", want: root},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Extract(root, tc.path, "raw")
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestExtractRejectsInvalidPointers(t *testing.T) {
	root := map[string]any{
		"output": map[string]any{
			"items":  []any{"first"},
			"status": "passed",
		},
	}

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "missing root marker",
			path:    "output.status",
			wantErr: `invalid pointer "output.status"`,
		},
		{
			name:    "key prefix without separator",
			path:    "$.outputs.status",
			wantErr: `missing key in path "outputs.status"`,
		},
		{
			name:    "descend into scalar",
			path:    "$.output.status.code",
			wantErr: `invalid pointer segment near "code"`,
		},
		{
			name:    "unterminated index",
			path:    "$.output.items[0",
			wantErr: `invalid pointer index "[0"`,
		},
		{
			name:    "non numeric index",
			path:    "$.output.items[first]",
			wantErr: `invalid pointer index "first"`,
		},
		{
			name:    "index on non array",
			path:    "$.output.status[0]",
			wantErr: `segment "status" is not an array`,
		},
		{
			name:    "index out of range",
			path:    "$.output.items[1]",
			wantErr: "array index 1 out of range",
		},
		{
			name:    "negative index",
			path:    "$.output.items[-1]",
			wantErr: "array index -1 out of range",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Extract(root, tc.path, "raw")
			require.EqualError(t, err, tc.wantErr)
			require.Nil(t, got)
		})
	}
}

func TestExtractDecoders(t *testing.T) {
	sdJWT := testSDJWTWithEmail(t)
	mdoc := base64.RawURLEncoding.EncodeToString(
		testMDocDeviceResponse(t, map[string]any{"family_name": "Trotter"}),
	)
	root := map[string]any{
		"text":            "plain",
		"json_text":       `{"nested":{"ok":true}}`,
		"number":          float64(7),
		"sdjwt_vp":        `{"query_0":["` + sdJWT + `","` + sdJWT + `"]}`,
		"sdjwt_other_key": `{"pid_credential":["` + sdJWT + `"]}`,
		"mdoc":            mdoc,
		"mdoc_vp":         `{"pid_mdoc":["` + mdoc + `"]}`,
	}

	value, err := Extract(root, "$.number", "")
	require.NoError(t, err)
	require.Equal(t, float64(7), value, "an empty decoder returns the raw value")

	value, err = Extract(root, "$.text", "string")
	require.NoError(t, err)
	require.Equal(t, "plain", value)

	value, err = Extract(root, "$.json_text", "json")
	require.NoError(t, err)
	require.Equal(t, map[string]any{"nested": map[string]any{"ok": true}}, value)

	value, err = Extract(root, "$.sdjwt_vp", "sdjwt.vp_token_presentations_json")
	require.NoError(t, err)
	presentations, ok := value.([]*SDJWTPresentation)
	require.True(t, ok)
	require.Len(t, presentations, 2)
	require.Equal(t, "person@example.test", presentations[1].Claims["email"])

	value, err = Extract(root, "$.sdjwt_other_key", "sdjwt.vp_token_json")
	require.NoError(t, err)
	presentation, ok := value.(*SDJWTPresentation)
	require.True(t, ok, "a single credential entry is used when query_0 is absent")
	require.Equal(t, "person@example.test", presentation.Claims["email"])

	value, err = Extract(root, "$.mdoc", "mdoc.presentation")
	require.NoError(t, err)
	mdocPresentation, ok := value.(*MDocPresentation)
	require.True(t, ok)
	element, found := mdocPresentation.Element(pidMDocTestDocType, "family_name")
	require.True(t, found)
	require.Equal(t, "Trotter", element.Value)

	value, err = Extract(root, "$.mdoc_vp", "mdoc.vp_token_json")
	require.NoError(t, err)
	mdocPresentation, ok = value.(*MDocPresentation)
	require.True(t, ok)
	require.Equal(t, pidMDocTestDocType, mdocPresentation.Documents[0].DocType)
}

func TestExtractDecoderErrors(t *testing.T) {
	root := map[string]any{
		"number":     float64(7),
		"bad_json":   `{"unterminated":`,
		"bad_vp":     `not-json`,
		"bad_member": `{"query_0":["not-an-sd-jwt"]}`,
	}

	tests := []struct {
		path    string
		decoder string
		wantErr string
	}{
		{path: "$.number", decoder: "string", wantErr: "wrong decoder string for float64"},
		{path: "$.number", decoder: "json", wantErr: "wrong decoder json for float64"},
		{path: "$.bad_json", decoder: "json", wantErr: "decode json:"},
		{
			path:    "$.number",
			decoder: "sdjwt.presentation",
			wantErr: "wrong decoder sdjwt.presentation for float64",
		},
		{
			path:    "$.number",
			decoder: "sdjwt.presentations",
			wantErr: "wrong decoder sdjwt.presentations for float64",
		},
		{
			path:    "$.number",
			decoder: "sdjwt.vp_token_json",
			wantErr: "wrong decoder sdjwt.vp_token_json for float64",
		},
		{path: "$.bad_vp", decoder: "sdjwt.vp_token_json", wantErr: "decode vp_token json:"},
		{
			path:    "$.number",
			decoder: "sdjwt.vp_token_presentations_json",
			wantErr: "wrong decoder sdjwt.vp_token_presentations_json for float64",
		},
		{
			path:    "$.bad_member",
			decoder: "sdjwt.vp_token_presentations_json",
			wantErr: "decode sdjwt.vp_token_presentations_json[0]: invalid SD-JWT presentation",
		},
		{
			path:    "$.number",
			decoder: "mdoc.vp_token_json",
			wantErr: "wrong decoder mdoc.vp_token_json for float64",
		},
		{path: "$.bad_vp", decoder: "mdoc.vp_token_json", wantErr: "decode vp_token json:"},
		{
			path:    "$.number",
			decoder: "mdoc.presentation",
			wantErr: "mdoc value is float64, expected string or bytes",
		},
		{path: "$.number", decoder: "base64", wantErr: `unsupported decoder "base64"`},
	}

	for _, tc := range tests {
		t.Run(tc.decoder+" "+tc.path, func(t *testing.T) {
			value, err := Extract(root, tc.path, tc.decoder)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, value)
		})
	}
}

func TestParseVPTokenJSONHelpers(t *testing.T) {
	sdJWT := testSDJWTWithEmail(t)
	mdoc := base64.RawURLEncoding.EncodeToString(
		testMDocDeviceResponse(t, map[string]any{"family_name": "Trotter"}),
	)

	presentation, err := ParseSDJWTVPTokenJSON(
		`{"query_0":["` + sdJWT + `"],"query_1":["ignored"]}`,
	)
	require.NoError(t, err)
	require.Equal(t, "person@example.test", presentation.Claims["email"])

	presentations, err := ParseSDJWTVPTokenPresentationsJSON(
		`{"query_0":["` + sdJWT + `","` + sdJWT + `"]}`,
	)
	require.NoError(t, err)
	require.Len(t, presentations, 2)

	mdocPresentation, err := ParseMDocVPTokenJSON(`{"pid":["` + mdoc + `"]}`)
	require.NoError(t, err)
	require.Equal(t, pidMDocTestDocType, mdocPresentation.Documents[0].DocType)
}

func TestParseVPTokenJSONRejectsMalformedTokens(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{name: "not json", raw: `[`, wantErr: "decode vp_token json:"},
		{
			name:    "query_0 missing among several keys",
			raw:     `{"query_1":["a"],"query_2":["b"]}`,
			wantErr: `vp_token json must contain "query_0" or exactly one credential entry`,
		},
		{
			name:    "not an array",
			raw:     `{"query_0":"token"}`,
			wantErr: `vp_token "query_0" must be an array`,
		},
		{name: "empty array", raw: `{"query_0":[]}`, wantErr: `vp_token "query_0" is empty`},
		{
			name:    "non string token",
			raw:     `{"query_0":["ok",1]}`,
			wantErr: `vp_token "query_0"[1] must be a string`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			presentation, err := ParseSDJWTVPTokenJSON(tc.raw)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, presentation)

			presentations, err := ParseSDJWTVPTokenPresentationsJSON(tc.raw)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, presentations)
		})
	}

	_, err := ParseMDocVPTokenJSON(`{}`)
	require.ErrorContains(t, err, "exactly one credential entry")
}

func TestParseSDJWTPresentationWithKeyBinding(t *testing.T) {
	email := testDisclosure(t, []any{"salt", "email", "person@example.test"})
	issuerJWT := testJWTPart(t, map[string]any{"alg": "ES256", "typ": "dc+sd-jwt"}) + "." +
		testJWTPart(t, map[string]any{
			"_sd":     []any{sha256Base64URL(email)},
			"_sd_alg": "sha-256",
			"vct":     "urn:eudi:pid:1",
		}) + ".signature"
	keyBindingJWT := testJWTPart(t, map[string]any{"alg": "ES256", "typ": "kb+jwt"}) + "." +
		testJWTPart(t, map[string]any{"nonce": "n-0S6_WzA2Mj", "aud": "verifier"}) + ".kb-signature"
	token := issuerJWT + "~" + email + "~" + keyBindingJWT

	presentation, err := ParseSDJWTPresentation(token)

	require.NoError(t, err)
	require.Equal(t, token, presentation.Raw)
	require.Equal(t, issuerJWT+"~"+email+"~", presentation.SDJWT)
	require.Equal(t, keyBindingJWT, presentation.KeyBindingJWT)
	require.Equal(t, "n-0S6_WzA2Mj", presentation.KeyBinding["nonce"])
	require.Equal(
		t,
		map[string]any{"alg": "ES256", "typ": "kb+jwt"},
		presentation.KeyBinding["_protected_header"],
	)
	require.Equal(t, "dc+sd-jwt", presentation.ProtectedHeaders["typ"])
	require.Equal(t, map[string]any{
		"email": "person@example.test",
		"vct":   "urn:eudi:pid:1",
	}, presentation.Claims, "_sd and _sd_alg are not exposed as claims")
	require.Equal(t, "sha-256", presentation.IssuerPayload["_sd_alg"])
	require.Equal(t, 1, presentation.DisclosureCount)
}

func TestParseSDJWTPresentationReconstructsArrayDisclosures(t *testing.T) {
	germany := testDisclosure(t, []any{"de-salt", "DE"})
	payload := testJWTPart(t, map[string]any{
		"nationalities": []any{
			map[string]any{"...": sha256Base64URL(germany)},
			map[string]any{"...": "undisclosed-digest"},
			"IT",
			map[string]any{"code": "FR"},
		},
	})
	token := testJWTPart(t, map[string]any{"alg": "none"}) + "." + payload + ".signature~" +
		germany + "~"

	presentation, err := ParseSDJWTPresentation(token)

	require.NoError(t, err)
	value, found := presentation.Claim("nationalities")
	require.True(t, found)
	require.Equal(t, []any{"DE", "IT", map[string]any{"code": "FR"}}, value)
	require.Empty(t, presentation.KeyBindingJWT)
	require.Nil(t, presentation.KeyBinding)
}

func TestParseSDJWTPresentationRejectsMalformedTokens(t *testing.T) {
	header := testJWTPart(t, map[string]any{"alg": "none"})
	jwt := func(payload map[string]any) string {
		return header + "." + testJWTPart(t, payload) + ".signature"
	}
	email := testDisclosure(t, []any{"salt", "email", "person@example.test"})
	emailAgain := testDisclosure(t, []any{"other-salt", "email", "other@example.test"})
	arrayItem := testDisclosure(t, []any{"salt", "DE"})
	nonStringName := testDisclosure(t, []any{"salt", 42, "value"})
	oneElement := testDisclosure(t, []any{"salt"})
	notArray := base64.RawURLEncoding.EncodeToString([]byte(`{"salt":"x"}`))
	invalidUTF8 := base64.RawURLEncoding.EncodeToString([]byte{0xff, 0xfe})
	emailDigest := sha256Base64URL(email)

	tests := []struct {
		name    string
		token   string
		wantErr string
	}{
		{name: "no tilde", token: jwt(map[string]any{}), wantErr: "invalid SD-JWT presentation"},
		{name: "issuer jwt with one segment", token: header + "~", wantErr: "invalid jwt"},
		{name: "issuer header not base64", token: "!!.e30.sig~", wantErr: "decode jwt header"},
		{
			name:    "issuer payload not json",
			token:   header + "." + base64.RawURLEncoding.EncodeToString([]byte("[]")) + ".sig~",
			wantErr: "decode jwt payload",
		},
		{
			name:    "trailing segment is not a jwt",
			token:   jwt(map[string]any{"_sd": []any{emailDigest}}) + "~" + email,
			wantErr: "SD-JWT presentation does not end with a KB-JWT or tilde",
		},
		{
			name:    "key binding jwt not decodable",
			token:   jwt(map[string]any{}) + "~bad.payload.sig",
			wantErr: "decode key binding jwt",
		},
		{
			name:    "empty disclosure in the middle",
			token:   jwt(map[string]any{"_sd": []any{emailDigest}}) + "~~" + email + "~",
			wantErr: "empty SD-JWT disclosure segment",
		},
		{
			name:    "empty disclosure before key binding",
			token:   jwt(map[string]any{}) + "~~" + jwt(map[string]any{"nonce": "n"}),
			wantErr: "empty SD-JWT disclosure segment",
		},
		{
			name: "duplicate disclosure",
			token: jwt(
				map[string]any{"_sd": []any{emailDigest}},
			) + "~" + email + "~" + email + "~",
			wantErr: "duplicate SD-JWT disclosure",
		},
		{
			name:    "disclosure not base64url",
			token:   jwt(map[string]any{}) + "~%%%~",
			wantErr: "decode disclosure",
		},
		{
			name:    "disclosure not utf-8",
			token:   jwt(map[string]any{}) + "~" + invalidUTF8 + "~",
			wantErr: "disclosure is not valid UTF-8",
		},
		{
			name:    "disclosure not a json array",
			token:   jwt(map[string]any{}) + "~" + notArray + "~",
			wantErr: "parse disclosure json",
		},
		{
			name:    "disclosure with wrong arity",
			token:   jwt(map[string]any{}) + "~" + oneElement + "~",
			wantErr: "disclosure must contain 2 or 3 elements",
		},
		{
			name:    "disclosure name not a string",
			token:   jwt(map[string]any{}) + "~" + nonStringName + "~",
			wantErr: "disclosure claim name must be a string",
		},
		{
			name:    "_sd not an array",
			token:   jwt(map[string]any{"_sd": emailDigest}) + "~",
			wantErr: "_sd must be an array",
		},
		{
			name:    "_sd digest not a string",
			token:   jwt(map[string]any{"_sd": []any{1}}) + "~",
			wantErr: "_sd digest must be a string",
		},
		{
			name: "array disclosure referenced from object",
			token: jwt(map[string]any{"_sd": []any{sha256Base64URL(arrayItem)}}) + "~" +
				arrayItem + "~",
			wantErr: "array disclosure referenced from object _sd",
		},
		{
			name: "object disclosure referenced from array",
			token: jwt(map[string]any{
				"nationalities": []any{map[string]any{"...": emailDigest}},
			}) + "~" + email + "~",
			wantErr: "object disclosure referenced from array",
		},
		{
			name: "array digest not a string",
			token: jwt(map[string]any{
				"nationalities": []any{map[string]any{"...": 1}},
			}) + "~",
			wantErr: "array disclosure digest must be a string",
		},
		{
			name: "same claim disclosed twice",
			token: jwt(map[string]any{
				"_sd": []any{emailDigest, sha256Base64URL(emailAgain)},
			}) + "~" + email + "~" + emailAgain + "~",
			wantErr: `duplicate disclosed claim "email"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			presentation, err := ParseSDJWTPresentation(tc.token)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, presentation)
		})
	}
}

func TestSDJWTPresentationClaim(t *testing.T) {
	presentation := &SDJWTPresentation{Claims: map[string]any{
		"email":   "person@example.test",
		"address": map[string]any{"country": "IT", "lines": []any{"Via Roma 1"}},
		"age":     nil,
	}}

	tests := []struct {
		name      string
		claim     string
		wantValue any
		wantFound bool
	}{
		{name: "top level", claim: "email", wantValue: "person@example.test", wantFound: true},
		{name: "nested", claim: "address.country", wantValue: "IT", wantFound: true},
		{name: "present null", claim: "age", wantValue: nil, wantFound: true},
		{name: "missing", claim: "family_name"},
		{name: "missing nested", claim: "address.locality"},
		{name: "through scalar", claim: "email.domain"},
		{name: "through array", claim: "address.lines.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, found := presentation.Claim(tc.claim)
			require.Equal(t, tc.wantFound, found)
			require.Equal(t, tc.wantValue, value)
		})
	}

	value, found := (*SDJWTPresentation)(nil).Claim("email")
	require.False(t, found)
	require.Nil(t, value)
}

func testDisclosure(t *testing.T, value []any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func testJWTPart(t *testing.T, value map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func testSDJWTWithEmail(t *testing.T) string {
	t.Helper()
	email := testDisclosure(t, []any{"salt", "email", "person@example.test"})
	payload := testJWTPart(t, map[string]any{"_sd": []any{sha256Base64URL(email)}})
	return testJWTPart(t, map[string]any{"alg": "none"}) + "." + payload + ".signature~" +
		email + "~"
}
