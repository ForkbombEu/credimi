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
	"regexp"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var PipelineRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/pipeline",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/queue",
			Handler:       HandlePipelineQueueEnqueue,
			RequestSchema: PipelineQueueInput{},
			Description:   "Queue a pipeline workflow for the runner semaphore",
		},
		{
			Method:         http.MethodPost,
			Path:           "/run-wallet-apk",
			Handler:        HandlePipelineRunWalletAPK,
			ResponseSchema: PipelineRunWalletAPKResponse{},
			Description:    "Create a temporary wallet APK version and queue a one-off pipeline run",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				apis.BodyLimit(1000 << 20),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/run-issuer",
			Handler:        HandlePipelineRunIssuer,
			ResponseSchema: PipelineRunIssuerResponse{},
			Description:    "Create temporary issuer credentials and queue a one-off pipeline run",
		},
		{
			Method:         http.MethodPost,
			Path:           "/run-verifier",
			Handler:        HandlePipelineRunVerifier,
			ResponseSchema: PipelineRunVerifierResponse{},
			Description:    "Create temporary verifier use cases and queue a one-off pipeline run",
		},
		{
			Method:      http.MethodGet,
			Path:        "/queue/{ticket}",
			Handler:     HandlePipelineQueueStatus,
			Description: "Get queued pipeline status by ticket",
		},
		{
			Method:      http.MethodDelete,
			Path:        "/queue/{ticket}",
			Handler:     HandlePipelineQueueCancel,
			Description: "Cancel a queued pipeline ticket",
		},
		{
			Method:  http.MethodGet,
			Path:    "/list-executions",
			Handler: HandleListPipelineExecutionOverview,
		},
		{
			Method:  http.MethodGet,
			Path:    "/list-executions/{id}",
			Handler: HandleListPipelineExecutionHistory,
		},
		{
			Method:      http.MethodGet,
			Path:        "/executions/{pipeline_id}/{workflow_id}/{run_id}",
			Handler:     HandleGetPipelineExecution,
			Description: "Get one pipeline execution with its child workflows",
		},
		{
			Method:      http.MethodPost,
			Path:        "/execute",
			Handler:     HandlePipelineExecute,
			Description: "Execute a pipeline synchronously and wait for result",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.OptionalAuthOrAPIKey(),
			},
			ExcludedMiddlewares: []string{
				middlewares.RequireAuthOrAPIKeyMiddlewareID,
			},
		},
	},
}

var PipelineTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/pipeline",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodPost,
			Path:        "/store-step-screenshots",
			Handler:     HandleStorePipelineStepScreenshots,
			Description: "Store Maestro screenshots produced by one pipeline step",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
				apis.BodyLimit(500 << 20),
			},
		},
		{
			Method:      http.MethodGet,
			Path:        "/get-yaml",
			Handler:     HandleGetPipelineYAML,
			Description: "Get a pipeline YAML from a pipeline ID",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/mobile-flow",
			Handler:        HandlePipelineMobileFlow,
			RequestSchema:  PipelineMobileFlowInput{},
			ResponseSchema: PipelineMobileFlowResponse{},
			Description:    "Run a wallet mobile action on the initialized device reserved by a running pipeline",
		},
		{
			Method:        http.MethodPost,
			Path:          "/pipeline-execution-results",
			Handler:       HandleSetPipelineExecutionResults,
			RequestSchema: PipelineResultInput{},
			Description:   "Create pipeline execution results record",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:        http.MethodPost,
			Path:          "/pipeline-execution-results/evidence",
			Handler:       HandleUpdatePipelineExecutionEvidence,
			RequestSchema: PipelineResultEvidenceInput{},
			Description:   "Update pipeline execution evidence fields",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:        http.MethodPost,
			Path:          "/pipeline-execution-results/report",
			Handler:       HandleUpdatePipelineExecutionReport,
			RequestSchema: PipelineResultReportInput{},
			Description:   "Update pipeline execution markdown report file",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/scoreboard/{namespace}",
			Handler: HandleGetPipelineScoreboard,
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/scoreboard/aggregate/start",
			Handler:        HandleStartAggregateScoreboard,
			ResponseSchema: StartAggregateScoreboardResponse{},
			Description:    "Start the aggregate scoreboard workflow (use ?schedule=300 per scheduling every N seconds)",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:      http.MethodDelete,
			Path:        "/scoreboard/aggregate/schedule/{schedule_id}",
			Handler:     HandleCancelAggregateScoreboardSchedule,
			Description: "Cancel a scheduled aggregate scoreboard workflow",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:      http.MethodGet,
			Path:        "/execution-details/{namespace}/{workflow_id}/{run_id}",
			Handler:     HandleGetExecutionDetails,
			Description: "Get detailed information about a specific execution",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:      http.MethodPost,
			Path:        "/scoreboard/save-results",
			Handler:     HandleSaveScoreboardResults,
			Description: "Refresh the aggregate scoreboard",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:        http.MethodPost,
			Path:          "/retention/delete-files",
			Handler:       HandleDeletePipelineResultFiles,
			RequestSchema: DeletePipelineResultFilesRequest{},
			Description:   "Delete retained pipeline result files older than the requested number of days",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/retention/schedule",
			Handler:        HandleSchedulePipelineRetentionWorkflow,
			RequestSchema:  SchedulePipelineRetentionRequest{},
			ResponseSchema: SchedulePipelineRetentionResponse{},
			Description:    "Schedule the pipeline retention workflow",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:         http.MethodDelete,
			Path:           "/retention/schedule",
			Handler:        HandleDeletePipelineRetentionSchedule,
			ResponseSchema: DeletePipelineRetentionScheduleResponse{},
			Description:    "Delete the pipeline retention schedule",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
	},
}

var pipelineTemporalClient = temporalclient.GetTemporalClientWithNamespace
var pipelineListQueuedRuns = listQueuedPipelineRuns

const pipelineListWorkflowsDefaultLimit = 1000

func HandleGetPipelineYAML() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		pipelineIdentifier := e.Request.URL.Query().Get("pipeline_identifier")
		if pipelineIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"pipeline_identifier",
				"pipeline_identifier is required",
				"missing pipeline_identifier",
			)
		}

		record, err := canonify.Resolve(e.App, pipelineIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_identifier",
				"pipeline not found",
				err.Error(),
			)
		}
		yaml := record.GetString("yaml")
		return e.String(http.StatusOK, yaml)
	}
}

type PipelineResultInput struct {
	Owner      string `json:"owner"`
	PipelineID string `json:"pipeline_id"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	Type       string `json:"type,omitempty"`
}

type PipelineResultEvidenceInput struct {
	WorkflowID           string           `json:"workflow_id"`
	RunID                string           `json:"run_id"`
	CredentialWellKnowns []map[string]any `json:"credential_well_knowns"`
	PresentationResults  []map[string]any `json:"presentation_results"`
}

type PipelineResultReportInput struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	Filename   string `json:"filename"`
	Markdown   string `json:"markdown"`
}

func pipelineRunType(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return pipelineinternal.RunTypeManual
	}
	return input
}

func setPipelineRunType(record *core.Record, coll *core.Collection, runType string) {
	if coll.Fields.GetByName("type") == nil {
		return
	}
	record.Set("type", pipelineRunType(runType))
}

func HandleUpdatePipelineExecutionReport() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineResultReportInput](e)
		if err != nil {
			return err
		}
		if strings.TrimSpace(input.WorkflowID) == "" || strings.TrimSpace(input.RunID) == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"workflow_id and run_id are required",
				"missing workflow_id or run_id",
			)
		}
		if strings.TrimSpace(input.Markdown) == "" {
			return apierror.New(
				http.StatusBadRequest,
				"report",
				"markdown is required",
				"missing markdown",
			)
		}

		record, apiErr := findPipelineResultByWorkflowRun(e, input.WorkflowID, input.RunID)
		if apiErr != nil {
			return apiErr
		}

		filename := sanitizePipelineReportFilename(input.Filename)
		file, err := filesystem.NewFileFromBytes([]byte(input.Markdown), filename)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"report",
				"failed to create report file",
				err.Error(),
			)
		}
		record.Set("report", []*filesystem.File{file})
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to save pipeline report",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, record.FieldsData())
	}
}

func HandleUpdatePipelineExecutionEvidence() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineResultEvidenceInput](e)
		if err != nil {
			return err
		}
		if strings.TrimSpace(input.WorkflowID) == "" || strings.TrimSpace(input.RunID) == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"workflow_id and run_id are required",
				"missing workflow_id or run_id",
			)
		}

		record, apiErr := findPipelineResultByWorkflowRun(e, input.WorkflowID, input.RunID)
		if apiErr != nil {
			return apiErr
		}

		record.Set("credential_well_knowns", input.CredentialWellKnowns)
		record.Set("presentation_results", input.PresentationResults)
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to save pipeline evidence",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, record.FieldsData())
	}
}

func findPipelineResultByWorkflowRun(
	e *core.RequestEvent,
	workflowID string,
	runID string,
) (*core.Record, *apierror.APIError) {
	coll, err := e.App.FindCollectionByNameOrId("pipeline_results")
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"collection",
			"failed to get collection",
			err.Error(),
		)
	}

	record, err := e.App.FindFirstRecordByFilter(
		coll,
		"workflow_id = {:workflow_id} && run_id = {:run_id}",
		dbx.Params{
			"workflow_id": workflowID,
			"run_id":      runID,
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierror.New(
				http.StatusNotFound,
				"pipeline",
				"pipeline execution result not found",
				"pipeline execution result not found for workflow_id and run_id",
			)
		}
		return nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline",
			"failed to lookup pipeline execution result",
			err.Error(),
		)
	}

	return record, nil
}

var unsafePipelineReportFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizePipelineReportFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "pipeline-report.md"
	}
	filename = unsafePipelineReportFilenameChars.ReplaceAllString(filename, "-")
	filename = strings.Trim(filename, ".-_")
	if filename == "" {
		return "pipeline-report.md"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename += ".md"
	}
	return filename
}

func HandleSetPipelineExecutionResults() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineResultInput](e)
		if err != nil {
			return err
		}

		pipeline, err := canonify.Resolve(e.App, input.PipelineID)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline",
				"pipeline not found",
				err.Error(),
			)
		}

		owner, err := canonify.Resolve(e.App, input.Owner)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"owner",
				"owner not found",
				err.Error(),
			)
		}

		coll, err := e.App.FindCollectionByNameOrId("pipeline_results")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"collection",
				"failed to get collection",
				err.Error(),
			)
		}
		runType := pipelineRunType(input.Type)
		if !pipelineinternal.ValidRunType(runType) {
			return apierror.New(
				http.StatusBadRequest,
				"type",
				"invalid pipeline result type",
				fmt.Sprintf("type must be one of %q, %q, or %q",
					pipelineinternal.RunTypeManual,
					pipelineinternal.RunTypeScheduled,
					pipelineinternal.RunTypeCI,
				),
			)
		}

		existing, err := e.App.FindFirstRecordByFilter(
			coll,
			"workflow_id = {:workflow_id} && run_id = {:run_id}",
			dbx.Params{
				"workflow_id": input.WorkflowID,
				"run_id":      input.RunID,
			},
		)
		if err == nil {
			if existing.GetString("owner") == owner.Id &&
				existing.GetString("pipeline") == pipeline.Id {
				return e.JSON(http.StatusOK, existing.FieldsData())
			}
			return apierror.New(
				http.StatusConflict,
				"pipeline",
				"pipeline execution result already exists",
				"pipeline execution result owner or pipeline mismatch",
			)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to lookup pipeline execution result",
				err.Error(),
			)
		}

		record := core.NewRecord(coll)
		record.Set("owner", owner.Id)
		record.Set("pipeline", pipeline.Id)
		record.Set("workflow_id", input.WorkflowID)
		record.Set("run_id", input.RunID)
		setPipelineRunType(record, coll, runType)
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to save pipeline record",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, record.FieldsData())
	}
}

type pipelineExecutionScope struct {
	Auth               *core.Record
	Organization       *core.Record
	Pipeline           *core.Record
	Namespace          string
	PipelineIdentifier string
}

func resolvePipelineExecutionScope(
	e *core.RequestEvent,
	pipelineID string,
) (*pipelineExecutionScope, *apierror.APIError) {
	if e.Auth == nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"auth",
			"authentication required",
			"user not authenticated",
		)
	}
	if strings.TrimSpace(pipelineID) == "" {
		return nil, apierror.New(
			http.StatusBadRequest,
			"pipeline",
			"pipeline ID is required",
			"missing pipeline ID in path parameter",
		)
	}

	organization, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"failed to get user organization",
			err.Error(),
		)
	}
	pipelineRecords, err := e.App.FindRecordsByFilter(
		"pipelines",
		"(id = {:id} && owner={:owner}) || (id = {:id} && published={:published})",
		"",
		-1,
		0,
		dbx.Params{
			"id":        pipelineID,
			"owner":     organization.Id,
			"published": true,
		},
	)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"pipelines",
			"failed to fetch pipelines",
			err.Error(),
		)
	}
	if len(pipelineRecords) == 0 {
		return nil, apierror.New(
			http.StatusNotFound,
			"pipeline",
			"pipeline not found",
			"pipeline is not available to the authenticated organization",
		)
	}

	pipelineRecord := pipelineRecords[0]
	pipelinePath, err := canonify.BuildPath(
		e.App,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline",
			"failed to build pipeline identifier",
			err.Error(),
		)
	}

	return &pipelineExecutionScope{
		Auth:               e.Auth,
		Organization:       organization,
		Pipeline:           pipelineRecord,
		Namespace:          organization.GetString("canonified_name"),
		PipelineIdentifier: strings.Trim(pipelinePath, "/"),
	}, nil
}

// HandleListPipelineExecutionHistory lists paginated executions for one pipeline,
// including each execution's child workflows.
func HandleListPipelineExecutionHistory() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		scope, apiErr := resolvePipelineExecutionScope(e, e.Request.PathValue("id"))
		if apiErr != nil {
			if apiErr.Code == http.StatusNotFound {
				return e.JSON(http.StatusOK, []*pipelineWorkflowSummary{})
			}
			return apiErr
		}

		limit, page := parsePageParams(e, 20, 0)
		itemOffset := page * limit
		statusFilter := e.Request.URL.Query().Get("status")

		includeQueued := shouldIncludeQueuedPipelineExecutions(statusFilter)
		if includeQueued {
			queuedRuns, err := pipelineListQueuedRuns(e.Request.Context(), scope.Namespace)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to list queued runs",
					err.Error(),
				)
			}

			queuedByPipelineID := mapQueuedRunsToPipelines(
				e.App,
				[]*core.Record{scope.Pipeline},
				queuedRuns,
			)
			queuedForPipeline := queuedByPipelineID[scope.Pipeline.Id]

			if queuedOnlyPipelineExecutions(statusFilter) {
				return e.JSON(http.StatusOK, paginateQueuedPipelineSummaries(
					e.App,
					queuedForPipeline,
					scope.Auth.GetString("Timezone"),
					limit,
					itemOffset,
				))
			}

			queuedCount := len(queuedForPipeline)
			if itemOffset < queuedCount {
				queuedSummaries := paginateQueuedPipelineSummaries(
					e.App,
					queuedForPipeline,
					scope.Auth.GetString("Timezone"),
					limit,
					itemOffset,
				)
				if len(queuedSummaries) >= limit {
					return e.JSON(http.StatusOK, queuedSummaries)
				}

				completedSummaries, apiErr := listPipelineExecutionHistoryPage(
					e.Request.Context(),
					e.App,
					scope,
					statusFilter,
					limit-len(queuedSummaries),
					0,
				)
				if apiErr != nil {
					return apiErr
				}
				return e.JSON(http.StatusOK, append(queuedSummaries, completedSummaries...))
			}

			itemOffset -= queuedCount
		}

		summaries, apiErr := listPipelineExecutionHistoryPage(
			e.Request.Context(),
			e.App,
			scope,
			statusFilter,
			limit,
			itemOffset,
		)
		if apiErr != nil {
			return apiErr
		}

		return e.JSON(http.StatusOK, summaries)
	}
}

func listPipelineExecutionHistoryPage(
	ctx context.Context,
	app core.App,
	scope *pipelineExecutionScope,
	statusFilter string,
	limit int,
	skip int,
) ([]*pipelineWorkflowSummary, *apierror.APIError) {
	temporalClient, err := pipelineTemporalClient(scope.Namespace)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"temporal",
			"unable to create temporal client",
			err.Error(),
		)
	}

	summaries, err := listPipelineExecutionHistory(ctx, pipelineExecutionHistoryRequest{
		App:                app,
		TemporalClient:     temporalClient,
		Namespace:          scope.Namespace,
		OwnerID:            scope.Organization.Id,
		UserTimezone:       scope.Auth.GetString("Timezone"),
		PipelineRecord:     scope.Pipeline,
		PipelineIdentifier: scope.PipelineIdentifier,
		StatusFilter:       statusFilter,
		Limit:              limit,
		Skip:               skip,
	})
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to list pipeline executions",
			err.Error(),
		)
	}
	return summaries, nil
}

// HandleGetPipelineExecution returns one exact pipeline execution and its child workflows.
func HandleGetPipelineExecution() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		scope, apiErr := resolvePipelineExecutionScope(
			e,
			e.Request.PathValue("pipeline_id"),
		)
		if apiErr != nil {
			return apiErr
		}

		workflowID := strings.TrimSpace(e.Request.PathValue("workflow_id"))
		runID := strings.TrimSpace(e.Request.PathValue("run_id"))
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"workflow ID and run ID are required",
				"missing workflow_id or run_id path parameter",
			)
		}

		temporalClient, err := pipelineTemporalClient(scope.Namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create temporal client",
				err.Error(),
			)
		}

		execution, apiErr := describeWorkflowExecution(
			e.Request.Context(),
			temporalClient,
			workflowID,
			runID,
		)
		if apiErr != nil {
			return apiErr
		}
		if execution.Type.Name != pipeline.NewPipelineWorkflow().Name() {
			return pipelineExecutionNotFound()
		}

		ref := workflowExecutionRef{WorkflowID: workflowID, RunID: runID}
		resultRecords, err := fetchPipelineResultRecords(
			e.App,
			scope.Organization.Id,
			[]workflowExecutionRef{ref},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"database",
				"failed to fetch pipeline results",
				err.Error(),
			)
		}
		resultRecord := resultRecords[ref]
		identifier := pipelineIdentifierFromSearchAttributes(execution.SearchAttributes)
		matchesSearchAttribute := identifier == scope.PipelineIdentifier
		matchesResultRecord := resultRecord != nil &&
			resultRecord.GetString("pipeline") == scope.Pipeline.Id
		if !matchesSearchAttribute && !matchesResultRecord {
			return pipelineExecutionNotFound()
		}
		if !matchesResultRecord {
			resultRecord = nil
		}

		childrenByParent, err := getChildWorkflowsByParents(
			e.Request.Context(),
			temporalClient,
			scope.Namespace,
			[]workflowExecutionRef{ref},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list child workflows",
				err.Error(),
			)
		}

		builder := newPipelineExecutionSummaryBuilder(
			e.App,
			temporalClient,
			scope.Auth.GetString("Timezone"),
		)
		summary, err := builder.Build(
			e.Request.Context(),
			scope.Pipeline,
			scope.PipelineIdentifier,
			execution,
			childrenByParent[ref],
			resultRecord,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to build pipeline execution summary",
				err.Error(),
			)
		}
		if summary == nil {
			return pipelineExecutionNotFound()
		}
		return e.JSON(http.StatusOK, summary)
	}
}

func pipelineExecutionNotFound() *apierror.APIError {
	return apierror.New(
		http.StatusNotFound,
		"workflow",
		"pipeline execution not found",
		"workflow execution does not belong to the requested pipeline",
	)
}

func shouldIncludeQueuedPipelineExecutions(statusFilter string) bool {
	if strings.TrimSpace(statusFilter) == "" {
		return true
	}

	for _, raw := range strings.Split(statusFilter, ",") {
		if strings.EqualFold(strings.TrimSpace(raw), statusStringQueued) {
			return true
		}
	}

	return false
}

func queuedOnlyPipelineExecutions(statusFilter string) bool {
	if strings.TrimSpace(statusFilter) == "" {
		return false
	}

	hasQueued := false
	hasNonQueued := false
	for _, raw := range strings.Split(statusFilter, ",") {
		normalized := strings.ToLower(strings.TrimSpace(raw))
		if normalized == "" {
			continue
		}
		if normalized == statusStringQueued {
			hasQueued = true
			continue
		}
		hasNonQueued = true
	}

	return hasQueued && !hasNonQueued
}

func paginateQueuedPipelineSummaries(
	app core.App,
	queuedRuns []QueuedPipelineRunAggregate,
	userTimezone string,
	limit int,
	offset int,
) []*pipelineWorkflowSummary {
	summaries := buildQueuedPipelineSummaries(
		app,
		queuedRuns,
		userTimezone,
		map[string]map[string]any{},
	)
	if len(summaries) == 0 || limit <= 0 || offset >= len(summaries) {
		return []*pipelineWorkflowSummary{}
	}
	if offset < 0 {
		offset = 0
	}

	end := offset + limit
	if end > len(summaries) {
		end = len(summaries)
	}

	return summaries[offset:end]
}

// HandleListPipelineExecutionOverview returns recent top-level executions grouped by pipeline.
func HandleListPipelineExecutionOverview() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			)
		}

		organization, err := pbutils.GetUserOrganization(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			)
		}
		namespace := organization.GetString("canonified_name")

		pipelineRecords, err := e.App.FindRecordsByFilter(
			"pipelines",
			"owner={:owner} || published={:published}",
			"",
			-1,
			0,
			dbx.Params{
				"owner":     organization.Id,
				"published": true,
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipelines",
				"failed to fetch pipelines",
				err.Error(),
			)
		}

		if len(pipelineRecords) == 0 {
			return e.JSON(http.StatusOK, map[string][]*WorkflowExecutionSummary{})
		}

		queuedRuns, err := pipelineListQueuedRuns(e.Request.Context(), namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list queued runs",
				err.Error(),
			)
		}
		queuedByPipelineID := mapQueuedRunsToPipelines(e.App, pipelineRecords, queuedRuns)

		pipelineMap := make(map[string]*core.Record, len(pipelineRecords))
		for _, pipelineRecord := range pipelineRecords {
			pipelineMap[pipelineRecord.Id] = pipelineRecord
		}

		pipelineIdentifierIndex := buildPipelineIdentifierIndex(e.App, pipelineMap)
		pipelineIdentifierByID := buildPipelineIdentifierByID(e.App, pipelineMap)

		temporalClient, err := pipelineTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create temporal client",
				err.Error(),
			)
		}

		executions, err := listPipelineWorkflowExecutions(
			context.Background(),
			temporalClient,
			namespace,
			nil,
			"",
			pipelineListWorkflowsDefaultLimit,
			0,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list workflows",
				err.Error(),
			)
		}

		pipelineIdentifiers := resolvePipelineIdentifiersForExecutions(executions)

		pipelineExecutionsByID := make(map[string][]*WorkflowExecution)
		for _, exec := range executions {
			if exec == nil || exec.Execution == nil {
				continue
			}
			ref := workflowExecutionRef{
				WorkflowID: exec.Execution.WorkflowID,
				RunID:      exec.Execution.RunID,
			}
			pipelineIdentifier := pipelineIdentifiers[ref]
			if pipelineIdentifier == "" {
				continue
			}
			pipelineRecord := pipelineIdentifierIndex[pipelineIdentifier]
			if pipelineRecord == nil {
				continue
			}
			pipelineExecutionsByID[pipelineRecord.Id] = append(
				pipelineExecutionsByID[pipelineRecord.Id],
				exec,
			)
		}

		if len(pipelineExecutionsByID) == 0 && len(queuedByPipelineID) == 0 {
			return e.JSON(http.StatusOK, map[string][]*pipelineWorkflowSummary{})
		}

		selectedExecutions := selectPipelineExecutionOverview(pipelineExecutionsByID, 5)
		selectedRefs := make([]workflowExecutionRef, 0)
		for _, selected := range selectedExecutions {
			selectedRefs = append(selectedRefs, workflowExecutionRefs(selected)...)
		}
		resultRecords, err := fetchPipelineResultRecords(
			e.App,
			organization.Id,
			selectedRefs,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"database",
				"failed to fetch pipeline results",
				err.Error(),
			)
		}

		response := make(map[string][]*pipelineWorkflowSummary, len(selectedExecutions))
		builder := newPipelineExecutionSummaryBuilder(
			e.App,
			temporalClient,
			authRecord.GetString("Timezone"),
		)

		for pipelineID, executions := range selectedExecutions {
			pipelineRecord := pipelineMap[pipelineID]
			pipelineIdentifier := pipelineIdentifierByID[pipelineID]
			for _, execution := range executions {
				ref, ok := workflowExecutionReference(execution)
				if !ok {
					continue
				}
				summary, err := builder.Build(
					e.Request.Context(),
					pipelineRecord,
					pipelineIdentifier,
					execution,
					nil,
					resultRecords[ref],
				)
				if err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"workflow",
						"failed to build pipeline execution summary",
						err.Error(),
					)
				}
				if summary != nil {
					response[pipelineID] = append(response[pipelineID], summary)
				}
			}
		}

		appendQueuedPipelineSummaries(
			e.App,
			response,
			queuedByPipelineID,
			authRecord.GetString("Timezone"),
			builder.runnerCache,
		)

		return e.JSON(http.StatusOK, response)
	}
}

func selectPipelineExecutionOverview(
	executionsByPipeline map[string][]*WorkflowExecution,
	limit int,
) map[string][]*WorkflowExecution {
	result := make(map[string][]*WorkflowExecution)
	for pipelineID, executions := range executionsByPipeline {
		var running, other []*WorkflowExecution
		for _, execution := range executions {
			if normalizeTemporalStatus(execution.Status) == string(WorkflowStatusRunning) {
				running = append(running, execution)
			} else {
				other = append(other, execution)
			}
		}

		selected := running
		remainingSlots := limit - len(running)
		if remainingSlots > 0 && len(other) > 0 {
			sort.Slice(other, func(i, j int) bool {
				return utils.TimeStringAfter(other[i].StartTime, other[j].StartTime)
			})
			if remainingSlots > len(other) {
				remainingSlots = len(other)
			}
			selected = append(selected, other[:remainingSlots]...)
		}
		sort.Slice(selected, func(i, j int) bool {
			return utils.TimeStringAfter(selected[i].StartTime, selected[j].StartTime)
		})
		if len(selected) > 0 {
			result[pipelineID] = selected
		}
	}

	return result
}
