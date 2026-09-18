// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// degreeSelectorEvidenceWithClaims builds selector evidence whose presentation
// discloses extra top-level claims besides the selected one.
func degreeSelectorEvidenceWithClaims(path []any, claims map[string]any) map[string]any {
	presented := map[string]any{"vct": "urn:credimi:degree:1"}
	for name, value := range claims {
		presented[name] = value
	}
	return map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{
				"id":     "degree",
				"format": "dc+sd-jwt",
				"meta":   map[string]any{"vct_values": []any{"urn:credimi:degree:1"}},
				"claims": []any{map[string]any{"path": path}},
			}},
		},
		"vp_token": map[string]any{"degree": []any{testSDJWTPresentation(presented)}},
	}
}

// TestArraySelectorFilterRejectsUnrequestedSiblingClaims covers the "disclose
// the selected element only" requirement that forbidden_values cannot reach:
// forbidden_values is matched against the string leaves of the selected claim,
// so a sibling top-level claim is invisible to it.
func TestArraySelectorFilterRejectsUnrequestedSiblingClaims(t *testing.T) {
	validator := OID4VPDCQLArraySelectorFilterValidator{}
	path := []any{"degrees", nil}
	params := map[string]any{
		"vct":              "urn:credimi:degree:1",
		"path":             path,
		"required_values":  []any{"Bachelor of Science"},
		"forbidden_claims": []any{"name", "address", "nationalities"},
	}
	degrees := []any{map[string]any{"type": "Bachelor of Science"}}

	selectedOnly := validator.Validate(context.Background(), Input{
		Value:  degreeSelectorEvidenceWithClaims(path, map[string]any{"degrees": degrees}),
		Params: params,
	})
	require.Equal(t, StatusPass, selectedOnly.Status, selectedOnly.Message)

	withSibling := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidenceWithClaims(path, map[string]any{
			"degrees": degrees,
			"name":    "Arthur Dent",
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, withSibling.Status, withSibling.Message)

	noNegativeCheck := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidenceWithClaims(path, map[string]any{"degrees": degrees}),
		Params: map[string]any{
			"vct":             "urn:credimi:degree:1",
			"path":            path,
			"required_values": []any{"Bachelor of Science"},
		},
	})
	require.Equal(t, StatusError, noNegativeCheck.Status, noNegativeCheck.Message)
}

// TestObjectPropertyFilterRejectsUnrequestedSiblingClaims covers the same
// requirement for the object-property case.
func TestObjectPropertyFilterRejectsUnrequestedSiblingClaims(t *testing.T) {
	validator := OID4VPDCQLObjectPropertyFilterValidator{}
	path := []any{"address", "street_address"}
	params := map[string]any{
		"vct":                  "urn:credimi:degree:1",
		"path":                 path,
		"required_properties":  []any{"street_address"},
		"forbidden_properties": []any{"locality"},
		"forbidden_claims":     []any{"name", "degrees"},
	}
	address := map[string]any{"street_address": "42 Market Street"}

	selectedOnly := validator.Validate(context.Background(), Input{
		Value:  degreeSelectorEvidenceWithClaims(path, map[string]any{"address": address}),
		Params: params,
	})
	require.Equal(t, StatusPass, selectedOnly.Status, selectedOnly.Message)

	withSibling := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidenceWithClaims(path, map[string]any{
			"address": address,
			"degrees": []any{map[string]any{"type": "Bachelor of Science"}},
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, withSibling.Status, withSibling.Message)
}
