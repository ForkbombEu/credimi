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

func TestSDJWTCompactSerializationValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		keyBinding bool
		wantStatus Status
	}{
		{
			name: "compact presentation with key binding",
			value: &evidence.SDJWTPresentation{
				Raw:           "issuer.jwt.signature~disclosure~kb.jwt.signature",
				SDJWT:         "issuer.jwt.signature~disclosure~",
				KeyBindingJWT: "kb.jwt.signature",
			},
			keyBinding: true,
			wantStatus: StatusPass,
		},
		{
			name: "missing key binding jwt",
			value: &evidence.SDJWTPresentation{
				Raw:   "issuer.jwt.signature~disclosure~",
				SDJWT: "issuer.jwt.signature~disclosure~",
			},
			keyBinding: true,
			wantStatus: StatusFail,
		},
		{
			name: "raw presentation does not end with key binding jwt",
			value: &evidence.SDJWTPresentation{
				Raw:           "issuer.jwt.signature~disclosure~different.jwt.signature",
				SDJWT:         "issuer.jwt.signature~disclosure~",
				KeyBindingJWT: "kb.jwt.signature",
			},
			keyBinding: true,
			wantStatus: StatusFail,
		},
		{
			name: "compact presentation without key binding ends in tilde",
			value: &evidence.SDJWTPresentation{
				Raw:   "issuer.jwt.signature~disclosure~",
				SDJWT: "issuer.jwt.signature~disclosure~",
			},
			wantStatus: StatusPass,
		},
		{
			name: "presentation without key binding omits final tilde",
			value: &evidence.SDJWTPresentation{
				Raw:   "issuer.jwt.signature~disclosure",
				SDJWT: "issuer.jwt.signature~disclosure",
			},
			wantStatus: StatusFail,
		},
	}

	validator := SDJWTCompactSerializationValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"key_binding": test.keyBinding},
			})

			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
