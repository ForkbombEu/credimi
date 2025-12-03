// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
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
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

var SchedulesRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/my/schedules",
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/start",
			Handler:        HandleStartSchedule,
			RequestSchema:  StartScheduleRequest{},
			ResponseSchema: StartScheduleResponse{},
			Description:    "Start a new schedule from an existing workflow",
			Summary:        "Start a new schedule from an existing workflow",
		},
		{
			Method:         http.MethodGet,
			Handler:        HandleListMySchedules,
			ResponseSchema: ListMySchedulesResponse{},
			Description:    "List all schedules for the authenticated user",
			Summary:        "Get a list of all schedules for the authenticated user",
		},

		{
			Method:         http.MethodPost,
			Path:           "/{scheduleId}/cancel",
			Handler:        HandleCancelSchedule,
			ResponseSchema: CancelScheduleResponse{},
			Description:    "Cancel a specific schedule",
			Summary:        "Cancel a specific schedule",
		},
		{
			Method:         http.MethodPost,
			Path:           "/{scheduleId}/pause",
			Handler:        HandlePauseSchedule,
			ResponseSchema: PauseScheduleResponse{},
			Description:    "Pause a specific schedule",
			Summary:        "Pause a specific schedule",
		},
		{
			Method:         http.MethodPost,
			Path:           "/{scheduleId}/resume",
			Handler:        HandleResumeSchedule,
			ResponseSchema: ResumeScheduleResponse{},
			Description:    "Resume a specific schedule",
			Summary:        "Resume a specific schedule",
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

type StartScheduleRequest struct {
	WorkflowID   string                      `json:"workflow_id"`
	RunID        string                      `json:"run_id"`
	ScheduleMode workflowengine.ScheduleMode `json:"schedule_mode"`
}

type StartScheduleResponse struct {
	Message      string                      `json:"message"`
	ScheduleID   string                      `json:"schedule_id"`
	ScheduleMode workflowengine.ScheduleMode `json:"schedule_mode"`
}

func HandleStartSchedule() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req StartScheduleRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("invalid JSON input", err)
		}

		// Validate schedule mode
		if err := validateScheduleMode(&req.ScheduleMode); err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"schedule",
				"invalid schedule mode",
				err.Error(),
			)
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

		timeZone := e.Auth.GetString("Timezone")
		info, err := workflowengine.GetWorkflowRunInfo(req.WorkflowID, req.RunID, namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow run info",
				err.Error(),
			).JSON(e)
		}

		scheduleInfo, err := workflowengine.StartScheduledWorkflowWithOptions(
			info,
			req.WorkflowID,
			namespace,
			req.ScheduleMode,
			timeZone,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to start scheduled workflow",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, StartScheduleResponse{
			Message:      "Schedule started successfully",
			ScheduleID:   scheduleInfo.ID,
			ScheduleMode: req.ScheduleMode,
		})
	}
}

func validateScheduleMode(mode *workflowengine.ScheduleMode) error {
	now := time.Now()
	switch mode.Mode {
	case "daily":

	case "weekly":
		if mode.Day == nil {
			d := int(now.Weekday())
			mode.Day = &d
		}
		if *mode.Day < 0 || *mode.Day > 6 {
			return fmt.Errorf("day must be between 0 (Sunday) and 6 (Saturday) for weekly mode")
		}

	case "monthly":
		if mode.Day == nil {
			d := now.Day()
			mode.Day = &d
		}
		if *mode.Day < 0 || *mode.Day > 30 {
			return fmt.Errorf("day must be between 0 and 30 for monthly mode")
		}
	default:
		return fmt.Errorf("invalid mode: must be 'daily', 'weekly', or 'monthly'")
	}

	return nil
}

func HandleListMySchedules() func(*core.RequestEvent) error {
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
		response := ListMySchedulesResponse{
			Schedules: schedules,
		}
		return e.JSON(http.StatusOK, response)
	}
}

func listScheduledWorkflows(namespace string) ([]*ScheduleInfoSummary, error) {
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

	var schedules []*ScheduleInfoSummary
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
		scheduleMode := workflowengine.ParseScheduleMode(schedInfo.Spec.Calendars)

		schedInfoSummary := ScheduleInfoSummary{
			ID:                 schedInfo.ID,
			ScheduleMode:       scheduleMode,
			WorkflowType:       schedInfo.WorkflowType,
			DisplayName:        displayName,
			OriginalWorkflowID: originalWorkflowID,
			NextActionTime:     schedInfo.NextActionTimes[0].Format("02/01/2006, 15:04:05"),
			Paused:             schedInfo.Paused,
		}

		schedules = append(schedules, &schedInfoSummary)
	}

	return schedules, nil
}

type scheduleAction func(ctx context.Context, handle client.ScheduleHandle) error

func HandleCancelSchedule() func(*core.RequestEvent) error {
	return handleSchedule(
		func(ctx context.Context, h client.ScheduleHandle) error {
			return h.Delete(ctx)
		},
		func(scheduleID, namespace string) any {
			return CancelScheduleResponse{
				Message:    "Schedule canceled successfully",
				ScheduleID: scheduleID,
				Status:     "canceled",
				Time:       time.Now().Format(time.RFC3339),
				Namespace:  namespace,
			}
		},
	)
}
func HandlePauseSchedule() func(*core.RequestEvent) error {
	return handleSchedule(
		func(ctx context.Context, h client.ScheduleHandle) error {
			return h.Pause(ctx, client.SchedulePauseOptions{
				Note: "Paused by user",
			})
		},
		func(scheduleID, namespace string) any {
			return PauseScheduleResponse{
				Message:    "Schedule paused successfully",
				ScheduleID: scheduleID,
				Status:     "paused",
				Time:       time.Now().Format(time.RFC3339),
				Namespace:  namespace,
			}
		},
	)
}
func HandleResumeSchedule() func(*core.RequestEvent) error {
	return handleSchedule(
		func(ctx context.Context, h client.ScheduleHandle) error {
			return h.Unpause(ctx, client.ScheduleUnpauseOptions{
				Note: "Resumed by user",
			})
		},
		func(scheduleID, namespace string) any {
			return ResumeScheduleResponse{
				Message:    "Schedule resumed successfully",
				ScheduleID: scheduleID,
				Status:     "running",
				Time:       time.Now().Format(time.RFC3339),
				Namespace:  namespace,
			}
		},
	)
}

func handleSchedule(
	action scheduleAction,
	makeResponse func(scheduleID, namespace string) any,
) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}

		scheduleID := e.Request.PathValue("scheduleId")
		if scheduleID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"scheduleId is required",
				"missing required parameters",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		ctx := context.Background()
		handle := c.ScheduleClient().GetHandle(ctx, scheduleID)

		if err := action(ctx, handle); err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"schedule",
					"schedule not found",
					err.Error(),
				).JSON(e)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to modify schedule",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, makeResponse(scheduleID, namespace))
	}
}
