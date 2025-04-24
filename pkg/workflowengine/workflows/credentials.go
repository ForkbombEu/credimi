// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for Credentials Issuers.
// It includes the CredentialsIssuersWorkflow, which validates and imports credential issuer metadata.
// The workflow performs various steps including checking the issuer, parsing JSON responses,
// storing credentials, and cleaning up invalid credentials.
package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/forkbombeu/didimo/pkg/internal/temporalclient"
	"github.com/forkbombeu/didimo/pkg/workflowengine"
	"github.com/forkbombeu/didimo/pkg/workflowengine/activities"
	"github.com/forkbombeu/didimo/pkg/workflowengine/workflows/credentials_config"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// CredentialsTaskQueue is the task queue for the credentials workflow.
const CredentialsTaskQueue = "CredentialsTaskQueue"

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
func (w *CredentialsIssuersWorkflow) Workflow(ctx workflow.Context, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())
	checkIssuer := activities.CheckCredentialsIssuerActivity{}
	var issuerResult workflowengine.ActivityResult
	err := workflow.ExecuteActivity(ctx, checkIssuer.Name(), workflowengine.ActivityInput{
		Config: map[string]string{
			"base_url": input.Payload["base_url"].(string),
		},
	}).Get(ctx, &issuerResult)
	if err != nil {
		logger.Error("CheckCredentialIssuer failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	rawJSON, ok := issuerResult.Output.(map[string]any)["rawJSON"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, fmt.Errorf("missing rawJSON in activity output")
	}

	parseJSON := activities.JSONActivity{
		StructRegistry: map[string]reflect.Type{
			"OpenidCredentialIssuerSchemaJson": reflect.TypeOf(credentials_config.OpenidCredentialIssuerSchemaJson{}),
		},
	}
	var result workflowengine.ActivityResult
	var issuerData *credentials_config.OpenidCredentialIssuerSchemaJson
	err = workflow.ExecuteActivity(ctx, parseJSON.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{
			"rawJSON":    rawJSON,
			"structType": "OpenidCredentialIssuerSchemaJson",
		},
	}).Get(ctx, &result)
	if err != nil {
		logger.Error("ParseJSON failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	jsonBytes, err := json.Marshal(result.Output)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	err = json.Unmarshal(jsonBytes, &issuerData)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	logs := make(map[string][]any)

	var validKeys []string
	for credKey, credential := range issuerData.CredentialConfigurationsSupported {

		castedCredential := activities.Credential(credential)
		HTTPActivity := activities.HTTPActivity{}
		storeInput := workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "POST",
				"url": fmt.Sprintf(
					"%s/%s",
					input.Config["app_url"].(string),
					"api/credentials_issuers/store-or-update-extracted-credentials"),
			},
			Payload: map[string]any{
				"body": map[string]any{
					"issuerID":   input.Payload["issuerID"].(string),
					"issuerName": *issuerData.Display[0].Name,
					"credKey":    credKey,
					"credential": castedCredential,
				},
			},
		}
		var storeResponse workflowengine.ActivityResult
		err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), storeInput).Get(ctx, &storeResponse)
		if err != nil {
			return workflowengine.WorkflowResult{Log: logs}, err
		}
		validKeys = append(validKeys, credKey)
		logs["StoredCredentials"] = append(logs["StoredCredentials"], storeResponse.Output.(map[string]any)["body"].(map[string]any)["key"])
	}

	HTTPActivity := activities.HTTPActivity{}
	cleanupInput := workflowengine.ActivityInput{
		Config: map[string]string{
			"method": "POST",
			"url":    fmt.Sprintf("%s/%s", input.Config["app_url"].(string), "api/credentials_issuers/cleanup_credentials"),
		},
		Payload: map[string]any{
			"body": map[string]any{
				"issuerID":  input.Payload["issuerID"].(string),
				"validKeys": validKeys,
			},
		},
	}
	var cleanupResponse workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), cleanupInput).Get(ctx, &cleanupResponse)
	logs["RemovedCredentials"] = append(logs["RemovedCredentials"], cleanupResponse.Output.(map[string]any)["body"].(map[string]any)["deleted"])
	if err != nil {
		return workflowengine.WorkflowResult{Log: logs}, err
	}

	return workflowengine.WorkflowResult{
		Message: "Successfully retrieved and stored and update credentials",
		Log:     logs,
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
	// Load environment variables.
	godotenv.Load()
	namespace := "default"
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	c, err := temporalclient.GetTemporalClientWithNamespace(
		namespace,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("unable to create client: %v", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:        "Credentials-Workflow-" + uuid.NewString(),
		TaskQueue: CredentialsTaskQueue,
	}
	if input.Config["Memo"] != nil {
		workflowOptions.Memo = input.Config["Memo"].(map[string]any)
	}

	// Start the workflow execution.
	_, err = c.ExecuteWorkflow(context.Background(), workflowOptions, w.Name(), input)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to start workflow: %v", err)
	}

	return workflowengine.WorkflowResult{}, nil
}
