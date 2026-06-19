// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	InternalPipeline "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	pip "github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const httpRequestStepUse = "http-request"
const pipelineExecuteWorkflowIDPrefix = "Pipeline-Execute-"

var PipelineExecuteTimeout = 2 * time.Minute

type PipelineExecuteResponse struct {
	WorkflowID string `json:"workflow_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	Result     any    `json:"result,omitempty"`
	Deeplink   string `json:"deeplink,omitempty"`
}

func HandlePipelineExecute() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// 1. Read request body
		bodyBytes, err := io.ReadAll(e.Request.Body)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"body",
				"failed to read request body",
				err.Error(),
			)
		}
		defer e.Request.Body.Close()

		yamlContent := string(bodyBytes)
		if strings.TrimSpace(yamlContent) == "" {
			return apierror.New(
				http.StatusBadRequest,
				"body",
				"request body cannot be empty",
				"empty body",
			)
		}

		// 2. Query parameters
		deeplinkFlag := e.Request.URL.Query().Get("deeplink") == "true"
		redirectFlag := e.Request.URL.Query().Get("redirect") == "true"

		if redirectFlag && !deeplinkFlag {
			return apierror.New(
				http.StatusBadRequest,
				"redirect",
				"redirect=true requires deeplink=true",
				"redirect without deeplink",
			)
		}

		// 3. Parse pipeline
		wfDef, err := InternalPipeline.ParseWorkflow(yamlContent)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"yaml",
				"failed to parse pipeline YAML",
				err.Error(),
			)
		}

		// 4. Validate pipeline steps
		for _, step := range wfDef.Steps {
			if step.Use != httpRequestStepUse {
				return apierror.New(
					http.StatusBadRequest,
					"yaml",
					fmt.Sprintf("pipeline contains invalid step type '%s'. Only 'http-request' steps are allowed", step.Use),
					"",
				)
			}
		}
		// 5. Determine Temporal namespace based on user organization (or default)
		namespace := "default"
		if e.Auth != nil {
			organization, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
			if err == nil {
				namespace = organization.GetString("canonified_name")
			}
		}

		// 6. Prepare input per il workflow
		workflowInput := pip.PipelineWorkflowInput{
			WorkflowDefinition: wfDef,
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{},
				ActivityOptions: &workflow.ActivityOptions{
					StartToCloseTimeout: PipelineExecuteTimeout,
				},
			},
		}

		// 7. start workflow execution
		// This endpoint executes ephemeral YAML directly, not a stored pipeline record.
		// Do not attach PipelineIdentifier search attributes or create pipeline_results.
		workflowOptions := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("%s%s-%s", pipelineExecuteWorkflowIDPrefix, canonify.CanonifyPlain(wfDef.Name), uuid.NewString()),
			TaskQueue: pip.PipelineTaskQueue,
		}

		temporalClient, err := pipelineTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				fmt.Sprintf("unable to get temporal client for namespace '%s': %v", namespace, err),
				err.Error(),
			)
		}

		ctx, cancel := context.WithTimeout(context.Background(), PipelineExecuteTimeout)
		defer cancel()

		we, err := temporalClient.ExecuteWorkflow(ctx, workflowOptions, "Dynamic Pipeline Workflow", workflowInput)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				fmt.Sprintf("failed to start workflow in namespace '%s': %v", namespace, err),
				err.Error(),
			)
		}
		e.App.Logger().Info("workflow started",
			"namespace", namespace,
			"workflowID", we.GetID(),
			"runID", we.GetRunID(),
		)

		// 8. Wait for workflow completion and get result
		var result workflowengine.WorkflowResult
		if err := we.Get(ctx, &result); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				fmt.Sprintf("workflow execution failed in namespace '%s': %v", namespace, err),
				err.Error(),
			)
		}

		// 9. Return response
		if !deeplinkFlag {
			return e.JSON(http.StatusOK, PipelineExecuteResponse{
				WorkflowID: result.WorkflowID,
				RunID:      result.WorkflowRunID,
				Result:     result.Output,
			})
		}

		deeplink, err := extractDeeplink(result.Output, wfDef.Steps)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"deeplink",
				err.Error(),
				"",
			)
		}

		if redirectFlag {
			return e.Redirect(http.StatusFound, deeplink)
		}

		return e.String(http.StatusOK, deeplink)
	}
}

func extractDeeplink(output any, steps []InternalPipeline.StepDefinition) (string, error) {
	outputMap, ok := output.(map[string]any)
	if !ok {
		return "", fmt.Errorf("output is not a map")
	}

	for i := len(steps) - 1; i >= 0; i-- {
		stepData, ok := outputMap[steps[i].ID].(map[string]any)
		if !ok {
			continue
		}

		outputs, ok := stepData["outputs"].(map[string]any)
		if !ok {
			continue
		}

		if deeplink, ok := outputs["deeplink"].(string); ok && deeplink != "" {
			return deeplink, nil
		}
	}

	return "", fmt.Errorf("deeplink not found in any step output")
}
