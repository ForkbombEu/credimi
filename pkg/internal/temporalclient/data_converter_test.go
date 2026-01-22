// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

func TestMaskSensitiveFields(t *testing.T) {
	input := map[string]any{
		"token": "secret",
		"nested": map[string]any{
			"password": "hidden",
			"value":    "keep",
		},
		"items": []any{
			map[string]any{"api_key": "abc"},
			"plain",
		},
	}

	masked := MaskSensitiveFields(input).(map[string]any)
	require.Equal(t, "***", masked["token"])
	require.Equal(t, "keep", masked["nested"].(map[string]any)["value"])
	require.Equal(t, "***", masked["nested"].(map[string]any)["password"])
	require.Equal(t, "***", masked["items"].([]any)[0].(map[string]any)["api_key"])
}

func TestPrettyMaskingCodecDecodeMasksJSON(t *testing.T) {
	codec := PrettyMaskingCodec{}
	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(converter.MetadataEncodingJSON),
		},
		Data: []byte(`{"token":"secret","value":1}`),
	}

	out, err := codec.Decode([]*commonpb.Payload{payload})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Contains(t, string(out[0].Data), "\"token\": \"***\"")
	require.Contains(t, string(out[0].Data), "\n  \"value\": 1\n")
}
