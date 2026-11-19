// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package signedtoken signs and verifies Credimi's own tokens:
// base64url(json).base64url(hmac), with a key derived from the users
// collection's auth token secret and a label naming the token kind, so
// rotating that secret revokes every token and a token of one kind is never
// accepted as another.
package signedtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

const minSecretLen = 32

var tokenEncoding = base64.RawURLEncoding

// ErrUnsigned rejects a token Credimi did not sign with the expected key.
var ErrUnsigned = errors.New("token not signed by Credimi")

// signingKey derives the signing key of label from the users collection's
// auth token secret.
func signingKey(app core.App, label string) ([]byte, error) {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, fmt.Errorf("find users collection: %w", err)
	}
	secret := users.AuthToken.Secret
	if len(secret) < minSecretLen {
		return nil, errors.New("users auth token secret is too short to sign tokens")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(label))
	return mac.Sum(nil), nil
}

func tokenMAC(key []byte, payload string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// Sign returns the token base64url(json).base64url(hmac) of v, signed with
// the key of label.
func Sign(app core.App, label string, v any) (string, error) {
	key, err := signingKey(app, label)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encode token: %w", err)
	}
	payload := tokenEncoding.EncodeToString(encoded)
	return payload + "." + tokenEncoding.EncodeToString(tokenMAC(key, payload)), nil
}

// Verify decodes into v a token signed with the key of label. It returns
// ErrUnsigned for a token that is malformed or signed otherwise, and other
// errors when the key cannot be derived.
func Verify(app core.App, label, token string, v any) error {
	key, err := signingKey(app, label)
	if err != nil {
		return err
	}
	payload, signature, ok := strings.Cut(token, ".")
	if !ok {
		return ErrUnsigned
	}
	mac, err := tokenEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(mac, tokenMAC(key, payload)) {
		return ErrUnsigned
	}
	encoded, err := tokenEncoding.DecodeString(payload)
	if err != nil || json.Unmarshal(encoded, v) != nil {
		return ErrUnsigned
	}
	return nil
}
