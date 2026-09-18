// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/stretchr/testify/require"
)

func TestMDocExactNamespaceElementsValidator(t *testing.T) {
	validator := MDocExactNamespaceElementsValidator{}
	params := map[string]any{
		"namespace": pidMDocType,
		"elements":  []string{"given_name", "family_name"},
	}

	tests := []struct {
		name         string
		presentation *evidence.MDocPresentation
		params       map[string]any
		want         Status
	}{
		{
			name: "accepts exactly requested elements",
			presentation: mdocValidatorPresentation(map[string]evidence.MDocElement{
				"given_name":  {Identifier: "given_name"},
				"family_name": {Identifier: "family_name"},
			}),
			params: params,
			want:   StatusPass,
		},
		{
			name: "rejects unrequested element",
			presentation: mdocValidatorPresentation(map[string]evidence.MDocElement{
				"given_name":  {Identifier: "given_name"},
				"family_name": {Identifier: "family_name"},
				"birth_date":  {Identifier: "birth_date"},
			}),
			params: params,
			want:   StatusFail,
		},
		{
			name: "rejects unrequested namespace",
			presentation: &evidence.MDocPresentation{
				Namespaces: map[string]map[string]evidence.MDocElement{
					pidMDocType: {
						"given_name":  {Identifier: "given_name"},
						"family_name": {Identifier: "family_name"},
					},
					"org.example": {"member": {Identifier: "member"}},
				},
			},
			params: params,
			want:   StatusFail,
		},
		{
			name: "rejects missing requested element",
			presentation: mdocValidatorPresentation(map[string]evidence.MDocElement{
				"given_name": {Identifier: "given_name"},
			}),
			params: params,
			want:   StatusFail,
		},
		{
			name: "rejects duplicate configured element",
			presentation: mdocValidatorPresentation(map[string]evidence.MDocElement{
				"given_name": {Identifier: "given_name"},
			}),
			params: map[string]any{
				"namespace": pidMDocType,
				"elements":  []string{"given_name", "given_name"},
			},
			want: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  tt.presentation,
				Params: tt.params,
			})
			require.Equal(t, tt.want, result.Status, result.Message)
		})
	}
}
