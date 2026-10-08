// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/safehttp"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
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
	credentialIssuerWaitForUpdateResult  = workflowengine.WaitForUpdateResult[map[string]any]
)

// credentialIssuerHTTPClient may only reach public addresses: the issuer URL
// is chosen by the caller.
var credentialIssuerHTTPClient = safehttp.NewClient(safehttp.Config{
	Timeout:      5 * time.Second,
	MaxRedirects: 10,
})

var fidesCredentialIssuersScheduleTriggerOptions = client.ScheduleTriggerOptions{
	Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
}

// IssuerURL is a struct that represents the URL of a credential issuer.
type IssuerURL struct {
	URL string `json:"credentialIssuerUrl"`
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
			return apierror.New(http.StatusBadRequest, "request", "invalid JSON input", err.Error())
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
		organization, err := pbutils.GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			)
		}
		orgName, err := pbutils.GetOrganizationCanonifiedName(e.App, organization)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get organization canonified name",
				err.Error(),
			)
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
		parsedURL, err := url.Parse(req.URL)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				fmt.Sprintf("credential_issuers_%s", req.URL),
				"invalid URL format",
				err.Error(),
			)
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
				)
			}
			isNew = true
		}
		credIssuerSchemaStr, apiErr := credentialIssuerReadSchemaFile(
			utils.GetEnvironmentVariable(
				"ROOT_DIR",
			) + "/" + workflows.CredentialIssuerSchemaPath,
		)
		if apiErr != nil {
			return apiErr
		}

		// Start the workflow
		opt := workflows.DefaultActivityOptions
		opt.RetryPolicy.MaximumAttempts = 1
		workflowInput := workflowengine.WorkflowInput{
			Config: workflowengine.WithAppConfig(e.App, map[string]any{
				"issuer_schema": credIssuerSchemaStr,
				"orgID":         organization,
			}),
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
					)
				}
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			)
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
					)
				}
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to create client",
				err.Error(),
			)
		}
		updateCtx, cancel := context.WithTimeout(e.Request.Context(), time.Minute)
		defer cancel()
		issuerResult, err := credentialIssuerWaitForUpdateResult(
			updateCtx,
			c,
			result.WorkflowID,
			result.WorkflowRunID,
			workflows.CredentialsIssuerDataUpdate,
		)

		if err != nil {
			if isNew {
				if err := e.App.Delete(record); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"credential_issuers",
						"failed to delete credential issuer",
						err.Error(),
					)
				}
			}
			details := workflowengine.ParseWorkflowError(err)
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				details.Summary,
				err.Error(),
			)
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
			)
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
			)
		}

		req, err := decodeImportFidesCredentialIssuersRequest(e.Request)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			)
		}

		organization, err := pbutils.GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			)
		}
		orgName, err := pbutils.GetOrganizationCanonifiedName(e.App, organization)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get organization canonified name",
				err.Error(),
			)
		}

		issuerSchema, apiErr := credentialIssuerReadSchemaFile(
			utils.GetEnvironmentVariable("ROOT_DIR") + "/" + workflows.CredentialIssuerSchemaPath,
		)
		if apiErr != nil {
			return apiErr
		}

		workflowInput := workflowengine.WorkflowInput{
			Config: workflowengine.WithAppConfig(e.App, map[string]any{
				"issuer_schema": issuerSchema,
				"orgID":         organization,
			}),
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
				)
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
			)
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

func checkEndpointExists(ctx context.Context, urlToCheck string) error {
	parsedURL, err := url.Parse(urlToCheck)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid or unsafe URL provided")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := credentialIssuerHTTPClient.Do(req)
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
