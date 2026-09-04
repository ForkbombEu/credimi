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

func TestSDJWTDisclosureDigestsSHA256Validator(t *testing.T) {
	tests := []struct {
		name         string
		presentation *evidence.SDJWTPresentation
		wantStatus   Status
	}{
		{
			name:         "accepts the default SHA-256 disclosure digest algorithm",
			presentation: newSDJWTPresentation(t, ""),
			wantStatus:   StatusPass,
		},
		{
			name:         "accepts the explicit SHA-256 disclosure digest algorithm",
			presentation: newSDJWTPresentation(t, "sha-256"),
			wantStatus:   StatusPass,
		},
		{
			name:         "rejects a non-SHA-256 disclosure digest algorithm",
			presentation: newSDJWTPresentation(t, "sha-384"),
			wantStatus:   StatusFail,
		},
	}

	validator := SDJWTDisclosureDigestsSHA256Validator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: tt.presentation})
			require.Equal(t, tt.wantStatus, result.Status, result.Message)
		})
	}
}

func newSDJWTPresentation(t *testing.T, algorithm string) *evidence.SDJWTPresentation {
	t.Helper()
	presentation := &evidence.SDJWTPresentation{
		Raw:             "issuer.jwt.signature~disclosure~",
		SDJWT:           "issuer.jwt.signature~disclosure~",
		DisclosureCount: 1,
		IssuerPayload: map[string]any{
			"_sd": []any{"disclosure-digest"},
		},
	}
	if algorithm != "" {
		presentation.IssuerPayload["_sd_alg"] = algorithm
	}
	return presentation
}
