// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLMetaFormatMismatchValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts mdoc metadata for an SD-JWT credential",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "format": "dc+sd-jwt",
					"meta": map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.1"},
				}}},
			}),
			wantStatus: StatusPass,
		},
		{
			name: "rejects SD-JWT metadata for an SD-JWT credential",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "format": "dc+sd-jwt",
					"meta": map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
				}}},
			}),
			wantStatus: StatusFail,
		},
		{
			name: "rejects a credential without metadata",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "format": "dc+sd-jwt",
				}}},
			}),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPDCQLMetaFormatMismatchValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
