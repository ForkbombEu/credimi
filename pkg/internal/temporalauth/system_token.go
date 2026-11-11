// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalauth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type tokenCache struct {
	mu        sync.Mutex
	now       func() time.Time
	token     string
	expiresAt time.Time
}

var systemTokens = &tokenCache{now: time.Now}

// SystemToken returns Credimi's own Temporal token, signed with
// SystemPermission and renewed refreshBefore its expiry. Its signature
// matches client.NewAPIKeyDynamicCredentials.
func SystemToken(context.Context) (string, error) {
	return systemTokens.get(DefaultSigner)
}

func (c *tokenCache) get(signer func() (*Signer, error)) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.token != "" && now.Before(c.expiresAt.Add(-refreshBefore)) {
		return c.token, nil
	}
	s, err := signer()
	if err != nil {
		return "", fmt.Errorf("sign Temporal system token: %w", err)
	}
	token, expiresAt, err := s.Sign(issuer, []string{SystemPermission}, now)
	if err != nil {
		return "", fmt.Errorf("sign Temporal system token: %w", err)
	}
	c.token, c.expiresAt = token, expiresAt
	return token, nil
}
