// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func requestedPathsEvidence(format string, paths ...[]any) map[string]any {
	claims := make([]any, 0, len(paths))
	for _, path := range paths {
		claims = append(claims, map[string]any{"path": path})
	}
	return map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{
				"id":     "pid",
				"format": format,
				"claims": claims,
			}},
		},
	}
}

func TestRequestedClaimPathsDistinguishesMixedFromSingleRequest(t *testing.T) {
	validator := OID4VPDCQLRequestedClaimPathsValidator{}
	mixed := map[string]any{
		"format": "dc+sd-jwt",
		"paths":  []any{[]any{"given_name"}, []any{"unavailable_claim"}},
	}
	single := map[string]any{
		"format": "dc+sd-jwt",
		"paths":  []any{[]any{"unavailable_claim"}},
	}

	mixedEvidence := requestedPathsEvidence(
		"dc+sd-jwt",
		[]any{"unavailable_claim"},
		[]any{"given_name"},
	)
	singleEvidence := requestedPathsEvidence("dc+sd-jwt", []any{"unavailable_claim"})

	require.Equal(
		t,
		StatusPass,
		validator.Validate(context.Background(), Input{Value: mixedEvidence, Params: mixed}).Status,
	)
	require.Equal(
		t,
		StatusPass,
		validator.Validate(
			context.Background(),
			Input{Value: singleEvidence, Params: single},
		).Status,
	)

	// The mixed-request assertion must reject the single-claim request, and the
	// single-claim assertion must reject the mixed request.
	require.Equal(
		t,
		StatusFail,
		validator.Validate(
			context.Background(),
			Input{Value: singleEvidence, Params: mixed},
		).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(
			context.Background(),
			Input{Value: mixedEvidence, Params: single},
		).Status,
	)
}

func TestRequestedClaimPathsRejectsSubstitutedPathAndFormat(t *testing.T) {
	validator := OID4VPDCQLRequestedClaimPathsValidator{}
	params := map[string]any{
		"format": "mso_mdoc",
		"paths": []any{
			[]any{"eu.europa.ec.eudi.pid.1", "given_name"},
			[]any{"eu.europa.ec.eudi.pid.1", "unavailable_element"},
		},
	}

	substituted := validator.Validate(context.Background(), Input{
		Value: requestedPathsEvidence(
			"mso_mdoc",
			[]any{"eu.europa.ec.eudi.pid.1", "given_name"},
			[]any{"eu.europa.ec.eudi.pid.1", "family_name"},
		),
		Params: params,
	})
	require.Equal(t, StatusFail, substituted.Status, substituted.Message)

	wrongFormat := validator.Validate(context.Background(), Input{
		Value: requestedPathsEvidence(
			"dc+sd-jwt",
			[]any{"eu.europa.ec.eudi.pid.1", "given_name"},
			[]any{"eu.europa.ec.eudi.pid.1", "unavailable_element"},
		),
		Params: params,
	})
	require.Equal(t, StatusFail, wrongFormat.Status, wrongFormat.Message)

	missingPaths := validator.Validate(context.Background(), Input{
		Value:  requestedPathsEvidence("mso_mdoc", []any{"eu.europa.ec.eudi.pid.1", "given_name"}),
		Params: map[string]any{"format": "mso_mdoc"},
	})
	require.Equal(t, StatusError, missingPaths.Status, missingPaths.Message)
}
