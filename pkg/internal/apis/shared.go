// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

type APIErrorDetail struct {
	Domain  string `json:"domain"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type APIError struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Errors  []APIErrorDetail `json:"errors"`
}

type APIErrorResponse struct {
	APIVersion string   `json:"apiVersion"`
	Error      APIError `json:"error"`
}


type SaveVariablesAndStartRequestInput map[string]struct {
	Format string      `json:"format" validate:"required"`
	Data   interface{} `json:"data" validate:"required"`
}
