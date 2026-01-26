// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"encoding/json"
	"strings"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

var sensitiveKeyParts = []string{
	"token",
	"password",
	"secret",
	"credential",
	"private",
	"api_key",
	"apikey",
	"authorization",
	"bearer",
	"client_secret",
}

// MaskSensitiveFields returns a deep-copied value with sensitive fields masked.
func MaskSensitiveFields(value any) any {
	switch v := value.(type) {
	case map[string]any:
		masked := make(map[string]any, len(v))
		for key, entry := range v {
			if isSensitiveKey(key) {
				masked[key] = "***"
				continue
			}
			masked[key] = MaskSensitiveFields(entry)
		}
		return masked
	case []any:
		masked := make([]any, len(v))
		for index, entry := range v {
			masked[index] = MaskSensitiveFields(entry)
		}
		return masked
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

// PrettyMaskingCodec masks sensitive fields in JSON payloads for display.
type PrettyMaskingCodec struct{}

// Encode is a no-op to avoid mutating payloads at write time.
func (PrettyMaskingCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	return payloads, nil
}

// Decode applies masking + pretty JSON formatting when possible.
func (PrettyMaskingCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	for _, payload := range payloads {
		encoding := string(payload.GetMetadata()[converter.MetadataEncoding])
		if encoding != converter.MetadataEncodingJSON {
			continue
		}
		var decoded any
		if err := json.Unmarshal(payload.GetData(), &decoded); err != nil {
			continue
		}
		masked := MaskSensitiveFields(decoded)
		pretty, err := json.MarshalIndent(masked, "", "  ")
		if err != nil {
			continue
		}
		payload.Data = pretty
	}
	return payloads, nil
}

// NewPrettyMaskingDataConverter returns a DataConverter that masks JSON payloads.
func NewPrettyMaskingDataConverter() converter.DataConverter {
	return converter.NewCodecDataConverter(
		converter.GetDefaultDataConverter(),
		PrettyMaskingCodec{},
	)
}
