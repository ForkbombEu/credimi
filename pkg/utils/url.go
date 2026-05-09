// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"net/url"
)

// JoinURL appends path parts to a base URL and returns the original base when
// the input cannot be parsed, avoiding nil dereferences in callers.
func JoinURL(base string, parts ...string) string {
	u, err := url.Parse(base)
	if err != nil || u == nil {
		return base
	}
	for _, p := range parts {
		u.Path, _ = url.JoinPath(u.Path, p)
	}
	return u.String()
}
