// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/runners"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

type PipelineQueueInput struct {
	PipelineIdentifier string `json:"pipeline_identifier"`
	YAML               string `json:"yaml"`
}

type PipelineQueueRunnerStatus struct {
	RunnerID          string                                   `json:"runner_id"`
	Status            workflows.MobileRunnerSemaphoreRunStatus `json:"status"`
	Position          int                                      `json:"position"`
	LineLen           int                                      `json:"line_len"`
	WorkflowID        string                                   `json:"workflow_id,omitempty"`
	RunID             string                                   `json:"run_id,omitempty"`
	WorkflowNamespace string                                   `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                   `json:"error_message,omitempty"`
}

type PipelineQueueStatusResponse struct {
	TicketID          string                                   `json:"ticket_id"`
	EnqueuedAt        *time.Time                               `json:"enqueued_at,omitempty"`
	RunnerIDs         []string                                 `json:"runner_ids,omitempty"`
	LeaderRunnerID    string                                   `json:"leader_runner_id,omitempty"`
	RequiredRunnerIDs []string                                 `json:"required_runner_ids,omitempty"`
	Status            workflows.MobileRunnerSemaphoreRunStatus `json:"status"`
	Position          int                                      `json:"position"`
	LineLen           int                                      `json:"line_len"`
	WorkflowID        string                                   `json:"workflow_id,omitempty"`
	RunID             string                                   `json:"run_id,omitempty"`
	WorkflowNamespace string                                   `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                   `json:"error_message,omitempty"`
	Runners           []PipelineQueueRunnerStatus              `json:"runners"`
}

var errRunTicketNotFound = errors.New("run ticket not found")

var ensureRunQueueSemaphoreWorkflow = ensureRunQueueSemaphoreWorkflowTemporal
var enqueueRunTicket = enqueueRunTicketTemporal
var queryRunTicketStatus = queryRunTicketStatusTemporal
var cancelRunTicket = cancelRunTicketTemporal

func HandlePipelineQueueEnqueue() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineQueueInput](e)
		if err != nil {
			return err
		}
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}
		pipelineIdentifier := strings.TrimSpace(input.PipelineIdentifier)
		if pipelineIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"pipeline_identifier",
				"pipeline_identifier is required",
				"missing pipeline_identifier",
			).JSON(e)
		}
		yaml := strings.TrimSpace(input.YAML)
		if yaml == "" {
			return apierror.New(
				http.StatusBadRequest,
				"yaml",
				"yaml is required",
				"missing yaml",
			).JSON(e)
		}

		if _, err := canonify.Resolve(e.App, pipelineIdentifier); err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_identifier",
				"pipeline not found",
				err.Error(),
			).JSON(e)
		}

		userID := e.Auth.Id
		userMail := e.Auth.GetString("email")
		userName := e.Auth.GetString("name")
		namespace, err := GetUserOrganizationCanonifiedName(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		runnerIDs, err := resolvePipelineRunnerIDs(yaml)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"yaml",
				"failed to parse pipeline yaml",
				err.Error(),
			).JSON(e)
		}
		if len(runnerIDs) == 0 {
			return apierror.New(
				http.StatusBadRequest,
				"runner_ids",
				"runner_ids are required",
				"no runner ids resolved from yaml",
			).JSON(e)
		}

		leaderRunnerID := runnerIDs[0]
		now := time.Now().UTC()
		ticketID := uuid.NewString()
		memo := map[string]any{
			"test":   "pipeline-run",
			"userID": userID,
		}
		config := buildPipelineQueueConfig(e, namespace, userName, userMail)

		runnerStatuses := make([]PipelineQueueRunnerStatus, 0, len(runnerIDs))
		for _, runnerID := range runnerIDs {
			if err := ensureRunQueueSemaphoreWorkflow(e.Request.Context(), runnerID); err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"semaphore",
					"failed to ensure runner semaphore",
					err.Error(),
				).JSON(e)
			}
			req := workflows.MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:           ticketID,
				OwnerNamespace:     namespace,
				EnqueuedAt:         now,
				RunnerID:           runnerID,
				RequiredRunnerIDs:  runnerIDs,
				LeaderRunnerID:     leaderRunnerID,
				PipelineIdentifier: pipelineIdentifier,
				YAML:               yaml,
				PipelineConfig:     config,
				Memo:               memo,
			}
			resp, err := enqueueRunTicket(e.Request.Context(), runnerID, req)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"semaphore",
					"failed to enqueue pipeline run",
					err.Error(),
				).JSON(e)
			}
			runnerStatuses = append(runnerStatuses, PipelineQueueRunnerStatus{
				RunnerID: runnerID,
				Status:   resp.Status,
				Position: resp.Position,
				LineLen:  resp.LineLen,
			})
		}

		status, position, lineLen, workflowID, runID, workflowNamespace, errorMessage :=
			aggregateRunQueueStatus(runnerStatuses)

		response := PipelineQueueStatusResponse{
			TicketID:          ticketID,
			EnqueuedAt:        &now,
			RunnerIDs:         runnerIDs,
			LeaderRunnerID:    leaderRunnerID,
			RequiredRunnerIDs: append([]string{}, runnerIDs...),
			Status:            status,
			Position:          position,
			LineLen:           lineLen,
			WorkflowID:        workflowID,
			RunID:             runID,
			WorkflowNamespace: workflowNamespace,
			ErrorMessage:      errorMessage,
			Runners:           runnerStatuses,
		}

		return e.JSON(http.StatusOK, response)
	}
}

func HandlePipelineQueueStatus() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}

		ticketID := strings.TrimSpace(e.Request.PathValue("ticket"))
		if ticketID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"ticket",
				"ticket is required",
				"missing ticket path parameter",
			).JSON(e)
		}

		runnerIDs := normalizeRunnerIDs(parseRunnerIDs(e.Request))
		if len(runnerIDs) == 0 {
			return apierror.New(
				http.StatusBadRequest,
				"runner_ids",
				"runner_ids are required",
				"missing runner_ids query parameter",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		statusViews := make([]workflows.MobileRunnerSemaphoreRunStatusView, 0, len(runnerIDs))
		runnerStatuses := make([]PipelineQueueRunnerStatus, 0, len(runnerIDs))
		for _, runnerID := range runnerIDs {
			status, err := queryRunTicketStatus(
				e.Request.Context(),
				runnerID,
				namespace,
				ticketID,
			)
			if err != nil {
				if errors.Is(err, errRunTicketNotFound) {
					return apierror.New(
						http.StatusNotFound,
						"ticket",
						"ticket not found",
						err.Error(),
					).JSON(e)
				}
				return apierror.New(
					http.StatusInternalServerError,
					"semaphore",
					"failed to query ticket status",
					err.Error(),
				).JSON(e)
			}
			if status.Status == workflowengine.MobileRunnerSemaphoreRunNotFound {
				return apierror.New(
					http.StatusNotFound,
					"ticket",
					"ticket not found",
					"ticket not found",
				).JSON(e)
			}
			statusViews = append(statusViews, status)
			runnerStatuses = append(runnerStatuses, runnerStatusFromView(runnerID, status))
		}

		status, position, lineLen, workflowID, runID, workflowNamespace, errorMessage :=
			aggregateRunQueueStatus(runnerStatuses)
		leaderRunnerID, requiredRunnerIDs := extractLeaderAndRequired(statusViews, runnerIDs)

		response := PipelineQueueStatusResponse{
			TicketID:          ticketID,
			RunnerIDs:         runnerIDs,
			LeaderRunnerID:    leaderRunnerID,
			RequiredRunnerIDs: requiredRunnerIDs,
			Status:            status,
			Position:          position,
			LineLen:           lineLen,
			WorkflowID:        workflowID,
			RunID:             runID,
			WorkflowNamespace: workflowNamespace,
			ErrorMessage:      errorMessage,
			Runners:           runnerStatuses,
		}

		return e.JSON(http.StatusOK, response)
	}
}

func HandlePipelineQueueCancel() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}

		ticketID := strings.TrimSpace(e.Request.PathValue("ticket"))
		if ticketID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"ticket",
				"ticket is required",
				"missing ticket path parameter",
			).JSON(e)
		}

		runnerIDs := normalizeRunnerIDs(parseRunnerIDs(e.Request))
		if len(runnerIDs) == 0 {
			return apierror.New(
				http.StatusBadRequest,
				"runner_ids",
				"runner_ids are required",
				"missing runner_ids query parameter",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		runnerStatuses := make([]PipelineQueueRunnerStatus, 0, len(runnerIDs))
		for _, runnerID := range runnerIDs {
			status, err := cancelRunTicket(
				e.Request.Context(),
				runnerID,
				workflows.MobileRunnerSemaphoreRunCancelRequest{
					TicketID:       ticketID,
					OwnerNamespace: namespace,
				},
			)
			if err != nil {
				if errors.Is(err, errRunTicketNotFound) {
					continue
				}
				return apierror.New(
					http.StatusInternalServerError,
					"semaphore",
					"failed to cancel ticket",
					err.Error(),
				).JSON(e)
			}
			runnerStatuses = append(runnerStatuses, runnerStatusFromView(runnerID, status))
		}

		status, position, lineLen, workflowID, runID, workflowNamespace, errorMessage :=
			aggregateRunQueueStatus(runnerStatuses)
		leaderRunnerID := ""
		if len(runnerIDs) > 0 {
			leaderRunnerID = runnerIDs[0]
		}

		response := PipelineQueueStatusResponse{
			TicketID:          ticketID,
			RunnerIDs:         runnerIDs,
			LeaderRunnerID:    leaderRunnerID,
			RequiredRunnerIDs: append([]string{}, runnerIDs...),
			Status:            status,
			Position:          position,
			LineLen:           lineLen,
			WorkflowID:        workflowID,
			RunID:             runID,
			WorkflowNamespace: workflowNamespace,
			ErrorMessage:      errorMessage,
			Runners:           runnerStatuses,
		}

		return e.JSON(http.StatusOK, response)
	}
}

func buildPipelineQueueConfig(
	e *core.RequestEvent,
	namespace string,
	userName string,
	userMail string,
) map[string]any {
	appURL := e.App.Settings().Meta.AppURL
	appName := e.App.Settings().Meta.AppName
	logoURL := utils.JoinURL(
		appURL,
		"logos",
		strings.ToLower(appName)+"_logo-transp_emblem.png",
	)
	return map[string]any{
		"namespace": namespace,
		"app_url":   appURL,
		"app_name":  appName,
		"app_logo":  logoURL,
		"user_name": userName,
		"user_mail": userMail,
	}
}

func resolvePipelineRunnerIDs(yaml string) ([]string, error) {
	info, err := runners.ParsePipelineRunnerInfo(yaml)
	if err != nil {
		return nil, err
	}
	globalRunnerID := ""
	if info.NeedsGlobalRunner {
		wfDef, err := pipeline.ParseWorkflow(yaml)
		if err != nil {
			return nil, err
		}
		globalRunnerID = strings.TrimSpace(wfDef.Runtime.GlobalRunnerID)
	}
	runnerIDs := runners.RunnerIDsWithGlobal(info, globalRunnerID)
	sort.Strings(runnerIDs)
	return runnerIDs, nil
}

func parseRunnerIDs(req *http.Request) []string {
	values := req.URL.Query()["runner_ids[]"]
	if len(values) == 0 {
		values = req.URL.Query()["runner_ids"]
	}
	return values
}

func normalizeRunnerIDs(values []string) []string {
	unique := map[string]struct{}{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			candidate := strings.TrimSpace(part)
			if candidate == "" {
				continue
			}
			unique[candidate] = struct{}{}
		}
	}
	out := make([]string, 0, len(unique))
	for candidate := range unique {
		out = append(out, candidate)
	}
	sort.Strings(out)
	return out
}

func runnerStatusFromView(
	runnerID string,
	status workflows.MobileRunnerSemaphoreRunStatusView,
) PipelineQueueRunnerStatus {
	return PipelineQueueRunnerStatus{
		RunnerID:          runnerID,
		Status:            status.Status,
		Position:          status.Position,
		LineLen:           status.LineLen,
		WorkflowID:        status.WorkflowID,
		RunID:             status.RunID,
		WorkflowNamespace: status.WorkflowNamespace,
		ErrorMessage:      status.ErrorMessage,
	}
}

func aggregateRunQueueStatus(
	statuses []PipelineQueueRunnerStatus,
) (
	workflows.MobileRunnerSemaphoreRunStatus,
	int,
	int,
	string,
	string,
	string,
	string,
) {
	aggregateStatus := workflowengine.MobileRunnerSemaphoreRunNotFound
	aggregatePriority := runStatusPriority(aggregateStatus)
	maxPosition := 0
	maxLineLen := 0
	workflowID := ""
	runID := ""
	workflowNamespace := ""
	errorMessage := ""

	for _, status := range statuses {
		if status.Position > maxPosition {
			maxPosition = status.Position
		}
		if status.LineLen > maxLineLen {
			maxLineLen = status.LineLen
		}
		priority := runStatusPriority(status.Status)
		if priority > aggregatePriority {
			aggregateStatus = status.Status
			aggregatePriority = priority
		}
		if status.Status == workflowengine.MobileRunnerSemaphoreRunRunning && workflowID == "" {
			workflowID = status.WorkflowID
			runID = status.RunID
			workflowNamespace = status.WorkflowNamespace
		}
		if status.Status == workflowengine.MobileRunnerSemaphoreRunFailed && errorMessage == "" {
			errorMessage = status.ErrorMessage
		}
	}

	return aggregateStatus, maxPosition, maxLineLen, workflowID, runID, workflowNamespace, errorMessage
}

func runStatusPriority(status workflows.MobileRunnerSemaphoreRunStatus) int {
	switch status {
	case workflowengine.MobileRunnerSemaphoreRunFailed:
		return 4
	case workflowengine.MobileRunnerSemaphoreRunCanceled:
		return 4
	case workflowengine.MobileRunnerSemaphoreRunRunning:
		return 3
	case workflowengine.MobileRunnerSemaphoreRunStarting:
		return 2
	case workflowengine.MobileRunnerSemaphoreRunQueued:
		return 1
	case workflowengine.MobileRunnerSemaphoreRunNotFound:
		return 0
	default:
		return 0
	}
}

func extractLeaderAndRequired(
	statuses []workflows.MobileRunnerSemaphoreRunStatusView,
	runnerIDs []string,
) (string, []string) {
	for _, status := range statuses {
		if status.LeaderRunnerID != "" {
			return status.LeaderRunnerID, copyStringSlice(status.RequiredRunnerIDs)
		}
	}
	leaderRunnerID := ""
	if len(runnerIDs) > 0 {
		leaderRunnerID = runnerIDs[0]
	}
	return leaderRunnerID, copyStringSlice(runnerIDs)
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func ensureRunQueueSemaphoreWorkflowTemporal(ctx context.Context, runnerID string) error {
	client, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return err
	}

	workflowID := workflows.MobileRunnerSemaphoreWorkflowID(runnerID)
	input := workflowengine.WorkflowInput{
		Payload: workflows.MobileRunnerSemaphoreWorkflowInput{
			RunnerID: runnerID,
			Capacity: 1,
		},
	}

	_, err = client.ExecuteWorkflow(
		ctx,
		tclient.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: workflows.MobileRunnerSemaphoreTaskQueue,
		},
		workflows.MobileRunnerSemaphoreWorkflowName,
		input,
	)
	if err != nil && !temporal.IsWorkflowExecutionAlreadyStartedError(err) {
		return err
	}

	return nil
}

func enqueueRunTicketTemporal(
	ctx context.Context,
	runnerID string,
	req workflows.MobileRunnerSemaphoreEnqueueRunRequest,
) (workflows.MobileRunnerSemaphoreEnqueueRunResponse, error) {
	client, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return workflows.MobileRunnerSemaphoreEnqueueRunResponse{}, err
	}

	workflowID := workflows.MobileRunnerSemaphoreWorkflowID(runnerID)
	handle, err := client.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   workflows.MobileRunnerSemaphoreEnqueueRunUpdate,
		UpdateID:     runQueueUpdateID("enqueue", runnerID, req.TicketID),
		Args:         []interface{}{req},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return workflows.MobileRunnerSemaphoreEnqueueRunResponse{}, err
	}

	var response workflows.MobileRunnerSemaphoreEnqueueRunResponse
	if err := handle.Get(ctx, &response); err != nil {
		return workflows.MobileRunnerSemaphoreEnqueueRunResponse{}, err
	}
	return response, nil
}

func queryRunTicketStatusTemporal(
	ctx context.Context,
	runnerID string,
	ownerNamespace string,
	ticketID string,
) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
	client, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return workflows.MobileRunnerSemaphoreRunStatusView{}, err
	}

	workflowID := workflows.MobileRunnerSemaphoreWorkflowID(runnerID)
	encoded, err := client.QueryWorkflow(
		ctx,
		workflowID,
		"",
		workflows.MobileRunnerSemaphoreRunStatusQuery,
		ownerNamespace,
		ticketID,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return workflows.MobileRunnerSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		return workflows.MobileRunnerSemaphoreRunStatusView{}, err
	}

	var status workflows.MobileRunnerSemaphoreRunStatusView
	if err := encoded.Get(&status); err != nil {
		return workflows.MobileRunnerSemaphoreRunStatusView{}, err
	}
	return status, nil
}

func cancelRunTicketTemporal(
	ctx context.Context,
	runnerID string,
	req workflows.MobileRunnerSemaphoreRunCancelRequest,
) (workflows.MobileRunnerSemaphoreRunStatusView, error) {
	client, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return workflows.MobileRunnerSemaphoreRunStatusView{}, err
	}

	workflowID := workflows.MobileRunnerSemaphoreWorkflowID(runnerID)
	handle, err := client.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   workflows.MobileRunnerSemaphoreCancelRunUpdate,
		UpdateID:     runQueueUpdateID("cancel", runnerID, req.TicketID),
		Args:         []interface{}{req},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return workflows.MobileRunnerSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		return workflows.MobileRunnerSemaphoreRunStatusView{}, err
	}

	var status workflows.MobileRunnerSemaphoreRunStatusView
	if err := handle.Get(ctx, &status); err != nil {
		return workflows.MobileRunnerSemaphoreRunStatusView{}, err
	}

	return status, nil
}

func runQueueUpdateID(prefix, runnerID, ticketID string) string {
	return prefix + "/" + runnerID + "/" + ticketID
}
