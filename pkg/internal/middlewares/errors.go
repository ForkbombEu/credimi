// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"errors"
	"log"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
)

func ErrorHandlingMiddleware(e *core.RequestEvent) error {
	err := e.Next()
	if err == nil {
		return nil
	}

	var apiError *apierror.APIError
	if errors.As(err, &apiError) {
		// apiError ci facciamo quello che vogliamo (sentry, mail, etc)
		log.Printf("Handled API error: %v", apiError)
		return e.JSON(apiError.Code, map[string]interface{}{
			"apiVersion": "2.0",
			"message":    apiError.Message,
			"error": map[string]interface{}{
				"code":    apiError.Code,
				"domain":  apiError.Domain,
				"reason":  apiError.Reason,
				"message": apiError.Message,
			},
		})
	}
	log.Printf("Unhandled error: %v", err)

	return e.JSON(500, map[string]interface{}{
		"apiVersion": "2.0",
		"message":    "Internal Server Error",
		"error": map[string]interface{}{
			"code":    500,
			"domain":  "internal",
			"reason":  "UnhandledException",
			"message": err.Error(),
		},
	})
}
