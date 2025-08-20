// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for Credentials Issuers.
// It includes the CredentialsIssuersWorkflow, which validates and imports credential issuer metadata.
// The workflow performs various steps including checking the issuer, parsing JSON responses,
// storing credentials, and cleaning up invalid credentials.
package workflows

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CredentialsTaskQueue is the task queue for the credentials workflow.
const (
	CredentialsTaskQueue       = "CredentialsTaskQueue"
	CredentialIssuerSchemaPath = "schemas/credentialissuer/openid-credential-issuer.schema.json"
	CredentialSchemaPath       = "schemas/credentialissuer/credential_config.schema.json"
)

// CredentialsIssuersWorkflow is a workflow that validates and imports credential issuer metadata.
type CredentialsIssuersWorkflow struct{}

// Name returns the name of the workflow.
func (w *CredentialsIssuersWorkflow) Name() string {
	return "Validate and import Credential Issuer metadata"
}

// GetOptions returns the activity options for the workflow.
func (w *CredentialsIssuersWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

// Workflow is the main workflow function for the CredentialsIssuersWorkflow.
// It performs the following steps:
//  1. Executes the CheckCredentialsIssuerActivity to validate the credentials issuer.
//  2. Parses the raw JSON response from the issuer using the JSONActivity.
//  3. Iterates through the credential configurations supported by the issuer and:
//     - Sends each credential to the "store-or-update-extracted-credentials" endpoint.
//     - Logs the stored credentials.
//  4. Executes a cleanup operation to remove invalid credentials by calling the
//     "cleanup_credentials" endpoint.
//  5. Returns a WorkflowResult containing a success message and logs.
//
// Parameters:
// - ctx: The workflow context.
// - input: The input for the workflow, containing configuration and payload data.
//
// Returns:
// - workflowengine.WorkflowResult: The result of the workflow execution, including logs.
// - error: An error if any step in the workflow fails.
func (w *CredentialsIssuersWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			input.Config["app_url"],
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}
	checkIssuer := activities.NewCheckCredentialsIssuerActivity()
	var issuerResult workflowengine.ActivityResult

	baseURL, ok := input.Payload["base_url"].(string)
	if !ok || baseURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"base_url",
			runMetadata,
		)
	}
	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			runMetadata,
		)
	}
	issuerSchema, ok := input.Config["issuer_schema"].(string)
	if !ok || issuerSchema == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"issuer_schema",
			runMetadata,
		)
	}
	issuerID, ok := input.Payload["issuerID"].(string)
	if !ok || issuerID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"issuerID",
			runMetadata,
		)
	}
	err := workflow.ExecuteActivity(ctx, checkIssuer.Name(), workflowengine.ActivityInput{
		Config: map[string]string{
			"base_url": baseURL,
		},
	}).Get(ctx, &issuerResult)
	if err != nil {
		logger.Error("CheckCredentialIssuer failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	source, ok := issuerResult.Output.(map[string]any)["source"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: source", checkIssuer.Name()),
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	rawJSON, ok := issuerResult.Output.(map[string]any)["rawJSON"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: rawJSON", checkIssuer.Name()),
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	parseJSON := activities.NewJSONActivity(
		map[string]reflect.Type{
			"map": reflect.TypeOf(
				map[string]any{},
			),
		},
	)

	logs := make(map[string][]any)
	var result workflowengine.ActivityResult
	var issuerData map[string]any
	invalidCred := make(map[string]bool)

	err = workflow.ExecuteActivity(ctx, parseJSON.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{
			"rawJSON":    rawJSON,
			"structType": "map",
		},
	}).Get(ctx, &result)
	if err != nil {
		logger.Error("ParseJSON failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	issuerData, err = decodeToMap(result.Output, runMetadata)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	validateJSON := activities.NewSchemaValidationActivity()
	validateErr := workflow.ExecuteActivity(ctx, validateJSON.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{
			"data":   issuerData,
			"schema": issuerSchema,
		},
	}).Get(ctx, nil)
	if validateErr != nil {
		details, err := extractAppErrorDetails(validateErr)
		if err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}

		logs["JSONSchemaError"] = details
		invalidCred, err = extractInvalidCredentialsFromErrorDetails(details, runMetadata)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
	}

	issuerName := ""
	if displayList, ok := issuerData["display"].([]any); ok && len(displayList) > 0 {
		if first, ok := displayList[0].(map[string]any); ok {
			if name, ok := first["name"].(string); ok {
				issuerName = name
			}
		}
	}

	credConfigs, ok := issuerData["credential_configurations_supported"].(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

		appErr := workflowengine.NewAppError(
			errCode,
			"rawJSON should contains credential_configurations_supported",
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	HTTPActivity := activities.NewHTTPActivity()
	validKeys := []string{}
	for credKey, credential := range credConfigs {
		conformant := true
		if invalidCred[credKey] {
			conformant = false
		}

		namespace, ok := input.Config["namespace"].(string)
		if !ok || namespace == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				"namespace",
				runMetadata,
			)
		}

		storeInput := workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "POST",
				"url": fmt.Sprintf(
					"%s/%s",
					appURL,
					"api/credentials_issuers/store-or-update-extracted-credentials"),
			},
			Payload: map[string]any{
				"body": map[string]any{
					"issuerID":   issuerID,
					"issuerName": issuerName,
					"credKey":    credKey,
					"credential": credential,
					"conformant": conformant,
					"orgID":      namespace,
				},
				"expected_status": 200,
			},
		}
		var storeResponse workflowengine.ActivityResult
		err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), storeInput).
			Get(ctx, &storeResponse)
		if err != nil {
			return workflowengine.WorkflowResult{Log: logs}, err
		}
		key, ok := storeResponse.Output.(map[string]any)["body"].(map[string]any)["key"]
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: body.key", HTTPActivity.Name()),
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}

		validKeys = append(validKeys, credKey)
		logs["StoredCredentials"] = append(
			logs["StoredCredentials"],
			key,
		)
	}

	cleanupInput := workflowengine.ActivityInput{
		Config: map[string]string{
			"method": "POST",
			"url": fmt.Sprintf(
				"%s/%s",
				appURL,
				"api/credentials_issuers/cleanup_credentials",
			),
		},
		Payload: map[string]any{
			"body": map[string]any{
				"issuerID":  issuerID,
				"validKeys": validKeys,
			},
			"expected_status": 200,
		},
	}
	var cleanupResponse workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), cleanupInput).
		Get(ctx, &cleanupResponse)
	logs["RemovedCredentials"] = append(
		logs["RemovedCredentials"],
		cleanupResponse.Output.(map[string]any)["body"].(map[string]any)["deleted"],
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			runMetadata,
			logs,
		)
	}

	return workflowengine.WorkflowResult{
		Message: fmt.Sprintf(
			"Successfully retrieved and stored and update credentials from '%s'",
			source,
		),
		Log: logs,
	}, nil
}

// Start initializes and starts the CredentialsIssuersWorkflow execution.
// It loads environment variables, configures the Temporal client with the specified namespace,
// and sets up workflow options including a unique workflow ID and optional memo.
// The workflow is then executed with the provided input.
//
// Parameters:
//   - input: A WorkflowInput object containing configuration and input data for the workflow.
//
// Returns:
//   - result: A WorkflowResult object (empty in this implementation).
//   - err: An error if the workflow fails to start or if there is an issue with the Temporal client.
//
// Errors:
//   - Returns an error if the Temporal client cannot be created or if the workflow execution fails.
func (w *CredentialsIssuersWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "Credentials-Workflow-" + uuid.NewString(),
		TaskQueue:                CredentialsTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}

func toStringSlice(input []any) []string {
	result := make([]string, len(input))
	for i, v := range input {
		result[i] = fmt.Sprint(v)
	}
	return result
}

func extractInvalidCredentialsFromErrorDetails(
	details []any,
	runMetadata workflowengine.WorkflowErrorMetadata,
) (map[string]bool, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityErrorDetails]
	invalidCred := map[string]bool{}

	rawMap, ok := details[0].(map[string]any)
	if !ok {
		wErr := workflowengine.NewAppError(errCode, "details[0] is not a map")
		return nil, workflowengine.NewWorkflowError(wErr, runMetadata)
	}

	causes, ok := rawMap["Causes"].([]any)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"details should contain causes from validation error",
		)
		return nil, workflowengine.NewWorkflowError(wErr, runMetadata)
	}

	for _, cause := range causes {
		causeMap, ok := cause.(map[string]any)
		if !ok {
			wErr := workflowengine.NewAppError(
				errCode,
				"each cause should be a map",
			)
			return nil, workflowengine.NewWorkflowError(wErr, runMetadata)
		}

		instanceLocation, ok := causeMap["InstanceLocation"].([]any)
		if !ok {
			wErr := workflowengine.NewAppError(
				errCode,
				"instanceLocation should be a string array",
			)
			return nil, workflowengine.NewWorkflowError(wErr, runMetadata)
		}

		instanceLocationStr := toStringSlice(instanceLocation)
		if len(instanceLocationStr) > 1 &&
			instanceLocationStr[0] == "credential_configurations_supported" {
			invalidCred[instanceLocationStr[1]] = true
		}
	}

	return invalidCred, nil
}

func decodeToMap(
	input any,
	runMetadata workflowengine.WorkflowErrorMetadata,
) (map[string]any, error) {
	b, err := json.Marshal(input)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
		appErr := workflowengine.NewAppError(
			errCode,
			err.Error(),
			input,
		)
		return nil, workflowengine.NewWorkflowError(
			appErr,
			runMetadata,
		)
	}
	var m map[string]any
	err = json.Unmarshal(b, &m)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
		appErr := workflowengine.NewAppError(errCode, err.Error(), string(b))
		return nil, workflowengine.NewWorkflowError(
			appErr,
			runMetadata,
		)
	}
	return m, err
}

func extractAppErrorDetails(err error) ([]any, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityError]
	var actErr *temporal.ActivityError
	if errors.As(err, &actErr) {
		var appErr *temporal.ApplicationError
		if errors.As(actErr.Unwrap(), &appErr) {
			var details []any
			derr := appErr.Details(&details)
			if derr == nil {
				return details, nil
			}
			return nil, workflowengine.NewAppError(errCode, derr.Error())
		}
		return nil, workflowengine.NewAppError(errCode, actErr.Unwrap().Error())
	}
	return nil, workflowengine.NewAppError(errCode, err.Error())
}
