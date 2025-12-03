// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var ConformanceCheckRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/conformance-check",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/deeplink",
			Handler: HandleGetConformanceCheckDeeplink,
		},
	},
}

func HandleGetConformanceCheckDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.URL.Query().Get("id")
		if id == "" {
			return apierror.New(
				http.StatusBadRequest,
				"request",
				"missing check id",
				"id parameter is required",
			).JSON(e)
		}

		redirect := e.Request.URL.Query().Get("redirect") == "true"

		parts := strings.Split(filepath.ToSlash(id), "/")
		var suite, standard, checkName string
		if len(parts) >= 2 {
			suite = parts[len(parts)-2]
			standard = parts[0]
			checkName = parts[len(parts)-1]
		}
		appURL := e.App.Settings().Meta.AppURL
		memo := map[string]any{
			"author":   suite,
			"standard": standard,
			"test":     checkName,
		}

		ao := &workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute,
			StartToCloseTimeout:    time.Second * 30,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 1.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    1},
		}

		// only EWC suite is supported
		templatePath := filepath.Join(
			utils.GetEnvironmentVariable("ROOT_DIR", true),
			workflows.EWCTemplateFolderPath,
			checkName+".yaml",
		)
		templateData, err := os.ReadFile(templatePath)

		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"file",
				"failed to read template file",
				err.Error(),
			).JSON(e)
		}

		input := workflowengine.WorkflowInput{
			Payload: workflows.StartCheckWorkflowPayload{
				Suite:     suite,
				CheckID:   id,
				SessionID: uuid.NewString(),
			},
			Config: map[string]any{
				"memo":      memo,
				"app_url":   appURL,
				"template":  string(templateData),
				"namespace": "default",
			},
			ActivityOptions: ao,
		}

		var w workflows.StartCheckWorkflow

		resStart, errStart := w.Start("default", input)
		if errStart != nil {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"failed to start get deeplink check",
				errStart.Error(),
			).JSON(e)
		}
		client, err := temporalclient.GetTemporalClientWithNamespace(
			"default",
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to get temporal client",
				err.Error(),
			).JSON(e)
		}
		result, err := workflowengine.WaitForWorkflowResult(
			client,
			resStart.WorkflowID,
			resStart.WorkflowRunID,
		)
		if err != nil {
			details := workflowengine.ParseWorkflowError(err)
			return e.JSON(http.StatusInternalServerError, map[string]any{
				"status":  http.StatusInternalServerError,
				"error":   "workflow",
				"reason":  "failed to get workflow result",
				"details": details,
			})
		}

		output, ok := result.Output.(map[string]any)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow output",
				"output is not a map",
			).JSON(e)
		}

		deeplink, ok := output["deeplink"].(string)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow output",
				"deeplink is not present in output",
			).JSON(e)
		}

		if redirect {
			e.Response.Header().Set("Location", deeplink)
			e.Response.WriteHeader(http.StatusMovedPermanently)
			return e.Next()
		}
		return e.String(http.StatusOK, deeplink)
	}
}
