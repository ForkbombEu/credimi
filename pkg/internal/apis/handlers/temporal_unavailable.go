// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
)

// temporalUnavailableError is the answer of a Credimi route-group handler that
// cannot obtain a Temporal client: the request may succeed once Temporal is
// reachable again, so it is a 503 and not a server fault.
func temporalUnavailableError() *apierror.APIError {
	return apierror.New(
		http.StatusServiceUnavailable,
		"temporal",
		"temporal_unavailable",
		"cannot connect to Temporal",
	)
}
