// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var CustomIntegrationsRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/custom-integrations",
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/run",
			Handler:       HandleRunCustomIntegration,
			RequestSchema: RunCustomIntegrationRequestInput{},
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

type RunCustomIntegrationRequestInput struct {
	Yaml string      `json:"yaml" validate:"required"`
	Data interface{} `json:"data"`
}

func HandleRunCustomIntegration() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[RunCustomIntegrationRequestInput](e)
		if err != nil {
			return err
		}

		authRecord := e.Auth
		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			).JSON(e)
		}

		appURL := e.App.Settings().Meta.AppURL

		var formJSON string
		if req.Data != nil {
			b, err := json.Marshal(req.Data)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"data",
					"failed to serialize data to JSON",
					err.Error(),
				).JSON(e)
			}
			formJSON = string(b)
		}

		memo := map[string]interface{}{
			"test":     "custom-integration",
			"author":   authRecord.Id,
		}

		result, err := processCustomChecks(
			req.Yaml,
			appURL,
			namespace,
			memo,
			formJSON,
		)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"custom-integration",
				"failed to process custom integration",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, workflowengine.WorkflowResult{
			WorkflowID: result.WorkflowID,
			RunID:      result.RunID,
		})
	}
}
