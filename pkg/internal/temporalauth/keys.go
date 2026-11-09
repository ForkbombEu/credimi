// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package temporalauth signs the JWTs that Temporal callers present and
// publishes the matching public keys as a JWKS file for Temporal's default
// claim mapper.
package temporalauth

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// KeysEnvVar holds a comma-separated list of base64-encoded PEM P-256
	// private keys. The first key signs; every key is published.
	KeysEnvVar = "CREDIMI_TEMPORAL_JWT_KEYS"
	// JWKSDirEnvVar overrides the directory the JWKS file is written to.
	JWKSDirEnvVar = "CREDIMI_TEMPORAL_JWKS_DIR"
	// JWKSFileName is the name of the JWKS file Temporal reads.
	JWKSFileName = "jwks.json"
	// TokenTTL is the lifetime of every signed token.
	TokenTTL = 15 * time.Minute
	// SystemPermission grants Credimi's own clients system-wide admin.
	SystemPermission = "temporal-system:admin"

	defaultJWKSDir = "temporal-jwks"
	refreshBefore  = 5 * time.Minute
	issuer         = "credimi"
)

// ErrKeysMissing is returned when no signing key is configured.
var ErrKeysMissing = errors.New(KeysEnvVar + " is not set")

// Signer signs Temporal tokens with the first key and publishes all keys.
type Signer struct {
	keys []signingKey
}

type signingKey struct {
	kid  string
	priv *ecdsa.PrivateKey
}

var defaultSigner = sync.OnceValues(func() (*Signer, error) {
	return ParseKeys(os.Getenv(KeysEnvVar))
})

// DefaultSigner returns the signer built from KeysEnvVar, parsed once per process.
func DefaultSigner() (*Signer, error) {
	return defaultSigner()
}

// JWKSDir returns the directory the JWKS file is written to.
func JWKSDir() string {
	if dir := strings.TrimSpace(os.Getenv(JWKSDirEnvVar)); dir != "" {
		return dir
	}
	return defaultJWKSDir
}

// ParseKeys parses a comma-separated list of base64-encoded PEM P-256 private keys.
func ParseKeys(value string) (*Signer, error) {
	signer := &Signer{}
	seen := map[string]struct{}{}
	for i, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		key, err := parseKey(entry)
		if err != nil {
			return nil, fmt.Errorf("%s entry %d: %w", KeysEnvVar, i+1, err)
		}
		if _, dup := seen[key.kid]; dup {
			return nil, fmt.Errorf("%s entry %d: duplicate key", KeysEnvVar, i+1)
		}
		seen[key.kid] = struct{}{}
		signer.keys = append(signer.keys, key)
	}
	if len(signer.keys) == 0 {
		return nil, ErrKeysMissing
	}
	return signer, nil
}

func parseKey(entry string) (signingKey, error) {
	pemBytes, err := base64.StdEncoding.DecodeString(entry)
	if err != nil {
		return signingKey{}, fmt.Errorf("decode base64: %w", err)
	}
	priv, err := jwt.ParseECPrivateKeyFromPEM(pemBytes)
	if err != nil {
		return signingKey{}, fmt.Errorf("parse private key: %w", err)
	}
	if priv.Curve.Params().Name != "P-256" {
		return signingKey{}, fmt.Errorf("curve %s is not P-256", priv.Curve.Params().Name)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return signingKey{}, fmt.Errorf("marshal public key: %w", err)
	}
	sum := sha256.Sum256(der)
	return signingKey{kid: hex.EncodeToString(sum[:])[:16], priv: priv}, nil
}

// Sign returns an ES256 token for subject with the given Temporal permissions
// and its expiry.
func (s *Signer) Sign(
	subject string,
	permissions []string,
	now time.Time,
) (string, time.Time, error) {
	expiresAt := now.Add(TokenTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss":         issuer,
		"sub":         subject,
		"permissions": permissions,
		"iat":         now.Unix(),
		"exp":         expiresAt.Unix(),
	})
	token.Header["kid"] = s.keys[0].kid
	signed, err := token.SignedString(s.keys[0].priv)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// JWKS returns the public keys as a JSON Web Key Set, in configuration order.
func (s *Signer) JWKS() ([]byte, error) {
	set := struct {
		Keys []jwk `json:"keys"`
	}{Keys: make([]jwk, 0, len(s.keys))}
	for _, key := range s.keys {
		point, err := key.priv.PublicKey.Bytes()
		if err != nil {
			return nil, fmt.Errorf("encode public key %s: %w", key.kid, err)
		}
		set.Keys = append(set.Keys, jwk{
			Kty: "EC",
			Crv: "P-256",
			Alg: "ES256",
			Use: "sig",
			Kid: key.kid,
			X:   base64.RawURLEncoding.EncodeToString(point[1:33]),
			Y:   base64.RawURLEncoding.EncodeToString(point[33:65]),
		})
	}
	return json.Marshal(set)
}

// WriteJWKS atomically replaces dir/JWKSFileName with the signer's JWKS.
func WriteJWKS(dir string, s *Signer) error {
	path := filepath.Join(dir, JWKSFileName)
	data, err := s.JWKS()
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	tmp, err := os.CreateTemp(dir, ".jwks-*.json")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	tmpName := tmp.Name()
	if err := writeAndClose(tmp, data); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeAndClose(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
