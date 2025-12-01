// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/sdk/client"
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

		namespace, err := GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
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
		namespace, err := GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization name",
				err.Error(),
			).JSON(e)
		}

		schedules, err := listScheduledWorkflows(namespace)
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

func listScheduledWorkflows(namespace string) ([]ScheduleInfoSummary, error) {
	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to create Temporal client for namespace %q: %w",
			namespace,
			err,
		)
	}

	ctx := context.Background()

	iter, err := c.ScheduleClient().List(ctx, client.ScheduleListOptions{
		PageSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	var schedules []ScheduleInfoSummary
	for iter.HasNext() {
		sched, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to list schedules: %w", err)
		}
		schedJSON, err := json.Marshal(sched)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schedule: %w", err)
		}
		var schedInfo ScheduleInfo
		if err := json.Unmarshal(schedJSON, &schedInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schedule: %w", err)
		}
		var displayName string
		if schedInfo.Memo != nil {
			if field, ok := schedInfo.Memo.Fields["test"]; ok {
				displayName = decodeFromTemporalPayload(*field.Data)
			}
		}
		var originalWorkflowID string
		if schedInfo.Memo != nil {
			if field, ok := schedInfo.Memo.Fields["original_workflow_id"]; ok {
				originalWorkflowID = decodeFromTemporalPayload(*field.Data)
			}
		}
		schedInfoSummary := ScheduleInfoSummary{
			ID:                 schedInfo.ID,
			Spec:               schedInfo.Spec,
			WorkflowType:       schedInfo.WorkflowType,
			DisplayName:        displayName,
			OriginalWorkflowID: originalWorkflowID,
			Paused:             schedInfo.Paused,
		}

		schedules = append(schedules, schedInfoSummary)
	}

	return schedules, nil
}
