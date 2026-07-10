// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const pipelineMobileFlowWorkflowIDPrefix = "Pipeline-Mobile-Flow-"

type PipelineMobileFlowInput struct {
	WorkflowID       string            `json:"workflow_id"       validate:"required"`
	RunID            string            `json:"run_id"            validate:"required"`
	OrganizationID   string            `json:"organization_id"   validate:"required"`
	ActionID         string            `json:"action_id"         validate:"required"`
	ActionParameters map[string]string `json:"action_parameters"`
}

type PipelineMobileFlowResponse struct {
	Success bool `json:"success"`
	Output  any  `json:"output,omitempty"`
	Error   any  `json:"error,omitempty"`
}

var pipelineMobileFlowTemporalClient = temporalclient.GetTemporalClientWithNamespace

func HandlePipelineMobileFlow() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineMobileFlowInput](e)
		if err != nil {
			return err
		}

		namespace := strings.TrimSpace(input.OrganizationID)
		temporalClient, err := pipelineMobileFlowTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to get temporal client",
				err.Error(),
			)
		}

		runnerID, device, apiErr := pipelineMobileFlowDevice(
			e.Request.Context(),
			temporalClient,
			strings.TrimSpace(input.WorkflowID),
			strings.TrimSpace(input.RunID),
		)
		if apiErr != nil {
			return apiErr
		}

		action, err := canonify.Resolve(e.App, strings.TrimSpace(input.ActionID))
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"action_id",
				"wallet action not found",
				err.Error(),
			)
		}
		if action == nil || action.Collection() == nil ||
			action.Collection().Name != "wallet_actions" {
			return apierror.New(
				http.StatusNotFound,
				"action_id",
				"wallet action not found",
				"resolved record is not a wallet action",
			)
		}
		actionCode := strings.TrimSpace(action.GetString("code"))
		if actionCode == "" {
			return apierror.New(
				http.StatusUnprocessableEntity,
				"action_id",
				"wallet action has no code",
				"wallet action code is empty",
			)
		}

		activityOptions := pipelineMobileFlowActivityOptions()
		mobileWorkflow := workflows.NewMobileAutomationWorkflow()
		run, err := temporalClient.ExecuteWorkflow(
			e.Request.Context(),
			client.StartWorkflowOptions{
				ID:        pipelineMobileFlowWorkflowIDPrefix + uuid.NewString(),
				TaskQueue: pipeline.PipelineTaskQueue,
				Memo: map[string]any{
					"test": fmt.Sprintf("mobile-flow: %s", strings.TrimSpace(input.ActionID)),
				},
			},
			mobileWorkflow.Name(),
			workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":   e.App.Settings().Meta.AppURL,
					"taskqueue": fmt.Sprintf("%s-TaskQueue", canonify.NormalizePath(runnerID)),
				},
				Payload: workflows.MobileAutomationWorkflowPayload{
					ActionID:   strings.TrimSpace(input.ActionID),
					ActionCode: actionCode,
					Serial:     workflowengine.AsString(device["serial"]),
					Type:       workflowengine.AsString(device["type"]),
					RunnerID:   runnerID,
					Parameters: input.ActionParameters,
				},
				ActivityOptions: &activityOptions,
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start mobile automation workflow",
				err.Error(),
			)
		}

		var result workflowengine.WorkflowResult
		waitErr := run.Get(e.Request.Context(), &result)

		return e.JSON(http.StatusOK, pipelineMobileFlowResponse(result, waitErr))
	}
}

func pipelineMobileFlowActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		ScheduleToCloseTimeout: 20 * time.Minute,
		StartToCloseTimeout:    20 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
}

func pipelineMobileFlowResponse(
	result workflowengine.WorkflowResult,
	err error,
) PipelineMobileFlowResponse {
	if err != nil {
		return PipelineMobileFlowResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return PipelineMobileFlowResponse{
		Success: true,
		Output:  result.Output,
	}
}

func pipelineMobileFlowDevice(
	ctx context.Context,
	temporalClient client.Client,
	workflowID string,
	runID string,
) (string, map[string]any, *apierror.APIError) {
	if workflowID == "" || runID == "" {
		return "", nil, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"workflow_id and run_id are required",
			"missing workflow execution identifier",
		)
	}

	description, err := temporalClient.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		return "", nil, apierror.New(
			http.StatusNotFound,
			"workflow",
			"pipeline workflow not found",
			err.Error(),
		)
	}
	info := description.GetWorkflowExecutionInfo()
	if info == nil || info.GetStatus() != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return "", nil, apierror.New(
			http.StatusConflict,
			"workflow",
			"pipeline workflow is not running",
			"mobile flow can only run while the pipeline workflow is active",
		)
	}
	if info.GetType() == nil || info.GetType().GetName() != "Dynamic Pipeline Workflow" {
		return "", nil, apierror.New(
			http.StatusUnprocessableEntity,
			"workflow",
			"workflow is not a dynamic pipeline",
			"mobile flow can only run from a dynamic pipeline workflow",
		)
	}

	attributes, err := decodeWorkflowSearchAttributes(info.GetSearchAttributes())
	if err != nil {
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to read pipeline runner metadata",
			err.Error(),
		)
	}
	runnerIDs := stringSlice(attributes[workflowengine.RunnerIdentifiersSearchAttribute])
	if len(runnerIDs) != 1 {
		return "", nil, apierror.New(
			http.StatusUnprocessableEntity,
			"runner_id",
			"pipeline must have exactly one reserved runner",
			fmt.Sprintf("found %d reserved runners", len(runnerIDs)),
		)
	}

	encoded, err := temporalClient.QueryWorkflow(
		ctx,
		workflowID,
		runID,
		pipeline.PipelineMobileDevicesQuery,
	)
	if err != nil {
		return "", nil, apierror.New(
			http.StatusConflict,
			"runner_id",
			"pipeline mobile device is not initialized",
			err.Error(),
		)
	}
	var devices map[string]any
	if err := encoded.Get(&devices); err != nil {
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed to read initialized pipeline devices",
			err.Error(),
		)
	}

	runnerID := canonify.NormalizePath(runnerIDs[0])
	device, ok := devices[runnerID].(map[string]any)
	if !ok {
		return "", nil, apierror.New(
			http.StatusConflict,
			"runner_id",
			"pipeline mobile device is not initialized",
			"run a mobile-automation step before calling this API",
		)
	}
	if workflowengine.AsString(device["type"]) == "" {
		return "", nil, apierror.New(
			http.StatusConflict,
			"runner_id",
			"pipeline mobile device is not initialized",
			"initialized device type is missing",
		)
	}

	return runnerID, device, nil
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				result = append(result, value)
			}
		}
		return result
	default:
		return nil
	}
}
