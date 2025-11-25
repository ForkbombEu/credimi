// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for the OpenID certification site.
// It includes the OpenIDNetWorkflow for conformance checks and the OpenIDNetLogsWorkflow
// for draining logs from the OpenID certification site.
package workflows

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// EudiwTaskQueue is the task queue for Eudiw workflows.
const (
	EudiwTaskQueue          = "EUDIWTaskQueue"
	EudiwTemplateFolderPath = "pkg/workflowengine/workflows/eudiw_config"
	EudiwStartCheckSignal   = "start-eudiw-check-signal"
	EudiwStopCheckSignal    = "stop-eudiw-check-signal"
	EudiwSubscription       = "eudiw-logs"
)

// EudiwWorkflow is a workflow that performs conformance checks on the EUDIW suite.
type EudiwWorkflow struct{}

type EudiwWorkflowPayload struct {
	ID       string `json:"id"        yaml:"id"        validate:"required"`
	Nonce    string `json:"nonce"     yaml:"nonce"     validate:"required"`
	UserMail string `json:"user_mail" yaml:"user_mail" validate:"required"`
}

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

	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: utils.JoinURL(
			input.Config["app_url"].(string),
			"my", "tests", "runs",
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}

	payload, err := workflowengine.DecodePayload[EudiwWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			runMetadata,
		)
	}
	template, ok := input.Config["template"].(string)
	if !ok || template == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"template",
			runMetadata,
		)
	}

	stepCIWorkflowActivity := activities.NewStepCIWorkflowActivity()
	stepCIInput := workflowengine.ActivityInput{
		Payload: activities.StepCIWorkflowActivityPayload{
			Data: map[string]any{
				"nonce": payload.Nonce,
				"id":    payload.ID,
			},
		},
		Config: map[string]string{
			"template": template,
		},
	}
	err = stepCIWorkflowActivity.Configure(&stepCIInput)
	if err != nil {
		logger.Error(" StepCI configure failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	var stepCIResult workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	result, ok := stepCIResult.Output.(map[string]any)["captures"].(map[string]any)
	if !ok {
		msg := fmt.Sprintf("unexpected output type: %T", stepCIResult.Output)
		return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
			msg,
			stepCIResult.Output,
			runMetadata,
		)
	}
	clientID, ok := result["client_id"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
			"client_id",
			stepCIResult.Output,
			runMetadata,
		)
	}
	requestURI, ok := result["request_uri"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
			"request_uri",
			stepCIResult.Output,
			runMetadata,
		)
	}
	transactionID, ok := result["transaction_id"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
			"transaction_id",
			stepCIResult.Output,
			runMetadata,
		)
	}
	baseURL := utils.JoinURL(
		input.Config["app_url"].(string),
		"tests",
		"wallet",
		"eudiw",
	) // TODO use the correct one
	u, err := url.Parse(baseURL)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
		appErr := workflowengine.NewAppError(errCode, baseURL)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	qr, err := BuildQRDeepLink(
		clientID,
		requestURI,
	)
	if err != nil {
		logger.Error("Failed to build QR deep link", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	query := u.Query()
	query.Set("workflow-id", workflow.GetInfo(ctx).WorkflowExecution.ID)
	query.Set("qr", qr)
	query.Set("namespace", input.Config["namespace"].(string))
	u.RawQuery = query.Encode()
	emailActivity := activities.NewSendMailActivity()

	emailInput := workflowengine.ActivityInput{
		Payload: activities.SendMailActivityPayload{
			Recipient: payload.UserMail,
			Subject:   "[CREDIMI] Action required to continue your conformance checks",
			Template:  activities.ContinueConformanceCheckEmailTemplate,
			Data: map[string]any{
				"AppName":          input.Config["app_name"],
				"AppLogo":          input.Config["app_logo"],
				"UserName":         input.Config["user_name"],
				"VerificationLink": u.String(),
			},
		},
	}
	err = emailActivity.Configure(&emailInput)
	if err != nil {
		logger.Error("Email activity configure failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	err = workflow.ExecuteActivity(ctx, emailActivity.Name(), emailInput).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to send mail to user ", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
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
		selector.AddFuture(timerFuture, func(_ workflow.Future) {
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

		HTTPActivity := activities.NewHTTPActivity()
		var checkResponse workflowengine.ActivityResult
		CheckStatusInput := workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodGet,
				URL: utils.JoinURL(
					"https://verifier-backend.eudiw.dev/ui/presentations",
					transactionID,
				),
			},
		}
		err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), CheckStatusInput).
			Get(ctx, &checkResponse)
		if err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}

		errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPResponse]
		outputMap, ok := checkResponse.Output.(map[string]any)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("unexpected output type: %T", checkResponse.Output),
				checkResponse.Output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}

		statusCode, ok := outputMap["status"].(float64)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				"missing or invalid status code",
				checkResponse.Output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}
		var events []map[string]any
		var eventsResponse workflowengine.ActivityResult
		getLogsInput := workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodGet,
				URL: utils.JoinURL(
					"https://verifier-backend.eudiw.dev/ui/presentations",
					transactionID,
					"events",
				),
				ExpectedStatus: 200,
			},
		}
		err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), getLogsInput).
			Get(ctx, &eventsResponse)
		if err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}
		events = workflowengine.AsSliceOfMaps(
			eventsResponse.Output.(map[string]any)["body"].(map[string]any)["events"],
		)
		triggerLogsInput := workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					input.Config["app_url"].(string),
					"api", "compliance", "send-eudiw-log-update",
				),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"workflow_id": workflow.GetInfo(ctx).WorkflowExecution.ID,
					"logs":        events,
				},
				ExpectedStatus: 200,
			},
		}

		err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), triggerLogsInput).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to send logs", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}
		errCode = errorcodes.Codes[errorcodes.EudiwCheckFailed]
		switch int(statusCode) {
		case 200:
			return workflowengine.WorkflowResult{
				Message: "Eudiw check completed successfully",
			}, nil

		case 400:
			continue

		case 500:
			failedErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"eudiw check failed with status code: %d",
					int(statusCode),
				),
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				failedErr,
				runMetadata,
			)

		default:
			failedErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"unexpected status code: %d",
					int(statusCode),
				),
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				failedErr,
				runMetadata,
			)
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
	namespace := DefaultNamespace
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	return workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
func BuildQRDeepLink(
	clientID, requestURI string,
) (string, error) {
	baseURL := "eudi-openid4vp://?client_id=%s&request_uri=%s"
	u, err := url.Parse(
		fmt.Sprintf(baseURL, url.QueryEscape(clientID), url.QueryEscape(requestURI)),
	)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
		appErr := workflowengine.NewAppError(errCode, baseURL)
		return "", workflowengine.NewWorkflowError(appErr, workflowengine.WorkflowErrorMetadata{
			WorkflowName: "EudiwWorkflow",
		})
	}
	return u.String(), nil
}
