// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

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

var IssuersRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/credentials_issuers",
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

var IssuerTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/credentials_issuers",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/store-or-update-extracted-credentials",
			Handler:       HandleCredentialIssuerStoreOrUpdateExtractedCredentials,
			RequestSchema: StoreOrUpdateCredentialsRequest{},
		},
		{
			Method:        http.MethodPost,
			Path:          "/cleanup-credentials",
			Handler:       HandleCredentialIssuerCleanupCredentials,
			RequestSchema: IssuerCleanupCredentialsRequest{},
		},
	},
}

// IssuerURL is a struct that represents the URL of a credential issuer.
type IssuerURL struct {
	URL string `json:"credentialIssuerUrl"`
}

type StoreOrUpdateCredentialsRequest struct {
	IssuerID   string         `json:"issuerID"`
	CredKey    string         `json:"credKey"`
	Credential map[string]any `json:"credential"`
	Conformant bool           `json:"conformant"`
	OrgID      string         `json:"orgID"`
}

type IssuerCleanupCredentialsRequest struct {
	IssuerID  string   `json:"issuerID"`
	ValidKeys []string `json:"validKeys"`
}

// HandleCredentialIssuerStartCheck handles the /start-check endpoint for credential issuers.
// It is expected that the request body will contain the URL of the credential issuer.
// The handler will check if the credential issuer exists or not.
// If the handler fails to start the workflow, it will return an error with the status code 500.
// If the handler fails to save the credential issuer, it will return an error with the status code 500.
// The response will contain the credential issuer URL and the workflow URL in a JSON object.
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
			).JSON(e)
		}

		// Check if a record with the given URL already exists
		collection, err := e.App.FindCollectionByNameOrId("credential_issuers")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credential_issuers",
				"failed to find credential issuers collection",
				err.Error(),
			).JSON(e)
		}
		organization, err := GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}
		orgName, err := GetOrganizationCanonifiedName(e.App, organization)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get organization canonified name",
				err.Error(),
			).JSON(e)
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
			).JSON(e)
		}
		parsedURL, err := url.Parse(req.URL)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				fmt.Sprintf("credential_issuers_%s", req.URL),
				"invalid URL format",
				err.Error(),
			).JSON(e)
		}
		var record *core.Record
		var isNew bool
		if len(existingRecords) > 0 {
			record = existingRecords[0]
		} else {
			// Create a new record

			record = core.NewRecord(collection)
			record.Set("url", parsedURL.String())
			record.Set("owner", organization)
			record.Set("imported", true)
			if err := e.App.Save(record); err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					fmt.Sprintf("credential_issuers_%s", req),
					"failed to save credential issuer",
					err.Error(),
				).JSON(e)
			}
			isNew = true
		}
		credIssuerSchemaStr, apiErr := readSchemaFile(
			utils.GetEnvironmentVariable(
				"ROOT_DIR",
			) + "/" + workflows.CredentialIssuerSchemaPath,
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		appURL := e.App.Settings().Meta.AppURL
		// Start the workflow
		opt := workflows.DefaultActivityOptions
		opt.RetryPolicy.MaximumAttempts = 1
		workflowInput := workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":       appURL,
				"issuer_schema": credIssuerSchemaStr,
				"orgID":         organization,
			},
			Payload: map[string]any{
				"issuerID": record.Id,
				"base_url": req.URL,
			},
			ActivityOptions: &opt,
		}
		w := workflows.CredentialsIssuersWorkflow{}

		result, err := w.Start(orgName, workflowInput)
		if err != nil {
			if isNew {
				if err := e.App.Delete(record); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credential_issuers",
						"failed to delete credential issuer",
						err.Error(),
					).JSON(e)
				}
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			).JSON(e)
		}
		workflowURL := fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			e.App.Settings().Meta.AppURL,
			result.WorkflowID,
			result.WorkflowRunID,
		)
		c, err := temporalclient.GetTemporalClientWithNamespace(orgName)
		if err != nil {
			if isNew {
				if err := e.App.Delete(record); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credential_issuers",
						"failed to delete credential issuer",
						err.Error(),
					).JSON(e)
				}
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to create client",
				err.Error(),
			).JSON(e)
		}
		issuerResult, err := workflowengine.WaitForPartialResult[map[string]any](
			c,
			result.WorkflowID,
			result.WorkflowRunID,
			workflows.CredentialsIssuerDataQuery,
			100*time.Millisecond,
			1*time.Minute,
		)

		if err != nil {
			if isNew {
				if err := e.App.Delete(record); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credential_issuers",
						"failed to delete credential issuer",
						err.Error(),
					).JSON(e)
				}
			}
			details := workflowengine.ParseWorkflowError(err.Error())
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				details.Summary,
				err.Error(),
			).JSON(e)
		}

		issuerName := getStringFromMap(issuerResult, "issuerName")
		if issuerName == "" {
			issuerName = parsedURL.Hostname()
		}
		logo := getStringFromMap(issuerResult, "logo")
		credentialsNumber, ok := issuerResult["credentialsNumber"].(float64)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"",
				"failed to parse credentials number",
				"unxexpected credentials number format",
			).JSON(e)
		}
		record.Set("name", issuerName)
		record.Set("logo_url", logo)
		record.Set("workflow_url", workflowURL)
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				fmt.Sprintf("credential_issuers_%s", req),
				"failed to save credential issuer",
				err.Error(),
			).JSON(e)
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

		return e.JSON(http.StatusOK, map[string]any{
			"credentialsNumber": credentialsNumber,
			"record":            record.FieldsData(),
		})
	}
}

// HandleCredentialIssuerStoreOrUpdateExtractedCredentials is an endpoint that handles
//
//	storing or updating an extracted credential from a credential issuer.
//
// It takes a StoreOrUpdateCredentialsRequest as input and returns the extracted credential key.
// If the credential does not exist, a new record is created.
// If the credential exists, the record is updated.
func HandleCredentialIssuerStoreOrUpdateExtractedCredentials() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body StoreOrUpdateCredentialsRequest

		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return apis.NewBadRequestError("invalid JSON body", err)
		}
		name, locale, logo, description := parseCredentialDisplay(body.Credential)
		var format string
		if credFormat, ok := body.Credential["format"].(string); ok {
			format = credFormat
		}

		collection, err := e.App.FindCollectionByNameOrId("credentials")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credentials",
				"failed to find credentials collection",
				err.Error(),
			).JSON(e)
		}
		existing, err := e.App.FindFirstRecordByFilter(collection,
			"name = {:key} && credential_issuer = {:issuerID}",
			map[string]any{
				"key":      body.CredKey,
				"issuerID": body.IssuerID,
			},
		)

		var record *core.Record
		if err != nil {
			// Create new record
			record = core.NewRecord(collection)
			record.Set("display_name", name)
			record.Set("logo", logo)

			record.Set("imported", true)
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
				).JSON(e)
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

			savedName := record.GetString("display_name")
			if savedName == orginalName {
				record.Set("display_name", name)
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
			).JSON(e)
		}
		record.Set("format", format)
		record.Set("locale", locale)
		record.Set("description", description)
		record.Set("json", string(credJSON))
		record.Set("name", body.CredKey)
		record.Set("credential_issuer", body.IssuerID)
		record.Set("conformant", body.Conformant)
		record.Set("owner", body.OrgID)

		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credentials",
				"failed to save credentials",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, map[string]any{"key": body.CredKey})
	}
}

// HandleCredentialIssuerCleanupCredentials handles requests to cleanup credentials associated with a credential issuer.
// It expects a JSON body with the following fields:
// - issuerID: the ID of the credential issuer
// - validKeys: a list of valid credential keys
//
// It will delete all credentials associated with the credential issuer that are not in the validKeys list.
// It returns a JSON response with a single field "deleted" containing a list of deleted credential keys.
func HandleCredentialIssuerCleanupCredentials() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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

		collection, err := e.App.FindCollectionByNameOrId("credentials")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credentials",
				"failed to find credentials collection",
				err.Error(),
			).JSON(e)
		}
		all, err := e.App.FindRecordsByFilter(collection,
			"credential_issuer = {:issuerID}",
			"", // sort
			0,  // page
			0,  // perPage
			dbx.Params{"issuerID": body.IssuerID},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credentials",
				"failed to find credentials",
				err.Error(),
			).JSON(e)
		}

		var deleted []string
		for _, rec := range all {
			key := rec.GetString("name")
			if !validSet[key] {
				if err := e.App.Delete(rec); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credentials",
						"failed to delete credentials",
						err.Error(),
					).JSON(e)
				}
				deleted = append(deleted, key)
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"deleted": deleted})
	}
}

func parseCredentialDisplay(cred map[string]any) (name, locale, logo, description string) {
	if displayList, ok := cred["display"].([]any); ok && len(displayList) > 0 {
		if first, ok := displayList[0].(map[string]any); ok {
			if n, ok := first["name"].(string); ok {
				name = n
			}
			if l, ok := first["locale"].(string); ok {
				locale = l
			}
			if d, ok := first["description"].(string); ok {
				description = d
			}
			if logoMap, ok := first["logo"].(map[string]any); ok {
				if uri, ok := logoMap["uri"].(string); ok {
					logo = uri
				}
			}
		}
	}
	return
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

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
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
func readSchemaFile(path string) (string, *apierror.APIError) {
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
	app.OnRecordAfterUpdateSuccess("features").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Get("name") != "updateIssuers" {
			return e.Next()
		}
		if e.Record.Get("active") == false {
			return e.Next()
		}
		envVariables := e.Record.Get("envVariables")
		if envVariables == nil {
			return e.Next()
		}
		result := struct {
			Interval string `json:"interval"`
		}{}
		errJSON := json.Unmarshal(e.Record.Get("envVariables").(types.JSONRaw), &result)
		if errJSON != nil {
			log.Fatal(errJSON)
		}
		if result.Interval == "" {
			return e.Next()
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

		return e.Next()
	})
}
