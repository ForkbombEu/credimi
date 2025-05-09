// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
)

type HandleConfirmSuccessRequestInput struct {
	WorkflowID string `json:"workflow_id" validate:"required"`
}

func HandleConfirmSuccess() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleConfirmSuccessRequestInput](e)
		if err != nil {
			return err
		}
		data := workflows.SignalData{Success: true}
		c, err := temporalclient.New()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.SignalWorkflow(
			context.Background(),
			req.WorkflowID,
			"",
			"openidnet-check-result-signal",
			data,
		); err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"signal",
				"failed to send success signal",
				err.Error(),
			)
		}
		return e.JSON(
			http.StatusOK,
			map[string]string{"message": "Workflow completed successfully"},
		)
	}
}

func HandleGetWorkflowsHistory() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth

		namespace, err := getUserNamespace(e.App, authRecord.Id)
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
		defer c.Close()

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

		namespace, err := getUserNamespace(e.App, authRecord.Id)
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
		defer c.Close()
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
		finalJson := make(map[string]interface{})
		err = json.Unmarshal(weJSON, &finalJson)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow execution",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, finalJson)
	}
}

func HandleGetWorkflows() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		namespace, err := getUserNamespace(e.App, authRecord.Id)
		if err != nil {
			return err
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
		defer c.Close()
		list, err := c.ListWorkflow(
			context.Background(),
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace: namespace,
			},
		)
		if err != nil {
			log.Println("Error listing workflows:", err)
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list workflows",
				err.Error(),
			)
		}
		listJSON, err := protojson.Marshal(list)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow list",
				err.Error(),
			)
		}
		finalJSON := make(map[string]interface{})
		err = json.Unmarshal(listJSON, &finalJSON)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow list",
				err.Error(),
			)
		}
		if finalJSON["executions"] == nil {
			finalJSON["executions"] = []map[string]interface{}{}
		}
		return e.JSON(http.StatusOK, finalJSON)
	}
}

type HandleNotifyFailureRequestInput struct {
	WorkflowID string `json:"workflow_id" validate:"required"`
	Reason     string `json:"reason"      validate:"required"`
}

func HandleNotifyFailure() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		log.Println("HandleNotifyFailure called")
		req, err := routing.GetValidatedInput[HandleNotifyFailureRequestInput](e)
		if err != nil {
			return err
		}
		data := workflows.SignalData{Success: false, Reason: req.Reason}
		c, err := temporalclient.New()
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}
		defer c.Close()

		if err := c.SignalWorkflow(
			context.Background(),
			req.WorkflowID,
			"",
			"openidnet-check-result-signal",
			data,
		); err != nil {
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
				http.StatusBadRequest,
				"signal",
				"failed to send failure signal",
				err.Error(),
			)
		}
		return e.JSON(
			http.StatusOK,
			map[string]string{"message": "Test failed", "reason": req.Reason},
		)
	}
}

type HandleSendLogUpdateRequestInput struct {
	WorkflowID string           `json:"workflow_id"`
	Logs       []map[string]any `json:"logs"`
}

func HandleSendLogUpdate() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleSendLogUpdateRequestInput](e)
		if err != nil {
			return err
		}
		if err := notifyLogsUpdate(e.App, req.WorkflowID+"openid4vp-wallet-logs", req.Logs); err != nil {
			return apierror.New(http.StatusBadRequest, "workflow", "failed to send realtime logs update", err.Error())
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Log update sent successfully"})
	}
}

type HandleSendTemporalSignalInput struct {
	WorkflowID string `json:"workflow_id"`
	Signal     string `json:"signal"`
}

func HandleSendTemporalSignal() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleSendTemporalSignalInput](e)
		if err != nil {
			return err
		}
		c, err := temporalclient.New()
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "temporal", "unable to create client", err.Error())
		}
		defer c.Close()
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

func getUserNamespace(app core.App, userID string) (string, error) {
	orgAuthCollection, err := app.FindCollectionByNameOrId("orgAuthorizations")
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"collection",
			"failed to find orgAuthorizations collection",
			err.Error(),
		)
	}

	authOrgRecords, err := app.FindRecordsByFilter(
		orgAuthCollection.Id,
		"user={:user}",
		"",
		0,
		0,
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
	if len(authOrgRecords) == 0 {
		return "", apierror.New(
			http.StatusNotFound,
			"get user namespace",
			"no orgAuthorizations record found",
			"no orgAuthorizations record found",
		)
	}

	ownerRoleRecord, err := app.FindFirstRecordByFilter("orgRoles", "name='owner'")
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"get user namespace",
			"failed to find orgRoles collection",
			err.Error(),
		)
	}

	if len(authOrgRecords) > 1 {
		for _, record := range authOrgRecords {
			if record.GetString("role") == ownerRoleRecord.Id {
				return record.GetString("organization"), nil
			}
		}
	}
	if authOrgRecords[0].GetString("role") == ownerRoleRecord.Id {
		return authOrgRecords[0].GetString("organization"), nil
	}
	return "default", nil
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
			return apierror.New(http.StatusBadRequest, "workflow", "invalid workflow ID", err.Error())
		}

		return apierror.New(http.StatusBadRequest, "signal", fmt.Sprintf("failed to send signal: %s", input.Signal), err.Error())
	}
	return nil
}

func sendOpenIDNetLogUpdateStart(app core.App, c client.Client, input HandleSendTemporalSignalInput) error {
	err := c.SignalWorkflow(
		context.Background(),
		input.WorkflowID,
		"",
		workflows.OpenIDNetStartCheckSignal,
		struct{}{},
	)
	if err != nil {
		canceledErr := &serviceerror.Canceled{}
		if errors.As(err, &canceledErr) {
			wf := c.GetWorkflow(context.Background(), input.WorkflowID, "")
			var result workflowengine.WorkflowResult

			err := wf.Get(context.Background(), &result)
			if err != nil {
				return apierror.New(http.StatusBadRequest, "workflow", "failed to get logs workflow result", err.Error())
			}

			if logsInterface, ok := result.Log.([]any); ok {
				logs := workflows.AsSliceOfMaps(logsInterface)
				id := strings.TrimSuffix(input.WorkflowID, "-logs")
				if err := notifyLogsUpdate(app, id+"openid4vp-wallet-logs", logs); err != nil {
					return apierror.New(http.StatusBadRequest, "workflow", "failed to send realtime logs update", err.Error())
				}
			} else {
				return apierror.New(http.StatusBadRequest, "workflow", "invalid log format", "logs are not in the expected format")
			}
		}
		notFound := &serviceerror.NotFound{}
		if errors.As(err, &notFound) {
			return apierror.New(http.StatusNotFound, "workflow", "workflow not found", err.Error())
		}
		invalidArgument := &serviceerror.InvalidArgument{}
		if errors.As(err, &invalidArgument) {
			return apierror.New(http.StatusBadRequest, "workflow", "invalid workflow ID", err.Error())
		}

		return apierror.New(http.StatusBadRequest, "signal", "failed to send start logs update signal", err.Error())
	}
	return nil
}
