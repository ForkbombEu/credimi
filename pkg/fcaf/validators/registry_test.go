// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/stretchr/testify/require"
)

func TestRegistryRejectsDuplicateIDs(t *testing.T) {
	_, err := NewRegistry(EvidencePresentValidator{}, EvidencePresentValidator{})

	require.ErrorContains(t, err, "duplicate validator id")
}

func TestEvidencePresentValidator(t *testing.T) {
	got := EvidencePresentValidator{}.Validate(context.Background(), Input{Value: "value"})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTClaimUTF8StringValidator(t *testing.T) {
	got := SDJWTClaimUTF8StringValidator{}.Validate(context.Background(), Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{"email": "person@example.test"}},
		Params: map[string]any{
			"claim": "email",
			"vectors": map[string]any{
				"positive": []string{"fixtures/fcaf/validators/sdjwt/email_utf8_positive.yaml"},
				"negative": []string{"fixtures/fcaf/validators/sdjwt/email_utf8_negative.yaml"},
			},
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTClaimRFC5322EmailValidator(t *testing.T) {
	got := SDJWTClaimRFC5322EmailValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"email": "person@example.test"},
		Params: map[string]any{
			"claim": "email",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTClaimStringPrefixValidator(t *testing.T) {
	got := SDJWTClaimStringPrefixValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"vct": "urn:eudi:pid:1"},
		Params: map[string]any{
			"claim":  "vct",
			"prefix": "urn:eudi:pid:",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestPIDSDJWTVCTValidator(t *testing.T) {
	got := PIDSDJWTVCTValidator{}.Validate(context.Background(), Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{
			"vct": "urn:eudi:pid:1",
		}},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestPIDSDJWTMandatoryClaimsValidator(t *testing.T) {
	got := PIDSDJWTMandatoryClaimsValidator{}.Validate(context.Background(), Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{
			"family_name":       "Trotter",
			"given_name":        "Filippo",
			"birthdate":         "1999-11-01",
			"place_of_birth":    map[string]any{"country": "IT"},
			"nationalities":     []any{"IT"},
			"date_of_expiry":    "2026-10-11",
			"issuing_authority": "GR Administrative authority",
			"issuing_country":   "GR",
			"email":             "person@example.test",
		}},
		Params: map[string]any{
			"required_elements": []string{
				"family_name",
				"given_name",
				"birthdate",
				"place_of_birth",
				"nationalities",
				"date_of_expiry",
				"issuing_authority",
				"issuing_country",
			},
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestMDocNamespaceElementPresentValidator(t *testing.T) {
	got := MDocNamespaceElementPresentValidator{}.Validate(context.Background(), Input{
		Value: &evidence.MDocPresentation{
			Documents: []evidence.MDocDocument{{
				DocType: "eu.europa.ec.eudi.pid.1",
			}},
			Namespaces: map[string]map[string]evidence.MDocElement{
				"eu.europa.ec.eudi.pid.1": {
					"email_address": {
						Identifier: "email_address",
						Value:      "person@example.test",
						MajorType:  3,
					},
				},
			},
		},
		Params: map[string]any{
			"namespace": "eu.europa.ec.eudi.pid.1",
			"element":   "email_address",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestJOSEJWEEncryptedResponseValidator(t *testing.T) {
	got := JOSEJWEEncryptedResponseValidator{}.Validate(context.Background(), Input{
		Value: "a.b.c.d.e",
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestOID4VPNonceStateBindingValidator(t *testing.T) {
	got := OID4VPNonceStateBindingValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{
			"request_nonce":  "n1",
			"response_nonce": "n1",
			"request_state":  "s1",
			"response_state": "s1",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestJSONFieldRequiredValidator(t *testing.T) {
	got := JSONFieldRequiredValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"email": "person@example.test"},
		Params: map[string]any{
			"field": "email",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTClaimPresentValidator(t *testing.T) {
	got := SDJWTClaimPresentValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"email": "person@example.test"},
		Params: map[string]any{
			"claim": "email",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}
