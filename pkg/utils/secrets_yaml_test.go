// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSecretsYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{
			name:  "valid",
			input: "api_key: secret\npin: \"1234\"\n",
			want:  map[string]string{"api_key": "secret", "pin": "1234"},
		},
		{name: "invalid", input: "- not\n- a map\n", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSecretsYAML(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
