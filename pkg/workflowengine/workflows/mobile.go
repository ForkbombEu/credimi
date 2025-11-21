// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi-extra/mobile"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

const MobileAutomationTaskQueue = "MobileAutomationTaskQueue"

// MobileAutomationWorkflow is a workflow that runs a mobile automation flow
type MobileAutomationWorkflow struct{}

// MobileAutomationWorkflowPayload is the payload for the mobile automation workflow
type MobileAutomationWorkflowPayload struct {
	ActionID         string            `json:"action_id,omitempty"          yaml:"action_id,omitempty"`
	VersionID        string            `json:"version_id,omitempty"         yaml:"version_id,omitempty"`
	ActionCode       string            `json:"action_code,omitempty"        yaml:"action_code,omitempty"`
	Video            bool              `json:"video,omitempty"              yaml:"video,omitempty"`
	StoredActionCode bool              `json:"stored_action_code,omitempty" yaml:"stored_action_code,omitempty"`
	EmulatorSerial   string            `json:"emulator_serial,omitempty"    yaml:"emulator_serial,omitempty"`
	Parameters       map[string]string `json:"parameters,omitempty"         yaml:"parameters,omitempty"`
}

type MobileAutomationWorkflowPipelinePayload struct {
	ActionID   string            `json:"action_id,omitempty"   yaml:"action_id,omitempty"`
	VersionID  string            `json:"version_id,omitempty"  yaml:"version_id,omitempty"`
	ActionCode string            `json:"action_code,omitempty" yaml:"action_code,omitempty"`
	Video      bool              `json:"video,omitempty"       yaml:"video,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
}

func (MobileAutomationWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

type MobileWorkflowOutput struct {
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

	var output MobileWorkflowOutput
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

	payload, err := workflowengine.DecodePayload[MobileAutomationWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
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
	mobileServerURL := utils.GetEnvironmentVariable("MAESTRO_WORKER", "http://localhost:8050")

	mobileActivity := activities.NewRunMobileFlowActivity()
	var mobileResponse workflowengine.ActivityResult
	mobileInput := workflowengine.ActivityInput{
		Payload: mobile.RunMobileFlowPayload{
			EmulatorSerial: payload.EmulatorSerial,
			Yaml:           payload.ActionCode,
			Recorded:       payload.Video,
			Parameters:     payload.Parameters,
			WorkflowId:     workflow.GetInfo(ctx).WorkflowExecution.ID,
		},
	}
	executeErr := workflow.ExecuteActivity(ctx, mobileActivity.Name(), mobileInput).
		Get(ctx, &mobileResponse)
	output.FlowOutput = mobileResponse.Output

	if payload.Video {
		var checkVideoResult workflowengine.ActivityResult
		checkVideoInput := workflowengine.ActivityInput{
			Payload: activities.CheckFileExistsActivityPayload{
				Path: filepath.Join("/credimi/workflows", runMetadata.WorkflowID, "video.mp4"),
			},
		}
		checkVideoActivity := activities.NewCheckFileExistsActivity()
		err := workflow.ExecuteActivity(ctx, checkVideoActivity.Name(), checkVideoInput).
			Get(ctx, &checkVideoResult)
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
				Payload: activities.HTTPActivityPayload{
					Method: http.MethodPost,
					URL: fmt.Sprintf(
						"%s/%s",
						mobileServerURL,
						"store-action-result",
					),
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: map[string]any{
						"result_path": filepath.Join(
							"/credimi",
							"workflows",
							runMetadata.WorkflowID,
							"video.mp4",
						),
						"action_identifier": payload.ActionID,
					},
					ExpectedStatus: 200,
				},
			}
			if !payload.StoredActionCode {
				walletIdentifier := deriveWalletIdentifier(payload.VersionID)
				storeResultInput.Payload.(activities.HTTPActivityPayload).
					Body.(map[string]any)["wallet_identifier"] = walletIdentifier
				storeResultInput.Payload.(activities.HTTPActivityPayload).Body.(map[string]any)["action_code"] = payload.ActionCode
			}
			HTTPActivity := activities.NewHTTPActivity()
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
				errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
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
