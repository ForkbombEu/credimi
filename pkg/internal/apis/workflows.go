// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"net/http"
	"reflect"

	"github.com/forkbombeu/didimo/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"github.com/forkbombeu/didimo/pkg/internal/apis/handlers"
	"github.com/forkbombeu/didimo/pkg/internal/routing"
)


func DefineComplianceRoutes(app core.App) []routing.RouteDefinition {
	startWorkflowInputType := reflect.TypeOf(handlers.OpenID4VPRequest{})
	saveInputType := reflect.TypeOf(handlers.SaveVariablesAndStartRequestInput{})
	confirmInputType := reflect.TypeOf(handlers.HandleConfirmSuccessRequestInput{})
	failureInputType := reflect.TypeOf(handlers.HandleNotifyFailureRequestInput{})
	sendLogUpdateStartInputType := reflect.TypeOf(handlers.HandleSendLogUpdateStartRequestInput{})
	sendLogUpdateInputType := reflect.TypeOf(handlers.HandleSendLogUpdateRequestInput{})


	return []routing.RouteDefinition{
		{Method: http.MethodPost, Path: "", Handler: handlers.HandleOpenID4VPTest(app), InputType: startWorkflowInputType},
		{Method: http.MethodPost, Path: "/confirm-success", Handler: handlers.HandleConfirmSuccess(app), InputType: confirmInputType},
		{Method: http.MethodPost, Path: "/{protocol}/{author}/save-variables-and-start", Handler: handlers.HandleSaveVariablesAndStart(app), InputType: saveInputType},
		{Method: http.MethodGet, Path: "/{workflowId}/{runId}/history", Handler: handlers.HandleGetWorkflowsHistory(app), InputType: nil},
		{Method: http.MethodPost, Path: "/notify-failure", Handler: handlers.HandleNotifyFailure(app), InputType: failureInputType},
		{Method: http.MethodPost, Path: "/send-log-update-start", Handler: handlers.HandleSendLogUpdateStart(app), InputType: sendLogUpdateStartInputType},
		{Method: http.MethodPost, Path: "/send-log-update", Handler: handlers.HandleSendLogUpdate(app), InputType: sendLogUpdateInputType},
	}
}

func AddComplianceChecks(app core.App) {
	complianceRoutes := DefineComplianceRoutes(app)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		basePath := "/api/compliance/check"

		g := se.Router.Group(basePath)
		g.Bind(apis.RequireAuth())
		g.Bind(&hook.Handler[*core.RequestEvent]{Func: middlewares.ErrorHandlingMiddleware})
		middlewares.RegisterRoutesWithValidation(app, g, complianceRoutes)

		return se.Next()
	})
}
