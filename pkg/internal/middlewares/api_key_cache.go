// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"crypto/hmac"
	"crypto/rand"
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
// against, so frequent callers skip the bcrypt scan over every key. Entries are
// keyed by an HMAC under a random per-process secret, so the cache holds no
// value that can be checked against a guessed key outside the process.
type verifiedAPIKeys struct {
	mu      sync.Mutex
	secret  []byte
	entries map[[sha256.Size]byte]verifiedAPIKey
}

func (c *verifiedAPIKeys) sum(apiKey string) [sha256.Size]byte {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(apiKey))
	var sum [sha256.Size]byte
	copy(sum[:], mac.Sum(nil))
	return sum
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
	cache := &verifiedAPIKeys{
		secret:  make([]byte, 32),
		entries: map[[sha256.Size]byte]verifiedAPIKey{},
	}
	if _, err := rand.Read(cache.secret); err != nil {
		panic("api key cache secret: " + err.Error())
	}
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
			sum := cache.sum(apiKey)

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
