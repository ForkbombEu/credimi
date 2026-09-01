// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/webpush"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var WebPushRoutes = routing.RouteGroup{
	BaseURL:                "/api/web-push",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/vapid-public-key",
			Handler:        HandleGetWebPushVAPIDPublicKey,
			ResponseSchema: WebPushVAPIDPublicKeyResponse{},
			Description:    "Get the public VAPID key used for web push subscriptions",
		},
		{
			Method:         http.MethodPost,
			Path:           "/pipeline-completed",
			Handler:        HandleWebPushPipelineCompleted,
			RequestSchema:  WebPushPipelineCompletedInput{},
			ResponseSchema: WebPushPipelineCompletedResponse{},
			Description:    "Notify organization members that a pipeline run completed",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
	},
}

type WebPushVAPIDPublicKeyResponse struct {
	PublicKey string `json:"public_key" validate:"required"`
}

type WebPushPipelineCompletedInput struct {
	AppURL       string `json:"app_url"                 validate:"required"`
	WorkflowID   string `json:"workflow_id"             validate:"required"`
	RunID        string `json:"run_id"                  validate:"required"`
	Result       string `json:"result"                  validate:"required"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type WebPushPipelineCompletedResponse struct {
	Sent int `json:"sent"`
}

func HandleGetWebPushVAPIDPublicKey() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		publicKey, _, err := webpush.GetVAPIDKeyPair(e.App)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"web_push",
				"failed to get VAPID public key",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, WebPushVAPIDPublicKeyResponse{PublicKey: publicKey})
	}
}

func HandleWebPushPipelineCompleted() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[WebPushPipelineCompletedInput](e)
		if err != nil {
			return err
		}

		result, apiErr := findPipelineResultByWorkflowRun(e, input.WorkflowID, input.RunID)
		if apiErr != nil {
			return apiErr
		}

		pipeline, err := e.App.FindRecordById("pipelines", result.GetString("pipeline"))
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to lookup pipeline",
				err.Error(),
			)
		}

		organization, err := e.App.FindRecordById("organizations", result.GetString("owner"))
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to lookup organization",
				err.Error(),
			)
		}

		appURL := strings.TrimSpace(input.AppURL)
		if appURL == "" {
			appURL = e.App.Settings().Meta.AppURL
		}
		duration := ""
		if startedAt := result.GetDateTime("created"); !startedAt.IsZero() {
			duration = time.Since(startedAt.Time()).Round(time.Second).String()
		}

		sent, err := webpush.NotifyPipelineRunCompletion(e.App, webpush.CompletionRequest{
			OrgID:        result.GetString("owner"),
			PipelineName: pipeline.GetString("name"),
			Organization: organization.GetString("name"),
			WorkflowID:   input.WorkflowID,
			RunID:        input.RunID,
			Result:       input.Result,
			Duration:     duration,
			Error:        input.ErrorMessage,
			AppURL:       appURL,
		})
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"web_push",
				"failed to send pipeline completion notifications",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, WebPushPipelineCompletedResponse{Sent: sent})
	}
}
