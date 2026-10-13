// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
)

func TestDecodeStringPayload(t *testing.T) {
	t.Setenv(temporalcrypto.SecretsEncryptionKeyEnv, "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=")
	encoded, err := temporalcrypto.DataConverter().ToPayload("hello")
	require.NoError(t, err)

	tests := []struct {
		name    string
		payload *commonpb.Payload
		want    string
	}{
		{name: "nil payload", payload: nil, want: ""},
		{name: "converter payload", payload: encoded, want: "hello"},
		{
			name:    "payload without encoding falls back to raw data",
			payload: &commonpb.Payload{Data: []byte(`"raw"`)},
			want:    "raw",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, DecodeStringPayload(tc.payload))
		})
	}
}
