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
