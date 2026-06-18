// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalcrypto

import (
	"bytes"
	"encoding/base64"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

func TestDataConverterUsesEnvKey(t *testing.T) {
	t.Setenv(SecretsEncryptionDisabledEnv, "")
	t.Setenv(SecretsEncryptionKeyEnv, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)))

	dc := DataConverter()
	payload, err := dc.ToPayload(map[string]any{"hello": "world"})
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if !bytes.Contains(payload.GetData(), []byte(`"hello":"world"`)) {
		t.Fatalf("expected payload to encode successfully, got %s", string(payload.GetData()))
	}
}

func TestDataConverterDisabledUsesDefaultConverter(t *testing.T) {
	t.Setenv(SecretsEncryptionDisabledEnv, "true")
	t.Setenv(SecretsEncryptionKeyEnv, "")

	dc := DataConverter()
	payload, err := dc.ToPayload(map[string]any{
		"secrets": map[string]any{"token": "plain"},
	})
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}
	if !bytes.Contains(payload.GetData(), []byte("plain")) {
		t.Fatalf("expected plaintext payload with default converter, got %s", string(payload.GetData()))
	}
}

func TestDataConverterPanicsWithoutKey(t *testing.T) {
	t.Setenv(SecretsEncryptionDisabledEnv, "")
	t.Setenv(SecretsEncryptionKeyEnv, "")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = DataConverter()
}

func TestDataConverterPanicsWithInvalidKey(t *testing.T) {
	t.Setenv(SecretsEncryptionDisabledEnv, "")
	t.Setenv(SecretsEncryptionKeyEnv, "invalid")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = DataConverter()
}

func TestSecretsJSONPayloadConverterSkipsEmptyAndEncryptedSecrets(t *testing.T) {
	key := bytes.Repeat([]byte{4}, 32)
	dc := NewDataConverter(key)

	alreadyEncrypted := map[string]any{
		encryptedSecretsMarker: true,
		"version":              float64(1),
		"alg":                  encryptedSecretsAlg,
		"nonce":                "abc",
		"ciphertext":           "def",
	}

	payload, err := dc.ToPayload(map[string]any{
		"secrets":        map[string]any{},
		"nested":         []any{map[string]any{"secrets": alreadyEncrypted}},
		"other_secrets":  map[string]any{"token": "visible"},
		"secret_values":  map[string]any{"token": "visible"},
		"plain_secrets":  "unchanged",
		"empty_secrets2": "",
	})
	if err != nil {
		t.Fatalf("ToPayload failed: %v", err)
	}

	if bytes.Contains(payload.GetData(), []byte(`other_secrets":{"token":"visible"}`)) == false {
		t.Fatalf("expected non-matching key to stay plaintext, got %s", string(payload.GetData()))
	}
}

func TestSecretsJSONPayloadConverterEncodeDecodeErrors(t *testing.T) {
	key := bytes.Repeat([]byte{5}, 32)
	c := NewSecretsJSONPayloadConverter(key)

	if _, err := c.ToPayload(map[string]any{"secrets": func() {}}); err == nil {
		t.Fatal("expected encode error")
	}

	payload := &commonpb.Payload{
		Metadata: map[string][]byte{converter.MetadataEncoding: []byte(converter.MetadataEncodingJSON)},
		Data:     []byte(`not-json`),
	}
	var target map[string]any
	if err := c.FromPayload(payload, &target); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestDecryptJSONValueValidationErrors(t *testing.T) {
	key := bytes.Repeat([]byte{6}, 32)

	tests := []struct {
		name  string
		value any
	}{
		{name: "not object", value: "bad"},
		{name: "bad alg", value: map[string]any{encryptedSecretsMarker: true, "alg": "bad", "version": float64(1), "nonce": "a", "ciphertext": "b"}},
		{name: "bad version", value: map[string]any{encryptedSecretsMarker: true, "alg": encryptedSecretsAlg, "version": float64(2), "nonce": "a", "ciphertext": "b"}},
		{name: "missing nonce", value: map[string]any{encryptedSecretsMarker: true, "alg": encryptedSecretsAlg, "version": float64(1), "ciphertext": "b"}},
		{name: "missing ciphertext", value: map[string]any{encryptedSecretsMarker: true, "alg": encryptedSecretsAlg, "version": float64(1), "nonce": "a"}},
		{name: "bad nonce", value: map[string]any{encryptedSecretsMarker: true, "alg": encryptedSecretsAlg, "version": float64(1), "nonce": "%", "ciphertext": "a"}},
		{name: "wrong nonce length", value: map[string]any{encryptedSecretsMarker: true, "alg": encryptedSecretsAlg, "version": float64(1), "nonce": base64.StdEncoding.EncodeToString(make([]byte, 4)), "ciphertext": "a"}},
		{name: "bad ciphertext", value: map[string]any{encryptedSecretsMarker: true, "alg": encryptedSecretsAlg, "version": float64(1), "nonce": base64.StdEncoding.EncodeToString(make([]byte, 12)), "ciphertext": "%"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decryptJSONValue(key, tc.value); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDecryptJSONValueRejectsTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	envelope, err := encryptJSONValue(key, map[string]any{"token": "secret"})
	if err != nil {
		t.Fatalf("encryptJSONValue failed: %v", err)
	}
	envelope["ciphertext"] = base64.StdEncoding.EncodeToString([]byte("tampered"))

	if _, err := decryptJSONValue(key, envelope); err == nil {
		t.Fatal("expected decrypt error")
	}
}

func TestIsEmptySecretsValueVariants(t *testing.T) {
	if !isEmptySecretsValue(nil) || !isEmptySecretsValue(map[string]any{}) || !isEmptySecretsValue([]any{}) || !isEmptySecretsValue("") {
		t.Fatal("expected empty variants to be true")
	}
	if isEmptySecretsValue("x") || isEmptySecretsValue([]any{1}) || isEmptySecretsValue(map[string]any{"a": 1}) {
		t.Fatal("expected non-empty variants to be false")
	}
}
