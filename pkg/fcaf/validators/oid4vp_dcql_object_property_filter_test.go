// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// addressDisclosureEvidence mirrors the Capture Wallet degree disclosure frame:
// `address` is a top-level disclosure whose value carries one nested object
// disclosure per property, so a Wallet can reveal a single property.
func addressDisclosureEvidence(
	t *testing.T,
	path []any,
	presentProperties ...string,
) map[string]any {
	t.Helper()
	properties := map[string]nestedDisclosure{
		"street_address": newNestedDisclosure(
			t,
			"salt-street",
			"street_address",
			"42 Market Street",
		),
		"locality":    newNestedDisclosure(t, "salt-locality", "locality", "Milliways"),
		"postal_code": newNestedDisclosure(t, "salt-postal", "postal_code", "12345"),
	}
	digests := make([]any, 0, len(properties))
	for _, name := range []string{"street_address", "locality", "postal_code"} {
		digests = append(digests, properties[name].digest)
	}
	address := newNestedDisclosure(t, "salt-address", "address", map[string]any{"_sd": digests})

	presented := make([]nestedDisclosure, 0, len(presentProperties))
	for _, name := range presentProperties {
		presented = append(presented, properties[name])
	}
	return nestedSelectorEvidence(path, nestedDisclosurePresentation(t, address, presented...))
}

func TestObjectPropertyFilterKeepsOnlyTheRequestedProperty(t *testing.T) {
	validator := OID4VPDCQLObjectPropertyFilterValidator{}
	path := []any{"address", "street_address"}
	params := map[string]any{
		"vct":                  "urn:credimi:degree:1",
		"path":                 path,
		"required_properties":  []any{"street_address"},
		"forbidden_properties": []any{"locality", "postal_code"},
	}

	filtered := validator.Validate(context.Background(), Input{
		Value:  addressDisclosureEvidence(t, path, "street_address"),
		Params: params,
	})
	require.Equal(t, StatusPass, filtered.Status, filtered.Message)

	unfiltered := validator.Validate(context.Background(), Input{
		Value:  addressDisclosureEvidence(t, path, "street_address", "locality", "postal_code"),
		Params: params,
	})
	require.Equal(t, StatusFail, unfiltered.Status, unfiltered.Message)

	missingProperty := validator.Validate(context.Background(), Input{
		Value:  addressDisclosureEvidence(t, path),
		Params: params,
	})
	require.Equal(t, StatusFail, missingProperty.Status, missingProperty.Message)
}

func TestObjectPropertyFilterRejectsUnboundEvidence(t *testing.T) {
	validator := OID4VPDCQLObjectPropertyFilterValidator{}
	params := map[string]any{
		"vct":                  "urn:credimi:degree:1",
		"path":                 []any{"address", "street_address"},
		"required_properties":  []any{"street_address"},
		"forbidden_properties": []any{"locality"},
	}

	otherPath := validator.Validate(context.Background(), Input{
		Value:  addressDisclosureEvidence(t, []any{"address", "locality"}, "street_address"),
		Params: params,
	})
	require.Equal(t, StatusFail, otherPath.Status, otherPath.Message)

	arrayPath := validator.Validate(context.Background(), Input{
		Value: addressDisclosureEvidence(t, []any{"address", "street_address"}, "street_address"),
		Params: map[string]any{
			"vct":                  "urn:credimi:degree:1",
			"path":                 []any{"address", nil, "street_address"},
			"required_properties":  []any{"street_address"},
			"forbidden_properties": []any{"locality"},
		},
	})
	require.Equal(t, StatusError, arrayPath.Status, arrayPath.Message)
}
