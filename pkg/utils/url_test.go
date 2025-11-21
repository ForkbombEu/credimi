// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"testing"
)

func TestJoinURL(t *testing.T) {
	tests := []struct {
		base   string
		parts  []string
		expect string
	}{
		{"https://example.com", []string{"api", "v1", "users"}, "https://example.com/api/v1/users"},
		{"https://example.com/", []string{"api", "v1"}, "https://example.com/api/v1"},
		{"https://example.com/base/", []string{"sub", "page"}, "https://example.com/base/sub/page"},
		{"https://example.com", []string{}, "https://example.com"},
	}

	for _, tt := range tests {
		got := JoinURL(tt.base, tt.parts...)
		if got != tt.expect {
			t.Errorf("joinURL(%q, %v) = %q; want %q", tt.base, tt.parts, got, tt.expect)
		}
	}
}
