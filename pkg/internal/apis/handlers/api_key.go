// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
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
			Method:      "GET",
			Path:        "/authenticate",
			Description: "Authenticate an API key and return Bearer token",
			Summary:     "Authenticate API Key",
			ResponseSchema: AuthenticateApiKeyResponseSchema{},
			Handler:     AuthenticateApiKey,
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

		if input.Name == "" {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"name_is_required",
				"name is required",
			)
		}

		apiKeyBytes, err := generateApiKeyBytes() 
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_generate_api_key",
				err.Error(),
			)
		}
		
		apiKey := b64EncodeApiKey(apiKeyBytes)
		if apiKey == "" {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_encode_api_key",
				"failed to encode API key",
			)
		}


		hashedKey, err := hashApiKey(apiKey)
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

		records, err := e.App.FindRecordsByFilter("api_keys", "", "", 0, 0)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_find_api_key_records",
				err.Error(),
			)
		}

		matchedRecord, err := getMatchApiKeyRecord(records, apiKey)
		if err != nil {
			return apierror.New(
				http.StatusUnauthorized,
				"request.validation",
				"invalid_api_key",	
				"Invalid API key provided",
			)
		}

		userId := matchedRecord.GetString("user")
		if userId == "" {
			return apierror.New(
				http.StatusUnauthorized,
				"request.validation",
				"user_not_found",
				"User associated with the API key not found",
			)
		}

		authRecord, err := e.App.FindRecordById("users", userId)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_find_user",
				err.Error(),
			)
		}
		if authRecord == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"request.validation",
				"user_not_found",
				"User associated with the API key not found",
			)
		}
		response, err := generateAuthenticateApiKeyResponse(apiKey, authRecord)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.internal_error",
				"failed_to_generate_auth_token",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, response)
	}

}
		


func generateApiKeyBytes() ([]byte, error) {
	apiKeyBytes := make([]byte, 32)
	if _, err := rand.Read(apiKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate API key bytes: %w", err)
	}
	return apiKeyBytes, nil
}

func b64EncodeApiKey(apiKeyBytes []byte) string {
	return base64.URLEncoding.EncodeToString(apiKeyBytes)
}

func hashApiKey(apiKey string) (string, error) {
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hashedKey), nil
}

func getMatchApiKeyRecord(records []*core.Record, apiKey string) (*core.Record, error) {
	for _, record := range records {
		record.NewAuthToken()
		storedHash := record.GetString("key")
		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(apiKey)); err == nil {
			return record, nil
		} 
	}
	
	return nil, fmt.Errorf("no matching API key record found")
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



