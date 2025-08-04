// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var ApiKeyRoutes = routing.RouteGroup{
	BaseURL:    "/api/apikey",
	Validation: true,
	Routes: []routing.RouteDefinition{
		{
			Method:         "POST",
			Path:           "/generate",
			Handler:        GenerateApiKey,
			RequestSchema:  GenerateApiKeyRequestSchema{},
			ResponseSchema: GenerateApiKeyResponseSchema{},
			Description:    "Generate a new API key for the authenticated user.",
			Summary:        "Generate API Key",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				apis.RequireAuth(),
				{Func: middlewares.ErrorHandlingMiddleware},
			},
		},
		{
			Method:         "GET",
			Path:           "/authenticate",
			Description:    "Authenticate an API key and return Bearer token",
			Summary:        "Authenticate API Key",
			ResponseSchema: AuthenticateApiKeyResponseSchema{},
			Handler:        AuthenticateApiKey,
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				{Func: middlewares.ErrorHandlingMiddleware},
			},
		},
	},
}

type GenerateApiKeyRequestSchema struct {
	Name string `json:"name" validate:"required"`
}
type GenerateApiKeyResponseSchema struct {
	ApiKey string `json:"api_key"`
}
type AuthenticateApiKeyResponseSchema struct {
	Message string `json:"message"`
	Token   string `json:"token"`
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

		service := NewApiKeyService(NewAppAdapter(e.App))
		apiKey, err := service.GenerateApiKey(user, input.Name)
		if err != nil {
			return err
		}

		return e.JSON(200, map[string]string{"api_key": apiKey})
	}
}

func AuthenticateApiKey() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		apiKey := e.Request.Header.Get("X-Api-Key")
		if apiKey == "" {
			return apierror.New(
				http.StatusUnauthorized,
				"request.validation",
				"api_key_required",
				"API key is required for authentication",
			)
		}

		service := NewApiKeyService(NewAppAdapter(e.App))
		authRecord, err := service.AuthenticateApiKey(apiKey)
		if err != nil {
			return err
		}

		response, err := generateAuthenticateApiKeyResponse(apiKey, authRecord)
		if err != nil {
			return err
		}

		return e.JSON(http.StatusOK, response)
	}
}

type HasAuthToken interface {
	NewAuthToken() (string, error)
}

func generateAuthenticateApiKeyResponse(apiKey string, authRecord HasAuthToken) (AuthenticateApiKeyResponseSchema, error) {
	token, err := authRecord.NewAuthToken()
	if err != nil {
		return AuthenticateApiKeyResponseSchema{}, apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_generate_auth_token",
			err.Error(),
		)
	}
	return AuthenticateApiKeyResponseSchema{
		Message: "API key authenticated successfully",
		Token:   token,
	}, nil
}
