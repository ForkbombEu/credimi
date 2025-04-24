// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"net/http"

	"github.com/forkbombeu/didimo/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"github.com/forkbombeu/didimo/pkg/internal/apis/handlers"
	"github.com/forkbombeu/didimo/pkg/internal/routing"
)

func AddComplianceChecks(app core.App) {
	routing.AddGroupRoutes(app, routing.RouteGroup{
		BaseUrl: "/api/compliance/check",
		Routes: []routing.RouteDefinition{
			{Method: http.MethodPost, Path: "", Handler: handlers.HandleOpenID4VPTest, Input: handlers.OpenID4VPRequest{}},
			{Method: http.MethodPost, Path: "/confirm-success", Handler: handlers.HandleConfirmSuccess, Input: handlers.HandleConfirmSuccessRequestInput{}},
			{Method: http.MethodPost, Path: "/{protocol}/{author}/save-variables-and-start", Handler: handlers.HandleSaveVariablesAndStart, Input: handlers.SaveVariablesAndStartRequestInput{}},
			{Method: http.MethodGet, Path: "/checks/{workflowId}/{runId}/history", Handler: handlers.HandleGetWorkflowsHistory, Input: nil},
			{Method: http.MethodGet, Path: "/checks/{workflowId}/{runId}", Handler: handlers.HandleGetWorkflow, Input: nil},
			{Method: http.MethodGet, Path: "/checks", Handler: handlers.HandleGetWorkflows, Input: nil},
			{Method: http.MethodPost, Path: "/notify-failure", Handler: handlers.HandleNotifyFailure, Input: handlers.HandleNotifyFailureRequestInput{}},
			{Method: http.MethodPost, Path: "/send-log-update-start", Handler: handlers.HandleSendLogUpdateStart, Input: handlers.HandleSendLogUpdateStartRequestInput{}},
			{Method: http.MethodPost, Path: "/send-log-update", Handler: handlers.HandleSendLogUpdate, Input: handlers.HandleSendLogUpdateRequestInput{}},
		},
		Middlewares: []*hook.Handler[*core.RequestEvent]{apis.RequireAuth(), {Func: middlewares.ErrorHandlingMiddleware}},
		Validation:  true,
	})
}
