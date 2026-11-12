// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalauth

import (
	"sync"
	"time"
)

const (
	// UIPermissionRole is the role the action author chat UI gets on the
	// namespace of the session it serves.
	UIPermissionRole = "write"
	// UISubject is the subject of the action author chat UI's tokens.
	UISubject = "credimi-action-author-ui"
)

// namespaceTokens caches one chat UI token per namespace.
type namespaceTokens struct {
	mu     sync.Mutex
	now    func() time.Time
	caches map[string]*tokenCache
}

var uiTokens = &namespaceTokens{now: time.Now, caches: map[string]*tokenCache{}}

// NamespaceToken returns the action author chat UI's token for namespace
// (UISubject, permission <namespace>:UIPermissionRole), cached per namespace
// and renewed refreshBefore its expiry.
func NamespaceToken(namespace string) (string, error) {
	return uiTokens.get(namespace, DefaultSigner)
}

func (n *namespaceTokens) get(namespace string, signer func() (*Signer, error)) (string, error) {
	n.mu.Lock()
	cache, ok := n.caches[namespace]
	if !ok {
		cache = &tokenCache{
			now:         n.now,
			subject:     UISubject,
			permissions: []string{namespace + ":" + UIPermissionRole},
		}
		n.caches[namespace] = cache
	}
	n.mu.Unlock()
	return cache.get(signer)
}
