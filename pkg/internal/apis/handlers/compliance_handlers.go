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
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
)

var ConformanceRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/compliance",
	Routes: []routing.RouteDefinition{
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
			Handler:             HandleSendOpenID4VPWalletLogUpdate,
			RequestSchema:       HandleSendLogUpdateRequestInput{},
			ExcludedMiddlewares: []string{middlewares.RequireAuthOrAPIKeyMiddlewareID},
		},
		{
			Method:              http.MethodPost,
			Path:                "/send-eudiw-log-update",
			Handler:             HandleSendEudiwLogUpdate,
			RequestSchema:       HandleSendLogUpdateRequestInput{},
			ExcludedMiddlewares: []string{middlewares.RequireAuthOrAPIKeyMiddlewareID},
		},
		{
			Method:              http.MethodPost,
			Path:                "/send-ewc-log-update",
			Handler:             HandleSendEWCLogUpdate,
			RequestSchema:       HandleSendLogUpdateRequestInput{},
			ExcludedMiddlewares: []string{middlewares.RequireAuthOrAPIKeyMiddlewareID},
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

// complianceTemporalClient resolves Temporal clients for compliance handlers.
var complianceTemporalClient = temporalclient.GetTemporalClientWithNamespace

// complianceNotifyLogsUpdate allows tests to bypass realtime notifications.
var complianceNotifyLogsUpdate = notifyLogsUpdate

func HandleGetWorkflowsHistory() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth

		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			)
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

		history, err := getWorkflowHistoryWithFallback(namespace, workflowID, runID)
		if err != nil {
			var notFound *serviceerror.NotFound
			var invalidArg *serviceerror.InvalidArgument
			switch {
			case errors.As(err, &notFound), errors.As(err, &invalidArg):
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow history not found",
					err.Error(),
				)
			default:
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to get workflow history",
					err.Error(),
				)
			}
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
		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
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

		exec, err := getWorkflowExecutionWithFallback(namespace, workflowID, runID)
		if err != nil {
			var notFound *serviceerror.NotFound
			var invalidArg *serviceerror.InvalidArgument
			switch {
			case errors.As(err, &notFound):
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow not found",
					err.Error(),
				)
			case errors.As(err, &invalidArg):
				return apierror.New(
					http.StatusBadRequest,
					"workflow",
					"invalid workflow ID",
					err.Error(),
				)
			default:
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to describe workflow execution",
					err.Error(),
				)
			}
		}

		finalJSON, err := marshalWorkflowExecution(exec)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to process workflow execution",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, finalJSON)
	}
}

func getWorkflowExecutionWithFallback(
	namespace, workflowID, runID string,
) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	exec, err := fetchWorkflowExecution(namespace, workflowID, runID)
	if err == nil && exec != nil {
		return exec, nil
	}

	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return fetchWorkflowExecution("default", workflowID, runID)
	}

	return nil, err
}

func fetchWorkflowExecution(
	namespace, workflowID, runID string,
) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	c, err := complianceTemporalClient(namespace)
	if err != nil {
		return nil, fmt.Errorf("get client for namespace %q: %w", namespace, err)
	}

	exec, err := c.DescribeWorkflowExecution(context.Background(), workflowID, runID)
	if err != nil {
		return nil, err
	}

	return exec, nil
}

func marshalWorkflowExecution(
	exec *workflowservice.DescribeWorkflowExecutionResponse,
) (map[string]any, error) {
	data, err := protojson.Marshal(exec)
	if err != nil {
		return nil, fmt.Errorf("marshal execution: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal execution: %w", err)
	}

	return result, nil
}

func getWorkflowHistoryWithFallback(
	namespace, workflowID, runID string,
) ([]map[string]interface{}, error) {
	history, err := fetchWorkflowHistory(namespace, workflowID, runID)
	if err == nil && len(history) > 0 {
		return history, nil
	}

	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return fetchWorkflowHistory("default", workflowID, runID)
	}

	return nil, err
}

func fetchWorkflowHistory(namespace, workflowID, runID string) ([]map[string]interface{}, error) {
	c, err := complianceTemporalClient(namespace)
	if err != nil {
		return nil, fmt.Errorf("get client for namespace %q: %w", namespace, err)
	}

	iter := c.GetWorkflowHistory(
		context.Background(),
		workflowID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	if iter == nil {
		return nil, fmt.Errorf("no history iterator for namespace %s", namespace)
	}

	var history []map[string]interface{}
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, err
		}
		eventData, err := protojson.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("marshal event: %w", err)
		}
		var eventMap map[string]interface{}
		if err := json.Unmarshal(eventData, &eventMap); err != nil {
			return nil, fmt.Errorf("unmarshal event: %w", err)
		}
		history = append(history, eventMap)
	}

	return history, nil
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

		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
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

		c, err := complianceTemporalClient(namespace)
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

func HandleSendOpenID4VPWalletLogUpdate() func(*core.RequestEvent) error {
	return sendRealtimeLogs(workflows.OpenID4VPWalletSubscription)
}

func HandleSendEudiwLogUpdate() func(*core.RequestEvent) error {
	return sendRealtimeLogs(workflows.EudiwSubscription)
}

func HandleSendEWCLogUpdate() func(*core.RequestEvent) error {
	return sendRealtimeLogs(workflows.EWCSubscription)
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
		c, err := complianceTemporalClient(req.Namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}
		switch req.Signal {
		case workflows.OpenID4VPWalletStartCheckSignal:
			err = sendOpenID4VPWalletLogUpdateStart(e.App, c, req)
		case workflows.OpenID4VCIIssuerStartCheckSignal:
			err = sendOpenIDNetConformanceLogUpdateStart(e.App, c, req)
		case workflows.OpenID4VPVerifierStartCheckSignal:
			err = sendOpenIDNetConformanceLogUpdateStart(e.App, c, req)
		case workflows.OpenID4VCIIssuerStopCheckSignal, workflows.OpenID4VPVerifierStopCheckSignal:
			err = nil
		case workflows.EwcStartCheckSignal:
			err = sendEWCLikeLogUpdateStart(e.App, c, req)
		case workflows.EwcStopCheckSignal:
			err = sendEWCLikeLogUpdateStop(c, req)
		default:
			err = sendTemporalSignal(c, req)
		}
		if err != nil {
			apiErr := &apierror.APIError{}
			if errors.As(err, &apiErr) {
				return apiErr
			}

			return err
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Signal sent successfully"})
	}
}

///

func sendRealtimeLogs(suiteSubscription string) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleSendLogUpdateRequestInput](e)
		if err != nil {
			return err
		}
		if err := complianceNotifyLogsUpdate(
			e.App,
			req.WorkflowID+suiteSubscription,
			req.Logs,
		); err != nil {
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
			err.Error(),
		)
	}
	return nil
}

func sendOpenID4VPWalletLogUpdateStart(
	app core.App,
	c client.Client,
	input HandleSendTemporalSignalInput,
) error {
	err := c.SignalWorkflow(
		context.Background(),
		input.WorkflowID,
		"",
		workflows.OpenID4VPWalletStartCheckSignal,
		struct{}{},
	)
	if err != nil {
		canceledErr := &serviceerror.Canceled{}
		notFound := &serviceerror.NotFound{}
		if errors.As(err, &canceledErr) ||
			(errors.As(err, &notFound) && err.Error() == "workflow execution already completed") {
			return sendCompletedWorkflowLogsUpdate(
				app,
				c,
				input.WorkflowID,
				strings.TrimSuffix(input.WorkflowID, "-log")+workflows.OpenID4VPWalletSubscription,
				extractOpenIDNetLogsFromResultOrError,
			)
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

func sendOpenIDNetConformanceLogUpdateStart(
	app core.App,
	c client.Client,
	input HandleSendTemporalSignalInput,
) error {
	exec, err := c.DescribeWorkflowExecution(context.Background(), input.WorkflowID, "")
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
			"workflow",
			"failed to describe OpenIDNet workflow",
			err.Error(),
		)
	}

	if exec.GetWorkflowExecutionInfo().GetStatus() == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return nil
	}

	return sendCompletedWorkflowLogsUpdate(
		app,
		c,
		input.WorkflowID,
		input.WorkflowID+workflows.OpenID4VPWalletSubscription,
		extractOpenIDNetLogsFromResultOrError,
	)
}

func sendEWCLikeLogUpdateStart(
	app core.App,
	c client.Client,
	input HandleSendTemporalSignalInput,
) error {
	var lastErr error
	for _, workflowID := range ewcLikeWorkflowIDs(input.WorkflowID) {
		err := c.SignalWorkflow(
			context.Background(),
			workflowID,
			"",
			workflows.EwcStartCheckSignal,
			struct{}{},
		)
		if err == nil {
			return nil
		}
		lastErr = err

		canceledErr := &serviceerror.Canceled{}
		if errors.As(err, &canceledErr) || isWorkflowExecutionAlreadyCompleted(err) {
			return sendCompletedWorkflowLogsUpdate(
				app,
				c,
				workflowID,
				ewcLikeLogsSubscription(workflowID),
				extractEWCLikeLogsFromResultOrError,
			)
		}

		notFound := &serviceerror.NotFound{}
		if errors.As(err, &notFound) {
			continue
		}
		return ewcLikeSignalError(err, workflows.EwcStartCheckSignal)
	}

	return apierror.New(http.StatusNotFound, "workflow", "workflow not found", lastErr.Error())
}

func sendEWCLikeLogUpdateStop(c client.Client, input HandleSendTemporalSignalInput) error {
	var lastErr error
	for _, workflowID := range ewcLikeWorkflowIDs(input.WorkflowID) {
		err := c.SignalWorkflow(
			context.Background(),
			workflowID,
			"",
			workflows.EwcStopCheckSignal,
			struct{}{},
		)
		if err == nil || isWorkflowExecutionAlreadyCompleted(err) {
			return nil
		}
		lastErr = err

		notFound := &serviceerror.NotFound{}
		if errors.As(err, &notFound) {
			continue
		}
		return ewcLikeSignalError(err, workflows.EwcStopCheckSignal)
	}

	return apierror.New(http.StatusNotFound, "workflow", "workflow not found", lastErr.Error())
}

func ewcLikeWorkflowIDs(workflowID string) []string {
	if strings.HasSuffix(workflowID, "-status") {
		return []string{workflowID, strings.TrimSuffix(workflowID, "-status")}
	}
	return []string{workflowID}
}

func ewcLikeLogsSubscription(workflowID string) string {
	return strings.TrimSuffix(workflowID, "-status") + workflows.EWCSubscription
}

func isWorkflowExecutionAlreadyCompleted(err error) bool {
	notFound := &serviceerror.NotFound{}
	return errors.As(err, &notFound) && err.Error() == "workflow execution already completed"
}

func ewcLikeSignalError(err error, signal string) error {
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
		fmt.Sprintf("failed to send signal: %s", signal),
		err.Error(),
	)
}

type workflowLogsExtractor func(workflowengine.WorkflowResult, error) []map[string]any

func sendCompletedWorkflowLogsUpdate(
	app core.App,
	c client.Client,
	workflowID string,
	subscription string,
	extractLogs workflowLogsExtractor,
) error {
	wf := c.GetWorkflow(context.Background(), workflowID, "")
	var result workflowengine.WorkflowResult
	getErr := wf.Get(context.Background(), &result)
	logs := extractLogs(result, getErr)
	if len(logs) == 0 {
		return nil
	}

	if err := complianceNotifyLogsUpdate(app, subscription, logs); err != nil {
		return apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to send realtime logs update",
			err.Error(),
		)
	}

	return nil
}

func extractOpenIDNetLogsFromResultOrError(
	result workflowengine.WorkflowResult,
	err error,
) []map[string]any {
	if logs := extractOpenIDNetLogsFromPayload(result.Log); len(logs) > 0 {
		return logs
	}
	if logs := extractOpenIDNetLogsFromPayload(result.Output); len(logs) > 0 {
		return logs
	}
	if err != nil {
		return extractOpenIDNetLogsFromWorkflowError(err)
	}
	return nil
}

func extractEWCLikeLogsFromWorkflowError(err error) []map[string]any {
	for current := err; current != nil; current = errors.Unwrap(current) {
		details := workflowengine.ParseWorkflowError(current)
		if logs := extractEWCLikeLogsFromPayload(
			workflowErrorDetailsPayload(details),
		); len(
			logs,
		) > 0 {
			return logs
		}
	}
	return nil
}

func extractEWCLikeLogsFromResultOrError(
	result workflowengine.WorkflowResult,
	err error,
) []map[string]any {
	if logs := extractEWCLikeLogsFromPayload(result.Log); len(logs) > 0 {
		return logs
	}
	if logs := extractEWCLikeLogsFromPayload(result.Output); len(logs) > 0 {
		return logs
	}
	if err != nil {
		return extractEWCLikeLogsFromWorkflowError(err)
	}
	return nil
}

func workflowErrorDetailsPayload(details workflowengine.WorkflowError) any {
	if details.Details == nil {
		return nil
	}
	if payload, ok := details.Details["payload"]; ok {
		return payload
	}
	return details.Details
}

func extractEWCLikeLogsFromPayload(payload any) []map[string]any {
	if logs := workflowengine.AsSliceOfMaps(payload); len(logs) > 0 && looksLikeEWCLikeLogs(logs) {
		return logs
	}

	if payloads, ok := payload.([]any); ok {
		for _, item := range payloads {
			if logs := extractEWCLikeLogsFromPayload(item); len(logs) > 0 {
				return logs
			}
		}
	}

	if payloadMap := workflowengine.AsMap(payload); payloadMap != nil {
		if nestedPayload, ok := payloadMap["payload"]; ok {
			if logs := extractEWCLikeLogsFromPayload(nestedPayload); len(logs) > 0 {
				return logs
			}
		}
		if logs := workflowengine.AsSliceOfMaps(
			payloadMap["logs"],
		); len(logs) > 0 &&
			looksLikeEWCLikeLogs(logs) {
			return logs
		}
		if logsResponse := workflowengine.AsMap(payloadMap["logs_response"]); logsResponse != nil {
			if logs := workflowengine.AsSliceOfMaps(
				logsResponse["logs"],
			); len(logs) > 0 &&
				looksLikeEWCLikeLogs(logs) {
				return logs
			}
		}
	}

	return nil
}

func looksLikeEWCLikeLogs(logs []map[string]any) bool {
	if len(logs) == 0 {
		return false
	}

	for _, logEntry := range logs {
		if _, hasMessage := logEntry["message"].(string); !hasMessage {
			continue
		}
		if _, hasTimestamp := logEntry["timestamp"].(string); hasTimestamp {
			return true
		}
		if _, hasLevel := logEntry["level"].(string); hasLevel {
			return true
		}
		if _, hasMetadata := logEntry["metadata"]; hasMetadata {
			return true
		}
		if _, hasData := logEntry["data"]; hasData {
			return true
		}
	}

	return false
}

func extractOpenIDNetLogsFromWorkflowError(err error) []map[string]any {
	for current := err; current != nil; current = errors.Unwrap(current) {
		details := workflowengine.ParseWorkflowError(current)
		if logs := extractOpenIDNetLogsFromPayload(
			workflowErrorDetailsPayload(details),
		); len(
			logs,
		) > 0 {
			return logs
		}
	}
	return nil
}

func extractOpenIDNetLogsFromPayload(payload any) []map[string]any {
	if logs := workflowengine.AsSliceOfMaps(
		payload,
	); len(logs) > 0 &&
		looksLikeOpenIDNetLogs(logs) {
		return logs
	}

	if payloads, ok := payload.([]any); ok {
		for _, item := range payloads {
			if logs := extractOpenIDNetLogsFromPayload(item); len(logs) > 0 {
				return logs
			}
		}
	}

	if payloadMap := workflowengine.AsMap(payload); payloadMap != nil {
		if nestedPayload, ok := payloadMap["payload"]; ok {
			if logs := extractOpenIDNetLogsFromPayload(nestedPayload); len(logs) > 0 {
				return logs
			}
		}
		if logs := workflowengine.AsSliceOfMaps(payloadMap["logs"]); len(logs) > 0 &&
			looksLikeOpenIDNetLogs(logs) {
			return logs
		}
	}

	return nil
}

func looksLikeOpenIDNetLogs(logs []map[string]any) bool {
	if len(logs) == 0 {
		return false
	}

	for _, logEntry := range logs {
		if _, hasMsg := logEntry["msg"]; hasMsg {
			return true
		}
		if _, hasSrc := logEntry["src"]; hasSrc {
			return true
		}
		if _, hasResult := logEntry["result"]; hasResult {
			return true
		}
	}

	return false
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
		c, err := complianceTemporalClient(namespace)
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
			apiErr := &apierror.APIError{}
			if errors.As(err, &apiErr) {
				return apiErr
			}
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
					case workflows.OpenIDConformanceSuite:
						return getDeeplinkOpenIDConformanceSuite(e, first)
					case workflows.EWCSuite:
						return getDeeplinkEWC(e, first)
					case workflows.WebuildSuite:
						return getDeeplinkEWC(e, first)
					case workflows.EudiwSuite:
						return getDeeplinkEudiw(e, first)
					default:
						return apierror.New(
							http.StatusBadRequest,
							"protocol",
							"unsupported suite",
							fmt.Sprintf(
								"author is %q, expected %s, %s, %s or %s",
								author,
								workflows.OpenIDConformanceSuite,
								workflows.EWCSuite,
								workflows.WebuildSuite,
								workflows.EudiwSuite,
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
					Deeplink any `json:"deeplink"`
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

func getDeeplinkEWC(e *core.RequestEvent, first map[string]any) error {
	if dataB64, ok := first["data"].(string); ok {
		decoded, _ := base64.StdEncoding.DecodeString(dataB64)
		var out struct {
			Output struct {
				Captures struct {
					Deeplink string `json:"deeplink"`
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
