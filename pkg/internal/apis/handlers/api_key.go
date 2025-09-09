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
	BaseURL: "/api/apikey",
	Routes: []routing.RouteDefinition{
		{
			Method:         "POST",
			Path:           "/generate",
			Handler:        GenerateApiKey,
			RequestSchema:  GenerateApiKeyRequest{},
			ResponseSchema: GenerateApiKeyResponse{},
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
			ResponseSchema: AuthenticateApiKeyResponse{},
			Handler:        AuthenticateApiKey,
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				{Func: middlewares.ErrorHandlingMiddleware},
			},
		},
	},
	AuthenticationRequired: false,
}

type GenerateApiKeyRequest struct {
	Name string `json:"name" validate:"required"`
}
type GenerateApiKeyResponse struct {
	ApiKey string `json:"api_key"`
}
type AuthenticateApiKeyResponse struct {
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
			).JSON(e)
		}

		input, err := routing.GetValidatedInput[GenerateApiKeyRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			).JSON(e)
		}

		service := NewApiKeyService(NewAppAdapter(e.App))
		apiKey, err := service.GenerateApiKey(user, input.Name)
		if err != nil {
			if apiErr, ok := err.(*apierror.APIError); ok {
				return apiErr.JSON(e)
			}
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
			).JSON(e)
		}

		service := NewApiKeyService(NewAppAdapter(e.App))
		authRecord, err := service.AuthenticateApiKey(apiKey)
		if err != nil {
			if apiErr, ok := err.(*apierror.APIError); ok {
				return apiErr.JSON(e)
			}
			return err
		}

		response, err := generateAuthenticateApiKeyResponse(authRecord)
		if err != nil {
			if apiErr, ok := err.(*apierror.APIError); ok {
				return apiErr.JSON(e)
			}
			return err
		}

		return e.JSON(http.StatusOK, response)
	}
}

type HasAuthToken interface {
	NewAuthToken() (string, error)
}

func generateAuthenticateApiKeyResponse(
	authRecord HasAuthToken,
) (AuthenticateApiKeyResponse, error) {
	token, err := authRecord.NewAuthToken()
	if err != nil {
		return AuthenticateApiKeyResponse{}, apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_generate_auth_token",
			err.Error(),
		)
	}
	return AuthenticateApiKeyResponse{
		Message: "API key authenticated successfully",
		Token:   token,
	}, nil
}
