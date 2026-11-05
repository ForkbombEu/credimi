// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"errors"
	"log"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/observability"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// ErrorHandlingMiddleware renders every error of a Credimi route group as an
// apierror.Response, and records server errors (5xx) as exceptions on the
// request span.
//
// Handlers return *apierror.APIError. PocketBase errors (apis.New*Error,
// e.BadRequestError and the other RequestEvent helpers) keep their status, so a
// stray one never turns into a 500; any other error is a 500.
func ErrorHandlingMiddleware(e *core.RequestEvent) error {
	err := e.Next()
	if err == nil {
		return nil
	}

	var apiError *apierror.APIError
	if errors.As(err, &apiError) {
		log.Printf("Handled API error: %v", apiError)
		observability.RecordServerError(e.Request.Context(), err, apiError.Code)
		return e.JSON(apiError.Code, apiError.Response())
	}

	var pbError *router.ApiError
	if errors.As(err, &pbError) {
		log.Printf("Handled PocketBase error: %v", pbError)
		observability.RecordServerError(e.Request.Context(), err, pbError.Status)
		apiError = apierror.New(
			pbError.Status,
			"request",
			http.StatusText(pbError.Status),
			pbError.Message,
		)
		return e.JSON(apiError.Code, apiError.Response())
	}

	log.Printf("Unhandled error: %v", err)
	observability.RecordServerError(e.Request.Context(), err, http.StatusInternalServerError)
	apiError = apierror.New(
		http.StatusInternalServerError,
		"internal",
		"UnhandledException",
		err.Error(),
	)
	response := apiError.Response()
	response.Message = http.StatusText(http.StatusInternalServerError)
	return e.JSON(apiError.Code, response)
}
