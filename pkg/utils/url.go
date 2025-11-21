// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"net/url"
)

func JoinURL(base string, parts ...string) string {
	u, _ := url.Parse(base)
	for _, p := range parts {
		u.Path, _ = url.JoinPath(u.Path, p)
	}
	return u.String()
}
