// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

const SecretsEncryptionKeyEnv = "CREDIMI_TEMPORAL_SECRETS_ENCRYPTION_KEY"
const SecretsEncryptionDisabledEnv = "CREDIMI_TEMPORAL_SECRETS_ENCRYPTION_DISABLED"

const (
	encryptedSecretsMarker = "__credimi_encrypted_secrets"
	encryptedSecretsAlg    = "AES-256-GCM"
)

type SecretsJSONPayloadConverter struct {
	key []byte
}

func DataConverter() converter.DataConverter {
	key, disabled, err := loadKeyFromEnv()
	if disabled {
		return converter.GetDefaultDataConverter()
	}
	if err != nil {
		panic(err)
	}
	return NewDataConverter(key)
}

func NewDataConverter(key []byte) converter.DataConverter {
	return converter.NewCompositeDataConverter(
		converter.NewNilPayloadConverter(),
		converter.NewByteSlicePayloadConverter(),
		converter.NewProtoJSONPayloadConverter(),
		converter.NewProtoPayloadConverter(),
		NewSecretsJSONPayloadConverter(key),
	)
}

func NewSecretsJSONPayloadConverter(key []byte) *SecretsJSONPayloadConverter {
	copied := append([]byte(nil), key...)
	return &SecretsJSONPayloadConverter{key: copied}
}

func (c *SecretsJSONPayloadConverter) Encoding() string {
	return converter.MetadataEncodingJSON
}

func (c *SecretsJSONPayloadConverter) ToPayload(value any) (*commonpb.Payload, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", converter.ErrUnableToEncode, err)
	}

	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, fmt.Errorf("%w: %v", converter.ErrUnableToEncode, err)
	}

	transformed, err := encryptSecretsFields(c.key, reflect.ValueOf(value), generic)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", converter.ErrUnableToEncode, err)
	}

	data, err = json.Marshal(transformed)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", converter.ErrUnableToEncode, err)
	}

	return &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(c.Encoding()),
		},
		Data: data,
	}, nil
}

func (c *SecretsJSONPayloadConverter) FromPayload(payload *commonpb.Payload, valuePtr any) error {
	var generic any
	if err := json.Unmarshal(payload.GetData(), &generic); err != nil {
		return fmt.Errorf("%w: %v", converter.ErrUnableToDecode, err)
	}

	transformed, err := decryptSecretsFields(c.key, generic)
	if err != nil {
		return fmt.Errorf("%w: %v", converter.ErrUnableToDecode, err)
	}

	data, err := json.Marshal(transformed)
	if err != nil {
		return fmt.Errorf("%w: %v", converter.ErrUnableToDecode, err)
	}

	if err := json.Unmarshal(data, valuePtr); err != nil {
		return fmt.Errorf("%w: %v", converter.ErrUnableToDecode, err)
	}

	return nil
}

func (c *SecretsJSONPayloadConverter) ToString(payload *commonpb.Payload) string {
	return string(payload.GetData())
}

func loadKeyFromEnv() ([]byte, bool, error) {
	if disabled := strings.EqualFold(strings.TrimSpace(os.Getenv(SecretsEncryptionDisabledEnv)), "true"); disabled {
		return nil, true, nil
	}

	raw := strings.TrimSpace(os.Getenv(SecretsEncryptionKeyEnv))
	if raw == "" {
		return nil, false, invalidKeyError()
	}

	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, false, invalidKeyError()
	}

	return key, false, nil
}

func invalidKeyError() error {
	return fmt.Errorf("%s must be a base64-encoded 32-byte key", SecretsEncryptionKeyEnv)
}

func encryptSecretsFields(key []byte, original reflect.Value, encoded any) (any, error) {
	original = dereferenceValue(original)
	if !original.IsValid() {
		return encoded, nil
	}

	if isBoundarySecretsCarrier(original.Type()) {
		return encryptStructFields(key, original, encoded, true)
	}

	switch original.Kind() {
	case reflect.Struct:
		return encryptStructFields(key, original, encoded, false)
	case reflect.Slice, reflect.Array:
		encodedSlice, ok := encoded.([]any)
		if !ok {
			return encoded, nil
		}
		out := make([]any, len(encodedSlice))
		copy(out, encodedSlice)
		for i := 0; i < len(encodedSlice) && i < original.Len(); i++ {
			next, err := encryptSecretsFields(key, original.Index(i), encodedSlice[i])
			if err != nil {
				return nil, err
			}
			out[i] = next
		}
		return out, nil
	case reflect.Map:
		encodedMap, ok := encoded.(map[string]any)
		if !ok {
			return encoded, nil
		}
		out := make(map[string]any, len(encodedMap))
		for k, v := range encodedMap {
			out[k] = v
		}
		for _, mapKey := range original.MapKeys() {
			if mapKey.Kind() != reflect.String {
				continue
			}
			keyString := mapKey.String()
			childEncoded, ok := out[keyString]
			if !ok {
				continue
			}
			next, err := encryptSecretsFields(key, original.MapIndex(mapKey), childEncoded)
			if err != nil {
				return nil, err
			}
			out[keyString] = next
		}
		return out, nil
	default:
		return encoded, nil
	}
}

func decryptSecretsFields(key []byte, value any) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			if k == "secrets" && isEncryptedSecretsEnvelope(v) {
				decrypted, err := decryptJSONValue(key, v)
				if err != nil {
					return nil, err
				}
				out[k] = decrypted
				continue
			}

			next, err := decryptSecretsFields(key, v)
			if err != nil {
				return nil, err
			}
			out[k] = next
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			next, err := decryptSecretsFields(key, item)
			if err != nil {
				return nil, err
			}
			out[i] = next
		}
		return out, nil
	default:
		return value, nil
	}
}

func encryptJSONValue(key []byte, value any) (map[string]any, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal secrets: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return map[string]any{
		encryptedSecretsMarker: true,
		"version":              float64(1),
		"alg":                  encryptedSecretsAlg,
		"nonce":                base64.StdEncoding.EncodeToString(nonce),
		"ciphertext":           base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func decryptJSONValue(key []byte, value any) (any, error) {
	envelope, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("encrypted secrets envelope is %T, expected object", value)
	}

	alg, ok := envelope["alg"].(string)
	if !ok || alg != encryptedSecretsAlg {
		return nil, fmt.Errorf("unsupported secrets algorithm")
	}

	version, ok := envelope["version"].(float64)
	if !ok || version != 1 {
		return nil, fmt.Errorf("unsupported secrets envelope version")
	}

	nonceValue, ok := envelope["nonce"].(string)
	if !ok {
		return nil, fmt.Errorf("missing secrets nonce")
	}
	ciphertextValue, ok := envelope["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("missing secrets ciphertext")
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceValue)
	if err != nil {
		return nil, fmt.Errorf("decode secrets nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextValue)
	if err != nil {
		return nil, fmt.Errorf("decode secrets ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid secrets nonce length")
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt secrets: %w", err)
	}

	var decoded any
	if err := json.Unmarshal(plaintext, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshal decrypted secrets: %w", err)
	}

	return decoded, nil
}

func isEncryptedSecretsEnvelope(value any) bool {
	envelope, ok := value.(map[string]any)
	if !ok {
		return false
	}
	marker, ok := envelope[encryptedSecretsMarker].(bool)
	return ok && marker
}

func isEmptySecretsValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case map[string]any:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	case string:
		return typed == ""
	default:
		return false
	}
}

func encryptStructFields(
	key []byte,
	original reflect.Value,
	encoded any,
	encryptSecrets bool,
) (any, error) {
	encodedMap, ok := encoded.(map[string]any)
	if !ok {
		return encoded, nil
	}

	out := make(map[string]any, len(encodedMap))
	for k, v := range encodedMap {
		out[k] = v
	}

	originalType := original.Type()
	for i := 0; i < original.NumField(); i++ {
		field := originalType.Field(i)
		if field.PkgPath != "" {
			continue
		}

		jsonName, include := jsonFieldName(field)
		if !include {
			continue
		}

		currentEncoded, ok := out[jsonName]
		if !ok {
			continue
		}

		if encryptSecrets && field.Name == "Secrets" && jsonName == "secrets" {
			if isEmptySecretsValue(currentEncoded) || isEncryptedSecretsEnvelope(currentEncoded) {
				continue
			}

			encrypted, err := encryptJSONValue(key, currentEncoded)
			if err != nil {
				return nil, err
			}
			out[jsonName] = encrypted
			continue
		}

		next, err := encryptSecretsFields(key, original.Field(i), currentEncoded)
		if err != nil {
			return nil, err
		}
		out[jsonName] = next
	}

	return out, nil
}

func dereferenceValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func isBoundarySecretsCarrier(typ reflect.Type) bool {
	return typ.PkgPath() == "github.com/forkbombeu/credimi/pkg/workflowengine" &&
		(typ.Name() == "WorkflowInput" || typ.Name() == "ActivityInput" || typ.Name() == "ActivityResult")
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	if tag == "" {
		return field.Name, true
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return field.Name, true
	}
	return name, true
}
