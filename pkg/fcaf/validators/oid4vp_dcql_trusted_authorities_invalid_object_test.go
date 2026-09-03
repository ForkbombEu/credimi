// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOID4VPDCQLTrustedAuthoritiesInvalidObjectValidator(t *testing.T) {
	v := OID4VPDCQLTrustedAuthoritiesInvalidObjectValidator{}
	bad := compactTestJWT(t, map[string]any{"dcql_query": map[string]any{"credentials": []any{map[string]any{"trusted_authorities": []any{map[string]any{}}}}}})
	good := compactTestJWT(t, map[string]any{"dcql_query": map[string]any{"credentials": []any{map[string]any{"trusted_authorities": []any{map[string]any{"type": "aki", "values": []any{"x"}}}}}}})
	require.Equal(t, StatusPass, v.Validate(context.Background(), Input{Value: bad}).Status)
	require.Equal(t, StatusFail, v.Validate(context.Background(), Input{Value: good}).Status)
}
