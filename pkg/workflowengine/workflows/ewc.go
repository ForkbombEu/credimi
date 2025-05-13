// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for the OpenID certification site.
// It includes the OpenIDNetWorkflow for conformance checks and the OpenIDNetLogsWorkflow
// for draining logs from the OpenID certification site.
package workflows

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// EWCTaskQueue is the task queue for EWC workflows.
const (
	EWCTaskQueue          = "EWCTaskQueue"
	EWCTemplateFolderPath = "pkg/workflowengine/workflows/ewc_config"
	EwcStartCheckSignal   = "start-ewc-check-signal"
	EwcStopCheckSignal    = "stop-ewc-check-signal"
)

// EWCWorkflow is a workflow that performs conformance checks on the OpenID certification site.
type EWCWorkflow struct{}

// Name returns the name of the EWCWorkflow.
func (EWCWorkflow) Name() string {
	return "Conformance check on EWC"
}

// GetOptions Configure sets up the workflow with the necessary options.
func (EWCWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

type EWCResponseBody struct {
	Status    string   `json:"status"`
	Reason    string   `json:"reason"`
	SessionID string   `json:"sessionId"`
	Claims    []string `json:"claims,omitempty"`
}

// Workflow is the main workflow function for the EWCWorkflow. It orchestrates
// the execution of various activities to perform conformance checks
// and send notifications to the user.
//
// Parameters:
//   - ctx: The workflow context used to manage workflow execution.
//   - input: The input data for the workflow, including payload and configuration.
//
// Returns:
//   - workflowengine.WorkflowResult: The result of the workflow execution, including
//     a message and log data.
//   - error: An error if the workflow fails at any step.
//
// Workflow Steps:
//  1. Execute the StepCIWorkflowActivity to perform initial checks and gets the QR code deep link.
//  2. Generate a URL with query parameters for the user to continue the process.
//  3. Configure and execute the SendMailActivity to notify the user via email.
//  4. Wait for a signal ("ewc-check-started") to start polling the API to getthe current status of the check.
//  5. Wait for either a signal ("ewc-check-stopped") to pause the workflow or the check result from the API
//  6. Process the response to determine the success or failure of the workflow.
//
// Notes:
//   - The workflow uses a selector to wait for either a signal or the next API call.
//   - If the signal data indicates failure, the workflow terminates with a failure message.
func (w *EWCWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	stepCIWorkflowActivity := activities.StepCIWorkflowActivity{}
	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"session_id": input.Payload["session_id"].(string),
		},
		Config: map[string]string{
			"template": input.Config["template"].(string),
		},
	}
	err := stepCIWorkflowActivity.Configure(&stepCIInput)
	if err != nil {
		logger.Error(" StepCI configure failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	var stepCIResult workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	result, ok := stepCIResult.Output.(map[string]any)
	if !ok {
		return workflowengine.WorkflowResult{}, fmt.Errorf(
			"unexpected output type: %T",
			stepCIResult.Output,
		)
	}
	deepLink, ok := result["deep_link"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, fmt.Errorf("missing deep_link in stepci response")
	}
	sessionID, ok := result["session_id"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, fmt.Errorf("missing session_id in stepci response")
	}
	baseURL := input.Payload["app_url"].(string) + "/tests/wallet/ewc"
	u, err := url.Parse(baseURL)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("unexpected error parsing URL: %w", err)
	}
	query := u.Query()
	query.Set("workflow-id", workflow.GetInfo(ctx).WorkflowExecution.ID)
	query.Set("qr", deepLink)
	u.RawQuery = query.Encode()
	emailActivity := activities.SendMailActivity{}

	emailInput := workflowengine.ActivityInput{
		Config: map[string]string{
			"recipient": input.Payload["user_mail"].(string),
		},
		Payload: map[string]any{
			"subject": "[CREDIMI] Action required to continue your conformance checks",
			"body": fmt.Sprintf(`
		<html>
			<body>
				<p>Please click on the following link:</p>
				<p><a href="%s" target="_blank" rel="noopener">%s</a></p>
			</body>
		</html>
	`, u.String(), u.String()),
		},
	}
	err = emailActivity.Configure(&emailInput)
	if err != nil {
		logger.Error("Email activity configure failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	err = workflow.ExecuteActivity(ctx, emailActivity.Name(), emailInput).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to send mail to user ", "error", err)
		return workflowengine.WorkflowResult{}, err
	}

	startSignalChan := workflow.GetSignalChannel(ctx, EwcStartCheckSignal)
	stopSignalChan := workflow.GetSignalChannel(ctx, EwcStopCheckSignal)
	selector := workflow.NewSelector(ctx)
	var isPolling bool
	var timerFuture workflow.Future
	var startTimer func()
	startTimer = func() {
		timerCtx, _ := workflow.WithCancel(ctx)
		timerFuture = workflow.NewTimer(timerCtx, time.Second)
		selector.AddFuture(timerFuture, func(f workflow.Future) {
			if isPolling {
				startTimer()
			}
		})
	}

	for {
		selector.AddReceive(startSignalChan, func(c workflow.ReceiveChannel, _ bool) {
			var signalData struct{}
			c.Receive(ctx, &signalData)
			isPolling = true
			startTimer()
		})

		selector.AddReceive(stopSignalChan, func(c workflow.ReceiveChannel, _ bool) {
			var signalData struct{}
			c.Receive(ctx, &signalData)
			isPolling = false
		})

		selector.Select(ctx)

		if !isPolling {
			continue
		}

		var HTTPGetActivity activities.HTTPActivity
		var response workflowengine.ActivityResult
		HTTPInput := workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "GET",
				"url":    input.Config["check_endpoint"].(string),
			},
			Payload: map[string]any{
				"query_params": map[string]any{
					"sessionId": sessionID,
				},
			},
		}
		err := workflow.ExecuteActivity(ctx, HTTPGetActivity.Name(), HTTPInput).Get(ctx, &response)
		if err != nil {
			logger.Error("HTTP GET failed", "error", err)
			return workflowengine.WorkflowResult{}, err
		}
		bodyJSON, err := json.Marshal(response.Output.(map[string]any)["body"])
		if err != nil {
			return workflowengine.WorkflowResult{}, fmt.Errorf("failed to re-marshal body: %w", err)
		}
		var parsed EWCResponseBody
		err = json.Unmarshal(bodyJSON, &parsed)
		if err != nil {
			logger.Error("Failed to parse JSON response", "error", err)
			return workflowengine.WorkflowResult{}, err
		}

		switch parsed.Status {
		case "success":
			return workflowengine.WorkflowResult{
				Message: "EWC check completed successfully",
				Log:     parsed.Claims,
			}, nil

		case "pending":
			if parsed.Reason != "ok" {
				return workflowengine.WorkflowResult{}, fmt.Errorf(
					"EWC check failed: %s",
					parsed.Reason,
				)
			}
		case "failed":
			return workflowengine.WorkflowResult{}, fmt.Errorf(
				"EWC check failed: %s",
				parsed.Reason,
			)

		default:
			return workflowengine.WorkflowResult{}, fmt.Errorf(
				"unexpected status from '%s': %s",
				input.Config["check_endpoint"].(string),
				parsed.Status,
			)
		}
	}
}

// Start initializes and starts the EWCWorkflow execution.
// It loads environment variables, configures the Temporal client with the specified namespace,
// and sets up workflow options including a unique workflow ID and optional memo.
// The workflow is then executed with the provided input.
//
// Parameters:
//   - input: A WorkflowInput object containing configuration and input data for the workflow.
//
// Returns:
//   - result: A WorkflowResult object (currently empty in this implementation).
//   - err: An error if the workflow fails to start or if there is an issue with the Temporal client.
//
// Notes:
//   - The namespace defaults to "default" if not provided in the input configuration.
//   - The workflow ID is generated using a UUID to ensure uniqueness.
//   - The Temporal client is closed after the workflow execution is initiated.
func (w *EWCWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "EWCWorkflow" + uuid.NewString(),
		TaskQueue:                EWCTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}
