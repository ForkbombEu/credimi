// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"crypto/sha256"
	"net/http"
	"strings"
	"sync"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

const maxVerifiedAPIKeys = 10000

// verifiedAPIKeys remembers which api_keys record and scope a key was verified
// against, so frequent callers skip the bcrypt scan over every key.
type verifiedAPIKeys struct {
	mu      sync.Mutex
	entries map[[sha256.Size]byte]verifiedAPIKey
}

type verifiedAPIKey struct {
	recordID string
	scope    string
}

func (c *verifiedAPIKeys) get(sum [sha256.Size]byte) (verifiedAPIKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[sum]
	return v, ok
}

func (c *verifiedAPIKeys) put(sum [sha256.Size]byte, v verifiedAPIKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= maxVerifiedAPIKeys {
		c.entries = map[[sha256.Size]byte]verifiedAPIKey{}
	}
	c.entries[sum] = v
}

func (c *verifiedAPIKeys) forget(sum [sha256.Size]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, sum)
}

// RequireCachedAPIKey accepts the same internal-admin and user-scoped keys as
// RequireInternalAdminOrAuth, for routes called every few seconds. A key is
// bcrypt-verified once per process; later requests reload its api_keys record
// by ID and check revocation, expiry, owner and scope again, so a revoked key
// is refused on its next request.
func RequireCachedAPIKey() *hook.Handler[*core.RequestEvent] {
	cache := &verifiedAPIKeys{entries: map[[sha256.Size]byte]verifiedAPIKey{}}
	return &hook.Handler[*core.RequestEvent]{
		Func: func(e *core.RequestEvent) error {
			apiKey := strings.TrimSpace(e.Request.Header.Get(apiKeyHeaderName))
			if apiKey == "" {
				return apierror.New(
					http.StatusUnauthorized,
					"request.validation",
					"authentication_required",
					"Credimi-Api-Key is required",
				)
			}
			sum := sha256.Sum256([]byte(apiKey))

			if v, ok := cache.get(sum); ok {
				record, err := e.App.FindRecordById("api_keys", v.recordID)
				if err != nil {
					cache.forget(sum)
					return apierror.New(
						http.StatusUnauthorized,
						"request.validation",
						"invalid_api_key",
						"Invalid API key provided",
					)
				}
				principal, apiErr := principalForAPIKey(e.App, record, v.scope)
				if apiErr != nil {
					cache.forget(sum)
					return apiErr
				}
				e.Auth = principal
				return e.Next()
			}

			record, apiErr := findAPIKeyRecord(e.App, apiKey)
			if apiErr != nil {
				return apiErr
			}
			scope := apiKeyScopeInternalAdmin
			principal, apiErr := principalForAPIKey(e.App, record, scope)
			if apiErr != nil {
				scope = apiKeyScopeUser
				if principal, apiErr = principalForAPIKey(e.App, record, scope); apiErr != nil {
					return apiErr
				}
			}
			cache.put(sum, verifiedAPIKey{recordID: record.Id, scope: scope})
			e.Auth = principal
			return e.Next()
		},
	}
}
