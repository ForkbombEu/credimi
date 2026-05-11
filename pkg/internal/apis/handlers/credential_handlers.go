// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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
		middlewares.RequireInternalAdminAPIKey(),
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/get-credential-offer",
			Handler:        HandleGetCredentialOffer,
			ResponseSchema: GetCredentialOfferResponse{},
		},
		{
			Method:  http.MethodDelete,
			Path:    "/temp/{record}",
			Handler: HandleDeleteTempCredential,
		},
	},
}

type credentialDeleteTempInput struct {
	ExpectedOwnerID    string `json:"expected_owner_id"`
	ExpectedIdentifier string `json:"expected_identifier"`
}

func HandleDeleteTempCredential() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		recordID := strings.TrimSpace(e.Request.PathValue("record"))
		if recordID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"credential",
				"credential record id is required",
				"missing record path parameter",
			).JSON(e)
		}

		var input credentialDeleteTempInput
		if e.Request.Body != nil {
			if err := json.NewDecoder(e.Request.Body).Decode(&input); err != nil &&
				!errors.Is(err, io.EOF) {
				return apierror.New(
					http.StatusBadRequest,
					"credential",
					"invalid delete validation payload",
					err.Error(),
				).JSON(e)
			}
		}

		record, err := e.App.FindRecordById("credentials", recordID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return e.JSON(http.StatusOK, map[string]any{"deleted": false})
			}
			return apierror.New(
				http.StatusInternalServerError,
				"credential",
				"failed to find credential",
				err.Error(),
			).JSON(e)
		}

		if apiErr := validateTempCredentialDeleteRequest(e.App, record, input); apiErr != nil {
			return apiErr.JSON(e)
		}

		if err := e.App.Delete(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credential",
				"failed to delete credential",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{"deleted": true})
	}
}

func validateTempCredentialDeleteRequest(
	app core.App,
	record *core.Record,
	input credentialDeleteTempInput,
) *apierror.APIError {
	expectedOwnerID := strings.TrimSpace(input.ExpectedOwnerID)
	expectedIdentifier := strings.TrimSpace(input.ExpectedIdentifier)
	if expectedOwnerID == "" || expectedIdentifier == "" {
		return apierror.New(
			http.StatusBadRequest,
			"credential",
			"delete validation payload is required",
			"expected_owner_id and expected_identifier are required",
		)
	}
	if record.GetString("owner") != expectedOwnerID {
		return apierror.New(
			http.StatusForbidden,
			"credential",
			"temporary credential owner mismatch",
			"credential owner does not match expected_owner_id",
		)
	}
	resolved, err := canonify.Resolve(app, expectedIdentifier)
	if err != nil {
		return apierror.New(
			http.StatusForbidden,
			"credential",
			"temporary credential identifier mismatch",
			err.Error(),
		)
	}
	if resolved.Id != record.Id {
		return apierror.New(
			http.StatusForbidden,
			"credential",
			"temporary credential identifier mismatch",
			"expected_identifier does not resolve to the requested record",
		)
	}
	return nil
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

		issuerRecord, err := e.App.FindRecordById(
			"credential_issuers",
			record.GetString("credential_issuer"),
		)
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
		response.CredentialOffer = fmt.Sprintf(
			"openid-credential-offer://?credential_offer=%s",
			encoded,
		)

		return e.JSON(http.StatusOK, response)
	}
}
