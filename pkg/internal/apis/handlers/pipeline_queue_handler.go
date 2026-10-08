// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/runqueue"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/semaphoreclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
)

type PipelineQueueInput struct {
	PipelineIdentifier string `json:"pipeline_identifier"`
	YAML               string `json:"yaml"`
}

type pipelineQueueRunnerStatus struct {
	DeviceID     string
	Status       workflows.MobileDeviceSemaphoreRunStatus
	Position     int
	LineLen      int
	WorkflowID   string
	RunID        string
	ErrorMessage string
	Cleanup      *workflows.MobileDeviceSemaphoreCleanupMetadata
}

type PipelineQueueResponse struct {
	TicketID     string                                   `json:"ticket_id,omitempty"`
	EnqueuedAt   *time.Time                               `json:"enqueued_at,omitempty"`
	DeviceIDs    []string                                 `json:"device_ids,omitempty"`
	Status       workflows.MobileDeviceSemaphoreRunStatus `json:"status,omitempty"`
	Position     *int                                     `json:"position,omitempty"`
	LineLen      *int                                     `json:"line_len,omitempty"`
	WorkflowID   string                                   `json:"workflow_id,omitempty"`
	RunID        string                                   `json:"run_id,omitempty"`
	PipelineURL  string                                   `json:"pipeline_url,omitempty"`
	RunURL       string                                   `json:"run_url,omitempty"`
	ErrorMessage string                                   `json:"error_message,omitempty"`
}

type queueRequestContext struct {
	ticketID  string
	deviceIDs []string
	namespace string
	ownerID   string
}

type pipelineQueueRunContext struct {
	pipelineRecord     *core.Record
	pipelineIdentifier string
	organizationRecord *core.Record
	userID             string
	userName           string
	userEmail          string
	yaml               string
	metadata           map[string]any
	runType            string
	cleanup            *workflows.MobileDeviceSemaphoreCleanupMetadata
	notification       *workflows.MobileDeviceSemaphoreNotification
}

func isQueueLimitExceeded(err error) bool {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Type() == workflows.MobileDeviceSemaphoreErrQueueLimitExceeded
	}
	return false
}

var ensureRunQueueSemaphoreWorkflow = func(ctx context.Context, deviceID string) error {
	c, err := queueTemporalClient(workflowengine.MobileDeviceSemaphoreDefaultNamespace)
	if err != nil {
		return err
	}
	return semaphoreclient.EnsureWorkflow(ctx, c, deviceID)
}

var enqueueRunTicket = func(
	ctx context.Context,
	deviceID string,
	req workflows.MobileDeviceSemaphoreEnqueueRunRequest,
) (workflows.MobileDeviceSemaphoreEnqueueRunResponse, error) {
	c, err := queueTemporalClient(workflowengine.MobileDeviceSemaphoreDefaultNamespace)
	if err != nil {
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{}, err
	}
	return semaphoreclient.EnqueueRun(ctx, c, deviceID, req)
}

var queryRunTicketStatus = queryRunTicketStatusTemporal

var cancelRunTicket = func(
	ctx context.Context,
	deviceID string,
	req workflows.MobileDeviceSemaphoreRunCancelRequest,
) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
	c, err := queueTemporalClient(workflowengine.MobileDeviceSemaphoreDefaultNamespace)
	if err != nil {
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}
	return semaphoreclient.CancelRun(ctx, c, deviceID, req)
}

var queueTemporalClient = temporalclient.GetTemporalClientWithNamespace

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
			)
		}
		pipelineIdentifier := strings.TrimSpace(input.PipelineIdentifier)
		if pipelineIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"pipeline_identifier",
				"pipeline_identifier is required",
				"missing pipeline_identifier",
			)
		}
		yaml := strings.TrimSpace(input.YAML)
		if yaml == "" {
			return apierror.New(
				http.StatusBadRequest,
				"yaml",
				"yaml is required",
				"missing yaml",
			)
		}

		pipelineRecord, err := canonify.Resolve(e.App, pipelineIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_identifier",
				"pipeline not found",
				err.Error(),
			)
		}

		orgRecord, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization record",
				err.Error(),
			)
		}
		response, apiErr := enqueuePipelineRun(e, pipelineQueueRunContext{
			pipelineRecord:     pipelineRecord,
			pipelineIdentifier: pipelineIdentifier,
			organizationRecord: orgRecord,
			userID:             e.Auth.Id,
			userName:           e.Auth.GetString("name"),
			userEmail:          e.Auth.GetString("email"),
			yaml:               yaml,
		})
		if apiErr != nil {
			return apiErr
		}
		return e.JSON(http.StatusOK, response)
	}
}

func enqueuePipelineRun(
	e *core.RequestEvent,
	runContext pipelineQueueRunContext,
) (PipelineQueueResponse, *apierror.APIError) {
	namespace := runContext.organizationRecord.GetString("canonified_name")
	if namespace == "" {
		return PipelineQueueResponse{}, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization canonified name",
			"missing organization canonified name",
		)
	}
	maxPipelinesInQueue := runContext.organizationRecord.GetInt("max_pipelines_in_queue")
	memo := map[string]any{
		"test":   "pipeline-run",
		"userID": runContext.userID,
	}
	runType := runContext.runType
	if runType == "" {
		runType = pipelineinternal.RunTypeManual
	}
	memo[pipelineinternal.RunTypeMemoKey] = runType
	if runContext.metadata != nil {
		memo["metadata"] = runContext.metadata
	}
	config := buildPipelineQueueConfig(e, namespace, runContext.userName, runContext.userEmail)
	// Every queue-started run notifies organization members on completion,
	// whether it starts directly or through the device semaphore.
	config[pipeline.CompletionNotificationConfigKey] = true
	applyPipelineQueueCleanupConfig(config, runContext.cleanup)
	deviceInfo, err := pipeline.ParsePipelineDeviceInfo(runContext.yaml)
	if err != nil {
		return PipelineQueueResponse{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}
	deviceIDs, err := resolvePipelineDeviceIDs(runContext.yaml, deviceInfo)
	if err != nil {
		return PipelineQueueResponse{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}
	if len(deviceIDs) == 0 && !deviceInfo.NeedsGlobalDevice {
		if githubPRConfig := buildPipelineGitHubPRCommentConfig(
			runContext.notification,
		); githubPRConfig != nil {
			config[pipeline.GitHubPRCommentConfigKey] = githubPRConfig
		}
		startResult, apiErr := startPipelineFromQueue(
			e,
			runContext.pipelineRecord,
			runContext.pipelineIdentifier,
			runContext.organizationRecord.Id,
			runContext.yaml,
			config,
			memo,
			runType,
		)
		if apiErr != nil {
			return PipelineQueueResponse{}, apiErr
		}
		response := PipelineQueueResponse{
			Status:     workflowengine.MobileDeviceSemaphoreRunRunning,
			WorkflowID: startResult.WorkflowID,
			RunID:      startResult.WorkflowRunID,
		}
		decoratePipelineQueueResponseURLs(
			&response,
			e.App.Settings().Meta.AppURL,
			runContext.pipelineIdentifier,
		)
		return response, nil
	}
	if len(deviceIDs) == 0 {
		return PipelineQueueResponse{}, apierror.New(
			http.StatusBadRequest,
			"device_ids",
			"device_ids are required",
			"no device ids resolved from yaml",
		)
	}
	if apiErr := validatePipelineRunnerAccess(
		e.App,
		runContext.organizationRecord.Id,
		deviceIDs,
	); apiErr != nil {
		return PipelineQueueResponse{}, apiErr
	}
	// Authorization first, then availability: parking a ticket on a semaphore
	// whose runner is not answering only surfaces as a stuck queue entry later.
	if apiErr := requireMobileDeviceRunnersOnline(
		e.Request.Context(),
		e.App,
		deviceIDs,
	); apiErr != nil {
		return PipelineQueueResponse{}, apiErr
	}

	leaderDeviceID := deviceIDs[0]
	if runContext.notification != nil && runContext.notification.GitHubPR != nil {
		runContext.notification.GitHubPR.DeviceTypes = buildGitHubPRDeviceTypes(
			e.App,
			deviceIDs,
			runContext.notification.GitHubPR.DeviceTypes,
		)
		if strings.TrimSpace(runContext.notification.GitHubPR.DeviceID) == "" {
			runContext.notification.GitHubPR.DeviceID = leaderDeviceID
		}
		if strings.TrimSpace(runContext.notification.GitHubPR.DeviceType) == "" {
			runContext.notification.GitHubPR.DeviceType = runContext.notification.GitHubPR.DeviceTypes[leaderDeviceID]
		}
	}
	now := time.Now().UTC()
	ticketID := uuid.NewString()

	rollbackEnqueuedTickets := func(deviceIDs []string) {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, deviceID := range deviceIDs {
			status, err := cancelRunTicket(
				rollbackCtx,
				deviceID,
				workflows.MobileDeviceSemaphoreRunCancelRequest{
					TicketID:       ticketID,
					OwnerNamespace: namespace,
				},
			)
			if err != nil {
				if errors.Is(err, semaphoreclient.ErrRunTicketNotFound) {
					continue
				}
				e.App.Logger().Warn(fmt.Sprintf(
					"failed to rollback run ticket %s for runner %s: %v",
					ticketID,
					deviceID,
					err,
				))
				continue
			}
			if status.Status == workflowengine.MobileDeviceSemaphoreRunNotFound {
				continue
			}
		}
	}

	rollbackDeviceIDs := make([]string, 0, len(deviceIDs))
	runnerStatuses := make([]pipelineQueueRunnerStatus, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		// Roll back every attempted runner because enqueue failures can be ambiguous (e.g. timeouts).
		rollbackDeviceIDs = append(rollbackDeviceIDs, deviceID)
		req := workflows.MobileDeviceSemaphoreEnqueueRunRequest{
			TicketID:            ticketID,
			OwnerNamespace:      namespace,
			EnqueuedAt:          now,
			DeviceID:            deviceID,
			RequiredDeviceIDs:   deviceIDs,
			LeaderDeviceID:      leaderDeviceID,
			MaxPipelinesInQueue: maxPipelinesInQueue,
			PipelineIdentifier:  runContext.pipelineIdentifier,
			YAML:                runContext.yaml,
			PipelineConfig:      config,
			Memo:                memo,
			Cleanup:             runContext.cleanup,
			Notification:        runContext.notification,
		}
		resp, err := enqueueRunTicket(e.Request.Context(), deviceID, req)
		if err != nil {
			rollbackEnqueuedTickets(rollbackDeviceIDs)
			if isQueueLimitExceeded(err) {
				return PipelineQueueResponse{}, apierror.New(
					http.StatusConflict,
					"queue_limit",
					"queue limit exceeded",
					err.Error(),
				)
			}
			return PipelineQueueResponse{}, apierror.New(
				http.StatusInternalServerError,
				"semaphore",
				"failed to enqueue pipeline run",
				err.Error(),
			)
		}
		runnerStatuses = append(runnerStatuses, pipelineQueueRunnerStatus{
			DeviceID: deviceID,
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
		deviceIDs,
		status,
		position,
		lineLen,
		workflowID,
		runID,
		errorMessage,
	)
	decoratePipelineQueueResponseURLs(
		&response,
		e.App.Settings().Meta.AppURL,
		runContext.pipelineIdentifier,
	)
	return response, nil
}

func HandlePipelineQueueStatus() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		requestContext, apiErr := parseQueueRequestContext(e)
		if apiErr != nil {
			return apiErr
		}

		runnerStatuses, apiErr := queryQueueRunnerStatuses(
			e.Request.Context(),
			requestContext.deviceIDs,
			requestContext.namespace,
			requestContext.ticketID,
		)
		if apiErr != nil {
			return apiErr
		}
		response := buildQueueStatusResponse(
			requestContext.ticketID,
			runnerStatuses,
		)
		decoratePipelineQueueResponseURLs(&response, e.App.Settings().Meta.AppURL, "")

		return e.JSON(http.StatusOK, response)
	}
}

func HandlePipelineQueueCancel() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		requestContext, apiErr := parseQueueRequestContext(e)
		if apiErr != nil {
			return apiErr
		}

		runnerStatuses, apiErr := cancelQueueRunnerStatuses(
			e.Request.Context(),
			requestContext.deviceIDs,
			requestContext.namespace,
			requestContext.ticketID,
		)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := cleanupCanceledQueueResources(
			e.App,
			requestContext.ownerID,
			runnerStatuses,
		); apiErr != nil {
			return apiErr
		}

		response := buildQueueStatusResponse(
			requestContext.ticketID,
			runnerStatuses,
		)

		if response.Status == workflowengine.MobileDeviceSemaphoreRunNotFound {
			response.Status = workflowengine.MobileDeviceSemaphoreRunCanceled
		}

		return e.JSON(http.StatusOK, response)
	}
}

// buildQueueEnqueueResponse maps the aggregate semaphore status into the enqueue response contract.
func buildQueueEnqueueResponse(
	ticketID string,
	enqueuedAt time.Time,
	deviceIDs []string,
	status workflows.MobileDeviceSemaphoreRunStatus,
	position int,
	lineLen int,
	workflowID string,
	runID string,
	errorMessage string,
) PipelineQueueResponse {
	switch status {
	case workflowengine.MobileDeviceSemaphoreRunFailed,
		workflowengine.MobileDeviceSemaphoreRunCanceled:
		msg := strings.TrimSpace(errorMessage)
		if msg == "" {
			msg = "queue failed"
		}
		return PipelineQueueResponse{
			Status:       workflowengine.MobileDeviceSemaphoreRunFailed,
			ErrorMessage: msg,
		}
	case workflowengine.MobileDeviceSemaphoreRunRunning:
		return PipelineQueueResponse{
			Status:     workflowengine.MobileDeviceSemaphoreRunRunning,
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
			DeviceIDs:  copyStringSlice(deviceIDs),
			Position:   &pos,
			LineLen:    &line,
		}
	}
}

func decoratePipelineQueueResponseURLs(
	response *PipelineQueueResponse,
	appURL string,
	pipelineIdentifier string,
) {
	if response == nil {
		return
	}
	if strings.TrimSpace(pipelineIdentifier) != "" {
		response.PipelineURL = buildPipelinePageURL(appURL, pipelineIdentifier)
	}
	if strings.TrimSpace(response.WorkflowID) != "" && strings.TrimSpace(response.RunID) != "" {
		response.RunURL = buildPipelineRunPageURL(appURL, response.WorkflowID, response.RunID)
	}
}

func buildPipelinePageURL(appURL string, pipelineIdentifier string) string {
	return utils.JoinURL(
		appURL,
		"my",
		"pipelines",
		strings.TrimPrefix(canonify.NormalizePath(pipelineIdentifier), "/"),
	)
}

func buildPipelineRunPageURL(appURL string, workflowID string, runID string) string {
	return utils.JoinURL(appURL, "my", "tests", "runs", workflowID, runID)
}

func buildPipelineQueueConfig(
	e *core.RequestEvent,
	namespace string,
	userName string,
	userMail string,
) map[string]any {
	return workflowengine.WithAppConfig(e.App, map[string]any{
		"namespace": namespace,
		"user_name": userName,
		"user_mail": userMail,
	})
}

func applyPipelineQueueCleanupConfig(
	config map[string]any,
	cleanup *workflows.MobileDeviceSemaphoreCleanupMetadata,
) {
	if config == nil || cleanup == nil {
		return
	}
	if cleanup.TempWalletVersionID != "" {
		config[walletAPKCleanupConfigKey] = map[string]any{
			"record_id":  cleanup.TempWalletVersionID,
			"owner_id":   cleanup.TempWalletVersionOwnerID,
			"identifier": cleanup.TempWalletVersionIdentifier,
			"cleanup":    true,
		}
	}
	if len(cleanup.TempCredentials) > 0 {
		credentials := make([]map[string]any, 0, len(cleanup.TempCredentials))
		for _, credential := range cleanup.TempCredentials {
			if strings.TrimSpace(credential.RecordID) == "" {
				continue
			}
			credentials = append(credentials, map[string]any{
				"record_id":  credential.RecordID,
				"owner_id":   credential.OwnerID,
				"identifier": credential.Identifier,
			})
		}
		if len(credentials) > 0 {
			config[issuerCITempCredentialsConfigKey] = map[string]any{
				"credentials": credentials,
				"cleanup":     true,
			}
		}
	}
	if len(cleanup.TempUseCaseVerifications) > 0 {
		useCases := make([]map[string]any, 0, len(cleanup.TempUseCaseVerifications))
		for _, useCase := range cleanup.TempUseCaseVerifications {
			if strings.TrimSpace(useCase.RecordID) == "" {
				continue
			}
			useCases = append(useCases, map[string]any{
				"record_id":  useCase.RecordID,
				"owner_id":   useCase.OwnerID,
				"identifier": useCase.Identifier,
			})
		}
		if len(useCases) > 0 {
			config[verifierCITempUseCasesConfigKey] = map[string]any{
				"use_cases": useCases,
				"cleanup":   true,
			}
		}
	}
}

// startPipelineFromQueue starts a non-runner pipeline and persists the pipeline result record.
func startPipelineFromQueue(
	e *core.RequestEvent,
	pipelineRecord *core.Record,
	pipelineIdentifier string,
	ownerID string,
	yaml string,
	config map[string]any,
	memo map[string]any,
	runType string,
) (workflowengine.WorkflowResult, *apierror.APIError) {
	result, err := startPipelineWorkflow(yaml, config, memo, pipelineIdentifier)
	if err != nil {
		return result, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}

	if _, err := pipelineresults.Create(e.App, pipelineresults.CreateInput{
		OwnerID:    ownerID,
		PipelineID: pipelineRecord.Id,
		WorkflowID: result.WorkflowID,
		RunID:      result.WorkflowRunID,
		RunType:    runType,
	}); err != nil {
		return result, apierror.New(
			http.StatusInternalServerError,
			"pipeline",
			"failed to save pipeline record",
			err.Error(),
		)
	}

	return result, nil
}

func resolvePipelineDeviceIDs(yaml string, info pipeline.PipelineDeviceInfo) ([]string, error) {
	globalDeviceID := ""
	if info.NeedsGlobalDevice {
		wfDef, err := pipelineinternal.ParseWorkflow(yaml)
		if err != nil {
			return nil, err
		}
		globalDeviceID = strings.TrimSpace(wfDef.Runtime.GlobalDeviceID)
	}
	deviceIDs := pipeline.DeviceIDsWithGlobal(info, globalDeviceID)
	sort.Strings(deviceIDs)
	return deviceIDs, nil
}

func validatePipelineRunnerAccess(
	app core.App,
	ownerID string,
	deviceIDs []string,
) *apierror.APIError {
	err := mobilerunner.ValidateDeviceAccess(app, ownerID, deviceIDs)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, mobilerunner.ErrDeviceNotFound):
		return apierror.New(http.StatusNotFound, "device_id", "device not found", err.Error())
	case errors.Is(err, mobilerunner.ErrDeviceRunnerNotFound):
		return apierror.New(
			http.StatusNotFound,
			"device_id",
			"device runner not found",
			err.Error(),
		)
	case errors.Is(err, mobilerunner.ErrDeviceNotAccessible):
		return apierror.New(
			http.StatusForbidden,
			"device_id",
			"device_id is not accessible",
			err.Error(),
		)
	default:
		return apierror.New(
			http.StatusInternalServerError,
			"device_id",
			"failed to validate device access",
			err.Error(),
		)
	}
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

	deviceIDs := mobilerunner.NormalizeDeviceIDs(parseDeviceIDs(e.Request))
	if len(deviceIDs) == 0 {
		return nil, apierror.New(
			http.StatusBadRequest,
			"device_ids",
			"device_ids are required",
			"missing device_ids query parameter",
		)
	}

	orgRecord, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization record",
			err.Error(),
		)
	}
	namespace := strings.TrimSpace(orgRecord.GetString("canonified_name"))
	if namespace == "" {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization canonified name",
			"missing organization canonified name",
		)
	}

	return &queueRequestContext{
		ticketID:  ticketID,
		deviceIDs: deviceIDs,
		namespace: namespace,
		ownerID:   orgRecord.Id,
	}, nil
}

func queryQueueRunnerStatuses(
	ctx context.Context,
	deviceIDs []string,
	namespace string,
	ticketID string,
) ([]pipelineQueueRunnerStatus, *apierror.APIError) {
	runnerStatuses := make([]pipelineQueueRunnerStatus, 0, len(deviceIDs))

	for _, deviceID := range deviceIDs {
		status, err := queryRunTicketStatus(ctx, deviceID, namespace, ticketID)
		if err != nil {
			if errors.Is(err, semaphoreclient.ErrRunTicketNotFound) {
				runnerStatuses = append(
					runnerStatuses,
					runnerStatusFromView(deviceID, runTicketNotFoundView(ticketID)),
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
		runnerStatuses = append(runnerStatuses, runnerStatusFromView(deviceID, status))
	}

	return runnerStatuses, nil
}

func cancelQueueRunnerStatuses(
	ctx context.Context,
	deviceIDs []string,
	namespace string,
	ticketID string,
) ([]pipelineQueueRunnerStatus, *apierror.APIError) {
	runnerStatuses := make([]pipelineQueueRunnerStatus, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		status, err := cancelRunTicket(
			ctx,
			deviceID,
			workflows.MobileDeviceSemaphoreRunCancelRequest{
				TicketID:       ticketID,
				OwnerNamespace: namespace,
			},
		)
		if err != nil {
			if errors.Is(err, semaphoreclient.ErrRunTicketNotFound) {
				continue
			}
			return nil, apierror.New(
				http.StatusInternalServerError,
				"semaphore",
				"failed to cancel ticket",
				err.Error(),
			)
		}
		runnerStatuses = append(runnerStatuses, runnerStatusFromView(deviceID, status))
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

func runTicketNotFoundView(ticketID string) workflows.MobileDeviceSemaphoreRunStatusView {
	return workflows.MobileDeviceSemaphoreRunStatusView{
		TicketID: ticketID,
		Status:   workflowengine.MobileDeviceSemaphoreRunNotFound,
	}
}

func parseDeviceIDs(req *http.Request) []string {
	values := req.URL.Query()["device_ids[]"]
	if len(values) == 0 {
		values = req.URL.Query()["device_ids"]
	}
	return values
}

func runnerStatusFromView(
	deviceID string,
	status workflows.MobileDeviceSemaphoreRunStatusView,
) pipelineQueueRunnerStatus {
	return pipelineQueueRunnerStatus{
		DeviceID:     deviceID,
		Status:       status.Status,
		Position:     status.Position,
		LineLen:      status.LineLen,
		WorkflowID:   status.WorkflowID,
		RunID:        status.RunID,
		ErrorMessage: status.ErrorMessage,
		Cleanup:      status.Cleanup,
	}
}

// cleanupCanceledQueueResources removes resources attached to a queue ticket
// only when every semaphore confirms the run never reached a running workflow.
func cleanupCanceledQueueResources(
	app core.App,
	ownerID string,
	statuses []pipelineQueueRunnerStatus,
) *apierror.APIError {
	cleanup, ok := canceledQueueCleanupMetadata(statuses)
	if !ok {
		return nil
	}
	if strings.TrimSpace(cleanup.TempWalletVersionID) != "" {
		if apiErr := deleteTempWalletVersionForOwner(
			app,
			cleanup.TempWalletVersionID,
			ownerID,
		); apiErr != nil {
			return apiErr
		}
	}
	for _, credential := range cleanup.TempCredentials {
		if strings.TrimSpace(credential.RecordID) == "" {
			continue
		}
		if apiErr := deleteTempCredentialForOwner(
			app,
			credential.RecordID,
			ownerID,
		); apiErr != nil {
			return apiErr
		}
	}
	for _, useCase := range cleanup.TempUseCaseVerifications {
		if strings.TrimSpace(useCase.RecordID) == "" {
			continue
		}
		if apiErr := deleteTempUseCaseVerificationForOwner(
			app,
			useCase.RecordID,
			ownerID,
		); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func canceledQueueCleanupMetadata(
	statuses []pipelineQueueRunnerStatus,
) (*workflows.MobileDeviceSemaphoreCleanupMetadata, bool) {
	var cleanup *workflows.MobileDeviceSemaphoreCleanupMetadata
	for _, status := range statuses {
		if status.WorkflowID != "" || status.RunID != "" {
			return nil, false
		}
		if status.Status == workflowengine.MobileDeviceSemaphoreRunRunning {
			return nil, false
		}
		if status.Cleanup != nil &&
			(strings.TrimSpace(status.Cleanup.TempWalletVersionID) != "" ||
				len(status.Cleanup.TempCredentials) > 0 ||
				len(status.Cleanup.TempUseCaseVerifications) > 0) {
			cleanup = status.Cleanup
		}
	}
	return cleanup, cleanup != nil
}

func deleteTempWalletVersionForOwner(
	app core.App,
	recordID string,
	ownerID string,
) *apierror.APIError {
	record, err := app.FindRecordById("wallet_versions", strings.TrimSpace(recordID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return apierror.New(
			http.StatusInternalServerError,
			"wallet_version",
			"failed to find temporary wallet version",
			err.Error(),
		)
	}
	if record.GetString("owner") != ownerID {
		return apierror.New(
			http.StatusForbidden,
			"wallet_version",
			"temporary wallet version owner mismatch",
			"queued cleanup does not belong to the authenticated organization",
		)
	}
	if err := app.Delete(record); err != nil {
		return apierror.New(
			http.StatusInternalServerError,
			"wallet_version",
			"failed to delete temporary wallet version",
			err.Error(),
		)
	}
	return nil
}

func deleteTempUseCaseVerificationForOwner(
	app core.App,
	recordID string,
	ownerID string,
) *apierror.APIError {
	record, err := app.FindRecordById("use_cases_verifications", strings.TrimSpace(recordID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return apierror.New(
			http.StatusInternalServerError,
			"use_case_verification",
			"failed to find temporary use case verification",
			err.Error(),
		)
	}
	if record.GetString("owner") != ownerID {
		return apierror.New(
			http.StatusForbidden,
			"use_case_verification",
			"temporary use case verification owner mismatch",
			"queued cleanup does not belong to the authenticated organization",
		)
	}
	if err := app.Delete(record); err != nil {
		return apierror.New(
			http.StatusInternalServerError,
			"use_case_verification",
			"failed to delete temporary use case verification",
			err.Error(),
		)
	}
	return nil
}

func aggregateRunQueueStatus(
	statuses []pipelineQueueRunnerStatus,
) (
	workflows.MobileDeviceSemaphoreRunStatus,
	int,
	int,
	string,
	string,
	string,
) {
	queueStatuses := make([]runqueue.DeviceStatus, 0, len(statuses))
	for _, status := range statuses {
		queueStatuses = append(queueStatuses, runqueue.DeviceStatus{
			DeviceID:     status.DeviceID,
			Status:       status.Status,
			Position:     status.Position,
			LineLen:      status.LineLen,
			WorkflowID:   status.WorkflowID,
			RunID:        status.RunID,
			ErrorMessage: status.ErrorMessage,
		})
	}

	aggregate := runqueue.AggregateDeviceStatuses(queueStatuses)

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

func queryRunTicketStatusTemporal(
	ctx context.Context,
	deviceID string,
	ownerNamespace string,
	ticketID string,
) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
	client, err := queueTemporalClient(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}

	workflowID := workflows.MobileDeviceSemaphoreWorkflowID(deviceID)
	encoded, err := client.QueryWorkflow(
		ctx,
		workflowID,
		"",
		workflows.MobileDeviceSemaphoreRunStatusQuery,
		ownerNamespace,
		ticketID,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return workflows.MobileDeviceSemaphoreRunStatusView{}, semaphoreclient.ErrRunTicketNotFound
		}
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}

	var status workflows.MobileDeviceSemaphoreRunStatusView
	if err := encoded.Get(&status); err != nil {
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}
	return status, nil
}

// startPipelineWorkflowTemporal runs the pipeline workflow directly via the Temporal client.
func startPipelineWorkflowTemporal(
	yaml string,
	config map[string]any,
	memo map[string]any,
	pipelineIdentifier string,
) (workflowengine.WorkflowResult, error) {
	w := pipeline.NewPipelineWorkflow()
	return w.Start(yaml, config, memo, pipelineIdentifier)
}
