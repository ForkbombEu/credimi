// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for Credentials Issuers.
// It includes the WalletWorkflow, which validates and imports credential issuer metadata.
// The workflow performs various steps including checking the issuer, parsing JSON responses,
// storing credentials, and cleaning up invalid credentials.
package workflows

import (
	"fmt"
	"reflect"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// WalletTaskQueue is the task queue for the wallet workflow.
const (
	WalletTaskQueue  = "WalletTaskQueue"
	AppleStoreAPIURL = "https://itunes.apple.com/lookup"
	AppMetadataQuery = "getAppMetadata"
)

// Wallet is a workflow that imports wallet metadata from app stores urls.
type WalletWorkflow struct{}

// Name returns the name of the workflow.
func (w *WalletWorkflow) Name() string {
	return "Import Wallet metadata from an app store URL(Google or Apple)"
}

// GetOptions returns the activity options for the workflow.
func (w *WalletWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *WalletWorkflow) Workflow(
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

	var metadata map[string]any
	var storeType string
	metadataReady := false

	workflow.SetQueryHandler(ctx, AppMetadataQuery, func() (map[string]any, error) {
		if !metadataReady {
			return nil, workflowengine.NotReadyError{}
		}
		return map[string]any{
			"metadata":  metadata,
			"storeType": storeType,
		}, nil
	})
	fullURL, ok := input.Payload["url"].(string)
	if !ok || fullURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"url",
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

	urlParser := activities.NewParseWalletURLActivity()
	var parsedResult workflowengine.ActivityResult
	err := workflow.ExecuteActivity(ctx, urlParser.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{
			"url": fullURL,
		},
	}).Get(ctx, &parsedResult)
	if err != nil {
		logger.Error("ParseWalletURL failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
	apiInput, ok := parsedResult.Output.(map[string]any)["api_input"].(string)
	if !ok {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: api_input", urlParser.Name()),
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	storeType, ok = parsedResult.Output.(map[string]any)["store_type"].(string)
	if !ok {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: store", urlParser.Name()),
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	httpActivity := activities.NewHTTPActivity()

	switch storeType {
	case "apple":
		var response workflowengine.ActivityResult
		err = workflow.ExecuteActivity(ctx, httpActivity.Name(), workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "GET",
				"url":    AppleStoreAPIURL,
			},
			Payload: map[string]any{
				"query_params": map[string]any{
					"id": apiInput,
				},
				"expected_status": 200,
			},
		}).Get(ctx, &response)
		if err != nil {
			logger.Error("HTTP failed", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}
		result, ok := response.Output.(map[string]any)["body"]
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: results", urlParser.Name()),
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}
		metadata = workflowengine.AsSliceOfMaps(result.(map[string]any)["results"])[0]

	case "google":
		docker := activities.NewDockerActivity()
		var result workflowengine.ActivityResult
		err = workflow.ExecuteActivity(ctx, docker.Name(), workflowengine.ActivityInput{
			Payload: map[string]any{
				"image": "ghcr.io/forkbombeu/appraccon:latest",
				"cmd":   []string{apiInput},
			},
		}).Get(ctx, &result)
		if err != nil {
			logger.Error("Docker failed", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}
		stdout, ok := result.Output.(map[string]any)["stdout"].(string)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: results", urlParser.Name()),
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}
		json := activities.NewJSONActivity(map[string]reflect.Type{
			"map": reflect.TypeOf(
				map[string]any{},
			),
		})
		var jsonResult workflowengine.ActivityResult
		err = workflow.ExecuteActivity(ctx, json.Name(), workflowengine.ActivityInput{
			Payload: map[string]any{
				"rawJSON":    stdout,
				"structType": "map",
			},
		}).Get(ctx, &jsonResult)
		if err != nil {
			logger.Error("ParseJSON failed", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
		}
		metadata, ok = jsonResult.Output.(map[string]any)
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: output", json.Name()),
				jsonResult.Output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
		}
	}
	namespace, ok := input.Config["namespace"].(string)
	if !ok || namespace == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"namespace",
			runMetadata,
		)
	}

	metadataReady = true
	// Code to store metdata directly into PB
	/*storeInput := workflowengine.ActivityInput{
		Config: map[string]string{
			"method": "POST",
			"url": fmt.Sprintf(
				"%s/%s",
				appURL,
				"api/wallet/store-or-update-wallet-data"),
		},
		Payload: map[string]any{
			"body": map[string]any{
				"metadata": metadata,
				"url":      fullURL,
				"type":     storeType,
				"orgID":    namespace,
			},
			"expected_status": 200,
		},
	}
	var storeResponse workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, httpActivity.Name(), storeInput).
		Get(ctx, &storeResponse)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	*/
	return workflowengine.WorkflowResult{
		Message: "Worflow completed successfully",
	}, nil
}

// Start initializes and starts the WalletWorkflow execution.
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
func (w *WalletWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "Wallet-Workflow-" + uuid.NewString(),
		TaskQueue:                WalletTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}
