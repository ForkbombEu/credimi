// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	credential_workflow "github.com/forkbombeu/credimi/pkg/credential_issuer/workflow"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/types"
	"go.temporal.io/sdk/client"
)

func AddComplianceChecks(app core.App) {
	routing.AddGroupRoutes(app, routing.RouteGroup{
		BaseURL: "/api/compliance",
		Routes: []routing.RouteDefinition{
			{
				Method:  http.MethodGet,
				Path:    "/checks",
				Handler: handlers.HandleGetWorkflows,
				Input:   nil,
			},
			{
				Method:  http.MethodGet,
				Path:    "/checks/{workflowId}/{runId}",
				Handler: handlers.HandleGetWorkflow,
				Input:   nil,
			},
			{
				Method:  http.MethodGet,
				Path:    "/checks/{workflowId}/{runId}/history",
				Handler: handlers.HandleGetWorkflowsHistory,
				Input:   nil,
			},
			{
				Method:  http.MethodPost,
				Path:    "/{protocol}/{version}/save-variables-and-start",
				Handler: handlers.HandleSaveVariablesAndStart,
				Input:   handlers.SaveVariablesAndStartRequestInput{},
			},
			{
				Method:  http.MethodPost,
				Path:    "/notify-failure",
				Handler: handlers.HandleNotifyFailure,
				Input:   handlers.HandleNotifyFailureRequestInput{},
			},
			{
				Method:  http.MethodPost,
				Path:    "/confirm-success",
				Handler: handlers.HandleConfirmSuccess,
				Input:   handlers.HandleConfirmSuccessRequestInput{},
			},
			{
				Method:  http.MethodPost,
				Path:    "/send-temporal-signal",
				Handler: handlers.HandleSendTemporalSignal,
				Input:   handlers.HandleSendTemporalSignalInput{},
			},
			{
				Method:              http.MethodPost,
				Path:                "/send-log-update",
				Handler:             handlers.HandleSendLogUpdate,
				Input:               handlers.HandleSendLogUpdateRequestInput{},
				ExcludedMiddlewares: []string{apis.DefaultRequireAuthMiddlewareId},
			},
			{
				Method:              http.MethodPost,
				Path:                "/send-eudiw-log-update",
				Handler:             handlers.HandleSendEudiwLogUpdate,
				Input:               handlers.HandleSendLogUpdateRequestInput{},
				ExcludedMiddlewares: []string{apis.DefaultRequireAuthMiddlewareId},
			},
		},
		Middlewares: []*hook.Handler[*core.RequestEvent]{
			// apis.RequireAuth(),
			{Func: middlewares.ErrorHandlingMiddleware},
		},
		Validation: true,
	})
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
				return apierror.New(
					http.StatusInternalServerError,
					"credential_issuers",
					"failed to find credential issuers collection",
					err.Error(),
				)
			}
			organization, err := handlers.GetUserOrganizationId(app, e.Auth.Id)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"organization",
					"failed to get user organization",
					err.Error())
			}
			existingRecords, err := app.FindRecordsByFilter(
				collection.Id,
				"url = {:url} && owner = {:owner}",
				"",
				1,
				0,
				dbx.Params{
					"url":   req.URL,
					"owner": organization,
				},
			)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"credential_issuers",
					"failed to find credential issuer",
					err.Error(),
				)
			}
			var issuerID string

			if len(existingRecords) > 0 {
				issuerID = existingRecords[0].Id
			} else {
				// Create a new record
				newRecord := core.NewRecord(collection)
				newRecord.Set("url", req.URL)
				newRecord.Set("owner", organization)
				if err := app.Save(newRecord); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credential_issuers",
						"failed to save credential issuer",
						err.Error(),
					)
				}

				issuerID = newRecord.Id
			}
			// Start the workflow
			workflowInput := workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":   app.Settings().Meta.AppURL,
					"namespace": organization,
				},
				Payload: map[string]any{
					"issuerID": issuerID,
					"base_url": req.URL,
				},
			}
			w := workflows.CredentialsIssuersWorkflow{}

			_, err = w.Start(workflowInput)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to start workflow",
					err.Error(),
				)
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

		se.Router.POST(
			"/api/credentials_issuers/store-or-update-extracted-credentials",
			func(e *core.RequestEvent) error {
				var body struct {
					IssuerID   string                `json:"issuerID"`
					IssuerName string                `json:"issuerName"`
					CredKey    string                `json:"credKey"`
					Credential activities.Credential `json:"credential"`
					OrgID      string                `json:"orgID"`
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
				record.Set("owner", body.OrgID)

				if err := app.Save(record); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credentials",
						"failed to save credentials",
						err.Error(),
					)
				}
				return e.JSON(http.StatusOK, map[string]any{"key": body.CredKey})
			},
		)
		se.Router.POST(
			"/api/credentials_issuers/cleanup_credentials",
			func(e *core.RequestEvent) error {
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
			},
		)
		return se.Next()
	})
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

func HookAtUserCreation(app *pocketbase.PocketBase) {
	app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		err := createNewOrganizationForUser(e.App, e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	})
}

func HookAtUserLogin(app *pocketbase.PocketBase) {
	app.OnRecordAuthRequest().BindFunc(func(e *core.RecordAuthRequestEvent) error {
		orgAuthCollection, err := e.App.FindCollectionByNameOrId("orgAuthorizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find orgAuthorizations collection", err)
		}
		user := e.Record
		if isSuperUser(e.App, user) {
			return e.Next()
		}
		_, orgNotFound := e.App.FindFirstRecordByFilter(
			orgAuthCollection.Id,
			"user = {:user}",
			dbx.Params{"user": user.Id},
		)
		if orgNotFound == nil {
			return e.Next()
		}
		err = createNewOrganizationForUser(e.App, user)
		if err != nil {
			return apis.NewInternalServerError("failed to create new organization for user", err)
		}
		return e.Next()
	})
}

func createNewOrganizationForUser(app core.App, user *core.Record) error {
	err := app.RunInTransaction(func(txApp core.App) error {
		orgCollection, err := txApp.FindCollectionByNameOrId("organizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find organizations collection", err)
		}

		newOrg := core.NewRecord(orgCollection)
		emailParts := strings.SplitN(user.Email(), "@", 2)
		if len(emailParts) != 2 {
			return apis.NewInternalServerError("invalid email format", nil)
		}

		newOrg.Set("name", emailParts[0]+"'s organization")
		txApp.Save(newOrg)

		ownerRoleRecord, err := txApp.FindFirstRecordByFilter("orgRoles", "name='owner'")
		if err != nil {
			return apis.NewInternalServerError("failed to find owner role", err)
		}

		orgAuthCollection, err := txApp.FindCollectionByNameOrId("orgAuthorizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find orgAuthorizations collection", err)
		}
		newOrgAuth := core.NewRecord(orgAuthCollection)
		newOrgAuth.Set("user", user.Id)
		newOrgAuth.Set("organization", newOrg.Id)
		newOrgAuth.Set("role", ownerRoleRecord.Id)
		txApp.Save(newOrgAuth)

		return nil
	})
	return err
}
