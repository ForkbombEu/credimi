// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

const fidesCredentialIssuersScheduleID = "fides-credential-issuers-import-schedule"

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
		{
			Method:        http.MethodPost,
			Path:          "/import-fides",
			Handler:       HandleCredentialIssuerImportFides,
			RequestSchema: ImportFidesCredentialIssuersRequest{},
		},
	},
}

var (
	credentialIssuerCheckWellKnownEndpoints = checkWellKnownEndpoints
	credentialIssuerReadSchemaFile          = readSchemaFile
	credentialIssuerStartWorkflow           = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewCredentialsIssuersWorkflow()
		return w.Start(namespace, input)
	}
	fidesCredentialIssuersStartWorkflow = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewFidesCredentialIssuersWorkflow()
		return w.Start(namespace, input)
	}
	fidesCredentialIssuersTemporalClient = temporalclient.GetTemporalClientWithNamespace
	credentialIssuerTemporalClient       = temporalclient.GetTemporalClientWithNamespace
	credentialIssuerWaitForPartialResult = workflowengine.WaitForPartialResult[map[string]any]
)

var fidesCredentialIssuersScheduleTriggerOptions = client.ScheduleTriggerOptions{
	Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
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
			Path:          "/store-or-update",
			Handler:       HandleCredentialIssuerStoreOrUpdate,
			RequestSchema: StoreOrUpdateCredentialIssuerRequest{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:        http.MethodPost,
			Path:          "/store-or-update-extracted-credentials",
			Handler:       HandleCredentialIssuerStoreOrUpdateExtractedCredentials,
			RequestSchema: StoreOrUpdateCredentialsRequest{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
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

type StoreOrUpdateCredentialIssuerRequest struct {
	URL   string `json:"url"`
	OrgID string `json:"orgID"`
	Name  string `json:"name,omitempty"`
	Logo  string `json:"logo,omitempty"`
}

type ImportFidesCredentialIssuersRequest struct {
	IntervalDays int `json:"interval_days" validate:"omitempty,min=1"`
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

		if err := credentialIssuerCheckWellKnownEndpoints(
			e.Request.Context(),
			req.URL,
		); err != nil {
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
		organization, err := pbutils.GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}
		orgName, err := pbutils.GetOrganizationCanonifiedName(e.App, organization)
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
		credIssuerSchemaStr, apiErr := credentialIssuerReadSchemaFile(
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
			Payload: workflows.CredentialsIssuersWorkflowPayload{
				IssuerID: record.Id,
				BaseURL:  req.URL,
			},
			ActivityOptions: &opt,
		}
		result, err := credentialIssuerStartWorkflow(orgName, workflowInput)
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
		workflowURL := utils.JoinURL(
			e.App.Settings().Meta.AppURL,
			"my",
			"tests",
			"runs",
			result.WorkflowID,
			result.WorkflowRunID,
		)

		c, err := credentialIssuerTemporalClient(orgName)
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
		issuerResult, err := credentialIssuerWaitForPartialResult(
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
			details := workflowengine.ParseWorkflowError(err)
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

func HandleCredentialIssuerImportFides() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"credential_issuers",
				"authentication required",
				"authenticated user or user API key is required",
			).JSON(e)
		}

		req, err := decodeImportFidesCredentialIssuersRequest(e.Request)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			).JSON(e)
		}

		organization, err := pbutils.GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}
		orgName, err := pbutils.GetOrganizationCanonifiedName(e.App, organization)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get organization canonified name",
				err.Error(),
			).JSON(e)
		}

		issuerSchema, apiErr := credentialIssuerReadSchemaFile(
			utils.GetEnvironmentVariable("ROOT_DIR") + "/" + workflows.CredentialIssuerSchemaPath,
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		workflowInput := workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":       e.App.Settings().Meta.AppURL,
				"issuer_schema": issuerSchema,
				"orgID":         organization,
			},
		}

		if req.IntervalDays > 0 {
			result, err := scheduleFidesCredentialIssuersImport(
				e.Request.Context(),
				orgName,
				workflowInput,
				req.IntervalDays,
			)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"schedule",
					"failed to schedule Fides credential issuers import",
					err.Error(),
				).JSON(e)
			}
			return e.JSON(http.StatusOK, result)
		}

		result, err := fidesCredentialIssuersStartWorkflow(orgName, workflowInput)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start Fides credential issuers import",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"workflow_id":     result.WorkflowID,
			"workflow_run_id": result.WorkflowRunID,
			"workflow_url": utils.JoinURL(
				e.App.Settings().Meta.AppURL,
				"my",
				"tests",
				"runs",
				result.WorkflowID,
				result.WorkflowRunID,
			),
		})
	}
}

func decodeImportFidesCredentialIssuersRequest(
	req *http.Request,
) (ImportFidesCredentialIssuersRequest, error) {
	var input ImportFidesCredentialIssuersRequest
	if req == nil || req.Body == nil || req.ContentLength == 0 {
		return input, nil
	}
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		return ImportFidesCredentialIssuersRequest{}, err
	}
	if input.IntervalDays < 0 {
		return ImportFidesCredentialIssuersRequest{}, fmt.Errorf(
			"interval_days must be greater than or equal to 1",
		)
	}
	return input, nil
}

func scheduleFidesCredentialIssuersImport(
	ctx context.Context,
	namespace string,
	input workflowengine.WorkflowInput,
	intervalDays int,
) (map[string]any, error) {
	c, err := fidesCredentialIssuersTemporalClient(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	options := buildFidesCredentialIssuersScheduleOptions(
		fidesCredentialIssuersScheduleID,
		input,
		intervalDays,
	)
	_, err = c.ScheduleClient().Create(ctx, options)
	if err != nil {
		if isScheduleAlreadyExistsError(err) {
			handle := c.ScheduleClient().GetHandle(ctx, fidesCredentialIssuersScheduleID)
			err = handle.Update(ctx, client.ScheduleUpdateOptions{
				DoUpdate: func(client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
					return &client.ScheduleUpdate{
						Schedule: buildFidesCredentialIssuersSchedule(input, intervalDays),
					}, nil
				},
			})
			if err == nil {
				err = handle.Trigger(ctx, fidesCredentialIssuersScheduleTriggerOptions)
			}
		}
	} else {
		handle := c.ScheduleClient().GetHandle(ctx, fidesCredentialIssuersScheduleID)
		err = handle.Trigger(ctx, fidesCredentialIssuersScheduleTriggerOptions)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to upsert Fides import schedule: %w", err)
	}

	return map[string]any{
		"message": fmt.Sprintf(
			"Fides credential issuers import triggered now and scheduled every %d day(s)",
			intervalDays,
		),
		"schedule_id":       fidesCredentialIssuersScheduleID,
		"workflowNamespace": namespace,
	}, nil
}

func buildFidesCredentialIssuersScheduleOptions(
	scheduleID string,
	input workflowengine.WorkflowInput,
	intervalDays int,
) client.ScheduleOptions {
	return client.ScheduleOptions{
		ID:      scheduleID,
		Spec:    buildFidesCredentialIssuersScheduleSpec(intervalDays),
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		Action:  buildFidesCredentialIssuersScheduleAction(input),
	}
}

func buildFidesCredentialIssuersSchedule(
	input workflowengine.WorkflowInput,
	intervalDays int,
) *client.Schedule {
	return &client.Schedule{
		Spec: &client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{
				Every: time.Duration(intervalDays) * 24 * time.Hour,
			}},
		},
		Policy: &client.SchedulePolicies{
			Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		},
		State:  &client.ScheduleState{},
		Action: buildFidesCredentialIssuersScheduleAction(input),
	}
}

func buildFidesCredentialIssuersScheduleSpec(intervalDays int) client.ScheduleSpec {
	return client.ScheduleSpec{
		Intervals: []client.ScheduleIntervalSpec{{
			Every: time.Duration(intervalDays) * 24 * time.Hour,
		}},
	}
}

func buildFidesCredentialIssuersScheduleAction(
	input workflowengine.WorkflowInput,
) *client.ScheduleWorkflowAction {
	return &client.ScheduleWorkflowAction{
		ID:        "Fides-Credential-Issuers-Scheduled",
		Workflow:  workflows.FidesCredentialIssuersWorkflowName,
		TaskQueue: workflows.FidesCredentialIssuersTaskQueue,
		Args: []interface{}{
			input,
		},
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
			record.Set("logo_url", logo)
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

			savedLogo := record.GetString("logo_url")
			if savedLogo == originalLogo {
				record.Set("logo_url", logo)
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

func parseCredentialDisplay(cred map[string]any) (name, locale, logo, description string) {
	displayList := credentialDisplayList(cred)
	if len(displayList) > 0 {
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
				} else if urlValue, ok := logoMap["url"].(string); ok {
					logo = urlValue
				}
			}
		}
	}
	return
}

func credentialDisplayList(cred map[string]any) []any {
	if metadata, ok := cred["credential_metadata"].(map[string]any); ok {
		if displayList, ok := metadata["display"].([]any); ok {
			return displayList
		}
	}
	if displayList, ok := cred["display"].([]any); ok {
		return displayList
	}
	return nil
}

func HandleCredentialIssuerStoreOrUpdate() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body StoreOrUpdateCredentialIssuerRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return apis.NewBadRequestError("invalid JSON body", err)
		}
		if strings.TrimSpace(body.URL) == "" {
			return apierror.New(
				http.StatusBadRequest,
				"credential_issuers",
				"missing credential issuer URL",
				"url is required",
			).JSON(e)
		}
		if strings.TrimSpace(body.OrgID) == "" {
			return apierror.New(
				http.StatusBadRequest,
				"credential_issuers",
				"missing organization",
				"orgID is required",
			).JSON(e)
		}

		collection, err := e.App.FindCollectionByNameOrId("credential_issuers")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credential_issuers",
				"failed to find credential issuers collection",
				err.Error(),
			).JSON(e)
		}

		record, err := e.App.FindFirstRecordByFilter(
			collection,
			"url = {:url} && owner = {:owner}",
			map[string]any{
				"url":   body.URL,
				"owner": body.OrgID,
			},
		)
		if err != nil {
			record = core.NewRecord(collection)
			record.Set("url", body.URL)
			record.Set("owner", body.OrgID)
			record.Set("imported", true)
		}
		if body.Name != "" {
			record.Set("name", body.Name)
		}
		if body.Logo != "" {
			record.Set("logo_url", body.Logo)
		}

		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"credential_issuers",
				"failed to save credential issuer",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"record": record.FieldsData(),
		})
	}
}

func checkWellKnownEndpoints(ctx context.Context, baseURL string) error {
	cleanURL := strings.TrimSpace(baseURL)
	if !strings.HasPrefix(cleanURL, "https://") && !strings.HasPrefix(cleanURL, "http://") {
		cleanURL = "https://" + cleanURL
	}
	cleanURL = strings.TrimRight(cleanURL, "/")

	if isFederationWellKnownURL(cleanURL) ||
		isCredentialIssuerWellKnownURL(cleanURL) {
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

func isFederationWellKnownURL(rawURL string) bool {
	const wellKnownPath = "/.well-known/openid-federation"

	return strings.Contains(rawURL, wellKnownPath+"/") ||
		strings.HasSuffix(rawURL, wellKnownPath)
}

func isCredentialIssuerWellKnownURL(rawURL string) bool {
	const wellKnownPath = "/.well-known/openid-credential-issuer"

	return strings.Contains(rawURL, wellKnownPath+"/") ||
		strings.HasSuffix(rawURL, wellKnownPath)
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
	resolver := net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, parsedURL.Hostname())
	if err != nil {
		return fmt.Errorf("could not resolve host: %w", err)
	}
	for _, addr := range ips {
		if isPrivateIP(addr.IP) {
			return fmt.Errorf("refusing to connect to private/internal IP: %s", addr.IP)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
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
