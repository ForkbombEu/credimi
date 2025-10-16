// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

const MobileAutomationTaskQueue = "MobileAutomationTaskQueue"

type MobileAutomationWorkflow struct {
}

func (MobileAutomationWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

type MobileWorflowOutput struct {
	TestRunURL     string `json:"test_run_url"`
	ResultVideoURL string `json:"result_video_url,omitempty"`
	FlowOutput     any    `json:"flow_output,omitempty"`
}

func (MobileAutomationWorkflow) Name() string {
	return "Run a mobile automation workflow"
}
func (w *MobileAutomationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

	var output MobileWorflowOutput
	testRunURL := fmt.Sprintf(
		"%s/my/tests/runs/%s/%s",
		input.Config["app_url"],
		workflow.GetInfo(ctx).WorkflowExecution.ID,
		workflow.GetInfo(ctx).WorkflowExecution.RunID,
	)
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI:   testRunURL,
	}
	output.TestRunURL = testRunURL

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			runMetadata,
		)
	}
	mobileServerURL := utils.GetEnvironmentVariable("MAESTRO_WORKER", "http://localhost:8050")
	var actionID string
	actionCode, actionCodeOk := input.Payload["action_code"].(string)
	versionID, versionIDOk := input.Payload["version_id"].(string)

	// If action_code is present, version_id is REQUIRED
	if actionCodeOk && actionCode != "" {
		if !versionIDOk || versionID == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
				"version_id",
				runMetadata,
			)
		}
	}
	// If action_code is NOT present -> action_id is REQUIRED
	if !actionCodeOk || actionCode == "" {
		actionIDValue, actionIDOk := input.Payload["action_id"].(string)
		if !actionIDOk || actionIDValue == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
				"action_id",
				runMetadata,
			)
		}
		actionID = actionIDValue

		if !versionIDOk {
			versionID = ""
		}
	}

	recorded, _ := input.Payload["recorded"].(bool)
	var parameters map[string]any
	if rawParams, exists := input.Payload["parameters"]; exists {
		var ok bool
		parameters, ok = rawParams.(map[string]any)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
				"parameters",
				runMetadata,
			)
		}
	}
	var HTTPActivity = activities.NewHTTPActivity()
	var response workflowengine.ActivityResult
	getDataInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": "POST",
			"url": fmt.Sprintf(
				"%s/%s",
				mobileServerURL,
				"fetch-apk-and-action",
			),
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": map[string]any{
				"version_identifier": versionID,
				"action_identifier":  actionID,
			},
			"expected_status": 200,
		},
	}
	err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), getDataInput).
		Get(ctx, &response)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			runMetadata,
		)
	}
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityError]
	apkPath, ok := response.Output.(map[string]any)["body"].(map[string]any)["apk_path"].(string)
	if !ok || apkPath == "" {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: 'apk_path'", errCode.Description),
			response.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			appErr,
			runMetadata,
		)
	}

	if actionCode == "" {
		actionCode, ok = response.Output.(map[string]any)["body"].(map[string]any)["code"].(string)
		if !ok || actionCode == "" {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: 'code'", errCode.Description),
				response.Output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}
	}
	mobileActivity := activities.NewMobileFlowActivity()
	var mobileResponse workflowengine.ActivityResult
	mobileInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"apk":        apkPath,
			"yaml":       actionCode,
			"recorded":   recorded,
			"parameters": parameters,
		},
	}
	executeErr := workflow.ExecuteActivity(ctx, mobileActivity.Name(), mobileInput).
		Get(ctx, &mobileResponse)
	var checkVideoResult workflowengine.ActivityResult
	checkVideoInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"path": "/tmp/credimi/video.mp4",
		},
	}
	output.FlowOutput = mobileResponse.Output
	if recorded {

		checkVideoActivity := activities.NewCheckFileExistsActivity()
		err = workflow.ExecuteActivity(ctx, checkVideoActivity.Name(), checkVideoInput).Get(ctx, &checkVideoResult)
		if err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}
		videoExists, ok := checkVideoResult.Output.(bool)
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				workflowengine.NewAppError(
					errCode,
					fmt.Sprintf(
						"%s: 'output' must be a boolean", errCode.Description),
					checkVideoResult,
				),
				runMetadata,
			)
		}
		if videoExists {
			storeResultInput := workflowengine.ActivityInput{
				Payload: map[string]any{
					"method": "POST",
					"url": fmt.Sprintf(
						"%s/%s",
						mobileServerURL,
						"store-action-result",
					),
					"headers": map[string]any{
						"Content-Type": "application/json",
					},
					"body": map[string]any{
						"result_path":       "/tmp/credimi/video.mp4",
						"action_identifier": actionID,
					},
					"expected_status": 200,
				},
			}
			if actionCodeOk {
				walletIdentifier := deriveWalletIdentifier(versionID)
				storeResultInput.Payload["body"].(map[string]any)["wallet_identifier"] = walletIdentifier
				storeResultInput.Payload["body"].(map[string]any)["action_code"] = actionCode
			}
			var storeResultResponse workflowengine.ActivityResult
			err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), storeResultInput).
				Get(ctx, &storeResultResponse)
			if err != nil {
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					err,
					runMetadata,
				)
			}
			resultURL, ok := storeResultResponse.Output.(map[string]any)["body"].(map[string]any)["result_url"].(string)
			if !ok || resultURL == "" {
				appErr := workflowengine.NewAppError(
					errCode,
					fmt.Sprintf(
						"%s: 'result_url'", errCode.Description),
					storeResultResponse.Output,
				)
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					appErr,
					runMetadata,
				)
			}
			output.ResultVideoURL = resultURL
		}
	}
	if executeErr != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			executeErr,
			runMetadata,
			map[string]any{
				"output": output,
			},
		)
	}

	return workflowengine.WorkflowResult{
		Output: output,
	}, nil
}

func deriveWalletIdentifier(versionID string) string {
	if versionID == "" {
		return ""
	}
	parts := strings.Split(versionID, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "/")
}
