// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	credential_workflow "github.com/forkbombeu/didimo/pkg/credential_issuer/workflow"
	"github.com/forkbombeu/didimo/pkg/internal/temporalclient"
	"github.com/forkbombeu/didimo/pkg/templateengine"
	"github.com/forkbombeu/didimo/pkg/workflowengine"
	"github.com/forkbombeu/didimo/pkg/workflowengine/activities"
	"github.com/forkbombeu/didimo/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"github.com/pocketbase/pocketbase/tools/types"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
)

// OpenID4VPTestInputFile is a struct that represents the input file for OpenID4VP tests.
type OpenID4VPTestInputFile struct {
	Variant json.RawMessage `json:"variant"`
	Form    any             `json:"form"`
}

// OpenID4VPRequest is a struct that represents the request body for OpenID4VP tests.
type OpenID4VPRequest struct {
	Input    OpenID4VPTestInputFile `json:"input"`
	UserMail string                 `json:"user_mail"`
	TestName string                 `json:"test_name"`
}

// IssuerURL is a struct that represents the URL of a credential issuer.
type IssuerURL struct {
	URL string `json:"credentialIssuerUrl"`
}

// HookCredentialWorkflow sets up routes and handlers for managing credential issuers
// and their associated workflows in the PocketBase application.
//
// This function registers the following endpoints:
// 1. POST /credentials_issuers/start-check
//   - Starts a workflow for a credential issuer based on the provided URL.
//   - If a record for the given URL already exists, it reuses it; otherwise, it creates a new record.
//   - Returns the credential issuer URL upon success.
//
// 2. POST /api/credentials_issuers/store-or-update-extracted-credentials
//   - Stores or updates credentials extracted from a workflow.
//   - Accepts details such as issuer ID, issuer name, credential key, and the credential itself.
//   - Updates an existing record if it matches the provided key and issuer ID, or creates a new one.
//
// 3. POST /api/credentials_issuers/cleanup_credentials
//   - Cleans up credentials associated with a specific issuer by removing records
//     that are not in the list of valid keys provided in the request.
//   - Returns a list of deleted keys upon success.
//
// Parameters:
// - app: The PocketBase application instance to which the routes and handlers are bound.
//
// This function ensures that all routes require authentication and handles errors
// gracefully by returning appropriate HTTP responses.
func HookCredentialWorkflow(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/credentials_issuers/start-check", func(e *core.RequestEvent) error {
			var req IssuerURL

			if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
				return apis.NewBadRequestError("invalid JSON input", err)
			}
			// Check if a record with the given URL already exists
			collection, err := app.FindCollectionByNameOrId("credential_issuers")
			if err != nil {
				return err
			}

			existingRecords, err := app.FindRecordsByFilter(
				collection.Id,
				"url = {:url} && owner = {:owner}",
				"",
				1,
				0,
				dbx.Params{"url": req.URL,
					"owner": e.Auth.Id},
			)
			if err != nil {
				return err
			}
			var issuerID string

			if len(existingRecords) > 0 {
				issuerID = existingRecords[0].Id
			} else {
				// Create a new record
				newRecord := core.NewRecord(collection)
				newRecord.Set("url", req.URL)
				newRecord.Set("owner", e.Auth.Id)
				if err := app.Save(newRecord); err != nil {
					return err
				}

				issuerID = newRecord.Id
			}
			// Start the workflow
			workflowInput := workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": app.Settings().Meta.AppURL,
				},
				Payload: map[string]any{
					"issuerID": issuerID,
					"base_url": req.URL,
				},
			}
			w := workflows.CredentialsIssuersWorkflow{}

			_, err = w.Start(workflowInput)
			if err != nil {
				return fmt.Errorf("error starting workflow for URL %s: %v", req.URL, err)
			}
			//
			// providers, err := app.FindCollectionByNameOrId("services")
			// if err != nil {
			// 	return err
			// }
			//
			// newRecord := core.NewRecord(providers)
			// newRecord.Set("credential_issuers", issuerID)
			// newRecord.Set("name", "TestName")
			// // Save the new record in providers
			// if err := app.Save(newRecord); err != nil {
			// 	return err
			// }
			return e.JSON(http.StatusOK, map[string]string{
				"credentialIssuerUrl": req.URL,
			})
		}).Bind(apis.RequireAuth())

		se.Router.POST("/api/credentials_issuers/store-or-update-extracted-credentials", func(e *core.RequestEvent) error {
			var body struct {
				IssuerID   string                `json:"issuerID"`
				IssuerName string                `json:"issuerName"`
				CredKey    string                `json:"credKey"`
				Credential activities.Credential `json:"credential"`
			}

			if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
				return apis.NewBadRequestError("invalid JSON body", err)
			}

			var name, locale, logo string
			if len(body.Credential.Display) > 0 {
				display := body.Credential.Display[0]
				name = display.Name
				if display.Locale != nil {
					locale = *display.Locale
				}
				if display.Logo != nil {
					// do not broke if URI is nil
					if display.Logo.Uri != nil {
						logo = *display.Logo.Uri
					}
				}
			}

			collection, err := app.FindCollectionByNameOrId("credentials")
			if err != nil {
				return err
			}
			existing, err := app.FindFirstRecordByFilter(collection,
				"key = {:key} && credential_issuer = {:issuerID}",
				map[string]any{
					"key":      body.CredKey,
					"issuerID": body.IssuerID,
				},
			)

			var record *core.Record
			if err != nil {
				// Create new record
				record = core.NewRecord(collection)
			} else {
				// Update existing record
				record = existing
			}

			// Marshal original credential JSON to store
			credJSON, _ := json.Marshal(body.Credential)

			record.Set("format", body.Credential.Format)
			record.Set("issuer_name", body.IssuerName)
			record.Set("name", name)
			record.Set("locale", locale)
			record.Set("logo", logo)
			record.Set("json", string(credJSON))
			record.Set("key", body.CredKey)
			record.Set("credential_issuer", body.IssuerID)

			if err := app.Save(record); err != nil {
				return err
			}
			return e.JSON(http.StatusOK, map[string]any{"key": body.CredKey})
		})
		se.Router.POST("/api/credentials_issuers/cleanup_credentials", func(e *core.RequestEvent) error {
			var body struct {
				IssuerID  string   `json:"issuerID"`
				ValidKeys []string `json:"validKeys"`
			}

			if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
				return apis.NewBadRequestError("invalid JSON body", err)
			}

			validSet := map[string]bool{}
			for _, key := range body.ValidKeys {
				validSet[key] = true
			}

			collection, err := app.FindCollectionByNameOrId("credentials")
			if err != nil {
				return err
			}
			all, err := app.FindRecordsByFilter(collection,
				"credential_issuer = {:issuerID}",
				"", // sort
				0,  // page
				0,  // perPage
				dbx.Params{"issuerID": body.IssuerID},
			)

			if err != nil {
				return apis.NewBadRequestError("failed to find records", err)
			}

			var deleted []string
			for _, rec := range all {
				key := rec.GetString("key")
				if !validSet[key] {
					if err := app.Delete(rec); err != nil {
						return apis.NewBadRequestError("failed to delete record", err)
					}
					deleted = append(deleted, key)
				}
			}
			return e.JSON(http.StatusOK, map[string]any{"deleted": deleted})
		})
		return se.Next()
	})

}

// AddOpenID4VPTestEndpoints registers multiple HTTP endpoints to the provided PocketBase application.
// These endpoints are used for testing OpenID4VP workflows and related functionalities.
//
// Endpoints:
// 1. POST /api/openid4vp-test
//   - Starts an OpenID4VP workflow using the provided input data.
//
// 2. POST /api/{protocol}/{author}/save-variables-and-start
//   - Saves variables and starts a workflow for a specific protocol and author.
//   - Supports JSON and variable-based input formats.
//
// 3. POST /wallet-test/confirm-success
//   - Sends a success signal to a specified workflow.
//
// 4. POST /wallet-test/notify-failure
//   - Sends a failure signal to a specified workflow with a reason for failure.
//
// 5. POST /wallet-test/send-log-update
//   - Sends real-time log updates for a specified workflow.
//
// 6. POST /wallet-test/send-log-update-start
//   - Signals the start of real-time log updates for a specified workflow.
//
// Parameters:
// - app: A pointer to the PocketBase application instance.
//
// Notes:
// - Some endpoints require authentication.
// - The function uses various utilities like JSON decoding, file reading, and workflow engine interactions.
// - Errors are returned as HTTP responses with appropriate status codes and messages.
func AddOpenID4VPTestEndpoints(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/openid4vp-test", func(e *core.RequestEvent) error {
			var req OpenID4VPRequest
			appURL := app.Settings().Meta.AppURL
			if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
				return apis.NewBadRequestError("invalid JSON input", err)
			}
			template, err := os.ReadFile(workflows.OpenIDNetStepCITemplatePath)
			if err != nil {
				return apis.NewBadRequestError("failed to open template file: %w", err)
			}
			// Start the workflow
			input := workflowengine.WorkflowInput{
				Payload: map[string]any{
					"variant":   string(req.Input.Variant),
					"form":      req.Input.Form,
					"user_mail": req.UserMail,
					"app_url":   appURL,
				},
				Config: map[string]any{
					"template": string(template),
				},
			}
			var w workflows.OpenIDNetWorkflow
			_, err = w.Start(input)
			if err != nil {
				return apis.NewBadRequestError("failed to start openidnet wallet workflow", err)
			}

			return e.JSON(http.StatusOK, map[string]bool{
				"started": true,
			})
		})

		se.Router.POST("/api/{protocol}/{author}/save-variables-and-start", func(e *core.RequestEvent) error {
			var req map[string]struct {
				Format string      `json:"format"`
				Data   interface{} `json:"data"`
			}
			appURL := app.Settings().Meta.AppURL

			User := e.Auth.Id
			email := e.Auth.GetString("email")

			namespace, err := getUserNamespace(app, User)
			if err != nil {
				return apis.NewBadRequestError("failed to get user namespace", err)
			}

			protocol := e.Request.PathValue("protocol")
			author := e.Request.PathValue("author")

			if protocol == "" || author == "" {
				return apis.NewBadRequestError("protocol and author are required", nil)
			}
			if protocol == "openid4vp_wallet" {
				protocol = "OpenID4VP_Wallet"
			}
			if protocol == "openid4vci_wallet" {
				protocol = "OpenID4VCI_Wallet"
			}
			if author == "openid_foundation" {
				author = "OpenID_foundation"
			}
			filepath := os.Getenv("ROOT_DIR") + "/config_templates/" + protocol + "/" + author + "/"

			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				return apis.NewBadRequestError("directory does not exist for test "+protocol+"/"+author, err)
			}

			if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
				return apis.NewBadRequestError("invalid JSON input", err)
			}

			for testName, testData := range req {
				memo := map[string]interface{}{
					"test":     testName,
					"standard": protocol,
					"author":   author,
				}
				if testData.Format == "json" {
					jsonData, ok := testData.Data.(string)
					if !ok {
						return apis.NewBadRequestError("invalid JSON format for test "+testName, nil)
					}

					var parsedData OpenID4VPTestInputFile
					if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
						return apis.NewBadRequestError("failed to parse JSON for test "+testName, err)
					}
					stepCItemplate, err := os.ReadFile(workflows.OpenIDNetStepCITemplatePath)
					if err != nil {
						return apis.NewBadRequestError("failed to open template file: %w", err)
					}
					// Start the workflow
					input := workflowengine.WorkflowInput{
						Payload: map[string]any{
							"variant":   string(parsedData.Variant),
							"form":      parsedData.Form,
							"user_mail": email,
							"app_url":   appURL,
						},
						Config: map[string]any{
							"template":  string(stepCItemplate),
							"namespace": namespace,
							"memo":      memo,
						},
					}
					var w workflows.OpenIDNetWorkflow
					_, err = w.Start(input)
					if err != nil {
						return apis.NewBadRequestError("failed to start openidnet wallet for test "+testName, err)
					}
				} else if testData.Format == "variables" {
					variables, ok := testData.Data.(map[string]interface{})
					values := make(map[string]interface{})
					configValues, err := app.FindCollectionByNameOrId("config_values")
					if err != nil {
						return err
					}
					if !ok {
						return apis.NewBadRequestError("invalid variables format for test "+testName, nil)
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
							log.Println("Failed to create related record:", err)
						}
						values[fieldName] = v["value"]
					}

					template, err := os.Open(filepath + testName)
					if err != nil {
						return apis.NewBadRequestError("failed to open template for test "+testName, err)
					}
					defer template.Close()

					templateFile, err := os.Open(filepath + testName)
					if err != nil {
						return apis.NewBadRequestError("failed to open template for test "+testName, err)
					}
					defer templateFile.Close()

					renderedTemplate, err := templateengine.RenderTemplate(templateFile, values)
					if err != nil {
						return apis.NewInternalServerError("failed to render template for test "+testName, err)
					}

					var parsedVariant OpenID4VPTestInputFile
					errParsing := json.Unmarshal([]byte(renderedTemplate), &parsedVariant)
					if errParsing != nil {
						return apis.NewBadRequestError("failed to unmarshal JSON for test "+testName, errParsing)
					}
					stepCItemplate, err := os.ReadFile(workflows.OpenIDNetStepCITemplatePath)

					if err != nil {
						return apis.NewBadRequestError("failed to open template file: %w", err)
					}
					// Start the workflow
					input := workflowengine.WorkflowInput{
						Payload: map[string]any{
							"variant":   string(parsedVariant.Variant),
							"form":      parsedVariant.Form,
							"user_mail": email,
							"app_url":   appURL,
						},
						Config: map[string]any{
							"template":  string(stepCItemplate),
							"namespace": namespace,
							"memo":      memo,
						},
					}
					var w workflows.OpenIDNetWorkflow
					_, err = w.Start(input)
					if err != nil {
						return apis.NewBadRequestError("failed to start openidnet wallet for test "+testName, err)
					}

				} else {
					return apis.NewBadRequestError("unsupported format for test "+testName, nil)
				}
			}

			return e.JSON(http.StatusOK, map[string]bool{
				"started": true,
			})
		}).Bind(apis.RequireAuth())

		se.Router.POST("/wallet-test/confirm-success", func(e *core.RequestEvent) error {
			var request struct {
				WorkflowID string `json:"workflow_id"`
			}
			if err := json.NewDecoder(e.Request.Body).Decode(&request); err != nil {
				return apis.NewBadRequestError("Invalid JSON input", err)
			}
			data := workflows.SignalData{
				Success: true,
			}

			c, err := temporalclient.New()
			if err != nil {
				return err
			}
			defer c.Close()

			err = c.SignalWorkflow(context.Background(), request.WorkflowID, "", "wallet-test-signal", data)
			if err != nil {
				return apis.NewBadRequestError("Failed to send success signal", err)
			}

			return e.JSON(http.StatusOK, map[string]string{"message": "Workflow completed successfully"})
		})

		se.Router.POST("/wallet-test/notify-failure", func(e *core.RequestEvent) error {
			var request struct {
				WorkflowID string `json:"workflow_id"`
				Reason     string `json:"reason"`
			}
			if err := json.NewDecoder(e.Request.Body).Decode(&request); err != nil {
				return apis.NewBadRequestError("Invalid JSON input", err)
			}
			data := workflows.SignalData{
				Success: false,
				Reason:  request.Reason,
			}

			c, err := temporalclient.New()
			if err != nil {
				return err
			}
			defer c.Close()

			err = c.SignalWorkflow(context.Background(), request.WorkflowID, "", "wallet-test-signal", data)
			if err != nil {
				return apis.NewBadRequestError("Failed to send failure signal", err)
			}

			return e.JSON(http.StatusOK, map[string]string{"message": "Test failed", "reason": request.Reason})
		})
		type LogUpdateRequest struct {
			WorkflowID string           `json:"workflow_id"`
			Logs       []map[string]any `json:"logs"`
		}

		se.Router.POST("/wallet-test/send-log-update", func(e *core.RequestEvent) error {
			var logData LogUpdateRequest
			if err := json.NewDecoder(e.Request.Body).Decode(&logData); err != nil {
				return apis.NewBadRequestError("invalid JSON input", err)
			}
			if err := notifyLogsUpdate(app, logData.WorkflowID+"openid4vp-wallet-logs", logData.Logs); err != nil {
				return apis.NewBadRequestError("failed to send real-time log update", err)
			}

			return e.JSON(http.StatusOK, map[string]string{
				"message": "Log update sent successfully",
			})
		})
		se.Router.POST("/wallet-test/send-log-update-start", func(e *core.RequestEvent) error {
			var request struct {
				WorkflowID string `json:"workflow_id"`
			}
			if err := json.NewDecoder(e.Request.Body).Decode(&request); err != nil {
				return apis.NewBadRequestError("Invalid JSON input", err)
			}

			c, err := temporalclient.New()
			if err != nil {
				return err
			}

			err = c.SignalWorkflow(context.Background(), request.WorkflowID+"-log", "", "wallet-test-start-log-update", struct{}{})
			if err != nil {
				return apis.NewBadRequestError("Failed to send start logs update signal", err)
			}

			return e.JSON(http.StatusOK, map[string]string{"message": "Realtime Logs update started successfully"})
		})

		return se.Next()
	})
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

// HookUpdateCredentialsIssuers sets up a hook in the PocketBase application to listen for
// successful updates to records in the "features" collection with the name "updateIssuers".
// When triggered, it checks if the "active" field is true and processes the "envVariables"
// field to determine a scheduling interval. Based on the interval, it creates a Temporal
// schedule to periodically execute the FetchIssuersWorkflow.
//
// Parameters:
//   - app: A pointer to the PocketBase application instance.
//
// Behavior:
//   - Listens for updates to the "features" collection.
//   - Verifies the record name is "updateIssuers" and the "active" field is true.
//   - Parses the "envVariables" field to extract the scheduling interval.
//   - Maps the interval to a predefined duration (e.g., every minute, hourly, daily, etc.).
//   - Creates a Temporal schedule with the specified interval to run the FetchIssuersWorkflow.
//
// Notes:
//   - If the "envVariables" field is missing or the interval is invalid, the hook exits without action.
//   - Logs fatal errors if JSON unmarshalling or Temporal client/schedule creation fails.
func HookUpdateCredentialsIssuers(app *pocketbase.PocketBase) {
	app.OnRecordAfterUpdateSuccess().BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Collection().Name != "features" || e.Record.Get("name") != "updateIssuers" {
			return nil
		}
		if e.Record.Get("active") == false {
			return nil
		}
		envVariables := e.Record.Get("envVariables")
		if envVariables == nil {
			return nil
		}
		result := struct {
			Interval string `json:"interval"`
		}{}
		errJSON := json.Unmarshal(e.Record.Get("envVariables").(types.JSONRaw), &result)
		if errJSON != nil {
			log.Fatal(errJSON)
		}
		if result.Interval == "" {
			return nil
		}
		var interval time.Duration
		switch result.Interval {
		case "every_minute":
			interval = time.Minute
		case "hourly":
			interval = time.Hour
		case "daily":
			interval = time.Hour * 24
		case "weekly":
			interval = time.Hour * 24 * 7
		case "monthly":
			interval = time.Hour * 24 * 30
		default:
			interval = time.Hour
		}
		workflowID := "schedule_workflow_id" + fmt.Sprintf("%d", time.Now().Unix())
		scheduleID := "schedule_id" + fmt.Sprintf("%d", time.Now().Unix())
		ctx := context.Background()

		temporalClient, err := client.Dial(client.Options{
			HostPort: client.DefaultHostPort,
		})
		if err != nil {
			log.Fatalln("Unable to create Temporal Client", err)
		}
		defer temporalClient.Close()
		scheduleHandle, err := temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID: scheduleID,
			Spec: client.ScheduleSpec{
				Intervals: []client.ScheduleIntervalSpec{
					{
						Every: interval,
					},
				},
			},
			Action: &client.ScheduleWorkflowAction{
				ID:        workflowID,
				Workflow:  credential_workflow.FetchIssuersWorkflow,
				TaskQueue: credential_workflow.FetchIssuersTaskQueue,
			},
		})
		if err != nil {
			log.Fatalln("Unable to create schedule", err)
		}
		_, _ = scheduleHandle.Describe(ctx)

		return nil
	})
}

// RouteWorkflow sets up the HTTP routes for managing workflows in the application.
// It defines the following endpoints:
//
// 1. GET /api/workflows
//   - Retrieves a list of workflows for the authenticated user.
//   - Requires authentication.
//   - Returns a JSON response containing the list of workflows.
//
// 2. GET /api/workflows/{workflowId}/{runId}
//   - Retrieves details of a specific workflow execution identified by `workflowId` and `runId`.
//   - Requires authentication.
//   - Returns a JSON response containing the workflow execution details.
//
// 3. GET /api/workflows/{workflowId}/{runId}/history
//   - Retrieves the history of a specific workflow execution identified by `workflowId` and `runId`.
//   - Requires authentication.
//   - Returns a JSON response containing the workflow execution history.
//
// Each route ensures that the user is authorized to access the requested namespace
// and handles errors such as missing parameters, unauthorized access, and internal server issues.
func RouteWorkflow(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/workflows", func(e *core.RequestEvent) error {
			authRecord := e.Auth
			namespace, err := getUserNamespace(e.App, authRecord.Id)
			if err != nil {
				return apis.NewUnauthorizedError("User is not authorized to access this organization", err)
			}
			c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
			if err != nil {
				return apis.NewInternalServerError("unable to create client", err)
			}
			defer c.Close()
			list, err := c.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{
				Namespace: namespace,
			})
			if err != nil {
				log.Println("Error listing workflows:", err)
				return apis.NewInternalServerError("failed to list workflows", err)
			}
			listJSON, err := protojson.Marshal(list)
			if err != nil {
				return apis.NewInternalServerError("failed to marshal workflow list", err)
			}
			finalJSON := make(map[string]interface{})
			err = json.Unmarshal(listJSON, &finalJSON)
			if err != nil {
				return apis.NewInternalServerError("failed to unmarshal workflow list", err)
			}
			if finalJSON["executions"] == nil {
				finalJSON["executions"] = []map[string]interface{}{}
			}
			return e.JSON(http.StatusOK, finalJSON)
		}).Bind(apis.RequireAuth())

		se.Router.GET("/api/workflows/{workflowId}/{runId}", func(e *core.RequestEvent) error {
			workflowID := e.Request.PathValue("workflowId")
			if workflowID == "" {
				return apis.NewBadRequestError("workflowId is required", nil)
			}
			runID := e.Request.PathValue("runId")
			if runID == "" {
				return apis.NewBadRequestError("runId is required", nil)
			}
			authRecord := e.Auth

			namespace, err := getUserNamespace(e.App, authRecord.Id)
			if err != nil {
				return apis.NewUnauthorizedError("User is not authorized to access this organization", err)
			}
			if namespace == "" {
				return apis.NewBadRequestError("organization is empty", nil)
			}

			c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
			if err != nil {
				return apis.NewInternalServerError("unable to create client", err)
			}
			defer c.Close()
			workflowExecution, err := c.DescribeWorkflowExecution(context.Background(), workflowID, runID)
			if err != nil {
				return apis.NewInternalServerError("failed to describe workflow execution", err)
			}
			weJSON, err := protojson.Marshal(workflowExecution)
			if err != nil {
				return apis.NewInternalServerError("failed to marshal workflow execution", err)
			}
			finalJSON := make(map[string]interface{})
			err = json.Unmarshal(weJSON, &finalJSON)
			if err != nil {
				return apis.NewInternalServerError("failed to unmarshal workflow execution", err)
			}
			return e.JSON(http.StatusOK, finalJSON)
		}).Bind(apis.RequireAuth())

		se.Router.GET("/api/workflows/{workflowId}/{runId}/history", func(e *core.RequestEvent) error {
			authRecord := e.Auth

			namespace, err := getUserNamespace(e.App, authRecord.Id)
			if err != nil {
				return apis.NewBadRequestError("failed to get user namespace", err)
			}

			workflowID := e.Request.PathValue("workflowId")
			if workflowID == "" {
				return apis.NewBadRequestError("workflowId is required", nil)
			}
			runID := e.Request.PathValue("runId")
			if runID == "" {
				return apis.NewBadRequestError("runId is required", nil)
			}

			c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
			if err != nil {
				return apis.NewInternalServerError("unable to create client", err)
			}
			defer c.Close()

			historyIterator := c.GetWorkflowHistory(context.Background(), workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
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
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// HookAtUserCreation sets up a hook in the PocketBase application to execute
// custom logic after a user record is successfully created. Specifically, it
// binds a function to the "users" collection that adds the newly created user
// to a default organization.
//
// Parameters:
//   - app: A pointer to the PocketBase application instance.
//
// The bound function will attempt to add the user to a default organization
// by calling `addUserToDefaultOrganization`. If an error occurs during this
// process, the error is returned and the hook execution is halted. Otherwise,
// the hook proceeds to the next middleware or handler by calling `e.Next()`.
func HookAtUserCreation(app *pocketbase.PocketBase) {
	app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		err := addUserToDefaultOrganization(e)
		if err != nil {
			return err
		}
		return e.Next()
	})
}

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

func addUserToDefaultOrganization(e *core.RecordEvent) error {
	user := e.Record
	errTx := e.App.RunInTransaction(func(txApp core.App) error {
		orgCollection, err := txApp.FindCollectionByNameOrId("organizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find organizations collection", err)
		}
		defaultOrgRecord, err := txApp.FindFirstRecordByFilter(orgCollection.Id, "name='default'")
		if err != nil {
			return apis.NewInternalServerError("failed to find default organization", err)
		}
		if defaultOrgRecord == nil {
			return apis.NewInternalServerError("default organization not found", nil)
		}
		orgAuthCollection, err := txApp.FindCollectionByNameOrId("orgAuthorizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find orgAuthorizations collection", err)
		}
		newOrgAuth := core.NewRecord(orgAuthCollection)
		newOrgAuth.Set("user", user.Id)
		newOrgAuth.Set("organization", defaultOrgRecord.Id)
		memberRoleRecord, err := txApp.FindFirstRecordByFilter("orgRoles", "name='member'")
		if err != nil {
			return apis.NewInternalServerError("failed to find owner role", err)
		}
		newOrgAuth.Set("role", memberRoleRecord.Id)
		err = txApp.Save(newOrgAuth)
		if err != nil {
			return apis.NewInternalServerError("failed to save orgAuthorization record", err)
		}
		return nil
	})

	if errTx != nil {
		return apis.NewInternalServerError("failed to add user to default organization", errTx)
	}
	return nil
}

// TODO: This function will be used when user will claim the organization
// func createNamespaceForUser(e *core.RecordEvent, user *core.Record) error {

// 	err := e.App.RunInTransaction(func(txApp core.App) error {
// 		orgCollection, err := txApp.FindCollectionByNameOrId("organizations")
// 		if err != nil {
// 			return apis.NewInternalServerError("failed to find organizations collection", err)
// 		}

// 		newOrg := core.NewRecord(orgCollection)
// 		newOrg.Set("name", user.Id)
// 		txApp.Save(newOrg)

// 		ownerRoleRecord, err := txApp.FindFirstRecordByFilter("orgRoles", "name='owner'")
// 		if err != nil {
// 			return apis.NewInternalServerError("failed to find owner role", err)
// 		}

// 		orgAuthCollection, err := txApp.FindCollectionByNameOrId("orgAuthorizations")
// 		if err != nil {
// 			return apis.NewInternalServerError("failed to find orgAuthorizations collection", err)
// 		}
// 		newOrgAuth := core.NewRecord(orgAuthCollection)
// 		newOrgAuth.Set("user", user.Id)
// 		newOrgAuth.Set("organization", newOrg.Id)
// 		newOrgAuth.Set("role", ownerRoleRecord.Id)
// 		txApp.Save(newOrgAuth)

// 		return nil
// 	})

// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
