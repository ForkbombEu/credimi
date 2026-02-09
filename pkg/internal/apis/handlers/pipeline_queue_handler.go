// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/runners"
	"github.com/forkbombeu/credimi/pkg/internal/runqueue"
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

type pipelineQueueRunnerStatus struct {
	RunnerID     string
	Status       workflows.MobileRunnerSemaphoreRunStatus
	Position     int
	LineLen      int
	WorkflowID   string
	RunID        string
	ErrorMessage string
}

type PipelineQueueResponse struct {
	TicketID     string                                   `json:"ticket_id,omitempty"`
	EnqueuedAt   *time.Time                               `json:"enqueued_at,omitempty"`
	RunnerIDs    []string                                 `json:"runner_ids,omitempty"`
	Status       workflows.MobileRunnerSemaphoreRunStatus `json:"status,omitempty"`
	Position     *int                                     `json:"position,omitempty"`
	LineLen      *int                                     `json:"line_len,omitempty"`
	WorkflowID   string                                   `json:"workflow_id,omitempty"`
	RunID        string                                   `json:"run_id,omitempty"`
	ErrorMessage string                                   `json:"error_message,omitempty"`
}

type queueRequestContext struct {
	ticketID  string
	runnerIDs []string
	namespace string
}

var errRunTicketNotFound = errors.New("run ticket not found")

func isQueueLimitExceeded(err error) bool {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Type() == workflows.MobileRunnerSemaphoreErrQueueLimitExceeded
	}
	return false
}

var ensureRunQueueSemaphoreWorkflow = ensureRunQueueSemaphoreWorkflowTemporal
var enqueueRunTicket = enqueueRunTicketTemporal
var queryRunTicketStatus = queryRunTicketStatusTemporal
var cancelRunTicket = cancelRunTicketTemporal

// startPipelineWorkflow starts a pipeline workflow and is stubbed in unit tests.
var startPipelineWorkflow = startPipelineWorkflowTemporal

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

		pipelineRecord, err := canonify.Resolve(e.App, pipelineIdentifier)
		if err != nil {
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
		orgRecord, err := GetUserOrganization(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization record",
				err.Error(),
			).JSON(e)
		}
		namespace := orgRecord.GetString("canonified_name")
		if namespace == "" {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				"missing organization canonified name",
			).JSON(e)
		}
		maxPipelinesInQueue := orgRecord.GetInt("max_pipelines_in_queue")
		memo := map[string]any{
			"test":   "pipeline-run",
			"userID": userID,
		}
		config := buildPipelineQueueConfig(e, namespace, userName, userMail)

		runnerInfo, err := runners.ParsePipelineRunnerInfo(yaml)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"yaml",
				"failed to parse pipeline yaml",
				err.Error(),
			).JSON(e)
		}
		runnerIDs, err := resolvePipelineRunnerIDs(yaml, runnerInfo)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"yaml",
				"failed to parse pipeline yaml",
				err.Error(),
			).JSON(e)
		}
		if len(runnerIDs) == 0 && !runnerInfo.NeedsGlobalRunner {
			startResult, apiErr := startPipelineFromQueue(
				e,
				pipelineRecord,
				orgRecord.Id,
				yaml,
				config,
				memo,
			)
			if apiErr != nil {
				return apiErr.JSON(e)
			}
			response := PipelineQueueResponse{
				Status:     workflowengine.MobileRunnerSemaphoreRunRunning,
				WorkflowID: startResult.WorkflowID,
				RunID:      startResult.WorkflowRunID,
			}
			return e.JSON(http.StatusOK, response)
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

		for _, runnerID := range runnerIDs {
			if err := ensureRunQueueSemaphoreWorkflow(e.Request.Context(), runnerID); err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"semaphore",
					"failed to ensure runner semaphore",
					err.Error(),
				).JSON(e)
			}
		}

		rollbackEnqueuedTickets := func(runnerIDs []string) {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for _, runnerID := range runnerIDs {
				status, err := cancelRunTicket(
					rollbackCtx,
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
					e.App.Logger().Warn(fmt.Sprintf(
						"failed to rollback run ticket %s for runner %s: %v",
						ticketID,
						runnerID,
						err,
					))
					continue
				}
				if status.Status == workflowengine.MobileRunnerSemaphoreRunNotFound {
					continue
				}
			}
		}

		rollbackRunnerIDs := make([]string, 0, len(runnerIDs))
		runnerStatuses := make([]pipelineQueueRunnerStatus, 0, len(runnerIDs))
		for _, runnerID := range runnerIDs {
			// Roll back every attempted runner because enqueue failures can be ambiguous (e.g. timeouts).
			rollbackRunnerIDs = append(rollbackRunnerIDs, runnerID)
			req := workflows.MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:            ticketID,
				OwnerNamespace:      namespace,
				EnqueuedAt:          now,
				RunnerID:            runnerID,
				RequiredRunnerIDs:   runnerIDs,
				LeaderRunnerID:      leaderRunnerID,
				MaxPipelinesInQueue: maxPipelinesInQueue,
				PipelineIdentifier:  pipelineIdentifier,
				YAML:                yaml,
				PipelineConfig:      config,
				Memo:                memo,
			}
			resp, err := enqueueRunTicket(e.Request.Context(), runnerID, req)
			if err != nil {
				rollbackEnqueuedTickets(rollbackRunnerIDs)
				if isQueueLimitExceeded(err) {
					return apierror.New(
						http.StatusConflict,
						"queue_limit",
						"queue limit exceeded",
						err.Error(),
					).JSON(e)
				}
				return apierror.New(
					http.StatusInternalServerError,
					"semaphore",
					"failed to enqueue pipeline run",
					err.Error(),
				).JSON(e)
			}
			runnerStatuses = append(runnerStatuses, pipelineQueueRunnerStatus{
				RunnerID: runnerID,
				Status:   resp.Status,
				Position: resp.Position,
				LineLen:  resp.LineLen,
			})
		}

		status, position, lineLen, workflowID, runID, errorMessage :=
			aggregateRunQueueStatus(runnerStatuses)
		response := buildQueueEnqueueResponse(
			ticketID,
			now,
			runnerIDs,
			status,
			position,
			lineLen,
			workflowID,
			runID,
			errorMessage,
		)

		return e.JSON(http.StatusOK, response)
	}
}

func HandlePipelineQueueStatus() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		requestContext, apiErr := parseQueueRequestContext(e)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		runnerStatuses, apiErr := queryQueueRunnerStatuses(
			e.Request.Context(),
			requestContext.runnerIDs,
			requestContext.namespace,
			requestContext.ticketID,
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		response := buildQueueStatusResponse(
			requestContext.ticketID,
			runnerStatuses,
		)

		return e.JSON(http.StatusOK, response)
	}
}

func HandlePipelineQueueCancel() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		requestContext, apiErr := parseQueueRequestContext(e)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		runnerStatuses, apiErr := cancelQueueRunnerStatuses(
			e.Request.Context(),
			requestContext.runnerIDs,
			requestContext.namespace,
			requestContext.ticketID,
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		response := buildQueueStatusResponse(
			requestContext.ticketID,
			runnerStatuses,
		)

		return e.JSON(http.StatusOK, response)
	}
}

// buildQueueEnqueueResponse maps the aggregate semaphore status into the enqueue response contract.
func buildQueueEnqueueResponse(
	ticketID string,
	enqueuedAt time.Time,
	runnerIDs []string,
	status workflows.MobileRunnerSemaphoreRunStatus,
	position int,
	lineLen int,
	workflowID string,
	runID string,
	errorMessage string,
) PipelineQueueResponse {
	switch status {
	case workflowengine.MobileRunnerSemaphoreRunFailed,
		workflowengine.MobileRunnerSemaphoreRunCanceled:
		msg := strings.TrimSpace(errorMessage)
		if msg == "" {
			msg = "queue failed"
		}
		return PipelineQueueResponse{
			Status:       workflowengine.MobileRunnerSemaphoreRunFailed,
			ErrorMessage: msg,
		}
	case workflowengine.MobileRunnerSemaphoreRunRunning:
		return PipelineQueueResponse{
			Status:     workflowengine.MobileRunnerSemaphoreRunRunning,
			WorkflowID: workflowID,
			RunID:      runID,
		}
	default:
		pos := position
		line := lineLen
		return PipelineQueueResponse{
			Status:     status,
			TicketID:   ticketID,
			EnqueuedAt: &enqueuedAt,
			RunnerIDs:  copyStringSlice(runnerIDs),
			Position:   &pos,
			LineLen:    &line,
		}
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

// startPipelineFromQueue starts a non-runner pipeline and persists the pipeline result record.
func startPipelineFromQueue(
	e *core.RequestEvent,
	pipelineRecord *core.Record,
	ownerID string,
	yaml string,
	config map[string]any,
	memo map[string]any,
) (workflowengine.WorkflowResult, *apierror.APIError) {
	var result workflowengine.WorkflowResult

	coll, err := e.App.FindCollectionByNameOrId("pipeline_results")
	if err != nil {
		return result, apierror.New(
			http.StatusInternalServerError,
			"collection",
			"failed to get collection",
			err.Error(),
		)
	}

	record := core.NewRecord(coll)
	record.Set("owner", ownerID)
	record.Set("pipeline", pipelineRecord.Id)

	result, err = startPipelineWorkflow(yaml, config, memo)
	if err != nil {
		return result, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}

	record.Set("workflow_id", result.WorkflowID)
	record.Set("run_id", result.WorkflowRunID)

	if err := e.App.Save(record); err != nil {
		return result, apierror.New(
			http.StatusInternalServerError,
			"pipeline",
			"failed to save pipeline record",
			err.Error(),
		)
	}

	return result, nil
}

func resolvePipelineRunnerIDs(yaml string, info runners.PipelineRunnerInfo) ([]string, error) {
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

func parseQueueRequestContext(e *core.RequestEvent) (*queueRequestContext, *apierror.APIError) {
	if e.Auth == nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"auth",
			"authentication required",
			"user not authenticated",
		)
	}

	ticketID := strings.TrimSpace(e.Request.PathValue("ticket"))
	if ticketID == "" {
		return nil, apierror.New(
			http.StatusBadRequest,
			"ticket",
			"ticket is required",
			"missing ticket path parameter",
		)
	}

	runnerIDs := normalizeRunnerIDs(parseRunnerIDs(e.Request))
	if len(runnerIDs) == 0 {
		return nil, apierror.New(
			http.StatusBadRequest,
			"runner_ids",
			"runner_ids are required",
			"missing runner_ids query parameter",
		)
	}

	namespace, err := GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization canonified name",
			err.Error(),
		)
	}

	return &queueRequestContext{
		ticketID:  ticketID,
		runnerIDs: runnerIDs,
		namespace: namespace,
	}, nil
}

func queryQueueRunnerStatuses(
	ctx context.Context,
	runnerIDs []string,
	namespace string,
	ticketID string,
) ([]pipelineQueueRunnerStatus, *apierror.APIError) {
	runnerStatuses := make([]pipelineQueueRunnerStatus, 0, len(runnerIDs))

	for _, runnerID := range runnerIDs {
		status, err := queryRunTicketStatus(ctx, runnerID, namespace, ticketID)
		if err != nil {
			if errors.Is(err, errRunTicketNotFound) {
				runnerStatuses = append(
					runnerStatuses,
					runnerStatusFromView(runnerID, runTicketNotFoundView(ticketID)),
				)
				continue
			}
			return nil, apierror.New(
				http.StatusInternalServerError,
				"semaphore",
				"failed to query ticket status",
				err.Error(),
			)
		}
		runnerStatuses = append(runnerStatuses, runnerStatusFromView(runnerID, status))
	}

	return runnerStatuses, nil
}

func cancelQueueRunnerStatuses(
	ctx context.Context,
	runnerIDs []string,
	namespace string,
	ticketID string,
) ([]pipelineQueueRunnerStatus, *apierror.APIError) {
	runnerStatuses := make([]pipelineQueueRunnerStatus, 0, len(runnerIDs))
	for _, runnerID := range runnerIDs {
		status, err := cancelRunTicket(
			ctx,
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
			return nil, apierror.New(
				http.StatusInternalServerError,
				"semaphore",
				"failed to cancel ticket",
				err.Error(),
			)
		}
		runnerStatuses = append(runnerStatuses, runnerStatusFromView(runnerID, status))
	}
	return runnerStatuses, nil
}

func buildQueueStatusResponse(
	ticketID string,
	runnerStatuses []pipelineQueueRunnerStatus,
) PipelineQueueResponse {
	status, position, lineLen, workflowID, runID, _ :=
		aggregateRunQueueStatus(runnerStatuses)
	pos := position
	line := lineLen
	response := PipelineQueueResponse{
		TicketID: ticketID,
		Status:   status,
		Position: &pos,
		LineLen:  &line,
	}
	if workflowID != "" {
		response.WorkflowID = workflowID
		response.RunID = runID
	}
	return response
}

func runTicketNotFoundView(ticketID string) workflows.MobileRunnerSemaphoreRunStatusView {
	return workflows.MobileRunnerSemaphoreRunStatusView{
		TicketID: ticketID,
		Status:   workflowengine.MobileRunnerSemaphoreRunNotFound,
	}
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
) pipelineQueueRunnerStatus {
	return pipelineQueueRunnerStatus{
		RunnerID:     runnerID,
		Status:       status.Status,
		Position:     status.Position,
		LineLen:      status.LineLen,
		WorkflowID:   status.WorkflowID,
		RunID:        status.RunID,
		ErrorMessage: status.ErrorMessage,
	}
}

func aggregateRunQueueStatus(
	statuses []pipelineQueueRunnerStatus,
) (
	workflows.MobileRunnerSemaphoreRunStatus,
	int,
	int,
	string,
	string,
	string,
) {
	queueStatuses := make([]runqueue.RunnerStatus, 0, len(statuses))
	for _, status := range statuses {
		queueStatuses = append(queueStatuses, runqueue.RunnerStatus{
			RunnerID:     status.RunnerID,
			Status:       status.Status,
			Position:     status.Position,
			LineLen:      status.LineLen,
			WorkflowID:   status.WorkflowID,
			RunID:        status.RunID,
			ErrorMessage: status.ErrorMessage,
		})
	}

	aggregate := runqueue.AggregateRunnerStatuses(queueStatuses)

	return aggregate.Status,
		aggregate.Position,
		aggregate.LineLen,
		aggregate.WorkflowID,
		aggregate.RunID,
		aggregate.ErrorMessage
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

// startPipelineWorkflowTemporal runs the pipeline workflow directly via the Temporal client.
func startPipelineWorkflowTemporal(
	yaml string,
	config map[string]any,
	memo map[string]any,
) (workflowengine.WorkflowResult, error) {
	w := pipeline.NewPipelineWorkflow()
	return w.Start(yaml, config, memo)
}

func runQueueUpdateID(prefix, runnerID, ticketID string) string {
	return prefix + "/" + runnerID + "/" + ticketID
}
