// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalauth

import (
	"crypto/elliptic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceTokenIsCachedPerNamespace(t *testing.T) {
	signer, err := ParseKeys(encodedKey(t, elliptic.P256()))
	require.NoError(t, err)
	t0 := time.Now()
	now := t0
	tokens := &namespaceTokens{
		now:    func() time.Time { return now },
		caches: map[string]*tokenCache{},
	}
	getSigner := func() (*Signer, error) { return signer, nil }

	first, err := tokens.get("runner-a", getSigner)
	require.NoError(t, err)
	claims := jwt.MapClaims{}
	_, _, err = jwt.NewParser().ParseUnverified(first, claims)
	require.NoError(t, err)
	assert.Equal(t, UISubject, claims["sub"])
	assert.Equal(t, []any{"runner-a:write"}, claims["permissions"])

	other, err := tokens.get("runner-b", getSigner)
	require.NoError(t, err)
	assert.NotEqual(t, first, other)

	// The same token within its life, a new one refreshBefore its expiry.
	now = t0.Add(TokenTTL - refreshBefore - time.Second)
	again, err := tokens.get("runner-a", getSigner)
	require.NoError(t, err)
	assert.Equal(t, first, again)
	now = t0.Add(TokenTTL - refreshBefore + time.Second)
	renewed, err := tokens.get("runner-a", getSigner)
	require.NoError(t, err)
	assert.NotEqual(t, first, renewed)
}

func TestNamespaceTokenReportsSignerError(t *testing.T) {
	tokens := &namespaceTokens{now: time.Now, caches: map[string]*tokenCache{}}
	_, err := tokens.get("runner-a", func() (*Signer, error) { return nil, ErrKeysMissing })
	require.ErrorIs(t, err, ErrKeysMissing)
}
