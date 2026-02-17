// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CredentialDeeplinkRequest struct {
	Yaml string `json:"yaml"`
}

var DeepLinkRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/get-deeplink",
			Handler:       HandleGetDeeplink,
			RequestSchema: CredentialDeeplinkRequest{},
		},
	},
}

// deeplinkTemporalClient resolves Temporal clients for deeplink requests.
var deeplinkTemporalClient = temporalclient.GetTemporalClientWithNamespace

// deeplinkWaitForWorkflowResult allows tests to stub workflow result polling.
var deeplinkWaitForWorkflowResult = workflowengine.WaitForWorkflowResult

// deeplinkStartWorkflow allows tests to stub workflow starts.
var deeplinkStartWorkflow = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
	return workflows.NewCustomCheckWorkflow().Start("default", input)
}

func HandleGetDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body CredentialDeeplinkRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return apis.NewBadRequestError("invalid JSON body", err)
		}

		appURL := e.App.Settings().Meta.AppURL

		memo := map[string]any{
			"test": "get-deeplink",
		}
		ao := &workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute,
			StartToCloseTimeout:    time.Second * 30,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 1.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    1},
		}
		input := workflowengine.WorkflowInput{
			Payload: workflows.CustomCheckWorkflowPayload{
				Yaml: body.Yaml,
			},
			Config: map[string]any{
				"memo":    memo,
				"app_url": appURL,
			},
			ActivityOptions: ao,
		}

		resStart, errStart := deeplinkStartWorkflow(input)
		if errStart != nil {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"failed to start get deeplink check",
				errStart.Error(),
			).JSON(e)
		}
		client, err := deeplinkTemporalClient("default")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to get temporal client",
				err.Error(),
			).JSON(e)
		}
		result, err := deeplinkWaitForWorkflowResult(
			client,
			resStart.WorkflowID,
			resStart.WorkflowRunID,
		)
		if err != nil {
			details := workflowengine.ParseWorkflowError(err)
			return e.JSON(http.StatusInternalServerError, map[string]any{
				"status":  http.StatusInternalServerError,
				"error":   "workflow",
				"reason":  "failed to get workflow result",
				"details": details,
			})
		}

		output, ok := result.Output.([]any)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow output",
				"output is not an array",
			).JSON(e)
		}
		steps, ok := output[0].(map[string]any)["steps"].([]any)
		if !ok || len(steps) == 0 {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow output",
				"steps are not present or empty",
			).JSON(e)
		}

		captures, ok := steps[0].(map[string]any)["captures"].(map[string]any)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow output",
				"captures are not present in step",
			).JSON(e)
		}

		deeplink, ok := captures["deeplink"].(string)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow output",
				"deeplink is not present in captures",
			).JSON(e)
		}

		// Return both the credential offer and the full workflow output
		return e.JSON(http.StatusOK, map[string]any{
			"deeplink": deeplink,
			"steps":    steps,
			"output":   output,
		})
	}
}
