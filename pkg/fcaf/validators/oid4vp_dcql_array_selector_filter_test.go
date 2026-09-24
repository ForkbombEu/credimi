// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func degreeSelectorEvidence(path []any, claim string, value any) map[string]any {
	return map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{
				"id":     "degree",
				"format": "dc+sd-jwt",
				"meta":   map[string]any{"vct_values": []any{"urn:credimi:degree:1"}},
				"claims": []any{map[string]any{"path": path}},
			}},
		},
		"vp_token": map[string]any{
			"degree": []any{testSDJWTPresentation(map[string]any{
				"vct": "urn:credimi:degree:1",
				claim: value,
			})},
		},
	}
}

func TestArraySelectorFilterRemovesElementWithoutTheSelectedKey(t *testing.T) {
	validator := OID4VPDCQLArraySelectorFilterValidator{}
	path := []any{"degrees", nil, "type"}
	params := map[string]any{
		"vct":              "urn:credimi:degree:1",
		"path":             path,
		"required_values":  []any{"Bachelor of Science", "Master of Science"},
		"forbidden_values": []any{"University of Betelgeuse"},
	}

	filtered := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidence(path, "degrees", []any{
			map[string]any{"type": "Bachelor of Science"},
			map[string]any{"type": "Master of Science"},
		}),
		Params: params,
	})
	require.Equal(t, StatusPass, filtered.Status, filtered.Message)

	unfiltered := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidence(path, "degrees", []any{
			map[string]any{"type": "Bachelor of Science", "university": "University of Betelgeuse"},
			map[string]any{"type": "Master of Science", "university": "University of Betelgeuse"},
			map[string]any{"university": "University of Betelgeuse"},
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, unfiltered.Status, unfiltered.Message)

	incomplete := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidence(path, "degrees", []any{
			map[string]any{"type": "Bachelor of Science"},
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, incomplete.Status, incomplete.Message)
}

func TestArraySelectorFilterRemovesArrayWithoutTheSelectedIndex(t *testing.T) {
	validator := OID4VPDCQLArraySelectorFilterValidator{}
	path := []any{"academic_programmes", nil, float64(1)}
	params := map[string]any{
		"vct":              "urn:credimi:degree:1",
		"path":             path,
		"required_values":  []any{"Doctor of Philosophy"},
		"forbidden_values": []any{"Bachelor of Science", "Master of Science"},
	}

	filtered := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidence(path, "academic_programmes", []any{
			[]any{"Doctor of Philosophy"},
		}),
		Params: params,
	})
	require.Equal(t, StatusPass, filtered.Status, filtered.Message)

	outOfRangeArrayKept := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidence(path, "academic_programmes", []any{
			[]any{"Bachelor of Science"},
			[]any{"Doctor of Philosophy"},
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, outOfRangeArrayKept.Status, outOfRangeArrayKept.Message)
}

func TestArraySelectorFilterAcceptsYAMLIntegerIndex(t *testing.T) {
	validator := OID4VPDCQLArraySelectorFilterValidator{}

	result := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidence(
			[]any{"academic_programmes", nil, float64(1)},
			"academic_programmes",
			[]any{[]any{"Doctor of Philosophy"}},
		),
		Params: map[string]any{
			"vct":              "urn:credimi:degree:1",
			"path":             []any{"academic_programmes", nil, 1},
			"required_values":  []any{"Doctor of Philosophy"},
			"forbidden_values": []any{"Bachelor of Science"},
		},
	})

	require.Equal(t, StatusPass, result.Status, result.Message)
}

func TestArraySelectorFilterRejectsUnboundEvidence(t *testing.T) {
	validator := OID4VPDCQLArraySelectorFilterValidator{}
	path := []any{"degrees", nil, "type"}
	params := map[string]any{
		"vct":              "urn:credimi:degree:1",
		"path":             path,
		"required_values":  []any{"Bachelor of Science"},
		"forbidden_values": []any{"University of Betelgeuse"},
	}

	otherPath := validator.Validate(context.Background(), Input{
		Value: degreeSelectorEvidence([]any{"degrees", "type"}, "degrees", []any{
			map[string]any{"type": "Bachelor of Science"},
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, otherPath.Status, otherPath.Message)

	missingParams := validator.Validate(context.Background(), Input{
		Value:  degreeSelectorEvidence(path, "degrees", []any{}),
		Params: map[string]any{"vct": "urn:credimi:degree:1", "path": path},
	})
	require.Equal(t, StatusError, missingParams.Status, missingParams.Message)
}
