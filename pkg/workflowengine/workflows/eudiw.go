// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for the OpenID certification site.
// It includes the OpenIDNetWorkflow for conformance checks and the OpenIDNetLogsWorkflow
// for draining logs from the OpenID certification site.
package workflows

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
)

// EudiwTaskQueue is the task queue for Eudiw workflows.
const EudiwTaskQueue = "EUDIWTaskQueue"

// EudiwWorkflow is a workflow that performs conformance checks on the OpenID certification site.
type EudiwWorkflow struct{}

// Name returns the name of the EudiwWorkflow.
func (EudiwWorkflow) Name() string {
	return "Conformance check on EUDIW"
}

// GetOptions Configure sets up the workflow with the necessary options.
func (EudiwWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

type EudiwResponseBody struct {
	Status    string   `json:"status"`
	Reason    string   `json:"reason"`
	SessionID string   `json:"sessionId"`
	Claims    []string `json:"claims,omitempty"`
}

const EudiwStartCheckSignal = "start-eudiw-check-signal"
const EudiwStopCheckSignal = "stop-eudiw-check-signal"

// Workflow is the main workflow function for the EudiwWorkflow. It orchestrates
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
//  4. Wait for a signal to start polling the API to get the current status of the check.
//  5. Wait for either a signal to pause the workflow or the check result from the API
//  6. Process the response to determine the success or failure of the workflow.
//
// Notes:
//   - The workflow uses a selector to wait for either a signal or the next API call.
//   - If the signal data indicates failure, the workflow terminates with a failure message.
func (w *EudiwWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	stepCIWorkflowActivity := activities.StepCIWorkflowActivity{}
	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"nonce": input.Payload["nonce"].(string),
			"id":    input.Payload["id"].(string),
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
		return workflowengine.WorkflowResult{}, fmt.Errorf("unexpected output type: %T", stepCIResult.Output)
	}
	clientID, ok := result["client_id"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, fmt.Errorf("missing client_id in stepci response")
	}
	requestUri, ok := result["request_uri"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, fmt.Errorf("missing request_uri in stepci response")
	}
	transactionID, ok := result["transaction_id"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, fmt.Errorf("missing transaction_id in stepci response")
	}
	baseURL := input.Payload["app_url"].(string) + "/tests/wallet/eudiw" //TODO use the correct one
	u, err := url.Parse(baseURL)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("unexpected error parsing URL: %v", err)
	}
	qr := fmt.Sprintf(
		"eudi-openid4vp://?client_id=%s&request_uri=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(requestUri),
	)
	query := u.Query()
	query.Set("workflow-id", workflow.GetInfo(ctx).WorkflowExecution.ID)
	query.Set("qr", qr)
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

	startSignalChan := workflow.GetSignalChannel(ctx, EudiwStartCheckSignal)
	stopSignalChan := workflow.GetSignalChannel(ctx, EudiwStopCheckSignal)
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

		var HTTPActivity activities.HTTPActivity
		var checkResponse workflowengine.ActivityResult
		CheckStatusInput := workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "GET",
				"url":    fmt.Sprintf("https://verifier-backend.eudiw.dev/ui/presentations/%s", transactionID),
			},
		}
		err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), CheckStatusInput).Get(ctx, &checkResponse)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
		outputMap, ok := checkResponse.Output.(map[string]any)
		if !ok {
			return workflowengine.WorkflowResult{}, fmt.Errorf("unexpected output type: %T", checkResponse.Output)
		}

		statusCode, ok := outputMap["status"].(float64)
		if !ok {
			return workflowengine.WorkflowResult{}, fmt.Errorf("missing or invalid status_code in response")
		}
		var events []map[string]any
		var eventsResponse workflowengine.ActivityResult
		getLogsInput := workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "GET",
				"url":    fmt.Sprintf("https://verifier-backend.eudiw.dev/ui/presentations/%s/events", transactionID),
			},
		}
		err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), getLogsInput).Get(ctx, &eventsResponse)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
		events = AsSliceOfMaps(eventsResponse.Output.(map[string]any)["body"].(map[string]any)["events"])
		triggerLogsInput := workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "POST",
				"url": fmt.Sprintf(
					"%s/%s",
					input.Payload["app_url"].(string),
					"api/compliance/send-eudiw-log-update",
				),
			},
			Payload: map[string]any{
				"headers": map[string]any{
					"Content-Type": "application/json",
				},
				"body": map[string]any{
					"workflow_id": workflow.GetInfo(ctx).WorkflowExecution.ID,
					"logs":        events,
				},
			},
		}

		err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), triggerLogsInput).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to send logs", "error", err)
			return workflowengine.WorkflowResult{}, err
		}
		switch int(statusCode) {
		case 200:
			return workflowengine.WorkflowResult{
				Message: "Eudiw check completed successfully",
			}, nil

		case 400:
			continue

		case 500:
			return workflowengine.WorkflowResult{
				Message: fmt.Sprintf("Eudiw check failed with status code %d", int(statusCode)),
			}, nil

		default:
			return workflowengine.WorkflowResult{}, fmt.Errorf("unexpected status code: %d", int(statusCode))
		}

	}
}

// Start initializes and starts the EudiwWorkflow execution.
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
func (w *EudiwWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "EudiWWorkflow" + uuid.NewString(),
		TaskQueue:                EudiwTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}
