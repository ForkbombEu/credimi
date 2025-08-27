// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
)

var ConformanceRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/compliance",
	Routes: []routing.RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/checks",
			Handler: HandleListMyChecks,
		},
		{
			Method:  http.MethodGet,
			Path:    "/checks/{workflowId}/{runId}",
			Handler: HandleGetWorkflow,
		},
		{
			Method:  http.MethodGet,
			Path:    "/checks/{workflowId}/{runId}/result",
			Handler: HandleGetWorkflowResult,
		},
		{
			Method:  http.MethodGet,
			Path:    "/checks/{workflowId}/{runId}/history",
			Handler: HandleGetWorkflowsHistory,
		},
		{
			Method:        http.MethodPost,
			Path:          "/{protocol}/{version}/save-variables-and-start",
			Handler:       HandleSaveVariablesAndStart,
			RequestSchema: SaveVariablesAndStartRequestInput{},
		},
		{
			Method:        http.MethodPost,
			Path:          "/send-temporal-signal",
			Handler:       HandleSendTemporalSignal,
			RequestSchema: HandleSendTemporalSignalInput{},
		},
		{
			Method:              http.MethodPost,
			Path:                "/send-openidnet-log-update",
			Handler:             HandleSendOpenIDNetLogUpdate,
			RequestSchema:       HandleSendLogUpdateRequestInput{},
			ExcludedMiddlewares: []string{apis.DefaultRequireAuthMiddlewareId},
		},
		{
			Method:              http.MethodPost,
			Path:                "/send-eudiw-log-update",
			Handler:             HandleSendEudiwLogUpdate,
			RequestSchema:       HandleSendLogUpdateRequestInput{},
			ExcludedMiddlewares: []string{apis.DefaultRequireAuthMiddlewareId},
		},
		{
			Method:  http.MethodGet,
			Path:    "/deeplink/{workflowId}/{runId}",
			Handler: HandleDeeplink,
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

func HandleGetWorkflowsHistory() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}

		workflowID := e.Request.PathValue("workflowId")
		if workflowID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflowId",
				"workflowId is required",
				"missing workflowId",
			)
		}
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runId",
				"runId is required",
				"missing runId",
			)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}

		historyIterator := c.GetWorkflowHistory(
			context.Background(),
			workflowID,
			runID,
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		)
		if historyIterator == nil {
			return apierror.New(
				http.StatusNotFound,
				"workflow",
				"workflow history not found",
				"not found",
			)
		}
		var history []map[string]interface{}
		for historyIterator.HasNext() {
			event, err := historyIterator.Next()
			if err != nil {
				notFound := &serviceerror.NotFound{}
				if errors.As(err, &notFound) {
					return apierror.New(
						http.StatusNotFound,
						"workflow",
						"workflow history not found",
						err.Error(),
					)
				}
				invalidArgument := &serviceerror.InvalidArgument{}
				if errors.As(err, &invalidArgument) {
					return apierror.New(
						http.StatusNotFound,
						"workflow",
						"workflow history not found",
						err.Error(),
					)
				}
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to iterate workflow history",
					err.Error(),
				)
			}
			eventData, err := protojson.Marshal(event)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to marshal history event",
					err.Error(),
				)
			}
			var eventMap map[string]interface{}
			if err := json.Unmarshal(eventData, &eventMap); err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to unmarshal history event",
					err.Error(),
				)
			}
			history = append(history, eventMap)
		}

		return e.JSON(http.StatusOK, history)
	}
}

func HandleGetWorkflow() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		workflowID := e.Request.PathValue("workflowId")
		if workflowID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflowId",
				"workflowId is required",
				"missing workflowId",
			)
		}
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runId",
				"runId is required",
				"missing runId",
			)
		}
		authRecord := e.Auth

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}
		workflowExecution, err := c.DescribeWorkflowExecution(
			context.Background(),
			workflowID,
			runID,
		)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow not found",
					err.Error(),
				)
			}
			invalidArgument := &serviceerror.InvalidArgument{}
			if errors.As(err, &invalidArgument) {
				return apierror.New(
					http.StatusBadRequest,
					"workflow",
					"invalid workflow ID",
					err.Error(),
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to describe workflow execution",
				err.Error(),
			)
		}
		weJSON, err := protojson.Marshal(workflowExecution)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow execution",
				err.Error(),
			)
		}
		finalJSON := make(map[string]interface{})
		err = json.Unmarshal(weJSON, &finalJSON)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow execution",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, finalJSON)
	}
}

func HandleGetWorkflowResult() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		workflowID := e.Request.PathValue("workflowId")
		if workflowID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflowId",
				"workflowId is required",
				"missing workflowId",
			)
		}
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runId",
				"runId is required",
				"missing runId",
			)
		}
		authRecord := e.Auth

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}

		// Get the workflow result
		wf := c.GetWorkflow(context.Background(), workflowID, runID)
		var result workflowengine.WorkflowResult

		err = wf.Get(context.Background(), &result)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow not found or not completed",
					err.Error(),
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow result",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, result)
	}
}

type Execution = map[string]any

func sortExecutionsByStartTime(executions []any) []any {
	slices.SortFunc(executions, func(execA, execB any) int {
		execAMap, okA := execA.(Execution)
		execBMap, okB := execB.(Execution)
		if !okA || !okB {
			return 0
		}
		startTimeA, okA := execAMap["startTime"].(string)
		startTimeB, okB := execBMap["startTime"].(string)
		if !okA || !okB {
			return 0
		}
		return strings.Compare(startTimeB, startTimeA)
	})
	return executions
}

//

type HandleNotifyFailureRequestInput struct {
	WorkflowID string `json:"workflow_id" validate:"required"`
	Namespace  string `json:"namespace"   validate:"required"`
	Reason     string `json:"reason"      validate:"required"`
}

type HandleSendLogUpdateRequestInput struct {
	WorkflowID string           `json:"workflow_id"`
	Logs       []map[string]any `json:"logs"`
}

func HandleSendOpenIDNetLogUpdate() func(*core.RequestEvent) error {
	return sendRealtimeLogs(workflows.OpenIDNetSubscription)
}

func HandleSendEudiwLogUpdate() func(*core.RequestEvent) error {
	return sendRealtimeLogs(workflows.EudiwSubscription)
}

type HandleSendTemporalSignalInput struct {
	WorkflowID string `json:"workflow_id" validate:"required"`
	Namespace  string `json:"namespace"   validate:"required"`
	Signal     string `json:"signal"      validate:"required"`
}

func HandleSendTemporalSignal() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleSendTemporalSignalInput](e)
		if err != nil {
			return err
		}
		c, err := temporalclient.GetTemporalClientWithNamespace(req.Namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}
		switch req.Signal {
		case workflows.OpenIDNetStartCheckSignal:
			err = sendOpenIDNetLogUpdateStart(e.App, c, req)
		default:
			err = sendTemporalSignal(c, req)
		}
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Signal sent successfully"})
	}
}

///

func GetUserOrganizationID(app core.App, userID string) (string, error) {
	orgAuthCollection, err := app.FindCollectionByNameOrId("orgAuthorizations")
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"collection",
			"failed to find orgAuthorizations collection",
			err.Error(),
		)
	}

	authOrgRecords, err := app.FindFirstRecordByFilter(
		orgAuthCollection.Id,
		"user={:user}",
		dbx.Params{"user": userID},
	)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"get user namespace",
			"failed to find orgAuthorizations record",
			err.Error(),
		)
	}
	return authOrgRecords.GetString("organization"), nil
}

func sendRealtimeLogs(suiteSubscription string) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleSendLogUpdateRequestInput](e)
		if err != nil {
			return err
		}
		if err := notifyLogsUpdate(e.App, req.WorkflowID+suiteSubscription, req.Logs); err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"failed to send realtime logs update",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Log update sent successfully"})
	}
}

func notifyLogsUpdate(app core.App, subscription string, data []map[string]any) error {
	rawData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	message := subscriptions.Message{
		Name: subscription,
		Data: rawData,
	}
	clients := app.SubscriptionsBroker().Clients()
	for _, client := range clients {
		if client.HasSubscription(subscription) {
			client.Send(message)
		}
	}
	return nil
}

func sendTemporalSignal(c client.Client, input HandleSendTemporalSignalInput) error {
	err := c.SignalWorkflow(context.Background(), input.WorkflowID, "", input.Signal, struct{}{})
	if err != nil {
		notFound := &serviceerror.NotFound{}
		if errors.As(err, &notFound) {
			return apierror.New(http.StatusNotFound, "workflow", "workflow not found", err.Error())
		}
		invalidArgument := &serviceerror.InvalidArgument{}
		if errors.As(err, &invalidArgument) {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"invalid workflow ID",
				err.Error(),
			)
		}

		return apierror.New(
			http.StatusBadRequest,
			"signal",
			fmt.Sprintf("failed to send signal: %s", input.Signal),
			err.Error())
	}
	return nil
}

func sendOpenIDNetLogUpdateStart(
	app core.App,
	c client.Client,
	input HandleSendTemporalSignalInput,
) error {
	err := c.SignalWorkflow(
		context.Background(),
		input.WorkflowID,
		"",
		workflows.OpenIDNetStartCheckSignal,
		struct{}{},
	)
	if err != nil {
		canceledErr := &serviceerror.Canceled{}
		notFound := &serviceerror.NotFound{}
		if errors.As(err, &canceledErr) ||
			(errors.As(err, &notFound) && err.Error() == "workflow execution already completed") {
			wf := c.GetWorkflow(context.Background(), input.WorkflowID, "")
			var result workflowengine.WorkflowResult

			err := wf.Get(context.Background(), &result)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"workflow",
					"failed to get logs workflow result",
					err.Error(),
				)
			}

			if logsInterface, ok := result.Log.([]any); ok {
				logs := workflows.AsSliceOfMaps(logsInterface)
				id := strings.TrimSuffix(input.WorkflowID, "-log")
				if err := notifyLogsUpdate(app, id+workflows.OpenIDNetSubscription, logs); err != nil {
					return apierror.New(
						http.StatusBadRequest,
						"workflow",
						"failed to send realtime logs update",
						err.Error(),
					)
				}
			} else {
				return apierror.New(http.StatusBadRequest, "workflow", "invalid log format", "logs are not in the expected format")
			}
		}

		if errors.As(err, &notFound) {
			return apierror.New(http.StatusNotFound, "workflow", "workflow not found", err.Error())
		}
		invalidArgument := &serviceerror.InvalidArgument{}
		if errors.As(err, &invalidArgument) {
			return apierror.New(
				http.StatusBadRequest,
				"workflow",
				"invalid workflow ID",
				err.Error(),
			)
		}

		return apierror.New(
			http.StatusBadRequest,
			"signal",
			"failed to send start logs update signal",
			err.Error(),
		)
	}
	return nil
}
func HandleDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		workflowID := e.Request.PathValue("workflowId")
		if workflowID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflowId",
				"workflowId is required",
				"missing workflowId",
			)
		}
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runId",
				"runId is required",
				"missing runId",
			)
		}

		namespace, err := GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return err
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			)
		}
		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}

		author, err := getWorkflowAuthor(c, workflowID, runID)
		if err != nil {
			return err
		}

		return handleDeeplinkFromHistory(e, c, workflowID, runID, author)
	}
}

func getWorkflowAuthor(c client.Client, workflowID, runID string) (string, error) {
	workflowExecution, err := c.DescribeWorkflowExecution(
		context.Background(),
		workflowID,
		runID,
	)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to describe workflow execution",
			err.Error(),
		)
	}
	weJSON, err := protojson.Marshal(workflowExecution)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to marshal workflow execution",
			err.Error(),
		)
	}
	var weMap map[string]any
	if err := json.Unmarshal(weJSON, &weMap); err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to unmarshal workflow execution",
			err.Error(),
		)
	}
	author := ""
	if workflowExecutionInfo, ok := weMap["workflowExecutionInfo"].(map[string]any); ok {
		if memo, ok := workflowExecutionInfo["memo"]; ok {
			if fields, ok := memo.(map[string]any)["fields"]; ok {
				if protoVal, ok := fields.(map[string]any)["author"]; ok {
					if protoMap, ok := protoVal.(map[string]any); ok {
						if protoData, ok := protoMap["data"].(string); ok {
							decoded, err := base64.StdEncoding.DecodeString(protoData)
							if err == nil {
								unquoted, err := strconv.Unquote(string(decoded))
								if err == nil {
									author = unquoted
								} else {
									author = string(decoded)
								}
							}
						}
					}
				}
			}
		}
	}
	return author, nil
}

func handleDeeplinkFromHistory(
	e *core.RequestEvent,
	c client.Client,
	workflowID, runID, author string,
) error {
	historyIterator := c.GetWorkflowHistory(
		context.Background(),
		workflowID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	if historyIterator == nil {
		return apierror.New(
			http.StatusNotFound,
			"workflow",
			"workflow history not found",
			"not found",
		)
	}

	for historyIterator.HasNext() {
		event, err := historyIterator.Next()
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to iterate workflow history",
				err.Error(),
			)
		}
		eventData, err := protojson.Marshal(event)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal history event",
				err.Error(),
			)
		}
		var eventMap map[string]any
		if err := json.Unmarshal(eventData, &eventMap); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal history event",
				err.Error(),
			)
		}

		if attrRaw, ok := eventMap["activityTaskCompletedEventAttributes"]; ok {
			attr := attrRaw.(map[string]any)
			if resultRaw, ok := attr["result"]; ok {
				resultMap := resultRaw.(map[string]any)
				if pls, ok := resultMap["payloads"].([]any); ok && len(pls) > 0 {
					first := pls[0].(map[string]any)
					switch author {
					case "openid_conformance_suite":
						return getDeeplinkOpenIDConformanceSuite(e, first)
					case "ewc":
						return getDeeplinkEWC(e, first)
					case "eudiw":
						return getDeeplinkEudiw(e, first)
					default:
						return apierror.New(
							http.StatusBadRequest,
							"protocol",
							"unsupported suite",
							fmt.Sprintf(
								"author is %q, expected openid_conformance_suite, ewc or eudiw",
								author,
							),
						)
					}
				}
			}
		}
	}

	return apierror.New(
		http.StatusNotFound,
		"deeplink",
		"no matching activity found",
		"no activity that have as result a deeplink found for this workflow run",
	)
}

func getDeeplinkOpenIDConformanceSuite(e *core.RequestEvent, first map[string]any) error {
	if dataB64, ok := first["data"].(string); ok {
		decoded, _ := base64.StdEncoding.DecodeString(dataB64)
		var out struct {
			Output struct {
				Captures struct {
					Result any `json:"result"`
				} `json:"captures"`
			} `json:"Output"`
		}
		json.Unmarshal(decoded, &out)
		return e.JSON(http.StatusOK, map[string]any{
			"deeplink": out.Output.Captures.Result,
		})
	}
	return nil
}

func getDeeplinkEWC(e *core.RequestEvent, first map[string]any) error {
	if dataB64, ok := first["data"].(string); ok {
		decoded, _ := base64.StdEncoding.DecodeString(dataB64)
		var out struct {
			Output struct {
				Captures struct {
					Deeplink string `json:"deep_link"`
				} `json:"captures"`
			} `json:"Output"`
		}
		json.Unmarshal(decoded, &out)
		return e.JSON(http.StatusOK, map[string]any{
			"deeplink": out.Output.Captures.Deeplink,
		})
	}
	return nil
}

func getDeeplinkEudiw(e *core.RequestEvent, first map[string]any) error {
	if dataB64, ok := first["data"].(string); ok {
		decoded, _ := base64.StdEncoding.DecodeString(dataB64)
		var out struct {
			Output struct {
				Captures struct {
					ClientID   string `json:"client_id"`
					RequestURI string `json:"request_uri"`
				} `json:"captures"`
			} `json:"Output"`
		}
		json.Unmarshal(decoded, &out)
		deeplink, err := workflows.BuildQRDeepLink(
			out.Output.Captures.ClientID,
			out.Output.Captures.RequestURI,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"deeplink",
				"failed to build QR deep link",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, map[string]any{
			"deeplink": deeplink,
		})
	}
	return nil
}
