// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
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
	"gopkg.in/yaml.v3"
)

type CredentialDeeplinkRequest struct {
	Yaml    string `json:"yaml"`
	Secrets string `json:"secrets,omitempty"`
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
		{
			Method:  http.MethodGet,
			Path:    "/credential/deeplink",
			Handler: HandleGetCredentialDeeplink,
		},
		{
			Method:  http.MethodGet,
			Path:    "/verification/deeplink",
			Handler: HandleVerificationDeeplink,
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

type deeplinkWorkflowResponse struct {
	Deeplink string
	Steps    []any
	Output   []any
}

func getDeeplinkFromYAML(
	app core.App,
	yaml string,
	secrets map[string]string,
) (deeplinkWorkflowResponse, error) {
	appURL := app.Settings().Meta.AppURL

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
			MaximumAttempts:    1,
		},
	}
	input := workflowengine.WorkflowInput{
		Payload: workflows.CustomCheckWorkflowPayload{
			Yaml:    yaml,
			Secrets: secrets,
		},
		Config: map[string]any{
			"memo":    memo,
			"app_url": appURL,
		},
		ActivityOptions: ao,
	}

	resStart, errStart := deeplinkStartWorkflow(input)
	if errStart != nil {
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start get deeplink check",
			errStart.Error(),
		)
	}
	client, err := deeplinkTemporalClient("default")
	if err != nil {
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusInternalServerError,
			"temporal",
			"failed to get temporal client",
			err.Error(),
		)
	}
	result, err := deeplinkWaitForWorkflowResult(
		client,
		resStart.WorkflowID,
		resStart.WorkflowRunID,
	)
	if err != nil {
		details := workflowengine.ParseWorkflowError(err)
		message := details.Message
		if message == "" {
			message = err.Error()
		}
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to get workflow result",
			message,
		)
	}

	output, ok := result.Output.([]any)
	if !ok {
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to get workflow output",
			"output is not an array",
		)
	}
	if len(output) == 0 {
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to get workflow output",
			"output is empty",
		)
	}
	steps, ok := output[0].(map[string]any)["steps"].([]any)
	if !ok || len(steps) == 0 {
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to get workflow output",
			"steps are not present or empty",
		)
	}

	captures, ok := steps[0].(map[string]any)["captures"].(map[string]any)
	if !ok {
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to get workflow output",
			"captures are not present in step",
		)
	}

	deeplink, ok := captures["deeplink"].(string)
	if !ok || deeplink == "" {
		return deeplinkWorkflowResponse{}, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to get workflow output",
			"deeplink is not present in captures",
		)
	}

	return deeplinkWorkflowResponse{
		Deeplink: deeplink,
		Steps:    steps,
		Output:   output,
	}, nil
}

func HandleGetDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body CredentialDeeplinkRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return apis.NewBadRequestError("invalid JSON body", err)
		}

		secrets, apiErr := parseSecretsYAML(body.Secrets)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		response, err := getDeeplinkFromYAML(e.App, body.Yaml, secrets)
		if err != nil {
			if apiErr, ok := err.(*apierror.APIError); ok {
				return apiErr.JSON(e)
			}

			return err
		}

		return e.JSON(http.StatusOK, map[string]any{
			"deeplink": response.Deeplink,
			"steps":    response.Steps,
			"output":   response.Output,
		})
	}
}

func HandleGetCredentialDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return handleRecordDeeplink(e, recordDeeplinkOptions{
			MissingIDReason:    "missing credential id",
			ResolveReason:      "failed to resolve credential path",
			MissingDomain:      "credential",
			ExpectedCollection: "credentials",
		})
	}
}

func HandleVerificationDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return handleRecordDeeplink(e, recordDeeplinkOptions{
			MissingIDReason:    "missing record id",
			ResolveReason:      "failed to resolve verification path",
			MissingDomain:      "use-case-verification",
			ExpectedCollection: "use_cases_verifications",
		})
	}
}

type recordDeeplinkOptions struct {
	MissingIDReason    string
	ResolveReason      string
	MissingDomain      string
	ExpectedCollection string
}

func handleRecordDeeplink(e *core.RequestEvent, opts recordDeeplinkOptions) error {
	id := e.Request.URL.Query().Get("id")
	if id == "" {
		return apierror.New(
			http.StatusBadRequest,
			"request",
			opts.MissingIDReason,
			"id parameter is required",
		).JSON(e)
	}

	redirect := e.Request.URL.Query().Get("redirect") == RedirectFlagTrue

	rec, err := canonify.Resolve(e.App, id)
	if err != nil {
		return apierror.New(
			http.StatusNotFound,
			"resolve",
			opts.ResolveReason,
			err.Error(),
		).JSON(e)
	}
	if rec.Collection() == nil || rec.Collection().Name != opts.ExpectedCollection {
		return apierror.New(
			http.StatusBadRequest,
			"record",
			"invalid record type",
			"id must resolve to a "+opts.ExpectedCollection+" record",
		).JSON(e)
	}

	deeplink, apiErr := deeplinkFromRecord(e.App, rec, opts.MissingDomain)
	if apiErr != nil {
		return apiErr.JSON(e)
	}

	if redirect {
		e.Response.Header().Set("Location", deeplink)
		e.Response.WriteHeader(http.StatusMovedPermanently)
		return e.Next()
	}

	return e.String(http.StatusOK, deeplink)
}

func deeplinkFromRecord(app core.App, rec *core.Record, missingDomain string) (string, *apierror.APIError) {
	yamlStr := rec.GetString("yaml")
	if yamlStr == "" {
		deeplink := rec.GetString("deeplink")
		if deeplink != "" {
			return deeplink, nil
		}

		return "", apierror.New(
			http.StatusInternalServerError,
			missingDomain,
			"deeplink not found",
			"field 'deeplink' is missing or empty",
		)
	}

	secrets, apiErr := parseSecretsYAML(rec.GetString("secrets"))
	if apiErr != nil {
		return "", apiErr
	}

	response, err := getDeeplinkFromYAML(app, yamlStr, secrets)
	if err != nil {
		if apiErr, ok := err.(*apierror.APIError); ok {
			return "", apiErr
		}

		return "", apierror.New(
			http.StatusInternalServerError,
			"deeplink",
			"failed to resolve deeplink",
			err.Error(),
		)
	}

	return response.Deeplink, nil
}

func parseSecretsYAML(secretsYAML string) (map[string]string, *apierror.APIError) {
	if secretsYAML == "" {
		return nil, nil
	}

	var secrets map[string]string
	if err := yaml.Unmarshal([]byte(secretsYAML), &secrets); err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"secrets",
			"invalid secrets yaml",
			err.Error(),
		)
	}

	return secrets, nil
}
