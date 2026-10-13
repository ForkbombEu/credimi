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

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
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

func TestSDJWTKBJWTAlgorithmEqualsValidator(t *testing.T) {
	tests := []struct {
		name       string
		algorithm  string
		wantStatus Status
	}{
		{name: "accepts beta Capture Wallet ES256", algorithm: "ES256", wantStatus: StatusPass},
		{name: "rejects another algorithm", algorithm: "ES384", wantStatus: StatusFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
			digest := sha256.Sum256([]byte(presentation.SDJWT))
			presentation.KeyBindingJWT = unsignedJWT(t,
				map[string]any{"alg": tt.algorithm, "typ": "kb+jwt"},
				map[string]any{
					"iat":     time.Now().Unix(),
					"aud":     "x509_hash:verifier.example",
					"nonce":   "fcaf-device-binding-012",
					"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
				},
			)

			result := SDJWTKBJWTAlgorithmEqualsValidator{}.Validate(
				context.Background(),
				Input{Value: presentation, Params: map[string]any{"algorithm": "ES256"}},
			)
			require.Equal(t, tt.wantStatus, result.Status, result.Message)
		})
	}
}

func testJSONBytes(t *testing.T, v map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}

func TestSDJWTKBJWTValidatorsReadEveryPresentation(t *testing.T) {
	valid := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	withoutKB := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	withoutKB.KeyBindingJWT = ""

	tests := []struct {
		name    string
		value   any
		status  Status
		message string
	}{
		{
			name:   "slice of presentation values",
			value:  []evidence.SDJWTPresentation{*valid, *valid},
			status: StatusPass,
		},
		{
			name:   "array of compact presentations",
			value:  []any{valid.Raw, valid.Raw},
			status: StatusPass,
		},
		{
			name:    "second presentation without KB-JWT is reported by index",
			value:   []*evidence.SDJWTPresentation{valid, withoutKB},
			status:  StatusFail,
			message: "presentation[1]: SD-JWT presentation does not contain a KB-JWT",
		},
		{
			name:    "array with non-string entry",
			value:   []any{valid.Raw, 1},
			status:  StatusFail,
			message: "SD-JWT presentation evidence is missing",
		},
		{
			name:    "array with malformed compact presentation",
			value:   []any{"not-an-sd-jwt"},
			status:  StatusFail,
			message: "SD-JWT presentation evidence is missing",
		},
		{
			name:    "empty array",
			value:   []any{},
			status:  StatusFail,
			message: "SD-JWT presentation evidence is missing",
		},
		{
			name:    "empty value slice",
			value:   []evidence.SDJWTPresentation{},
			status:  StatusFail,
			message: "SD-JWT presentation evidence is missing",
		},
		{
			name:    "nil presentation in pointer slice",
			value:   []*evidence.SDJWTPresentation{valid, nil},
			status:  StatusFail,
			message: "SD-JWT presentation evidence is missing",
		},
		{
			name:    "unsupported value",
			value:   42,
			status:  StatusFail,
			message: "SD-JWT presentation evidence is missing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			present := SDJWTKBJWTPresentValidator{}.Validate(
				context.Background(),
				Input{Value: test.value},
			)
			require.Equal(t, test.status, present.Status, present.Message)
			require.Contains(t, present.Message, test.message)

			algorithm := SDJWTKBJWTAlgorithmEqualsValidator{}.Validate(
				context.Background(),
				Input{Value: test.value, Params: map[string]any{"algorithm": "ES256"}},
			)
			require.Equal(t, test.status, algorithm.Status, algorithm.Message)
			require.Contains(t, algorithm.Message, test.message)
		})
	}
}

func TestSDJWTKBJWTAlgorithmEqualsRequiresAlgorithm(t *testing.T) {
	result := SDJWTKBJWTAlgorithmEqualsValidator{}.Validate(
		context.Background(),
		Input{Value: newKeyBindingPresentation(t, keyBindingFixtureOptions{})},
	)

	require.Equal(t, StatusError, result.Status)
	require.Equal(t, "algorithm is required", result.Message)
}
