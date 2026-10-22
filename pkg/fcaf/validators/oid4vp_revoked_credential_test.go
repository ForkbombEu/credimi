// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VCIRevocationPrecedesRequestValidator(t *testing.T) {
	const revokedAt = "2026-10-22T09:00:00.000Z"
	revocation := func(results ...any) map[string]any {
		return map[string]any{
			"session_id":       "issuance",
			"at":               revokedAt,
			"requested_status": "revoked",
			"results":          results,
		}
	}
	revoked := map[string]any{
		"uri":         "https://tsl.example/1",
		"idx":         7,
		"status":      1,
		"ok":          true,
		"http_status": 200,
	}
	refused := map[string]any{
		"uri":         "https://tsl.example/1",
		"idx":         7,
		"status":      1,
		"ok":          false,
		"http_status": 500,
	}
	session := func(events ...any) map[string]any {
		return map[string]any{"events": events}
	}
	retrieved := func(at string) map[string]any {
		return map[string]any{"type": "vp_request_retrieved", "at": at}
	}

	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{
			"request retrieved after the exp window",
			map[string]any{
				"revocation": revocation(revoked),
				"presentation_session": session(
					map[string]any{"type": "vp_session_created", "at": "2026-10-22T09:03:50Z"},
					retrieved("2026-10-22T09:04:05.120Z"),
				),
			},
			StatusPass,
		},
		{
			"request retrieved inside the exp window",
			map[string]any{
				"revocation":           revocation(revoked),
				"presentation_session": session(retrieved("2026-10-22T09:02:00Z")),
			},
			StatusBlocked,
		},
		{
			"request retrieved before the revocation",
			map[string]any{
				"revocation":           revocation(revoked),
				"presentation_session": session(retrieved("2026-10-22T08:59:00Z")),
			},
			StatusBlocked,
		},
		{
			"one reference the status list refused",
			map[string]any{
				"revocation":           revocation(revoked, refused),
				"presentation_session": session(retrieved("2026-10-22T09:10:00Z")),
			},
			StatusBlocked,
		},
		{
			"no reference revoked",
			map[string]any{
				"revocation":           revocation(),
				"presentation_session": session(retrieved("2026-10-22T09:10:00Z")),
			},
			StatusBlocked,
		},
		{
			"another status requested",
			map[string]any{
				"revocation": map[string]any{
					"at":               revokedAt,
					"requested_status": "suspended",
					"results":          []any{revoked},
				},
				"presentation_session": session(retrieved("2026-10-22T09:10:00Z")),
			},
			StatusBlocked,
		},
		{
			"wallet never retrieved the request",
			map[string]any{"revocation": revocation(revoked), "presentation_session": session()},
			StatusBlocked,
		},
		{
			"unparseable retrieval time",
			map[string]any{
				"revocation":           revocation(revoked),
				"presentation_session": session(retrieved("yesterday")),
			},
			StatusFail,
		},
		{"evidence is not an object", "revoked", StatusFail},
	}

	validator := OID4VCIRevocationPrecedesRequestValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"min_delay_seconds": 180},
			})
			require.Equal(t, test.status, result.Status, result.Message)
		})
	}

	t.Run("min_delay_seconds is required", func(t *testing.T) {
		result := validator.Validate(context.Background(), Input{Value: map[string]any{}})
		require.Equal(t, StatusError, result.Status, result.Message)
	})
}

func TestOID4VPWalletErrorConcealsReasonValidator(t *testing.T) {
	session := func(response map[string]any) map[string]any {
		return map[string]any{
			"raw": map[string]any{
				"presentation_response_decrypted": response,
				// The verifier's own reply must never count as the Wallet's.
				"presentation_response_verifier_http": map[string]any{
					"body": map[string]any{
						"error":             "invalid_request",
						"error_description": "credential revoked",
					},
				},
			},
		}
	}
	terms := []any{"revoked", "revocation", "status list"}

	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{"bare error", session(map[string]any{"error": "access_denied", "state": "s"}), StatusPass},
		{
			"error with a neutral description",
			session(
				map[string]any{
					"error":             "access_denied",
					"error_description": "no matching credential",
				},
			),
			StatusPass,
		},
		{
			"description names the revocation",
			session(
				map[string]any{
					"error":             "access_denied",
					"error_description": "The credential was REVOKED",
				},
			),
			StatusFail,
		},
		{
			"error uri names the status list",
			session(
				map[string]any{
					"error":     "invalid_request",
					"error_uri": "https://wallet.example/errors/status list check",
				},
			),
			StatusFail,
		},
		{"no error and no presentation", session(map[string]any{"state": "s"}), StatusFail},
		{
			"presentation alongside an error",
			session(
				map[string]any{
					"error":    "access_denied",
					"vp_token": map[string]any{"pid": []any{"token"}},
				},
			),
			StatusFail,
		},
		{"session is not an object", "access_denied", StatusFail},
	}

	validator := OID4VPWalletErrorConcealsReasonValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"terms": terms},
			})
			require.Equal(t, test.status, result.Status, result.Message)
		})
	}

	t.Run("terms are required", func(t *testing.T) {
		result := validator.Validate(
			context.Background(),
			Input{Value: session(map[string]any{"error": "access_denied"})},
		)
		require.Equal(t, StatusError, result.Status, result.Message)
	})
}
