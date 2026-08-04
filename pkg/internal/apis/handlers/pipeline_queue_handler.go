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
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
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

var errRunTicketNotFound = errors.New("run ticket not found")

func isQueueLimitExceeded(err error) bool {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Type() == workflows.MobileDeviceSemaphoreErrQueueLimitExceeded
	}
	return false
}

var ensureRunQueueSemaphoreWorkflow = ensureRunQueueSemaphoreWorkflowTemporal
var enqueueRunTicket = enqueueRunTicketTemporal
var queryRunTicketStatus = queryRunTicketStatusTemporal
var cancelRunTicket = cancelRunTicketTemporal
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
	memo[pipelineinternal.PublishedMemoKey] = runContext.pipelineRecord.GetBool("published")
	config := buildPipelineQueueConfig(e, namespace, runContext.userName, runContext.userEmail)
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

	for _, deviceID := range deviceIDs {
		if err := ensureRunQueueSemaphoreWorkflow(e.Request.Context(), deviceID); err != nil {
			return PipelineQueueResponse{}, apierror.New(
				http.StatusInternalServerError,
				"semaphore",
				"failed to ensure runner semaphore",
				err.Error(),
			)
		}
	}

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
				if errors.Is(err, errRunTicketNotFound) {
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
	setPipelineRunType(record, coll, runType)

	result, err = startPipelineWorkflow(yaml, config, memo, pipelineIdentifier)
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
	for _, deviceID := range normalizeDeviceIDs(deviceIDs) {
		record, err := canonify.Resolve(app, deviceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apierror.New(
					http.StatusNotFound,
					"device_id",
					"runner not found",
					"mobile runner "+deviceID+" was not found",
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"device_id",
				"failed to resolve device_id",
				err.Error(),
			)
		}
		if record.Collection() == nil || record.Collection().Name != mobileDevicesCollection {
			return apierror.New(
				http.StatusNotFound,
				"device_id",
				"device not found",
				"mobile device "+deviceID+" was not found",
			)
		}
		runner, runnerErr := app.FindRecordById("mobile_runners", record.GetString("runner"))
		if runnerErr != nil {
			return apierror.New(
				http.StatusNotFound,
				"device_id",
				"device runner not found",
				runnerErr.Error(),
			)
		}
		if record.GetString("owner") == ownerID || runner.GetBool("published") {
			continue
		}
		return apierror.New(
			http.StatusForbidden,
			"device_id",
			"device_id is not accessible",
			"mobile device "+deviceID+" is private and does not belong to the caller organization",
		)
	}
	return nil
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

	deviceIDs := normalizeDeviceIDs(parseDeviceIDs(e.Request))
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
			if errors.Is(err, errRunTicketNotFound) {
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

func normalizeDeviceIDs(values []string) []string {
	unique := map[string]struct{}{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			candidate := canonify.NormalizePath(part)
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

func ensureRunQueueSemaphoreWorkflowTemporal(ctx context.Context, deviceID string) error {
	deviceID = canonify.NormalizePath(deviceID)

	client, err := queueTemporalClient(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return err
	}

	workflowID := workflows.MobileDeviceSemaphoreWorkflowID(deviceID)
	input := workflowengine.WorkflowInput{
		Payload: workflows.MobileDeviceSemaphoreWorkflowInput{
			DeviceID: deviceID,
			Capacity: 1,
		},
	}

	_, err = client.ExecuteWorkflow(
		ctx,
		tclient.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: workflows.MobileDeviceSemaphoreTaskQueue,
		},
		workflows.MobileDeviceSemaphoreWorkflowName,
		input,
	)
	if err != nil && !temporal.IsWorkflowExecutionAlreadyStartedError(err) {
		return err
	}

	return nil
}

func enqueueRunTicketTemporal(
	ctx context.Context,
	deviceID string,
	req workflows.MobileDeviceSemaphoreEnqueueRunRequest,
) (workflows.MobileDeviceSemaphoreEnqueueRunResponse, error) {
	deviceID = canonify.NormalizePath(deviceID)
	req.DeviceID = canonify.NormalizePath(req.DeviceID)
	req.RequiredDeviceIDs = normalizeDeviceIDs(req.RequiredDeviceIDs)
	req.LeaderDeviceID = canonify.NormalizePath(req.LeaderDeviceID)

	client, err := queueTemporalClient(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{}, err
	}

	workflowID := workflows.MobileDeviceSemaphoreWorkflowID(deviceID)
	handle, err := client.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   workflows.MobileDeviceSemaphoreEnqueueRunUpdate,
		UpdateID:     runQueueUpdateID("enqueue", deviceID, req.TicketID),
		Args:         []interface{}{req},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{}, err
	}

	var response workflows.MobileDeviceSemaphoreEnqueueRunResponse
	if err := handle.Get(ctx, &response); err != nil {
		return workflows.MobileDeviceSemaphoreEnqueueRunResponse{}, err
	}
	return response, nil
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
			return workflows.MobileDeviceSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}

	var status workflows.MobileDeviceSemaphoreRunStatusView
	if err := encoded.Get(&status); err != nil {
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}
	return status, nil
}

func cancelRunTicketTemporal(
	ctx context.Context,
	deviceID string,
	req workflows.MobileDeviceSemaphoreRunCancelRequest,
) (workflows.MobileDeviceSemaphoreRunStatusView, error) {
	client, err := queueTemporalClient(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}

	workflowID := workflows.MobileDeviceSemaphoreWorkflowID(deviceID)
	handle, err := client.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   workflows.MobileDeviceSemaphoreCancelRunUpdate,
		UpdateID:     runQueueUpdateID("cancel", deviceID, req.TicketID),
		Args:         []interface{}{req},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return workflows.MobileDeviceSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		return workflows.MobileDeviceSemaphoreRunStatusView{}, err
	}

	var status workflows.MobileDeviceSemaphoreRunStatusView
	if err := handle.Get(ctx, &status); err != nil {
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

func runQueueUpdateID(prefix, deviceID, ticketID string) string {
	deviceID = canonify.NormalizePath(deviceID)
	return prefix + "/" + deviceID + "/" + ticketID
}
