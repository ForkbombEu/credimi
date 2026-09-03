// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLTrustedAuthoritiesNonArrayValidator(t *testing.T) {
	validator := OID4VPDCQLTrustedAuthoritiesNonArrayValidator{}
	invalid := compactTestJWT(t, map[string]any{
		"dcql_query": map[string]any{"credentials": []any{map[string]any{"trusted_authorities": "not-an-array"}}},
	})
	valid := compactTestJWT(t, map[string]any{
		"dcql_query": map[string]any{"credentials": []any{map[string]any{"trusted_authorities": []any{}}}},
	})
	require.Equal(t, StatusPass, validator.Validate(context.Background(), Input{Value: invalid}).Status)
	require.Equal(t, StatusFail, validator.Validate(context.Background(), Input{Value: valid}).Status)
}
