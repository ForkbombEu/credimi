// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type PipelineInput struct {
	Yaml string `json:"yaml"`
}

var PipelineRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/pipeline",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/start",
			Handler:       HandlePipelineStart,
			RequestSchema: PipelineInput{},
			Description:   "Start a pipeline workflow from a YAML file",
		},
	},
}

var PipelineTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/pipeline",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/get-yaml",
			Handler:     HandleGetPipelineYAML,
			Description: "Get a pipeline YAML from a pipeline ID",
		},
	},
}

func HandlePipelineStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineInput](e)
		if err != nil {
			return err
		}
		appURL := e.App.Settings().Meta.AppURL
		appName := e.App.Settings().Meta.AppName
		logoURL := utils.JoinURL(
			appURL,
			"logos",
			fmt.Sprintf("%s_logo-transp_emblem.png", strings.ToLower(appName)),
		)

		var userID, userMail, userName, namespace string

		if e.Auth != nil {
			userID = e.Auth.Id
			userMail = e.Auth.GetString("email")
			userName = e.Auth.GetString("name")
			namespace, err = GetUserOrganizationCanonifiedName(e.App, userID)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"organization",
					"unable to get user organization canonified name",
					err.Error(),
				).JSON(e)
			}
		}

		memo := map[string]any{
			"test":   "pipeline-run",
			"userID": userID,
		}
		config := map[string]any{
			"namespace": namespace,
			"app_url":   appURL,
			"app_name":  appName,
			"app_logo":  logoURL,
			"user_name": userName,
			"user_mail": userMail,
		}
		w := pipeline.NewPipelineWorkflow()
		result, err := w.Start(input.Yaml, config, memo)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, map[string]any{
			"message": "Workflow started successfully",
			"result":  result,
		})
	}
}

func HandleGetPipelineYAML() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		pipelineIdentifier := e.Request.URL.Query().Get("pipeline_identifier")
		if pipelineIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"pipeline_identifier",
				"pipeline_identifier is required",
				"missing pipeline_identifier",
			).JSON(e)
		}

		record, err := canonify.Resolve(e.App, pipelineIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_identifier",
				"pipeline not found",
				err.Error(),
			).JSON(e)
		}
		yaml := record.GetString("yaml")
		return e.String(http.StatusOK, yaml)
	}
}
