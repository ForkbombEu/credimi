// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apierror

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

type APIError struct {
	Code    int    `json:"status"`
	Domain  string `json:"error"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("[%s:%s] %s", e.Domain, e.Reason, e.Message)
}

func New(code int, domain, reason, message string) *APIError {
	return &APIError{
		Code:    code,
		Domain:  domain,
		Reason:  reason,
		Message: message,
	}
}

func (e *APIError) JSON(r *core.RequestEvent) error {
	return r.JSON(e.Code, e)
}
