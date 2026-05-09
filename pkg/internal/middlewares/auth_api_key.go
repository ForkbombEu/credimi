// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"golang.org/x/crypto/bcrypt"
)

const (
	RequireAuthOrAPIKeyMiddlewareID = "requireAuthOrAPIKey"

	apiKeyHeaderName         = "Credimi-Api-Key"
	apiKeyScopeFieldName     = "key_type"
	apiKeyScopeUser          = "user"
	apiKeyScopeInternalAdmin = "internal_admin"
	internalAdminOwnerTable  = "_superusers"
	userOwnerTable           = "users"
)

// RequireAuthOrAPIKey accepts either a Bearer token or a user-scoped API key.
func RequireAuthOrAPIKey() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id: RequireAuthOrAPIKeyMiddlewareID,
		Func: func(e *core.RequestEvent) error {
			authHeader := strings.TrimSpace(e.Request.Header.Get("Authorization"))
			if authHeader == "" {
				apiKey := strings.TrimSpace(e.Request.Header.Get(apiKeyHeaderName))
				if apiKey == "" {
					return apierror.New(
						http.StatusUnauthorized,
						"request.validation",
						"authentication_required",
						"Bearer token or Credimi-Api-Key is required",
					).JSON(e)
				}

				principal, apiErr := authenticateAPIKeyByScope(e.App, apiKey, apiKeyScopeUser)
				if apiErr != nil {
					return apiErr.JSON(e)
				}
				e.Auth = principal

				return e.Next()
			}

			if err := apis.RequireAuth().Func(e); err == nil {
				return nil
			}
			if strings.HasPrefix(authHeader, "Bearer ") {
				e.Request.Header.Set(
					"Authorization",
					strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")),
				)
				return apis.RequireAuth().Func(e)
			}
			e.Request.Header.Set("Authorization", "Bearer "+authHeader)
			return apis.RequireAuth().Func(e)
		},
	}
}

// RequireInternalAdminAPIKey requires an internal-admin API key.
func RequireInternalAdminAPIKey() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Func: func(e *core.RequestEvent) error {
			apiKey := strings.TrimSpace(e.Request.Header.Get(apiKeyHeaderName))
			if apiKey == "" {
				return apierror.New(
					http.StatusUnauthorized,
					"request.validation",
					"api_key_required",
					"Credimi-Api-Key is required",
				).JSON(e)
			}

			principal, apiErr := authenticateAPIKeyByScope(e.App, apiKey, apiKeyScopeInternalAdmin)
			if apiErr != nil {
				return apiErr.JSON(e)
			}
			e.Auth = principal

			return e.Next()
		},
	}
}

// RequireInternalAdminOrAuth accepts either an internal-admin API key or a user auth context.
func RequireInternalAdminOrAuth() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Func: func(e *core.RequestEvent) error {
			apiKey := strings.TrimSpace(e.Request.Header.Get(apiKeyHeaderName))
			if apiKey != "" {
				principal, apiErr := authenticateAPIKeyByScope(
					e.App,
					apiKey,
					apiKeyScopeInternalAdmin,
				)
				if apiErr == nil {
					e.Auth = principal
					return e.Next()
				}
			}

			return RequireAuthOrAPIKey().Func(e)
		},
	}
}

func authenticateAPIKeyByScope(
	app core.App,
	apiKey string,
	requiredScope string,
) (*core.Record, *apierror.APIError) {
	records, err := app.FindRecordsByFilter("api_keys", "", "", 0, 0)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_find_api_key_records",
			err.Error(),
		)
	}

	matched := findMatchingAPIKeyRecord(records, apiKey)
	if matched == nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"invalid_api_key",
			"Invalid API key provided",
		)
	}

	if matched.GetBool("revoked") {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"revoked_api_key",
			"API key is revoked",
		)
	}

	expiresAt := matched.GetDateTime("expires_at")
	if !expiresAt.IsZero() && expiresAt.Time().Before(time.Now().UTC()) {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"expired_api_key",
			"API key is expired",
		)
	}

	userID := matched.GetString("user")
	superuserID := matched.GetString("superuser")
	hasUser := strings.TrimSpace(userID) != ""
	hasSuperuser := strings.TrimSpace(superuserID) != ""
	if hasUser == hasSuperuser && !(requiredScope == apiKeyScopeInternalAdmin && hasSuperuser) {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"invalid_api_key_owner",
			"API key owner must be exactly one of user or superuser",
		)
	}

	scope := matched.GetString(apiKeyScopeFieldName)
	if scope == "" {
		if hasSuperuser {
			scope = apiKeyScopeInternalAdmin
		} else {
			scope = apiKeyScopeUser
		}
	}
	if scope != requiredScope {
		return nil, apierror.New(
			http.StatusForbidden,
			"request.validation",
			"insufficient_api_key_scope",
			"API key does not have required scope",
		)
	}

	ownerCollection := userOwnerTable
	ownerID := userID
	if scope == apiKeyScopeInternalAdmin {
		if superuserID != "" {
			ownerCollection = internalAdminOwnerTable
			ownerID = superuserID
		}
	}

	principal, err := app.FindRecordById(ownerCollection, ownerID)
	if err != nil {
		if scope == apiKeyScopeInternalAdmin && userID != "" {
			principal, err = app.FindRecordById(userOwnerTable, userID)
		}
	}
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_find_principal",
			err.Error(),
		)
	}
	if principal == nil {
		if scope == apiKeyScopeInternalAdmin && userID != "" {
			principal, _ = app.FindRecordById(userOwnerTable, userID)
		}
	}
	if principal == nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"principal_not_found",
			"Principal associated with the API key not found",
		)
	}

	return principal, nil
}

func findMatchingAPIKeyRecord(records []*core.Record, apiKey string) *core.Record {
	for _, record := range records {
		if record == nil {
			continue
		}
		hash := record.GetString("key")
		if hash == "" {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey)); err == nil {
			return record
		}
	}
	return nil
}
