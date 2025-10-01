// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var WorkflowsRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/workflow",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/start-scheduled-workflow",
			Handler:       HandleScheduledWorkflowStart,
			RequestSchema: StartScheduledWorkflowRequest{},
		},
		{
			Method:  http.MethodGet,
			Path:    "/list-scheduled-workflows",
			Handler: HandleListScheduledWorkflows,
		},
	},
}

type StartScheduledWorkflowRequest struct {
	WorkflowID string `json:"workflowID"`
	RunID      string `json:"runID"`
	Interval   string `json:"interval"`
}

func HandleScheduledWorkflowStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req StartScheduledWorkflowRequest

		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("invalid JSON input", err)
		}

		namespace, err := GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}
		info, err := workflowengine.GetWorkflowRunInfo(req.WorkflowID, req.RunID, namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow run info",
				err.Error(),
			).JSON(e)
		}

		var interval time.Duration
		switch req.Interval {
		case "every_minute":
			interval = time.Minute
		case "hourly":
			interval = time.Hour
		case "daily":
			interval = time.Hour * 24
		case "weekly":
			interval = time.Hour * 24 * 7
		case "monthly":
			interval = time.Hour * 24 * 30
		default:
			interval = time.Hour
		}
		err = workflowengine.StartScheduledWorkflowWithOptions(
			info,
			req.WorkflowID,
			namespace,
			interval,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to start scheduled workflow",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, "scheduled workflow started successfully")
	}
}

func HandleListScheduledWorkflows() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		namespace, err := GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}

		schedules, err := workflowengine.ListScheduledWorkflows(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to list scheduled workflows",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, schedules)
	}
}
