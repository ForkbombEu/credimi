// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLMetaOmittedValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts a DCQL credential without meta",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "format": "dc+sd-jwt",
				}}},
			}),
			wantStatus: StatusPass,
		},
		{
			name: "rejects a DCQL credential with meta",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "format": "dc+sd-jwt",
					"meta": map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
				}}},
			}),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPDCQLMetaOmittedValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
