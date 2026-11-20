// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// credentials_match must judge the presented credential, not only the presence
// of a vp_token entry: a Wallet that answers a query with another format, vct or
// docType did not match the query. The run review found the mode passing on
// presence alone.
func TestDCQLCredentialsMatchChecksFormatAndMeta(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		meta         map[string]any
		presentation any
		want         Status
		wantMessage  string
	}{
		{
			name:         "sd-jwt with the requested vct matches",
			format:       "dc+sd-jwt",
			meta:         map[string]any{"vct_values": []any{"urn:eu.europa.ec.eudi:pid:1"}},
			presentation: signedSDJWTPresentation(),
			want:         StatusPass,
		},
		{
			name:         "sd-jwt with another vct does not match",
			format:       "dc+sd-jwt",
			meta:         map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
			presentation: testSDJWTPresentation(map[string]any{"vct": "urn:eudi:mdl:1"}),
			want:         StatusFail,
			wantMessage:  "vct",
		},
		{
			name:         "sd-jwt without a vct does not satisfy vct_values",
			format:       "dc+sd-jwt",
			meta:         map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
			presentation: unsignedSDJWTPresentation(),
			want:         StatusFail,
			wantMessage:  "vct",
		},
		{
			// A present-but-empty meta imposes no vct constraint, so any
			// dc+sd-jwt presentation satisfies it (WS_RP_MS_ProtocolMessages__076).
			name:         "sd-jwt with empty meta matches",
			format:       "dc+sd-jwt",
			meta:         map[string]any{},
			presentation: signedSDJWTPresentation(),
			want:         StatusPass,
		},
		{
			name:         "mdoc with the requested doctype matches",
			format:       "mso_mdoc",
			meta:         map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.1"},
			presentation: validMDocPresentation(t),
			want:         StatusPass,
		},
		{
			name:         "mdoc with another doctype does not match",
			format:       "mso_mdoc",
			meta:         map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.2"},
			presentation: validMDocPresentation(t),
			want:         StatusFail,
			wantMessage:  "docType",
		},
		{
			name:         "sd-jwt token does not answer an mdoc query",
			format:       "mso_mdoc",
			meta:         map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.1"},
			presentation: signedSDJWTPresentation(),
			want:         StatusFail,
			wantMessage:  "not a valid mso_mdoc DeviceResponse",
		},
		{
			name:         "mdoc token does not answer a dc+sd-jwt query",
			format:       "dc+sd-jwt",
			meta:         map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
			presentation: validMDocPresentation(t),
			want:         StatusFail,
			wantMessage:  "not a valid dc+sd-jwt presentation",
		},
		{
			name:         "a non-string presentation is not a credential",
			format:       "dc+sd-jwt",
			meta:         map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
			presentation: map[string]any{"vp_token": "not-a-token"},
			want:         StatusFail,
			wantMessage:  "is not a dc+sd-jwt presentation",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evidence := map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id":     "pid",
					"format": test.format,
					"meta":   test.meta,
				}}},
				"vp_token": map[string]any{"pid": []any{test.presentation}},
			}
			result := DCQLResponseConstraintsValidator{}.Validate(
				context.Background(),
				Input{Value: evidence, Params: map[string]any{"mode": "credentials_match"}},
			)
			require.Equal(t, test.want, result.Status, result.Message)
			if test.wantMessage != "" {
				require.Contains(t, result.Message, test.wantMessage)
			}
		})
	}
}

// The status-reference probes (Metadata 084-100) fail because the Wallet kept
// and shared the credential the probe pinned by document_number, not because a
// claim value differed from what the query asked for.
func TestClaimsValuesNoMatchNamesTheSharedCredential(t *testing.T) {
	probe := func(withPresentation bool) map[string]any {
		credential := validSDJWTCredentialQuery("pid")
		credential["claims"] = []any{map[string]any{
			"path":   []any{"document_number"},
			"values": []any{"CREDIMI-SDJWT-MISSING-INDEX"},
		}}
		evidence := map[string]any{
			"dcql_query": map[string]any{"credentials": []any{credential}},
		}
		if withPresentation {
			evidence["vp_token"] = map[string]any{"pid": []any{
				testSDJWTPresentation(map[string]any{
					"vct":             "urn:eu.europa.ec.eudi:pid:1",
					"document_number": "CREDIMI-SDJWT-OTHER",
				}),
			}}
		}
		return evidence
	}
	validator := DCQLResponseConstraintsValidator{}
	params := map[string]any{
		"mode":                "claims_values_no_match",
		"expected_claim_path": []any{"document_number"},
		"expected_value":      "CREDIMI-SDJWT-MISSING-INDEX",
	}

	t.Run("shared credential is named as the probe's credential", func(t *testing.T) {
		result := validator.Validate(context.Background(), Input{
			Value:  probe(true),
			Params: params,
		})
		require.Equal(t, StatusFail, result.Status)
		require.Contains(t, result.Message, "wallet shared the credential pinned by the probe")
		require.Contains(t, result.Message, "kept a credential it had to reject")
		require.NotContains(t, result.Message, "mismatched claim values")
	})

	t.Run("withheld credential keeps the pass message", func(t *testing.T) {
		result := validator.Validate(context.Background(), Input{
			Value:  probe(false),
			Params: params,
		})
		require.Equal(t, StatusPass, result.Status)
		require.Contains(t, result.Message, "wallet returned no credential")
	})

	t.Run("a query without a value restriction keeps the generic message", func(t *testing.T) {
		credential := validSDJWTCredentialQuery("pid")
		credential["claims"] = []any{map[string]any{
			"path":   []any{"document_number"},
			"values": []any{"CREDIMI-SDJWT-MISSING-INDEX"},
		}}
		result := validator.Validate(context.Background(), Input{
			Value: map[string]any{
				"dcql_query": map[string]any{"credentials": []any{credential}},
				"vp_token":   map[string]any{"pid": []any{signedSDJWTPresentation()}},
			},
			Params: map[string]any{"mode": "claims_values_no_match"},
		})
		require.Equal(t, StatusFail, result.Status)
		require.Contains(t, result.Message, "mismatched claim values")
	})
}
