// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSDJWTKBJWTPresentRejectsNoneAlgorithm(t *testing.T) {
	presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	digest := sha256.Sum256([]byte(presentation.SDJWT))
	presentation.KeyBindingJWT = unsignedJWT(t,
		map[string]any{"alg": "none", "typ": "kb+jwt"},
		map[string]any{
			"iat":     time.Now().Unix(),
			"aud":     "x509_hash:verifier.example",
			"nonce":   "fcaf-device-binding-012",
			"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
		},
	)

	result := SDJWTKBJWTPresentValidator{}.Validate(
		context.Background(),
		Input{Value: presentation},
	)

	require.Equal(t, StatusFail, result.Status, result.Message)
	require.Contains(t, result.Message, `"none"`)
}

func TestSDJWTKBJWTPresentRejectsEmptySignatureSegment(t *testing.T) {
	presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	digest := sha256.Sum256([]byte(presentation.SDJWT))
	header := base64.RawURLEncoding.EncodeToString(
		testJSONBytes(t, map[string]any{"alg": "ES256", "typ": "kb+jwt"}),
	)
	payload := base64.RawURLEncoding.EncodeToString(
		testJSONBytes(t, map[string]any{
			"iat":     time.Now().Unix(),
			"aud":     "x509_hash:verifier.example",
			"nonce":   "fcaf-test",
			"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
		}),
	)
	presentation.KeyBindingJWT = header + "." + payload + "."

	result := SDJWTKBJWTPresentValidator{}.Validate(
		context.Background(),
		Input{Value: presentation},
	)

	require.Equal(t, StatusFail, result.Status, result.Message)
	require.Contains(t, result.Message, "signature segment")
}

func TestSDJWTKBJWTPresentRejectsNonBase64URLSignature(t *testing.T) {
	presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	digest := sha256.Sum256([]byte(presentation.SDJWT))
	header := base64.RawURLEncoding.EncodeToString(
		testJSONBytes(t, map[string]any{"alg": "ES256", "typ": "kb+jwt"}),
	)
	payload := base64.RawURLEncoding.EncodeToString(
		testJSONBytes(t, map[string]any{
			"iat":     time.Now().Unix(),
			"aud":     "x509_hash:verifier.example",
			"nonce":   "fcaf-test",
			"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
		}),
	)
	presentation.KeyBindingJWT = header + "." + payload + ".not+valid+base64url"

	result := SDJWTKBJWTPresentValidator{}.Validate(
		context.Background(),
		Input{Value: presentation},
	)

	require.Equal(t, StatusFail, result.Status, result.Message)
	require.Contains(t, result.Message, "signature segment")
}

func TestSDJWTKBJWTPresentRejectsUnboundSDHash(t *testing.T) {
	presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	digest := sha256.Sum256([]byte("different-presentation~"))
	header := base64.RawURLEncoding.EncodeToString(
		testJSONBytes(t, map[string]any{"alg": "ES256", "typ": "kb+jwt"}),
	)
	payload := base64.RawURLEncoding.EncodeToString(
		testJSONBytes(t, map[string]any{
			"iat":     time.Now().Unix(),
			"aud":     "x509_hash:verifier.example",
			"nonce":   "fcaf-test",
			"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
		}),
	)
	presentation.KeyBindingJWT = header + "." + payload + ".c2lnbmF0dXJl"

	result := SDJWTKBJWTPresentValidator{}.Validate(
		context.Background(),
		Input{Value: presentation},
	)

	require.Equal(t, StatusFail, result.Status, result.Message)
	require.Contains(t, result.Message, "sd_hash does not bind")
}

func TestSDJWTKBJWTPresentRejectsTrailingPayloadBytes(t *testing.T) {
	presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	digest := sha256.Sum256([]byte(presentation.SDJWT))
	header := base64.RawURLEncoding.EncodeToString(
		testJSONBytes(t, map[string]any{"alg": "ES256", "typ": "kb+jwt"}),
	)
	payload := base64.RawURLEncoding.EncodeToString(
		append(
			testJSONBytes(t, map[string]any{
				"iat":     time.Now().Unix(),
				"aud":     "x509_hash:verifier.example",
				"nonce":   "fcaf-test",
				"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
			}),
			[]byte("trailing-garbage")...,
		),
	)
	presentation.KeyBindingJWT = header + "." + payload + ".c2lnbmF0dXJl"

	result := SDJWTKBJWTPresentValidator{}.Validate(
		context.Background(),
		Input{Value: presentation},
	)

	require.Equal(t, StatusFail, result.Status, result.Message)
	require.Contains(t, result.Message, "trailing bytes")
}

func testJSONBytes(t *testing.T, v map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}
