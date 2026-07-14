// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

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

func TestSDJWTIssuerX509HeaderValidator(t *testing.T) {
	certificate := testX509Certificate(t)
	tests := []struct {
		name       string
		headers    map[string]any
		wantStatus Status
	}{
		{
			name:       "valid x5c chain",
			headers:    map[string]any{"x5c": []any{certificate}},
			wantStatus: StatusPass,
		},
		{
			name:       "missing x5c chain",
			headers:    map[string]any{},
			wantStatus: StatusFail,
		},
		{
			name:       "invalid certificate",
			headers:    map[string]any{"x5c": []any{"not-base64"}},
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SDJWTIssuerX509HeaderValidator{}.Validate(context.Background(), Input{
				Value: &evidence.SDJWTPresentation{ProtectedHeaders: tt.headers},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func testX509Certificate(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(3600, 0),
	}
	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(der)
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

func TestJSONFieldEqualsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      map[string]any
		expected   any
		wantStatus Status
	}{
		{
			name:       "matching value",
			value:      map[string]any{"request_uri_method": "post"},
			expected:   "post",
			wantStatus: StatusPass,
		},
		{
			name:       "different value",
			value:      map[string]any{"request_uri_method": "get"},
			expected:   "post",
			wantStatus: StatusFail,
		},
		{
			name:       "missing field",
			value:      map[string]any{},
			expected:   "post",
			wantStatus: StatusFail,
		},
		{
			name:       "different type",
			value:      map[string]any{"request_uri_method": 1},
			expected:   "1",
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSONFieldEqualsValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "request_uri_method",
					"value": tt.expected,
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestEvidenceNonEmptyValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{name: "screenshot URLs", value: []any{"https://example.test/selection.png"}, wantStatus: StatusPass},
		{name: "empty screenshot URLs", value: []any{}, wantStatus: StatusFail},
		{name: "missing evidence", value: nil, wantStatus: StatusFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvidenceNonEmptyValidator{}.Validate(context.Background(), Input{Value: tt.value})
			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJSONFieldPresenceValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		field      string
		present    bool
		wantStatus Status
	}{
		{
			name:       "required field present",
			value:      map[string]any{"vp_token": map[string]any{}},
			field:      "vp_token",
			present:    true,
			wantStatus: StatusPass,
		},
		{
			name:       "forbidden field absent",
			value:      map[string]any{"vp_token": map[string]any{}},
			field:      "access_token",
			present:    false,
			wantStatus: StatusPass,
		},
		{
			name:       "required field absent",
			value:      map[string]any{},
			field:      "vp_token",
			present:    true,
			wantStatus: StatusFail,
		},
		{
			name:       "forbidden field present",
			value:      map[string]any{"access_token": "token"},
			field:      "access_token",
			present:    false,
			wantStatus: StatusFail,
		},
		{
			name:       "non-object evidence",
			value:      "vp_token=value",
			field:      "vp_token",
			present:    true,
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSONFieldPresenceValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field":   tt.field,
					"present": tt.present,
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTHeaderFieldEqualsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name:       "matching typ",
			value:      "eyJ0eXAiOiJvYXV0aC1hdXRoei1yZXErand0IiwiYWxnIjoiRVMyNTYifQ.e30.signature",
			wantStatus: StatusPass,
		},
		{
			name:       "different typ",
			value:      "eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiJ9.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "missing typ",
			value:      "eyJhbGciOiJFUzI1NiJ9.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "not compact JWT",
			value:      "invalid",
			wantStatus: StatusFail,
		},
		{
			name:       "not string",
			value:      map[string]any{},
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTHeaderFieldEqualsValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "typ",
					"value": "oauth-authz-req+jwt",
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTPayloadFieldEqualsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name:       "matching response type",
			value:      "e30.eyJyZXNwb25zZV90eXBlIjoidnBfdG9rZW4ifQ.signature",
			wantStatus: StatusPass,
		},
		{
			name:       "different response type",
			value:      "e30.eyJyZXNwb25zZV90eXBlIjoiY29kZSJ9.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "missing response type",
			value:      "e30.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "not compact JWT",
			value:      "invalid",
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTPayloadFieldEqualsValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "response_type",
					"value": "vp_token",
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTPayloadObjectKeysAllowedValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantStatus Status
	}{
		{
			name: "defined metadata keys",
			value: "e30." +
				"eyJjbGllbnRfbWV0YWRhdGEiOnsiandrcyI6e30sInZwX2Zvcm1hdHNfc3VwcG9ydGVkIjp7fX19." +
				"signature",
			wantStatus: StatusPass,
		},
		{
			name: "undefined metadata key",
			value: "e30." +
				"eyJjbGllbnRfbWV0YWRhdGEiOnsiandrcyI6e30sInVua25vd24iOnRydWV9fQ." +
				"signature",
			wantStatus: StatusFail,
		},
		{
			name:       "missing metadata",
			value:      "e30.e30.signature",
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTPayloadObjectKeysAllowedValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "client_metadata",
					"allowed_keys": []string{
						"jwks",
						"vp_formats_supported",
						"encrypted_response_enc_values_supported",
					},
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTPayloadFieldPresenceValidator(t *testing.T) {
	token := "e30.eyJjbGllbnRfaWQiOiJjbGllbnQtMSJ9.signature"
	tests := []struct {
		name       string
		field      string
		present    bool
		wantStatus Status
	}{
		{name: "required field present", field: "client_id", present: true, wantStatus: StatusPass},
		{name: "forbidden field absent", field: "iss", present: false, wantStatus: StatusPass},
		{name: "required field absent", field: "iss", present: true, wantStatus: StatusFail},
		{
			name:       "forbidden field present",
			field:      "client_id",
			present:    false,
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTPayloadFieldPresenceValidator{}.Validate(context.Background(), Input{
				Value: token,
				Params: map[string]any{
					"field":   tt.field,
					"present": tt.present,
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
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
