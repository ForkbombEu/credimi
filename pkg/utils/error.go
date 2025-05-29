// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"fmt"
	"strings"
)

type CredimiError struct {
	Code      string
	Component string
	Location  string
	Message   string
	Context   []string
}

func (e CredimiError) Error() string {
	base := fmt.Sprintf("%s: %s %s: %s", e.Code, e.Component, e.Location, e.Message)
	if len(e.Context) > 0 {
		return fmt.Sprintf("%s (%s)", base, strings.Join(e.Context, ", "))
	}
	return base
}
