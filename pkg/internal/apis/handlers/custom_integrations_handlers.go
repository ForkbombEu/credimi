// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var CustomIntegrationsRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/custom-integrations",
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/run",
			Handler:       HandleRunCustomIntegration,
			RequestSchema: RunCustomIntegrationRequestInput{},
			OperationID:   "custom-integration.run",
			Description:   "Run a custom integration",
			Summary:       "Run a custom integration",
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

type RunCustomIntegrationRequestInput struct {
	Yaml           string `json:"yaml"                     validate:"required"`
	Data           any    `json:"data"`
	TimeoutSeconds *int   `json:"timeoutSeconds,omitempty"`
}

func HandleRunCustomIntegration() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[RunCustomIntegrationRequestInput](e)
		if err != nil {
			return err
		}

		authRecord := e.Auth
		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			).JSON(e)
		}

		timeout := 30 * time.Second
		if req.TimeoutSeconds != nil {
			timeout = time.Duration(*req.TimeoutSeconds) * time.Second
		}

		appURL := e.App.Settings().Meta.AppURL

		var formJSON string
		if req.Data != nil {
			b, err := json.Marshal(req.Data)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"data",
					"failed to serialize data to JSON",
					err.Error(),
				).JSON(e)
			}
			formJSON = string(b)
		}

		memo := map[string]interface{}{
			"test":   "custom-integration",
			"author": authRecord.Id,
		}

		result, err := processCustomChecks(
			req.Yaml,
			appURL,
			namespace,
			memo,
			formJSON,
			timeout,
		)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"custom-integration",
				"failed to process custom integration",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, workflowengine.WorkflowResult{
			WorkflowID:    result.WorkflowID,
			WorkflowRunID: result.WorkflowRunID,
		})
	}
}

func processCustomChecks(
	testData string,
	appURL string,
	namespace string,
	memo map[string]interface{},
	formJSON string,
	timeout time.Duration, // per-attempt timeout
) (workflowengine.WorkflowResult, error) {
	yaml := testData
	if yaml == "" {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"yaml is empty",
			"yaml is empty",
		)
	}

	const maxAttempts int32 = 5

	totalTimeout := time.Duration(maxAttempts) * timeout

	input := workflowengine.WorkflowInput{
		Payload: workflows.CustomCheckWorkflowPayload{
			Yaml: yaml,
		},
		Config: map[string]any{
			"memo":    memo,
			"app_url": appURL,
			"env":     formJSON,
		},
		ActivityOptions: &workflow.ActivityOptions{
			ScheduleToCloseTimeout: totalTimeout,
			StartToCloseTimeout:    timeout,

			RetryPolicy: &temporal.RetryPolicy{
				BackoffCoefficient: 1.0,
				MaximumAttempts:    maxAttempts,
			},
		},
	}

	w := workflows.NewCustomCheckWorkflow()

	results, errStart := w.Start(namespace, input)
	if errStart != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start check",
			errStart.Error(),
		)
	}

	if authorVal, ok := memo["author"]; ok {
		results.Author = fmt.Sprintf("%v", authorVal)
	}
	return results, nil
}
