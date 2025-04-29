// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/encoding/protojson"

	temporalclient "github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	engine "github.com/forkbombeu/credimi/pkg/templateengine"
	workflowengine "github.com/forkbombeu/credimi/pkg/workflowengine"
)

type SaveVariablesAndStartRequestInput map[string]struct {
	Format string      `json:"format" validate:"required"`
	Data   interface{} `json:"data" validate:"required"`
}

type openID4VPTestInputFile struct {
	Variant json.RawMessage `json:"variant"`
	Form    any             `json:"form"`
}

func HandleSaveVariablesAndStart(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[SaveVariablesAndStartRequestInput](e)
		if err != nil {
			return err
		}

		if len(req) == 0 {
			return apierror.New(http.StatusBadRequest, "request.body.missing", "Request body cannot be empty", "input is required")
		}

		appURL := app.Settings().Meta.AppURL
		userID := e.Auth.Id
		email := e.Auth.GetString("email")
		namespace, err := getUserNamespace(app, userID)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "user namespace", "failed to get user namespace", err.Error())
		}

		protocol := e.Request.PathValue("protocol")
		author := e.Request.PathValue("author")
		if protocol == "" || author == "" {
			return apierror.New(http.StatusBadRequest, "protocol and author", "protocol and author are required", "missing parameters")
		}
		protocol, author = normalizeProtocolAndAuthor(protocol, author)

		dirPath := os.Getenv("ROOT_DIR") + "/config_templates/" + protocol + "/" + author + "/"
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return apierror.New(http.StatusBadRequest, "directory", "directory does not exist for test "+os.Getenv("ROOT_DIR")+protocol+"/"+author, err.Error())
		}

		for testName, testData := range req {
			memo := map[string]interface{}{
				"test":     testName,
				"standard": protocol,
				"author":   author,
			}

			switch testData.Format {
			case "json":
				if err := processJSONChecks(app, e, testData, email, appURL, namespace, memo); err != nil {
					return apierror.New(http.StatusBadRequest, "json", "failed to process JSON checks", err.Error())
				}
			case "variables":
				if err := processVariablesTest(app, e, testName, testData, email, appURL, namespace, dirPath, memo); err != nil {
					return apierror.New(http.StatusBadRequest, "variables", "failed to process variables test", err.Error())
				}
			default:
				return apierror.New(http.StatusBadRequest, "format", "unsupported format for test "+testName, "unsupported format")
			}
		}

		return e.JSON(http.StatusOK, map[string]bool{"started": true})
	}
}

type HandleConfirmSuccessRequestInput struct {
	WorkflowID string `json:"workflow_id" validate:"required"`
}

func HandleConfirmSuccess(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		log.Println("HandleConfirmSuccess called")
		req, err := routing.GetValidatedInput[HandleConfirmSuccessRequestInput](e)
		if err == nil {
			return err
		}

		data := workflows.SignalData{Success: true}
		c, err := temporalclient.New()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.SignalWorkflow(context.Background(), req.WorkflowID, "", "wallet-test-signal", data); err != nil {
			// return apis.NewBadRequestError("failed to send success signal", err)
			return apierror.New(http.StatusBadRequest, "signal", "failed to send success signal", err.Error())
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Workflow completed successfully"})
	}
}

func HandleGetWorkflowsHistory(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth

		namespace, err := getUserNamespace(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "user namespace", "failed to get user namespace", err.Error())
		}

		workflowID := e.Request.PathValue("workflowId")
		if workflowID == "" {
			return apierror.New(http.StatusBadRequest, "workflowId", "workflowId is required", "missing workflowId")
		}
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(http.StatusBadRequest, "runId", "runId is required", "missing runId")
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "temporal", "unable to create client", err.Error())
		}
		defer c.Close()

		historyIterator := c.GetWorkflowHistory(context.Background(), workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		if historyIterator == nil {
			return apierror.New(http.StatusNotFound, "workflow", "workflow history not found", "not found")
		}
		var history []map[string]interface{}
		for historyIterator.HasNext() {
			event, err := historyIterator.Next()
			if err != nil {
				if _, ok := err.(*serviceerror.NotFound); ok {
					return apierror.New(http.StatusNotFound, "workflow", "workflow history not found", err.Error())
				} else if _, ok := err.(*serviceerror.InvalidArgument); ok {
					return apierror.New(http.StatusNotFound, "workflow", "workflow history not found", err.Error())
				}
				return apierror.New(http.StatusInternalServerError, "workflow", "failed to iterate workflow history", err.Error())

			}
			eventData, err := protojson.Marshal(event)
			if err != nil {
				return apierror.New(http.StatusInternalServerError, "workflow", "failed to marshal history event", err.Error())
			}
			var eventMap map[string]interface{}
			if err := json.Unmarshal(eventData, &eventMap); err != nil {
				return apierror.New(http.StatusInternalServerError, "workflow", "failed to unmarshal history event", err.Error())
			}
			history = append(history, eventMap)
		}

		return e.JSON(http.StatusOK, history)
	}
}

func HandleGetWorkflow(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		workflowID := e.Request.PathValue("workflowId")
		if workflowID == "" {
			return apierror.New(http.StatusBadRequest, "workflowId", "workflowId is required", "missing workflowId")
		}
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(http.StatusBadRequest, "runId", "runId is required", "missing runId")
		}
		authRecord := e.Auth

		namespace, err := getUserNamespace(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "user namespace", "failed to get user namespace", err.Error())
		}
		if namespace == "" {
			return apierror.New(http.StatusBadRequest, "organization", "organization is empty", "missing organization")
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apis.NewInternalServerError("unable to create client", err)
		}
		defer c.Close()
		workflowExecution, err := c.DescribeWorkflowExecution(context.Background(), workflowID, runID)
		if err != nil {
			if _, ok := err.(*serviceerror.NotFound); ok {
				return apierror.New(http.StatusNotFound, "workflow", "workflow not found", err.Error())
			}
			if _, ok := err.(*serviceerror.InvalidArgument); ok {
				return apierror.New(http.StatusBadRequest, "workflow", "invalid workflow ID", err.Error())
			}
			return apierror.New(http.StatusInternalServerError, "workflow", "failed to describe workflow execution", err.Error())
		}
		weJSON, err := protojson.Marshal(workflowExecution)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "workflow", "failed to marshal workflow execution", err.Error())
		}
		finalJson := make(map[string]interface{})
		err = json.Unmarshal(weJSON, &finalJson)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "workflow", "failed to unmarshal workflow execution", err.Error())
		}
		return e.JSON(http.StatusOK, finalJson)
	}
}

func HandleGetWorkflows(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		namespace, err := getUserNamespace(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "user namespace", "failed to get user namespace", err.Error())
		}
		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "temporal", "unable to create client", err.Error())
		}
		defer c.Close()
		list, err := c.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{
			Namespace: namespace,
		})
		if err != nil {
			log.Println("Error listing workflows:", err)
			return apierror.New(http.StatusInternalServerError, "workflow", "failed to list workflows", err.Error())
		}
		listJSON, err := protojson.Marshal(list)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "workflow", "failed to marshal workflow list", err.Error())
		}
		finalJSON := make(map[string]interface{})
		err = json.Unmarshal(listJSON, &finalJSON)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "workflow", "failed to unmarshal workflow list", err.Error())
		}
		if finalJSON["executions"] == nil {
			finalJSON["executions"] = []map[string]interface{}{}
		}
		return e.JSON(http.StatusOK, finalJSON)
	}

}

type HandleNotifyFailureRequestInput struct {
	WorkflowID string `json:"workflow_id" validate:"required"`
	Reason     string `json:"reason" validate:"required"`
}

func HandleNotifyFailure(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		log.Println("HandleNotifyFailure called")
		req, err := routing.GetValidatedInput[HandleNotifyFailureRequestInput](e)
		if err != nil {
			return err
		}
		data := workflows.SignalData{Success: false, Reason: req.Reason}
		c, err := temporalclient.New()
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "temporal", "unable to create client", err.Error())
		}
		defer c.Close()

		if err := c.SignalWorkflow(context.Background(), req.WorkflowID, "", "wallet-test-signal", data); err != nil {
			if _, ok := err.(*serviceerror.NotFound); ok {
				return apierror.New(http.StatusNotFound, "workflow", "workflow not found", err.Error())
			} else if _, ok := err.(*serviceerror.InvalidArgument); ok {
				return apierror.New(http.StatusBadRequest, "workflow", "invalid workflow ID", err.Error())
			}
			return apierror.New(http.StatusBadRequest, "signal", "failed to send failure signal", err.Error())
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Test failed", "reason": req.Reason})
	}
}

type HandleSendLogUpdateStartRequestInput struct {
	WorkflowID string `json:"workflow_id"`
}

func HandleSendLogUpdateStart(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleSendLogUpdateStartRequestInput](e)
		if err != nil {
			return err
		}

		c, err := temporalclient.New()
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "temporal", "unable to create client", err.Error())
		}
		defer c.Close()

		err = c.SignalWorkflow(context.Background(), req.WorkflowID+"-log", "", "wallet-test-start-log-update", struct{}{})
		if err != nil {
			if _, ok := err.(*serviceerror.NotFound); ok {
				return apierror.New(http.StatusNotFound, "workflow", "workflow not found", err.Error())
			} else if _, ok := err.(*serviceerror.InvalidArgument); ok {
				return apierror.New(http.StatusBadRequest, "workflow", "invalid workflow ID", err.Error())
			}
			return apierror.New(http.StatusBadRequest, "signal", "failed to send start logs update signal", err.Error())
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Realtime Logs update started successfully"})
	}
}

type HandleSendLogUpdateRequestInput struct {
	WorkflowID string           `json:"workflow_id"`
	Logs       []map[string]any `json:"logs"`
}

func HandleSendLogUpdate(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[HandleSendLogUpdateRequestInput](e)
		if err != nil {
			return err
		}
		if err := notifyLogsUpdate(app, req.WorkflowID+"openid4vp-wallet-logs", req.Logs); err != nil {
			return apis.NewBadRequestError("failed to send real-time log update", err)
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Log update sent successfully"})
	}
}

///

func getUserNamespace(app core.App, userID string) (string, error) {
	orgAuthCollection, err := app.FindCollectionByNameOrId("orgAuthorizations")
	if err != nil {
		return "", apis.NewInternalServerError("failed to find orgAuthorizations collection", err)
	}

	authOrgRecords, err := app.FindRecordsByFilter(orgAuthCollection.Id, "user={:user}", "", 0, 0, dbx.Params{"user": userID})
	if err != nil {
		return "", apis.NewInternalServerError("failed to find orgAuthorizations records", err)
	}
	if len(authOrgRecords) == 0 {
		return "", apis.NewInternalServerError("user is not authorized to access any organization", nil)
	}

	ownerRoleRecord, err := app.FindFirstRecordByFilter("orgRoles", "name='owner'")
	if err != nil {
		return "", apis.NewInternalServerError("failed to find owner role", err)
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

func processJSONChecks(app core.App, e *core.RequestEvent, testData struct {
	Format string      `json:"format" validate:"required"`
	Data   interface{} `json:"data" validate:"required"`
}, email, appURL string, namespace interface{}, memo map[string]interface{}) error {
	jsonData, ok := testData.Data.(string)
	if !ok {
		return apis.NewBadRequestError("invalid JSON format", nil)
	}

	var parsedData openID4VPTestInputFile
	if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
		return apis.NewBadRequestError("failed to parse JSON input", err)
	}

	templateStr, err := readTemplateFile(os.Getenv("ROOT_DIR") + "/" + workflows.OpenIDNetStepCITemplatePath)
	if err != nil {
		return apis.NewBadRequestError(err.Error(), err)
	}

	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"variant":   string(parsedData.Variant),
			"form":      parsedData.Form,
			"user_mail": email,
			"app_url":   appURL,
		},
		Config: map[string]any{
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
		},
	}
	var workflow workflows.OpenIDNetWorkflow
	if _, err = workflow.Start(input); err != nil {
		return apis.NewBadRequestError("failed to start workflow for json test", err)
	}
	return nil
}

func processVariablesTest(app core.App, e *core.RequestEvent, testName string, testData struct {
	Format string      `json:"format" validate:"required"`
	Data   interface{} `json:"data" validate:"required"`
}, email, appURL string, namespace interface{}, dirPath string, memo map[string]interface{}) error {
	variables, ok := testData.Data.(map[string]interface{})
	if !ok {
		return apis.NewBadRequestError("invalid variables format for test "+testName, nil)
	}

	values := make(map[string]interface{})
	configValues, err := app.FindCollectionByNameOrId("config_values")
	if err != nil {
		return err
	}

	for credimiID, variable := range variables {
		v, ok := variable.(map[string]interface{})
		if !ok {
			return apis.NewBadRequestError("invalid variable format for test "+testName, nil)
		}
		fieldName, ok := v["fieldName"].(string)
		if !ok {
			return apis.NewBadRequestError("invalid fieldName format for test "+testName, nil)
		}

		record := core.NewRecord(configValues)
		record.Set("credimi_id", credimiID)
		record.Set("value", v["value"])
		record.Set("field_name", fieldName)
		record.Set("template_path", testName)
		if err := app.Save(record); err != nil {
			return apis.NewBadRequestError("failed to save variable for test "+testName, err)
		}
		values[fieldName] = v["value"]
	}

	templatePath := dirPath + testName
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return apis.NewBadRequestError("failed to open template for test "+testName, err)
	}

	renderedTemplate, err := engine.RenderTemplate(bytes.NewReader(templateData), values)
	if err != nil {
		return apis.NewInternalServerError("failed to render template for test "+testName, err)
	}

	var parsedVariant openID4VPTestInputFile
	if err := json.Unmarshal([]byte(renderedTemplate), &parsedVariant); err != nil {
		return apis.NewBadRequestError("failed to unmarshal JSON for test "+testName, err)
	}

	templateStr, err := readTemplateFile(os.Getenv("ROOT_DIR") + "/" + workflows.OpenIDNetStepCITemplatePath)
	if err != nil {
		return apis.NewBadRequestError(err.Error(), err)
	}

	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"variant":   string(parsedVariant.Variant),
			"form":      parsedVariant.Form,
			"user_mail": email,
			"app_url":   appURL,
		},
		Config: map[string]any{
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
		},
	}
	var workflow workflows.OpenIDNetWorkflow
	if _, err = workflow.Start(input); err != nil {
		return apis.NewBadRequestError("failed to start workflow for variables test "+testName, err)
	}
	return nil
}

func readTemplateFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to open template file: %w", err)
	}
	return string(data), nil
}

func normalizeProtocolAndAuthor(protocol, author string) (string, string) {
	switch protocol {
	case "openid4vp_wallet":
		protocol = "OpenID4VP_Wallet"
	case "openid4vci_wallet":
		protocol = "OpenID4VCI_Wallet"
	}
	if author == "openid_foundation" {
		author = "OpenID_foundation"
	}
	return protocol, author
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
