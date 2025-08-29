// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	credential_workflow "github.com/forkbombeu/credimi/pkg/credential_issuer/workflow"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/types"
	"go.temporal.io/sdk/client"
)

// IssuerURL is a struct that represents the URL of a credential issuer.
type IssuerURL struct {
	URL string `json:"credentialIssuerUrl"`
}

var IssuersRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/credentials_issuers",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/start-check",
			Handler:       HandleCredentialIssuerStartCheck,
			RequestSchema: IssuerURL{},
		},
	},
}

func HandleCredentialIssuerStartCheck() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req IssuerURL

		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("invalid JSON input", err)
		}

		if err := checkWellKnownEndpoints(e.Request.Context(), req.URL); err != nil {
			return apierror.New(
				http.StatusNotFound,
				"credential_issuers",
				"credential issuer endpoints not accessible",
				err.Error(),
			)
		}

		// Check if a record with the given URL already exists
		collection, err := e.App.FindCollectionByNameOrId("credential_issuers")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credential_issuers",
				"failed to find credential issuers collection",
				err.Error(),
			)
		}
		organization, err := handlers.GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error())
		}
		existingRecords, err := e.App.FindRecordsByFilter(
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
			parsedURL, err := url.Parse(req.URL)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					fmt.Sprintf("credential_issuers_%s", req.URL),
					"invalid URL format",
					err.Error(),
				)
			}
			newRecord := core.NewRecord(collection)
			newRecord.Set("url", req.URL)
			newRecord.Set("owner", organization)
			newRecord.Set("name", parsedURL.Hostname())
			newRecord.Set("imported", true)
			if err := e.App.Save(newRecord); err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					fmt.Sprintf("credential_issuers_%s", req),
					"failed to save credential issuer",
					err.Error(),
				)
			}

			issuerID = newRecord.Id
		}
		credIssuerSchemaStr, err := readSchemaFile(
			utils.GetEnvironmentVariable(
				"ROOT_DIR",
			) + "/" + workflows.CredentialIssuerSchemaPath,
		)
		if err != nil {
			return err
		}

		appURL := e.App.Settings().Meta.AppURL
		// Start the workflow
		workflowInput := workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":       appURL,
				"issuer_schema": credIssuerSchemaStr,
				"namespace":     organization,
			},
			Payload: map[string]any{
				"issuerID": issuerID,
				"base_url": req.URL,
			},
		}
		w := workflows.CredentialsIssuersWorkflow{}

		result, err := w.Start(workflowInput)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			)
		}
		workflowURL := fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			e.App.Settings().Meta.AppURL,
			result.WorkflowID,
			result.WorkflowRunID,
		)
		record, err := e.App.FindRecordById("credential_issuers", issuerID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				fmt.Sprintf("credential_issuers_%s", req),
				"failed to get credential issuer",
				err.Error(),
			)
		}
		record.Set("workflow_url", workflowURL)
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				fmt.Sprintf("credential_issuers_%s", req),
				"failed to save credential issuer",
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
			"workflowUrl":         workflowURL,
		})
	}
}

func checkWellKnownEndpoints(ctx context.Context, baseURL string) error {
	cleanURL := strings.TrimSpace(baseURL)
	if !strings.HasPrefix(cleanURL, "https://") && !strings.HasPrefix(cleanURL, "http://") {
		cleanURL = "https://" + cleanURL
	}
	cleanURL = strings.TrimRight(cleanURL, "/")

	if strings.HasSuffix(cleanURL, "/.well-known/openid-federation") ||
		strings.HasSuffix(cleanURL, "/.well-known/openid-credential-issuer") {
		if err := checkEndpointExists(ctx, cleanURL); err == nil {
			return nil
		}

		return fmt.Errorf("%s is not accessible", cleanURL)
	}

	federationURL := cleanURL + "/.well-known/openid-federation"
	if err := checkEndpointExists(ctx, federationURL); err == nil {
		return nil
	}

	issuerURL := cleanURL + "/.well-known/openid-credential-issuer"
	if err := checkEndpointExists(ctx, issuerURL); err == nil {
		return nil
	}

	return fmt.Errorf(
		`neither .well-known/openid-federation  
		 nor .well-known/openid-credential-issuer endpoints are accessible at %s`,
		cleanURL,
	)
}

func isPrivateIP(ip net.IP) bool {
	privateBlocks := []*net.IPNet{
		// IPv4 private ranges
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
		// IPv6 loopback and link-local
		{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
		{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)},
	}
	for _, block := range privateBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return ip.IsLoopback()
}

func checkEndpointExists(ctx context.Context, urlToCheck string) error {
	parsedURL, err := url.Parse(urlToCheck)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid or unsafe URL provided")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme")
	}

	ips, err := net.LookupIP(parsedURL.Hostname())
	if err != nil {
		return fmt.Errorf("could not resolve host: %w", err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("refusing to connect to private/internal IP: %s", ip)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("endpoint returned status %d", resp.StatusCode)
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
		se.Router.POST(
			"/api/credentials_issuers/store-or-update-extracted-credentials",
			func(e *core.RequestEvent) error {
				var body struct {
					IssuerID   string         `json:"issuerID"`
					IssuerName string         `json:"issuerName"`
					CredKey    string         `json:"credKey"`
					Credential map[string]any `json:"credential"`
					Conformant bool           `json:"conformant"`
					OrgID      string         `json:"orgID"`
				}

				if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
					return apis.NewBadRequestError("invalid JSON body", err)
				}
				var name, locale, logo, format string
				if displayList, ok := body.Credential["display"].([]any); ok &&
					len(displayList) > 0 {
					if first, ok := displayList[0].(map[string]any); ok {
						if credName, ok := first["name"].(string); ok {
							name = credName
						}
						if displayLogo, ok := first["logo"].(map[string]any); ok {
							// do not broke if URI is nil
							if uri, ok := displayLogo["uri"].(string); ok {
								logo = uri
							}
						}
					}
				}
				if credFormat, ok := body.Credential["format"].(string); ok {
					format = credFormat
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
					record.Set("name", name)
					record.Set("logo", logo)
				} else {
					// Update existing record
					record = existing
					var savedCred map[string]any
					err := json.Unmarshal([]byte(record.GetString("json")), &savedCred)
					if err != nil {
						return apierror.New(
							http.StatusInternalServerError,
							"credentials",
							"failed to unmarshal credentials",
							err.Error(),
						)
					}
					var orginalName, originalLogo string
					if displayList, ok := savedCred["display"].([]any); ok &&
						len(displayList) > 0 {
						if first, ok := displayList[0].(map[string]any); ok {
							if credName, ok := first["name"].(string); ok {
								orginalName = credName
							}
							if displayLogo, ok := first["logo"].(map[string]any); ok {
								// do not broke if URI is nil
								if uri, ok := displayLogo["uri"].(string); ok {
									originalLogo = uri
								}
							}
						}
					}

					savedName := record.GetString("name")
					if savedName == orginalName {
						record.Set("name", name)
					}

					savedLogo := record.GetString("logo")
					if savedLogo == originalLogo {
						record.Set("logo", logo)
					}
				}

				credJSON, err := json.Marshal(body.Credential)
				if err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credentials",
						"failed to marshal credentials",
						err.Error(),
					)
				}
				record.Set("format", format)
				record.Set("issuer_name", body.IssuerName)
				record.Set("locale", locale)
				record.Set("json", string(credJSON))
				record.Set("key", body.CredKey)
				record.Set("credential_issuer", body.IssuerID)
				record.Set("conformant", body.Conformant)
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

type WalletURL struct {
	URL string `json:"walletURL"`
}

func HookWalletWorkflow(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/wallet/start-check", func(e *core.RequestEvent) error {
			var req WalletURL

			if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
				return apis.NewBadRequestError("invalid JSON input", err)
			}
			organization, err := handlers.GetUserOrganizationID(app, e.Auth.Id)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"organization",
					"failed to get user organization",
					err.Error())
			}

			// Start the workflow
			workflowInput := workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":   app.Settings().Meta.AppURL,
					"namespace": organization,
				},
				Payload: map[string]any{
					"url": req.URL,
				},
			}
			w := workflows.WalletWorkflow{}

			workflowInfo, err := w.Start(workflowInput)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to start workflow",
					err.Error(),
				)
			}
			client, err := temporalclient.GetTemporalClientWithNamespace(
				organization,
			)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"temporal",
					"failed to get temporal client",
					err.Error(),
				)
			}
			result, err := workflowengine.WaitForPartialResult[map[string]any](
				client,
				workflowInfo.WorkflowID,
				workflowInfo.WorkflowRunID,
				workflows.AppMetadataQuery,
				100*time.Millisecond,
				30*time.Second,
			)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to get partial workflow result",
					err.Error(),
				)
			}
			storeType := getStringFromMap(result, "storeType")
			metadata, ok := result["metadata"].(map[string]any)
			if !ok {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to get partial workflow result",
					"failed to get metadata",
				)
			}
			var name, logo, appleAppID, googleAppID, playstoreURL, appstoreURL, homeURL string
			description := getStringFromMap(metadata, "description")
			switch storeType {
			case "google":
				name = getStringFromMap(metadata, "title")
				logo = getStringFromMap(metadata, "icon")
				googleAppID = getStringFromMap(metadata, "appId")
				homeURL = getStringFromMap(metadata, "developerWebsite")
				playstoreURL = req.URL

			case "apple":
				name = getStringFromMap(metadata, "trackName")
				logo = getStringFromMap(metadata, "artworkUrl100")
				appleAppID = getStringFromMap(metadata, "bundleId")
				homeURL = getStringFromMap(metadata, "sellerUrl")
				appstoreURL = req.URL
			}

			return e.JSON(http.StatusOK, map[string]any{
				"type":          storeType,
				"name":          name,
				"description":   description,
				"logo":          logo,
				"google_app_id": googleAppID,
				"apple_app_id":  appleAppID,
				"playstore_url": playstoreURL,
				"appstore_url":  appstoreURL,
				"home_url":      homeURL,
				"owner":         organization,
			})
		}).Bind(apis.RequireAuth())
		return se.Next()
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

type StartScheduledWorkflowRequest struct {
	WorkflowID string `json:"workflowID"`
	RunID      string `json:"runID"`
	Interval   string `json:"interval"`
}

func HookStartScheduledWorkflow(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/start-scheduled-workflow", func(e *core.RequestEvent) error {
			var req StartScheduledWorkflowRequest

			if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
				return apis.NewBadRequestError("invalid JSON input", err)
			}

			namespace, err := handlers.GetUserOrganizationID(app, e.Auth.Id)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"organization",
					"failed to get user organization",
					err.Error())
			}
			info, err := workflowengine.GetWorkflowRunInfo(req.WorkflowID, req.RunID, namespace)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to get workflow run info",
					err.Error(),
				)
			}

			var interval time.Duration
			switch req.Interval {
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
			err = workflowengine.StartScheduledWorkflowWithOptions(
				info,
				req.WorkflowID,
				namespace,
				interval,
			)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"schedule",
					"failed to start scheduled workflow",
					err.Error(),
				)
			}
			return e.JSON(http.StatusOK, "scheduled workflow started successfully")
		}).Bind(apis.RequireAuth())

		se.Router.GET("/list-scheduled-workflows", func(e *core.RequestEvent) error {
			namespace, err := handlers.GetUserOrganizationID(app, e.Auth.Id)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"organization",
					"failed to get user organization",
					err.Error())
			}

			schedules, err := workflowengine.ListScheduledWorkflows(namespace)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"schedule",
					"failed to list scheduled workflows",
					err.Error(),
				)
			}
			return e.JSON(http.StatusOK, schedules)
		}).Bind(apis.RequireAuth())

		return se.Next()
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

func readSchemaFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", apierror.New(
			http.StatusBadRequest,
			"file",
			"failed to read  JSON schema file",
			err.Error(),
		)
	}
	return string(data), nil
}
func getStringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
