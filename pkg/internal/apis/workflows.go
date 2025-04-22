// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/forkbombeu/didimo/pkg/internal/middlewares"
	temporalclient "github.com/forkbombeu/didimo/pkg/internal/temporal_client"
	engine "github.com/forkbombeu/didimo/pkg/template_engine"
	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
	"github.com/forkbombeu/didimo/pkg/workflow_engine/workflows"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/enums/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func AddComplianceChecks(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		g := se.Router.Group("/api/compliance/check")

		g.Bind(apis.RequireAuth())

		g.POST("", HandleOpenID4VPTest(app))
		g.POST("/{protocol}/{author}/save-variables-and-start",
			HandleSaveVariablesAndStart(app)).Bind(&hook.Handler[*core.RequestEvent]{
			Func: middlewares.ValidateInputMiddleware[*SaveVariablesAndStartRequestInput](),
		})
		g.POST("/confirm-success", HandleConfirmSuccess(app))
		g.POST("/notify-failure", HandleNotifyFailure(app))
		g.POST("/send-log-update", HandleSendLogUpdate(app))
		g.POST("/send-log-update-start", HandleSendLogUpdateStart(app))

		g.GET("/{workflowId}/{runId}/history", HandleGetWorkflowsHistory(app))

		return se.Next()
	})
}

type OpenID4VPTestInputFile struct {
	Variant json.RawMessage `json:"variant"`
	Form    any             `json:"form"`
}

type OpenID4VPRequest struct {
	Input    OpenID4VPTestInputFile `json:"input"`
	UserMail string                 `json:"user_mail"`
	TestName string                 `json:"test_name"`
}

type IssuerURL struct {
	URL string `json:"credentialIssuerUrl"`
}

func HandleOpenID4VPTest(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req OpenID4VPRequest
		if err := decodeJSON(e.Request.Body, &req); err != nil {
			return err
		}

		appURL := app.Settings().Meta.AppURL
		templateStr, err := readTemplateFile(os.Getenv("ROOT_DIR") + "/" + workflows.OpenIDNetStepCITemplatePath)
		if err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		input := workflowengine.WorkflowInput{
			Payload: map[string]any{
				"variant":   string(req.Input.Variant),
				"form":      req.Input.Form,
				"user_mail": req.UserMail,
				"app_url":   appURL,
			},
			Config: map[string]any{
				"template": templateStr,
			},
		}

		var workflow workflows.OpenIDNetWorkflow
		if _, err = workflow.Start(input); err != nil {
			return apis.NewBadRequestError("failed to start openidnet wallet workflow", err)
		}

		return e.JSON(http.StatusOK, map[string]bool{"started": true})
	}
}

func HandleSaveVariablesAndStart(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req SaveVariablesAndStartRequestInput
		if err := decodeJSON(e.Request.Body, &req); err != nil {
			return err
		}

		appURL := app.Settings().Meta.AppURL
		userID := e.Auth.Id
		email := e.Auth.GetString("email")
		namespace, err := getUserNamespace(app, userID)
		if err != nil {
			return apis.NewBadRequestError("failed to get user namespace", err)
		}

		protocol := e.Request.PathValue("protocol")
		author := e.Request.PathValue("author")
		if protocol == "" || author == "" {
			return apis.NewBadRequestError("protocol and author are required", nil)
		}
		protocol, author = normalizeProtocolAndAuthor(protocol, author)

		dirPath := os.Getenv("ROOT_DIR") + "/config_templates/" + protocol + "/" + author + "/"
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return apis.NewBadRequestError("directory does not exist for test "+os.Getenv("ROOT_DIR")+protocol+"/"+author, err)
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
					return err
				}
			case "variables":
				if err := processVariablesTest(app, e, testName, testData, email, appURL, namespace, dirPath, memo); err != nil {
					return err
				}
			default:
				return apis.NewBadRequestError("unsupported format for test "+testName, nil)
			}
		}

		return e.JSON(http.StatusOK, map[string]bool{"started": true})
	}
}

func processJSONChecks(app core.App, e *core.RequestEvent, testData struct {
	Format string      `json:"format" validate:"required"`
	Data   interface{} `json:"data" validate:"required"`
}, email, appURL string, namespace interface{}, memo map[string]interface{}) error {
	jsonData, ok := testData.Data.(string)
	if !ok {
		return apis.NewBadRequestError("invalid JSON format", nil)
	}

	var parsedData OpenID4VPTestInputFile
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

	var parsedVariant OpenID4VPTestInputFile
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

func HandleConfirmSuccess(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req struct {
			WorkflowID string `json:"workflow_id"`
		}
		if err := decodeJSON(e.Request.Body, &req); err != nil {
			return err
		}

		data := workflows.SignalData{Success: true}
		c, err := temporalclient.GetTemporalClient()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.SignalWorkflow(context.Background(), req.WorkflowID, "", "wallet-test-signal", data); err != nil {
			return apis.NewBadRequestError("failed to send success signal", err)
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Workflow completed successfully"})
	}
}

func HandleNotifyFailure(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req struct {
			WorkflowID string `json:"workflow_id"`
			Reason     string `json:"reason"`
		}
		if err := decodeJSON(e.Request.Body, &req); err != nil {
			return err
		}
		data := workflows.SignalData{Success: false, Reason: req.Reason}
		c, err := temporalclient.GetTemporalClient()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.SignalWorkflow(context.Background(), req.WorkflowID, "", "wallet-test-signal", data); err != nil {
			return apis.NewBadRequestError("failed to send failure signal", err)
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Test failed", "reason": req.Reason})
	}
}

func HandleSendLogUpdate(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req struct {
			WorkflowID string           `json:"workflow_id"`
			Logs       []map[string]any `json:"logs"`
		}
		if err := decodeJSON(e.Request.Body, &req); err != nil {
			return err
		}
		if err := notifyLogsUpdate(app, req.WorkflowID+"openid4vp-wallet-logs", req.Logs); err != nil {
			return apis.NewBadRequestError("failed to send real-time log update", err)
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Log update sent successfully"})
	}
}

func HandleSendLogUpdateStart(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req struct {
			WorkflowID string `json:"workflow_id"`
		}
		if err := decodeJSON(e.Request.Body, &req); err != nil {
			return err
		}

		c, err := temporalclient.GetTemporalClient()
		if err != nil {
			return err
		}
		defer c.Close()

		err = c.SignalWorkflow(context.Background(), req.WorkflowID+"-log", "", "wallet-test-start-log-update", struct{}{})
		if err != nil {
			return apis.NewBadRequestError("failed to send start logs update signal", err)
		}
		return e.JSON(http.StatusOK, map[string]string{"message": "Realtime Logs update started successfully"})
	}
}

func HandleGetWorkflowsHistory(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth

		namespace, err := getUserNamespace(e.App, authRecord.Id)
		if err != nil {
			return apis.NewBadRequestError("failed to get user namespace", err)
		}

		workflowId := e.Request.PathValue("workflowId")
		if workflowId == "" {
			return apis.NewBadRequestError("workflowId is required", nil)
		}
		runId := e.Request.PathValue("runId")
		if runId == "" {
			return apis.NewBadRequestError("runId is required", nil)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		defer c.Close()
		if err != nil {
			return apis.NewInternalServerError("unable to create client", err)
		}

		historyIterator := c.GetWorkflowHistory(context.Background(), workflowId, runId, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		var history []map[string]interface{}
		for historyIterator.HasNext() {
			event, err := historyIterator.Next()
			if err != nil {
				return apis.NewInternalServerError("failed to iterate workflow history", err)
			}
			eventData, err := protojson.Marshal(event)
			if err != nil {
				return apis.NewInternalServerError("failed to marshal history event", err)
			}
			var eventMap map[string]interface{}
			if err := json.Unmarshal(eventData, &eventMap); err != nil {
				return apis.NewInternalServerError("failed to unmarshal history event", err)
			}
			history = append(history, eventMap)
		}

		return e.JSON(http.StatusOK, history)
	}
}
