// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package utils

import (
	"testing"
)

func TestCredimiError_ErrorWithoutContext(t *testing.T) {
	err := CredimiError{
		Code:      "CRE123",
		Component: "Auth",
		Location:  "LoginHandler",
		Message:   "Invalid credentials",
	}

	expected := "CRE123: Auth LoginHandler: Invalid credentials"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCredimiError_ErrorWithContext(t *testing.T) {
	err := CredimiError{
		Code:      "CRE456",
		Component: "DB",
		Location:  "PocketBase",
		Message:   "Connection timeout",
		Context:   []string{"retrying", "attempt=3"},
	}

	expected := "CRE456: DB PocketBase: Connection timeout (retrying, attempt=3)"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
