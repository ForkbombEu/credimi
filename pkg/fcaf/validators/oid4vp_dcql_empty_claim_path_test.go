// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLEmptyClaimPathValidator(t *testing.T) {
	validator := OID4VPDCQLEmptyClaimPathValidator{}
	emptyPath := compactTestJWT(
		t,
		map[string]any{
			"dcql_query": map[string]any{
				"credentials": []any{
					map[string]any{"claims": []any{map[string]any{"path": []any{}}}},
				},
			},
		},
	)
	nonEmptyPath := compactTestJWT(
		t,
		map[string]any{
			"dcql_query": map[string]any{
				"credentials": []any{
					map[string]any{"claims": []any{map[string]any{"path": []any{"given_name"}}}},
				},
			},
		},
	)

	require.Equal(
		t,
		StatusPass,
		validator.Validate(context.Background(), Input{Value: emptyPath}).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(context.Background(), Input{Value: nonEmptyPath}).Status,
	)
}
