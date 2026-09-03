// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLTrustedAuthoritiesMissingTypeValidator(t *testing.T) {
	validator := OID4VPDCQLTrustedAuthoritiesMissingTypeValidator{}
	missingType := compactTestJWT(t, map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{
				"trusted_authorities": []any{map[string]any{
					"values": []any{"https://example.com/authority"},
				}},
			}},
		},
	})
	withType := compactTestJWT(t, map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{
				"trusted_authorities": []any{map[string]any{
					"type":   "aki",
					"values": []any{"https://example.com/authority"},
				}},
			}},
		},
	})

	require.Equal(
		t,
		StatusPass,
		validator.Validate(context.Background(), Input{Value: missingType}).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(context.Background(), Input{Value: withType}).Status,
	)
}
