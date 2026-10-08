// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

const (
	pipelineRetentionDefaultDays     = 30
	pipelineRetentionDefaultInterval = 1
	pipelineRetentionScheduleID      = "pipeline-retention-schedule"
)

var pipelineRetentionImmediateTriggerOptions = client.ScheduleTriggerOptions{
	Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
}

type SchedulePipelineRetentionRequest struct {
	OlderThanDays int `json:"older_than_days" validate:"omitempty,min=1"`
	IntervalDays  int `json:"interval_days"   validate:"omitempty,min=1"`
}

type SchedulePipelineRetentionResponse struct {
	Message           string `json:"message"`
	ScheduleID        string `json:"schedule_id"`
	WorkflowNamespace string `json:"workflowNamespace"`
}

type DeletePipelineRetentionScheduleResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	ScheduleID        string `json:"schedule_id"`
	WorkflowNamespace string `json:"workflowNamespace"`
}

func HandleSchedulePipelineRetentionWorkflow() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[SchedulePipelineRetentionRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			)
		}

		olderThanDays := input.OlderThanDays
		if olderThanDays == 0 {
			olderThanDays = pipelineRetentionDefaultDays
		}

		intervalDays := input.IntervalDays
		if intervalDays == 0 {
			intervalDays = pipelineRetentionDefaultInterval
		}

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
		scheduleID := pipelineRetentionScheduleID
		options := buildPipelineRetentionScheduleOptions(
			scheduleID,
			e.App,
			olderThanDays,
			intervalDays,
		)

		_, err = c.ScheduleClient().Create(ctx, options)
		if err != nil {
			if isScheduleAlreadyExistsError(err) {
				handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
				err = handle.Update(ctx, client.ScheduleUpdateOptions{
					DoUpdate: func(client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
						return &client.ScheduleUpdate{Schedule: buildPipelineRetentionSchedule(
							e.App,
							olderThanDays,
							intervalDays,
						)}, nil
					},
				})
				if err == nil {
					err = handle.Trigger(ctx, pipelineRetentionImmediateTriggerOptions)
				}
			}
		} else {
			handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
			err = handle.Trigger(ctx, pipelineRetentionImmediateTriggerOptions)
		}
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to upsert retention schedule",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, SchedulePipelineRetentionResponse{
			Message: fmt.Sprintf(
				"Pipeline retention triggered now and scheduled every %d day(s) with older_than_days=%d",
				intervalDays,
				olderThanDays,
			),
			ScheduleID:        scheduleID,
			WorkflowNamespace: workflows.DefaultNamespace,
		})
	}
}

func HandleDeletePipelineRetentionSchedule() func(*core.RequestEvent) error {
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
		handle := c.ScheduleClient().GetHandle(ctx, pipelineRetentionScheduleID)

		if err := handle.Delete(ctx); err != nil {
			var notFound *serviceerror.NotFound
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"schedule",
					"retention schedule not found",
					err.Error(),
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to delete retention schedule",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, DeletePipelineRetentionScheduleResponse{
			Success:           true,
			Message:           "Pipeline retention schedule deleted successfully",
			ScheduleID:        pipelineRetentionScheduleID,
			WorkflowNamespace: namespace,
		})
	}
}

func buildPipelineRetentionScheduleOptions(
	scheduleID string,
	app core.App,
	olderThanDays int,
	intervalDays int,
) client.ScheduleOptions {
	return client.ScheduleOptions{
		ID:      scheduleID,
		Spec:    buildPipelineRetentionScheduleSpec(intervalDays),
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		Action:  buildPipelineRetentionScheduleAction(app, olderThanDays),
	}
}

func buildPipelineRetentionSchedule(
	app core.App,
	olderThanDays int,
	intervalDays int,
) *client.Schedule {
	return &client.Schedule{
		Spec: &client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{
				Every: time.Duration(intervalDays) * 24 * time.Hour,
			}},
		},
		Policy: buildPipelineRetentionSchedulePolicy(),
		State:  buildPipelineRetentionScheduleState(),
		Action: buildPipelineRetentionScheduleAction(app, olderThanDays),
	}
}

func buildPipelineRetentionScheduleSpec(intervalDays int) client.ScheduleSpec {
	return client.ScheduleSpec{
		Intervals: []client.ScheduleIntervalSpec{{
			Every: time.Duration(intervalDays) * 24 * time.Hour,
		}},
	}
}

func buildPipelineRetentionSchedulePolicy() *client.SchedulePolicies {
	return &client.SchedulePolicies{
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
	}
}

func buildPipelineRetentionScheduleState() *client.ScheduleState {
	return &client.ScheduleState{}
}

func buildPipelineRetentionScheduleAction(
	app core.App,
	olderThanDays int,
) *client.ScheduleWorkflowAction {
	workflowName := workflows.NewPipelineRetentionWorkflow().Name()

	return &client.ScheduleWorkflowAction{
		ID:        pipelineRetentionScheduleID,
		Workflow:  workflowName,
		TaskQueue: workflows.PipelineRetentionTaskQueue,
		Args: []interface{}{
			workflowengine.WorkflowInput{
				Payload: workflows.PipelineRetentionWorkflowInput{
					OlderThanDays: olderThanDays,
					DryRun:        false,
				},
				Config: workflowengine.WithAppConfig(app, map[string]any{}),
			},
		},
	}
}

func isScheduleAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	var alreadyExists *serviceerror.AlreadyExists
	if errors.As(err, &alreadyExists) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already registered") ||
		strings.Contains(msg, "already exists")
}
