// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type GetCredentialOfferResponse struct {
	CredentialOffer string `json:"credential_offer"`
	Dynamic         bool   `json:"dynamic"`
	Code            string `json:"code"`
}

var CredentialTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/credential",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/get-credential-offer",
			Handler:        HandleGetCredentialOffer,
			ResponseSchema: GetCredentialOfferResponse{},
		},
	},
}

func HandleGetCredentialOffer() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		credentialIdentifier := e.Request.URL.Query().Get("credential_identifier")
		if credentialIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"credential_identifier",
				"credential_identifier is required",
				"missing credential_identifier",
			).JSON(e)
		}

		record, err := canonify.Resolve(e.App, credentialIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"credential_identifier",
				"credential not found",
				err.Error(),
			).JSON(e)
		}

		var response GetCredentialOfferResponse
		code := record.GetString("yaml")
		if code != "" {
			response.Dynamic = true
			response.Code = code
			return e.JSON(http.StatusOK, response)
		}

		deeplink := record.GetString("deeplink")
		if deeplink != "" {
			response.CredentialOffer = deeplink
			return e.JSON(http.StatusOK, response)
		}

		issuerRecord, err := e.App.FindRecordById("credential_issuers", record.GetString("credential_issuer"))
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"credential_issuer",
				"issuer not found",
				err.Error(),
			).JSON(e)
		}
		data := map[string]any{
			"credential_configuration_ids": []string{record.GetString("name")},
			"credential_issuer":            issuerRecord.GetString("url"),
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"json",
				"unable to marshal json",
				err.Error(),
			)
		}

		encoded := url.QueryEscape(string(jsonData))
		response.CredentialOffer = fmt.Sprintf("openid-credential-offer://?credential_offer=%s", encoded)

		return e.JSON(http.StatusOK, response)
	}
}
