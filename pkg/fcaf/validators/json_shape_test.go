// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONFieldTypeValidator(t *testing.T) {
	request := map[string]any{
		"credential_configuration_id": "urn:x",
		"proofs":                      map[string]any{"jwt": []any{"a.b.c"}},
		"count":                       2.0,
		"jwt":                         redacted(true),
	}
	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{"string", request, map[string]any{"field": "credential_configuration_id", "type": "string"}, StatusPass},
		{"object", request, map[string]any{"field": "proofs", "type": "object"}, StatusPass},
		{"number is not a string", request, map[string]any{"field": "count", "type": "string"}, StatusFail},
		{"string is not an object", request, map[string]any{"field": "credential_configuration_id", "type": "object"}, StatusFail},
		{"missing", request, map[string]any{"field": "proof", "type": "object"}, StatusFail},
		{"redacted value hides its type", request, map[string]any{"field": "jwt", "type": "string"}, StatusInconclusive},
		{"input not an object", "x", map[string]any{"field": "proofs", "type": "object"}, StatusFail},
		{"unknown type", request, map[string]any{"field": "proofs", "type": "map"}, StatusError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONFieldTypeValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}

func TestJSONObjectExactlyOneMemberValidator(t *testing.T) {
	params := map[string]any{"field": "proofs", "members": []any{"jwt", "attestation"}}
	withProofs := func(proofs any) map[string]any { return map[string]any{"proofs": proofs} }
	for _, tt := range []struct {
		name   string
		value  any
		status Status
	}{
		{"jwt only", withProofs(map[string]any{"jwt": redacted(true)}), StatusPass},
		{"attestation only", withProofs(map[string]any{"attestation": redacted(true)}), StatusPass},
		{"both", withProofs(map[string]any{"jwt": redacted(true), "attestation": redacted(true)}), StatusFail},
		{"neither", withProofs(map[string]any{"di_vp": []any{}}), StatusFail},
		{"redacted as absent", withProofs(map[string]any{"jwt": redacted(false), "attestation": redacted(true)}), StatusPass},
		{"null member", withProofs(map[string]any{"jwt": nil}), StatusFail},
		{"proofs not an object", withProofs("a.b.c"), StatusFail},
		{"proofs redacted", withProofs(redacted(true)), StatusFail},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONObjectExactlyOneMemberValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}
