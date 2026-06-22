// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type GetUseCaseVerificationDeeplinkResponse struct {
	Code    string            `json:"code"`
	Secrets map[string]string `json:"secrets,omitempty"`
}

var VerifierTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/verifier",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		middlewares.RequireInternalAdminAPIKey(),
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/get-use-case-verification-deeplink",
			Handler:        HandleGetUseCaseVerificationDeeplink,
			ResponseSchema: GetUseCaseVerificationDeeplinkResponse{},
		},
		{
			Method:  http.MethodDelete,
			Path:    "/temp-use-case/{record}",
			Handler: HandleDeleteTempUseCaseVerification,
		},
	},
}

func HandleDeleteTempUseCaseVerification() func(*core.RequestEvent) error {
	return handleTempRecordDelete("use_cases_verifications", "use case verification")
}

func HandleGetUseCaseVerificationDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		useCaseIdentifier := e.Request.URL.Query().Get("use_case_identifier")
		if useCaseIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"use_case_identifier",
				"use_case_identifier is required",
				"missing use_case_identifier",
			)
		}

		record, err := canonify.Resolve(e.App, useCaseIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"use_case_identifier",
				"use case verification not found",
				err.Error(),
			)
		}

		secrets, apiErr := parseSecretsYAML(record.GetString("secrets"))
		if apiErr != nil {
			return apiErr
		}

		var response GetUseCaseVerificationDeeplinkResponse
		response.Code = record.GetString("yaml")
		response.Secrets = secrets

		return e.JSON(http.StatusOK, response)
	}
}
