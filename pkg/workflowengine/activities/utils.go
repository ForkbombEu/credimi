// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"net/http"
	"strings"
)

func TrimInput(s string) string {
	return strings.TrimSpace(s)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
