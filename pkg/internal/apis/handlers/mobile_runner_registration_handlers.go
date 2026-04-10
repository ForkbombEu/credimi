// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var MobileRunnerRegistrationRoutes = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/preview-id",
			Handler:        HandlePreviewMobileRunnerID,
			RequestSchema:  PreviewMobileRunnerIDRequest{},
			ResponseSchema: PreviewMobileRunnerIDResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "",
			Handler:        HandleUpsertMobileRunner,
			RequestSchema:  UpsertMobileRunnerRequest{},
			ResponseSchema: UpsertMobileRunnerResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
	},
}

type PreviewMobileRunnerIDRequest struct {
	Organization string `json:"organization,omitempty"`
	Name         string `json:"name"              validate:"required"`
}

type PreviewMobileRunnerIDResponse struct {
	Organization   string `json:"organization"`
	CanonifiedName string `json:"canonified_name"`
	RunnerID       string `json:"runner_id"`
}

type UpsertMobileRunnerRequest struct {
	RunnerID     string `json:"runner_id,omitempty"`
	Organization string `json:"organization,omitempty"`
	Name         string `json:"name"                 validate:"required"`
	IP           string `json:"ip"                   validate:"required"`
	Description  string `json:"description,omitempty"`
	Type         string `json:"type,omitempty"`
	Port         string `json:"port,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Published    *bool  `json:"published,omitempty"`
}

type UpsertMobileRunnerResponse struct {
	ID             string `json:"id"`
	Organization   string `json:"organization"`
	Name           string `json:"name"`
	CanonifiedName string `json:"canonified_name"`
	RunnerID       string `json:"runner_id"`
	IP             string `json:"ip"`
	Description    string `json:"description,omitempty"`
	Type           string `json:"type,omitempty"`
	Port           string `json:"port,omitempty"`
	Serial         string `json:"serial,omitempty"`
	Published      bool   `json:"published"`
}

func HandlePreviewMobileRunnerID() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PreviewMobileRunnerIDRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			).JSON(e)
		}

		owner, apiErr := resolveMobileRunnerOwner(e.App, e.Auth, input.Organization)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		preview, apiErr := previewMobileRunnerIdentifier(e.App, owner, input.Name)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		return e.JSON(http.StatusOK, preview)
	}
}

func HandleUpsertMobileRunner() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[UpsertMobileRunnerRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			).JSON(e)
		}

		owner, apiErr := resolveMobileRunnerOwner(e.App, e.Auth, input.Organization)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		normalizedRunnerID := canonify.NormalizePath(input.RunnerID)
		record, apiErr := resolveExistingMobileRunner(e.App, owner, normalizedRunnerID)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		if normalizedRunnerID != "" && record == nil {
			preview, previewErr := previewMobileRunnerIdentifier(e.App, owner, input.Name)
			if previewErr != nil {
				return previewErr.JSON(e)
			}
			if preview.RunnerID != canonicalRunnerIDPath(normalizedRunnerID) {
				return apierror.New(
					http.StatusConflict,
					"runner_id",
					"runner_id_conflict",
					fmt.Sprintf(
						"runner_id %q does not match the next available id %q",
						canonicalRunnerIDPath(normalizedRunnerID),
						preview.RunnerID,
					),
				).JSON(e)
			}
		}

		if record != nil && strings.TrimSpace(record.GetString("name")) != strings.TrimSpace(input.Name) {
			return apierror.New(
				http.StatusConflict,
				"name",
				"runner_name_conflict",
				"name does not match the existing runner_id",
			).JSON(e)
		}

		if record == nil {
			collection, err := e.App.FindCollectionByNameOrId("mobile_runners")
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"collection",
					"mobile_runners collection not found",
					err.Error(),
				).JSON(e)
			}
			record = core.NewRecord(collection)
			record.Set("owner", owner.Id)
		}

		record.Set("name", strings.TrimSpace(input.Name))
		record.Set("ip", strings.TrimSpace(input.IP))
		record.Set("description", strings.TrimSpace(input.Description))
		record.Set("type", strings.TrimSpace(input.Type))
		record.Set("port", strings.TrimSpace(input.Port))
		record.Set("serial", strings.TrimSpace(input.Serial))
		if input.Published != nil {
			record.Set("published", *input.Published)
		}

		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"failed_to_save_mobile_runner",
				err.Error(),
			).JSON(e)
		}

		path, err := canonify.BuildPath(
			e.App,
			record,
			canonify.CanonifyPaths["mobile_runners"],
			"",
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"failed_to_build_runner_id",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, UpsertMobileRunnerResponse{
			ID:             record.Id,
			Organization:   owner.GetString("canonified_name"),
			Name:           record.GetString("name"),
			CanonifiedName: record.GetString("canonified_name"),
			RunnerID:       canonicalRunnerIDPath(path),
			IP:             record.GetString("ip"),
			Description:    record.GetString("description"),
			Type:           record.GetString("type"),
			Port:           record.GetString("port"),
			Serial:         record.GetString("serial"),
			Published:      record.GetBool("published"),
		})
	}
}

func resolveMobileRunnerOwner(
	app core.App,
	auth *core.Record,
	requestedOrganization string,
) (*core.Record, *apierror.APIError) {
	if auth == nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"auth",
			"authentication_required",
			"authentication is required",
		)
	}

	if isSuperuserAuth(auth) {
		orgCanon := strings.TrimSpace(requestedOrganization)
		if orgCanon == "" {
			return nil, apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization_required",
				"organization is required for admin authentication",
			)
		}

		record, err := app.FindFirstRecordByFilter(
			"organizations",
			"canonified_name={:canonified_name}",
			dbx.Params{"canonified_name": orgCanon},
		)
		if err != nil {
			status := http.StatusInternalServerError
			reason := "failed_to_find_organization"
			message := err.Error()
			if errors.Is(err, sql.ErrNoRows) {
				status = http.StatusNotFound
				reason = "organization_not_found"
				message = "organization not found"
			}
			return nil, apierror.New(status, "organization", reason, message)
		}

		return record, nil
	}

	record, err := GetUserOrganization(app, auth.Id)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"failed_to_find_user_organization",
			err.Error(),
		)
	}

	return record, nil
}

func isSuperuserAuth(auth *core.Record) bool {
	if auth == nil || auth.Collection() == nil {
		return false
	}

	return auth.Collection().Name == "_superusers"
}

func previewMobileRunnerIdentifier(
	app core.App,
	owner *core.Record,
	name string,
) (PreviewMobileRunnerIDResponse, *apierror.APIError) {
	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	if err != nil {
		return PreviewMobileRunnerIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"collection",
			"mobile_runners collection not found",
			err.Error(),
		)
	}

	record := core.NewRecord(collection)
	record.Set("owner", owner.Id)
	record.Set("name", strings.TrimSpace(name))

	canonifiedName, err := canonify.Canonify(
		record.GetString("name"),
		canonify.MakeExistsFunc(app, "mobile_runners", record, ""),
	)
	if err != nil {
		return PreviewMobileRunnerIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"name",
			"failed_to_canonify_runner_name",
			err.Error(),
		)
	}

	record.Set("canonified_name", canonifiedName)
	path, err := canonify.BuildPath(app, record, canonify.CanonifyPaths["mobile_runners"], "")
	if err != nil {
		return PreviewMobileRunnerIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed_to_build_runner_id",
			err.Error(),
		)
	}

	return PreviewMobileRunnerIDResponse{
		Organization:   owner.GetString("canonified_name"),
		CanonifiedName: canonifiedName,
		RunnerID:       canonicalRunnerIDPath(path),
	}, nil
}

func resolveExistingMobileRunner(
	app core.App,
	owner *core.Record,
	runnerID string,
) (*core.Record, *apierror.APIError) {
	if runnerID == "" {
		return nil, nil
	}

	record, err := canonify.Resolve(app, runnerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed_to_resolve_runner_id",
			err.Error(),
		)
	}

	if record.Collection() == nil || record.Collection().Name != "mobile_runners" {
		return nil, apierror.New(
			http.StatusBadRequest,
			"runner_id",
			"invalid_runner_id",
			"runner_id does not reference a mobile runner",
		)
	}

	if record.GetString("owner") != owner.Id {
		return nil, apierror.New(
			http.StatusForbidden,
			"runner_id",
			"runner_id_owner_mismatch",
			"runner_id does not belong to the resolved organization",
		)
	}

	return record, nil
}

func canonicalRunnerIDPath(path string) string {
	normalized := canonify.NormalizePath(path)
	if normalized == "" {
		return ""
	}

	return "/" + normalized
}
