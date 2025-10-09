// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

type MobileAutomationWorkflow struct {
}

func (MobileAutomationWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (MobileAutomationWorkflow) Name() string {
	return "Run a mobile automation workflow"
}
func (w *MobileAutomationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)
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
	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			runMetadata,
		)
	}
	actionID, ok := input.Payload["action_id"].(string)
	if !ok || actionID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"action_uid",
			runMetadata,
		)
	}
	walletID, ok := input.Payload["wallet_id"].(string)
	if !ok || walletID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"wallet_id",
			runMetadata,
		)
	}
	var HTTPActivity = activities.NewHTTPActivity()
	var response workflowengine.ActivityResult
	getDataInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": "POST",
			"url": fmt.Sprintf(
				"%s/%s",
				input.Config["app_url"].(string),
				"api/wallet/get-apk-and-action",
			),
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": map[string]any{
				"wallet_identifier": walletID,
				"action_identifier": actionID,
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
	ActionYAML, ok := response.Output.(map[string]any)["body"].(map[string]any)["code"].(string)
	if !ok || ActionYAML == "" {
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
	mobileActivity := activities.NewMobileFlowActivity()
	var mobileResponse workflowengine.ActivityResult
	mobileInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"apk":  apkPath,
			"yaml": ActionYAML,
		},
	}
	err = workflow.ExecuteActivity(ctx, mobileActivity.Name(), mobileInput).
		Get(ctx, &mobileResponse)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			runMetadata,
		)
	}
	storeResultInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": "POST",
			"url": fmt.Sprintf(
				"%s/%s",
				input.Config["app_url"].(string),
				"api/wallet/store-action-result",
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
	err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), storeResultInput).
		Get(ctx, &response)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			runMetadata,
		)
	}
	return workflowengine.WorkflowResult{
		Output: mobileResponse.Output,
	}, nil
}
