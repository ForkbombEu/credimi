// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLArraySelectorFilterValidator(t *testing.T) {
	validator := OID4VPDCQLArraySelectorFilterValidator{}
	newEvidence := func(degrees []any) map[string]any {
		return map[string]any{
			"dcql_query": map[string]any{
				"credentials": []any{map[string]any{
					"id":     "degree",
					"format": "dc+sd-jwt",
					"meta":   map[string]any{"vct_values": []any{"urn:credimi:degree:1"}},
					"claims": []any{map[string]any{
						"path": []any{"degrees", nil, "type"},
					}},
				}},
			},
			"vp_token": map[string]any{
				"degree": []any{testSDJWTPresentation(map[string]any{
					"vct":     "urn:credimi:degree:1",
					"degrees": degrees,
				})},
			},
		}
	}

	valid := validator.Validate(context.Background(), Input{Value: newEvidence([]any{
		map[string]any{"type": "Bachelor of Science"},
		map[string]any{"type": "Master of Science"},
	})})
	require.Equal(t, StatusPass, valid.Status, valid.Message)

	unfiltered := validator.Validate(context.Background(), Input{Value: newEvidence([]any{
		map[string]any{"type": "Bachelor of Science"},
		map[string]any{"type": "Master of Science"},
		map[string]any{"university": "University of Betelgeuse"},
	})})
	require.Equal(t, StatusFail, unfiltered.Status, unfiltered.Message)

	wrongQuery := newEvidence([]any{
		map[string]any{"type": "Bachelor of Science"},
		map[string]any{"type": "Master of Science"},
	})
	wrongQuery["dcql_query"].(map[string]any)["credentials"].([]any)[0].(map[string]any)["claims"] = []any{
		map[string]any{"path": []any{"degrees", "type"}},
	}
	result := validator.Validate(context.Background(), Input{Value: wrongQuery})
	require.Equal(t, StatusFail, result.Status, result.Message)
}
