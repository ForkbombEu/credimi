// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	temporalclient "github.com/forkbombeu/didimo/pkg/internal/temporal_client"
	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
	"github.com/forkbombeu/didimo/pkg/workflow_engine/activities"
	"github.com/forkbombeu/didimo/pkg/workflow_engine/workflows/credentials_config"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const CredentialsTaskQueue = "CredentialsTaskQueue"

type CredentialsIssuersWorkflow struct{}

func (w *CredentialsIssuersWorkflow) Name() string {
	return "Validate and import Credential Issuer metadata"
}

func (w *CredentialsIssuersWorkflow) GetOptions() workflow.ActivityOptions {
	return ActivityOptions
}

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
		return workflowengine.WorkflowResult{}, fmt.Errorf("Missing rawJSON in activity output")
	}

	parseJSON := activities.JsonActivity{
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
		HTTPActivity := activities.HttpActivity{}
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

	HTTPActivity := activities.HttpActivity{}
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
