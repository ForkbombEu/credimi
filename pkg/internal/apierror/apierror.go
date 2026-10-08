// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apierror

import (
	"fmt"
)

// APIVersion is stamped on every error response of a Credimi route group.
const APIVersion = "2.0"

// APIError is the error returned by handlers of Credimi route groups.
// ErrorHandlingMiddleware renders it as the "error" object of a Response.
type APIError struct {
	Code    int    `json:"code"`
	Domain  string `json:"domain"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// Response is the JSON body of every error answered by a Credimi route group.
type Response struct {
	APIVersion string    `json:"apiVersion"`
	Message    string    `json:"message"`
	Error      *APIError `json:"error"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("[%s:%s] %s", e.Domain, e.Reason, e.Message)
}

// Response wraps the error in the response envelope, using its message as the
// top-level message.
func (e *APIError) Response() Response {
	return Response{APIVersion: APIVersion, Message: e.Message, Error: e}
}

func New(code int, domain, reason, message string) *APIError {
	return &APIError{
		Code:    code,
		Domain:  domain,
		Reason:  reason,
		Message: message,
	}
}
