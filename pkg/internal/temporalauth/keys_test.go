// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodedKey(t *testing.T, curve elliptic.Curve) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return base64.StdEncoding.EncodeToString(pemBytes)
}

func sec1Key(t *testing.T) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	return base64.StdEncoding.EncodeToString(pemBytes)
}

func TestParseKeysErrors(t *testing.T) {
	valid := encodedKey(t, elliptic.P256())
	notPEM := base64.StdEncoding.EncodeToString([]byte("not a pem"))
	tests := []struct {
		name    string
		value   string
		missing bool
		want    string
	}{
		{name: "empty", value: "", missing: true},
		{name: "only separators", value: " , ", missing: true},
		{name: "not base64", value: valid + ",%%%", want: "entry 2"},
		{name: "not PEM", value: notPEM, want: "entry 1"},
		{name: "P-384", value: encodedKey(t, elliptic.P384()), want: "entry 1"},
		{name: "duplicate", value: valid + "," + valid, want: "entry 2: duplicate key"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseKeys(tc.value)
			require.Error(t, err)
			if tc.missing {
				assert.ErrorIs(t, err, ErrKeysMissing)
				return
			}
			assert.Contains(t, err.Error(), KeysEnvVar+" "+tc.want)
		})
	}
}

func TestParseKeysAcceptsPKCS8AndSEC1(t *testing.T) {
	signer, err := ParseKeys(" " + encodedKey(t, elliptic.P256()) + " , " + sec1Key(t) + ",")
	require.NoError(t, err)
	require.Len(t, signer.keys, 2)
	assert.Len(t, signer.keys[0].kid, 16)
	assert.NotEqual(t, signer.keys[0].kid, signer.keys[1].kid)
}

func publicKeysFromJWKS(t *testing.T, data []byte) map[string]*ecdsa.PublicKey {
	t.Helper()
	var set struct {
		Keys []jwk `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(data, &set))
	keys := map[string]*ecdsa.PublicKey{}
	for _, k := range set.Keys {
		assert.Equal(t, "EC", k.Kty)
		assert.Equal(t, "P-256", k.Crv)
		assert.Equal(t, "ES256", k.Alg)
		x, err := base64.RawURLEncoding.DecodeString(k.X)
		require.NoError(t, err)
		y, err := base64.RawURLEncoding.DecodeString(k.Y)
		require.NoError(t, err)
		keys[k.Kid] = &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		}
	}
	return keys
}

func TestSignVerifiesAgainstJWKS(t *testing.T) {
	signer, err := ParseKeys(encodedKey(t, elliptic.P256()) + "," + encodedKey(t, elliptic.P256()))
	require.NoError(t, err)
	data, err := signer.JWKS()
	require.NoError(t, err)
	keys := publicKeysFromJWKS(t, data)
	require.Len(t, keys, 2)

	t0 := time.Now().Truncate(time.Second)
	token, expiresAt, err := signer.Sign("s", []string{"ns:write"}, t0)
	require.NoError(t, err)
	assert.Equal(t, t0.Add(TokenTTL), expiresAt)

	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (any, error) {
		assert.Equal(t, signer.keys[0].kid, tok.Header["kid"])
		key, ok := keys[tok.Header["kid"].(string)]
		require.True(t, ok)
		return key, nil
	}, jwt.WithValidMethods([]string{"ES256"}))
	require.NoError(t, err)
	claims := parsed.Claims.(jwt.MapClaims)
	assert.Equal(t, "s", claims["sub"])
	assert.Equal(t, "credimi", claims["iss"])
	assert.Equal(t, []any{"ns:write"}, claims["permissions"])
	assert.InDelta(t, float64(t0.Add(15*time.Minute).Unix()), claims["exp"], 0)
}

func TestWriteJWKSReplacesFileAtomically(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "jwks")
	first, err := ParseKeys(encodedKey(t, elliptic.P256()))
	require.NoError(t, err)
	second, err := ParseKeys(encodedKey(t, elliptic.P256()))
	require.NoError(t, err)

	for _, signer := range []*Signer{first, second} {
		require.NoError(t, WriteJWKS(dir, signer))
		want, err := signer.JWKS()
		require.NoError(t, err)
		path := filepath.Join(dir, JWKSFileName)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, ".jwks-*"))
	require.NoError(t, err)
	assert.Empty(t, leftovers)
}

func TestTokenCacheRenewsBeforeExpiry(t *testing.T) {
	signer, err := ParseKeys(encodedKey(t, elliptic.P256()))
	require.NoError(t, err)
	t0 := time.Now()
	now := t0
	cache := &tokenCache{
		now:         func() time.Time { return now },
		subject:     issuer,
		permissions: []string{SystemPermission},
	}
	getSigner := func() (*Signer, error) { return signer, nil }

	first, err := cache.get(getSigner)
	require.NoError(t, err)
	now = t0.Add(9 * time.Minute)
	again, err := cache.get(getSigner)
	require.NoError(t, err)
	assert.Equal(t, first, again)

	now = t0.Add(11 * time.Minute)
	renewed, err := cache.get(getSigner)
	require.NoError(t, err)
	assert.NotEqual(t, first, renewed)
}

func TestTokenCacheReportsSignerError(t *testing.T) {
	cache := &tokenCache{now: time.Now, subject: issuer}
	_, err := cache.get(func() (*Signer, error) { return nil, ErrKeysMissing })
	require.ErrorIs(t, err, ErrKeysMissing)
	assert.Contains(t, err.Error(), "sign Temporal token for credimi")
}
