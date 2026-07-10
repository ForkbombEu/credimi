// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var fcafWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
	return workflows.NewFCAFAssessmentWorkflow().Start(namespace, input)
}

var fcafTemporalClient = temporalclient.GetTemporalClientWithNamespace

var fcafWorkflowWaitForResult = workflowengine.WaitForWorkflowResult

var FCAFRouteGroup = routing.RouteGroup{
	BaseURL:                "/api/fcaf",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/run",
			Handler:        HandleRunFCAF,
			RequestSchema:  RunFCAFInput{},
			ResponseSchema: RunFCAFResponse{},
			Description:    "Start an FCAF assessment workflow",
		},
	},
}

type RunFCAFInput struct {
	TestIDs           []string       `json:"test_ids,omitempty"`
	Suite             string         `json:"suite,omitempty"`
	CatalogRoot       string         `json:"catalog_root,omitempty"`
	RunnerID          string         `json:"runner_id,omitempty"`
	Runtime           map[string]any `json:"runtime,omitempty"`
	WaitForCompletion bool           `json:"wait_for_completion,omitempty"`
}

type RunFCAFResponse struct {
	WorkflowID string                         `json:"workflow_id,omitempty"`
	RunID      string                         `json:"run_id,omitempty"`
	Report     map[string]any                 `json:"report,omitempty"`
	Summary    map[string]any                 `json:"summary,omitempty"`
	Result     *workflowengine.WorkflowResult `json:"result,omitempty"`
}

func HandleRunFCAF() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[RunFCAFInput](e)
		if err != nil {
			return err
		}

		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			)
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			)
		}

		suite := req.Suite
		if suite == "" {
			suite = "wallet_solution/relying_party"
		}
		runtime := req.Runtime
		if runtime == nil {
			runtime = map[string]any{}
		}
		if req.RunnerID != "" {
			if existing, ok := runtime["runner_id"].(string); ok && existing != "" && existing != req.RunnerID {
				return apierror.New(
					http.StatusBadRequest,
					"runner_id",
					"runner_id conflicts with runtime.runner_id",
					"remove runtime.runner_id or make it match runner_id",
				)
			}
			runtime["runner_id"] = req.RunnerID
		}

		input := workflowengine.WorkflowInput{
			Payload: workflows.FCAFAssessmentWorkflowPayload{
				TestIDs:     req.TestIDs,
				Suite:       suite,
				CatalogRoot: req.CatalogRoot,
				Runtime:     runtime,
				RunnerID:    req.RunnerID,
			},
			Config: map[string]any{
				"app_url": e.App.Settings().Meta.AppURL,
			},
		}

		result, err := fcafWorkflowStart(namespace, input)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"failed to start fcaf workflow",
				err.Error(),
			)
		}

		if !req.WaitForCompletion {
			return e.JSON(http.StatusOK, RunFCAFResponse{
				WorkflowID: result.WorkflowID,
				RunID:      result.WorkflowRunID,
				Result:     &result,
			})
		}

		temporalClient, err := fcafTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to get temporal client",
				err.Error(),
			)
		}
		finalResult, err := fcafWorkflowWaitForResult(
			temporalClient,
			result.WorkflowID,
			result.WorkflowRunID,
		)
		if err != nil {
			failure := workflowengine.ParseWorkflowError(err)
			report := workflowengine.ExtractOutputFromError(err)
			if len(report) == 0 && failure.Details != nil {
				if rawOutput, ok := failure.Details["output"]; ok {
					report, _ = normalizeFCAFReport(rawOutput)
				}
			}
			if len(report) > 0 {
				summary, _ := report["summary"].(map[string]any)
				return e.JSON(http.StatusConflict, RunFCAFResponse{
					WorkflowID: result.WorkflowID,
					RunID:      result.WorkflowRunID,
					Result: &workflowengine.WorkflowResult{
						WorkflowID:    result.WorkflowID,
						WorkflowRunID: result.WorkflowRunID,
						Message:       failure.Summary,
						Errors:        failure,
						Output:        report,
					},
					Report:  report,
					Summary: summary,
				})
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to wait for fcaf workflow result",
				err.Error(),
			)
		}

		report, err := normalizeFCAFReport(finalResult.Output)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to decode fcaf workflow output",
				err.Error(),
			)
		}
		summary, _ := report["summary"].(map[string]any)
		return e.JSON(http.StatusOK, RunFCAFResponse{
			WorkflowID: result.WorkflowID,
			RunID:      result.WorkflowRunID,
			Result:     &finalResult,
			Report:     report,
			Summary:    summary,
		})
	}
}

func normalizeFCAFReport(raw any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}
	if report, ok := raw.(map[string]any); ok {
		return report, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return report, nil
}
