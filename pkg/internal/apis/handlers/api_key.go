// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later


package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"golang.org/x/crypto/bcrypt"
)

var ApiKeyRoutes = routing.RouteGroup{
	BaseURL: "/api/apikey",
	Routes: []routing.RouteDefinition{
		{
			Method:        "POST",
			Path:          "/generate",
			Handler:       GenerateApiKey,
			RequestSchema: GenerateApiKeyRequestSchema{},
			Description:   "Generate a new API key for the authenticated user.",
			Summary:       "Generate API Key",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				apis.RequireAuth(),
				{Func: middlewares.ErrorHandlingMiddleware},
			},
		},
	},
	Validation: true,
}

type GenerateApiKeyRequestSchema struct {
	Name string `json:"name" validate:"required"`
}

func GenerateApiKey() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		user := e.Auth.Id
		if user == "" {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"user_not_authenticated",
				"user must be authenticated to generate an API key",
			)
		}

		input, err := routing.GetValidatedInput[GenerateApiKeyRequestSchema](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			)
		}

		if input.Name == "" {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"name_is_required",
				"name is required",
			)
		}

		apiKeyBytes := make([]byte, 32)
		if _, err := rand.Read(apiKeyBytes); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_generate_api_key",
				err.Error(),
			)
		}
		apiKey := base64.URLEncoding.EncodeToString(apiKeyBytes)

		hashedKey, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_hash_api_key",
				err.Error(),
			)
		}

		apiKeysCollection, err := e.App.FindCollectionByNameOrId("api_keys")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_find_api_keys_collection",
				err.Error(),
			)
		}

		record := core.NewRecord(apiKeysCollection)
		record.Set("user", user)
		record.Set("key", string(hashedKey))
		record.Set("name", input.Name)

		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_create_api_key_record",
				err.Error(),
			)
		}

		return e.JSON(200, map[string]string{"api_key": apiKey})
	}
}
