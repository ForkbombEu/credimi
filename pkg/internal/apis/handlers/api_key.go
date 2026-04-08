// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var ApiKeyRoutes = routing.RouteGroup{
	BaseURL: "/api/apikey",
	Routes: []routing.RouteDefinition{
		{
			Method:         "POST",
			Path:           "/generate",
			OperationID:    "apiKey.generate",
			Handler:        GenerateApiKey,
			RequestSchema:  GenerateApiKeyRequest{},
			ResponseSchema: GenerateApiKeyResponse{},
			Description:    "Generate a new API key for the authenticated user or superuser.",
			Summary:        "Generate API Key",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireAuthOrAPIKey(),
				{Func: middlewares.ErrorHandlingMiddleware},
			},
		},
		{
			Method:         "GET",
			Path:           "/authenticate",
			OperationID:    "apiKey.authenticate",
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

type APIKeyGenerationService interface {
	GenerateApiKey(userID, name string) (string, error)
	GenerateInternalAdminAPIKey(superuserID, name string) (string, error)
}

func GenerateApiKey() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil || strings.TrimSpace(e.Auth.Id) == "" {
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
		apiKey, err := generateAPIKeyForPrincipal(service, e.Auth, input.Name)
		if err != nil {
			apiErr := &apierror.APIError{}
			if errors.As(err, &apiErr) {
				return apiErr.JSON(e)
			}
			return err
		}

		return e.JSON(200, map[string]string{"api_key": apiKey})
	}
}

func generateAPIKeyForPrincipal(
	service APIKeyGenerationService,
	auth *core.Record,
	name string,
) (string, error) {
	if auth == nil || strings.TrimSpace(auth.Id) == "" {
		return "", apierror.New(
			http.StatusBadRequest,
			"request.validation",
			"user_not_authenticated",
			"user must be authenticated to generate an API key",
		)
	}

	collectionName := ""
	if auth.Collection() != nil {
		collectionName = auth.Collection().Name
	}

	switch collectionName {
	case apiKeyUserCollection:
		return service.GenerateApiKey(auth.Id, name)
	case apiKeySuperuserCollection:
		return service.GenerateInternalAdminAPIKey(auth.Id, name)
	default:
		return "", apierror.New(
			http.StatusForbidden,
			"request.validation",
			"unsupported_auth_collection",
			"API key generation is supported only for users or superusers",
		)
	}
}

func AuthenticateApiKey() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		apiKey := e.Request.Header.Get("Credimi-Api-Key")
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
			apiErr := &apierror.APIError{}
			if errors.As(err, &apiErr) {
				return apiErr.JSON(e)
			}
			return err
		}

		response, err := generateAuthenticateApiKeyResponse(authRecord)
		if err != nil {
			apiErr := &apierror.APIError{}
			if errors.As(err, &apiErr) {
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
