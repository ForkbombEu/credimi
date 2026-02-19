// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
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

// EWCWorkflow is a workflow that performs conformance checks on the EWC suite.
type EWCWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

var ewcStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

type EWCWorkflowPayload struct {
	SessionID string `json:"session_id" yaml:"session_id" validate:"required"`
	UserMail  string `json:"user_mail"  yaml:"user_mail"  validate:"required"`
}

func NewEWCWorkflow() *EWCWorkflow {
	w := &EWCWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

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

func (w *EWCWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// ExecuteWorkflow is the main workflow function for the EWCWorkflow. It orchestrates
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
func (w *EWCWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	payload, err := workflowengine.DecodePayload[EWCWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	template, ok := input.Config["template"].(string)
	if !ok || template == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"template",
			input.RunMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}
	checkEndpoint, ok := input.Config["check_endpoint"].(string)
	if !ok || checkEndpoint == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"check_endpoint",
			input.RunMetadata,
		)
	}
	stepCIPayload := activities.StepCIWorkflowActivityPayload{
		Data: map[string]any{"session_id": payload.SessionID},
	}
	suite, ok := input.Config["memo"].(map[string]any)["author"].(string)
	if !ok {
		return workflowengine.WorkflowResult{},
			workflowengine.NewMissingConfigError(
				"author",
				input.RunMetadata,
			)
	}
	cfg := StepCIAndEmailConfig{
		AppURL:        appURL,
		AppName:       input.Config["app_name"].(string),
		AppLogo:       input.Config["app_logo"].(string),
		UserName:      input.Config["user_name"].(string),
		UserMail:      payload.UserMail,
		Template:      template,
		StepCIPayload: stepCIPayload,
		Namespace:     input.Config["namespace"].(string),
		RunMetadata:   input.RunMetadata,
		Suite:         suite,
	}

	result, err := RunStepCIAndSendMail(ctx, cfg)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	sessionID, ok := result.Captures["session_id"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
			"session_id",
			result.Captures,
			input.RunMetadata,
		)
	}

	interval := time.Second
	workflowResult, err := pollEWCCheck(
		ctx,
		interval,
		checkEndpoint,
		sessionID,
		input.RunMetadata,
		false,
	)
	return workflowResult, err
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
	namespace := DefaultNamespace
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	return ewcStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

// EWCStatusWorkflow is a workflow that checks the status of an EWC check.
type EWCStatusWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// EWCStatusWorkflowPayload is the payload for the EWCStatusWorkflow.
type EWCStatusWorkflowPayload struct {
	SessionID string `json:"session_id"`
}

func NewEWCStatusWorkflow() *EWCStatusWorkflow {
	w := &EWCStatusWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

// Name returns the human-readable name for this workflow.
func (EWCStatusWorkflow) Name() string {
	return "Drain  EWC check status conformance endpoint"
}

// GetOptions returns the default activity options for this workflow.
func (EWCStatusWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *EWCStatusWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// Workflow continuously polls EWC Status and pushes updates to the backend.
// It can be paused/resumed by Temporal signals.
func (w *EWCStatusWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	payload, err := workflowengine.DecodePayload[EWCStatusWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}

	interval := time.Second
	if v, ok := input.Config["interval"].(float64); ok {
		interval = time.Duration(v)
	}

	checkEndpoint, ok := input.Config["check_endpoint"].(string)
	if !ok || checkEndpoint == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"check_endpoint",
			input.RunMetadata,
		)
	}

	result, err := pollEWCCheck(
		ctx,
		interval,
		checkEndpoint,
		payload.SessionID,
		input.RunMetadata,
		true,
	)
	return result, err
}

func pollEWCCheck(
	ctx workflow.Context,
	interval time.Duration,
	checkEndpoint string,
	sessionID string,
	runMetadata *workflowengine.WorkflowErrorMetadata,
	startImmediately bool,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	startSignalChan := workflow.GetSignalChannel(ctx, EwcStartCheckSignal)
	stopSignalChan := workflow.GetSignalChannel(ctx, EwcStopCheckSignal)
	pipelineCancelChan := workflow.GetSignalChannel(ctx, PipelineCancelSignal)
	selector := workflow.NewSelector(ctx)

	// If flag is true → start polling right away
	isPolling := startImmediately
	var canceled bool

	var timerFuture workflow.Future
	var startTimer func()

	startTimer = func() {
		timerCtx, _ := workflow.WithCancel(ctx)
		timerFuture = workflow.NewTimer(timerCtx, interval)
		selector.AddFuture(timerFuture, func(_ workflow.Future) {
			if isPolling {
				startTimer()
			}
		})
	}

	// Automatically start timer if startImmediately == true
	if startImmediately {
		startTimer()
		logger.Info("EWC polling started (startImmediately=true)")
	}

	httpInput := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodGet,
			URL:    checkEndpoint,
			QueryParams: map[string]string{
				"sessionId": sessionID,
			},
			ExpectedStatus: 200,
		},
	}
	var signalData struct{}
	selector.AddReceive(pipelineCancelChan, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &signalData)
		canceled = true
	})
	selector.AddReceive(startSignalChan, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &signalData)

		if !isPolling {
			isPolling = true
			startTimer()
			logger.Info("EWC polling started (signal)")
		}
	})

	selector.AddReceive(stopSignalChan, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &signalData)
		isPolling = false
		logger.Info("EWC polling stopped (signal)")
	})

	for {
		selector.Select(ctx)

		if canceled {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowCancellationError(
				runMetadata,
			)
		}

		if !isPolling {
			continue
		}

		httpActivity := activities.NewHTTPActivity()
		var response workflowengine.ActivityResult

		err := workflow.ExecuteActivity(ctx, httpActivity.Name(), httpInput).Get(ctx, &response)
		if err != nil {
			logger.Error("EWC HTTP check failed", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}

		bodyJSON, err := json.Marshal(response.Output.(map[string]any)["body"])
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
			appErr := workflowengine.NewAppError(errCode, err.Error(), response.Output)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}

		var parsed EWCResponseBody
		if err := json.Unmarshal(bodyJSON, &parsed); err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
			appErr := workflowengine.NewAppError(errCode, err.Error(), bodyJSON)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}

		errCode := errorcodes.Codes[errorcodes.EWCCheckFailed]
		failedErr := workflowengine.NewAppError(errCode, parsed.Reason, parsed)

		switch parsed.Status {
		case "success":
			return workflowengine.WorkflowResult{
				Message: "EWC check completed successfully",
				Log:     parsed.Claims,
			}, nil

		case "pending":
			if parsed.Reason != "ok" {
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					failedErr,
					runMetadata,
				)
			}
			// continue polling

		case "failed":
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				failedErr,
				runMetadata,
			)

		default:
			failedErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("unexpected status from '%s': %s", checkEndpoint, parsed.Status),
				parsed,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				failedErr,
				runMetadata,
			)
		}
	}
}
