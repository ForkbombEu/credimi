// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

const (
	newsScanDefaultHourUTC   = 6
	newsScanDefaultMaxDrafts = 3
	newsScanCatchupWindow    = 12 * time.Hour
)

var newsScanTriggerOptions = client.ScheduleTriggerOptions{
	Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
}

// NewsAgentInternalRoutes hold operator routes for the news agent daily scan
// schedule. They accept the internal admin API key only.
var NewsAgentInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/news-agent",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/schedule",
			Handler:        HandleScheduleNewsScan,
			RequestSchema:  ScheduleNewsScanRequest{},
			ResponseSchema: ScheduleNewsScanResponse{},
			Description:    "Create or update the daily news agent scan schedule",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:         http.MethodDelete,
			Path:           "/schedule",
			Handler:        HandleDeleteNewsScanSchedule,
			ResponseSchema: DeleteNewsScanScheduleResponse{},
			Description:    "Delete the daily news agent scan schedule",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
	},
}

type ScheduleNewsScanRequest struct {
	HourUTC    *int     `json:"hour_utc"    validate:"omitempty,min=0,max=23"`
	SourceIDs  []string `json:"source_ids"`
	MaxDrafts  int      `json:"max_drafts"  validate:"omitempty,min=1,max=10"`
	TriggerNow bool     `json:"trigger_now"`
}

type ScheduleNewsScanResponse struct {
	Message           string `json:"message"`
	ScheduleID        string `json:"schedule_id"`
	WorkflowNamespace string `json:"workflowNamespace"`
	Cron              string `json:"cron"`
}

type DeleteNewsScanScheduleResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	ScheduleID        string `json:"schedule_id"`
	WorkflowNamespace string `json:"workflowNamespace"`
}

func HandleScheduleNewsScan() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ScheduleNewsScanRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			)
		}

		hour := newsScanDefaultHourUTC
		if input.HourUTC != nil {
			hour = *input.HourUTC
		}
		maxDrafts := input.MaxDrafts
		if maxDrafts == 0 {
			maxDrafts = newsScanDefaultMaxDrafts
		}
		sourceIDs := input.SourceIDs
		if sourceIDs == nil {
			sourceIDs = []string{}
		}
		cron := fmt.Sprintf("0 %d * * *", hour)
		scanInput := workflows.NewsDailyScanInput{SourceIDs: sourceIDs, MaxDrafts: maxDrafts}

		namespace := workflows.DefaultNamespace
		c, err := scheduleTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to create temporal client",
				err.Error(),
			)
		}

		ctx := e.Request.Context()
		scheduleID := workflows.NewsDailyScanScheduleID
		_, err = c.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID:            scheduleID,
			Spec:          client.ScheduleSpec{CronExpressions: []string{cron}},
			Overlap:       enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
			CatchupWindow: newsScanCatchupWindow,
			Action:        buildNewsScanScheduleAction(scanInput),
		})
		if err != nil && isScheduleAlreadyExistsError(err) {
			handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
			err = handle.Update(ctx, client.ScheduleUpdateOptions{
				DoUpdate: func(update client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
					return &client.ScheduleUpdate{Schedule: &client.Schedule{
						Spec: &client.ScheduleSpec{CronExpressions: []string{cron}},
						Policy: &client.SchedulePolicies{
							Overlap:       enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
							CatchupWindow: newsScanCatchupWindow,
						},
						State:  update.Description.Schedule.State,
						Action: buildNewsScanScheduleAction(scanInput),
					}}, nil
				},
			})
		}
		if err == nil && input.TriggerNow {
			err = c.ScheduleClient().GetHandle(ctx, scheduleID).Trigger(ctx, newsScanTriggerOptions)
		}
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to upsert news scan schedule",
				err.Error(),
			)
		}

		message := fmt.Sprintf("News scan scheduled daily at %02d:00 UTC", hour)
		if input.TriggerNow {
			message += " and triggered now"
		}
		return e.JSON(http.StatusOK, ScheduleNewsScanResponse{
			Message:           message,
			ScheduleID:        scheduleID,
			WorkflowNamespace: namespace,
			Cron:              cron,
		})
	}
}

func HandleDeleteNewsScanSchedule() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		namespace := workflows.DefaultNamespace

		c, err := scheduleTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to create temporal client",
				err.Error(),
			)
		}

		ctx := e.Request.Context()
		handle := c.ScheduleClient().GetHandle(ctx, workflows.NewsDailyScanScheduleID)

		if err := handle.Delete(ctx); err != nil {
			var notFound *serviceerror.NotFound
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"schedule",
					"news scan schedule not found",
					err.Error(),
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to delete news scan schedule",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, DeleteNewsScanScheduleResponse{
			Success:           true,
			Message:           "News scan schedule deleted successfully",
			ScheduleID:        workflows.NewsDailyScanScheduleID,
			WorkflowNamespace: namespace,
		})
	}
}

func buildNewsScanScheduleAction(
	input workflows.NewsDailyScanInput,
) *client.ScheduleWorkflowAction {
	return &client.ScheduleWorkflowAction{
		ID:        workflows.NewsDailyScanScheduleID,
		Workflow:  workflows.NewsDailyScanWorkflowName,
		TaskQueue: workflows.NewsAgentTaskQueue,
		Args:      []interface{}{input},
	}
}
