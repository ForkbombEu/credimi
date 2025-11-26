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
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SignalData represents the data structure for signals used in the workflow.
type SignalData struct {
	Success bool
	Reason  string
}

// OpenIDNetTaskQueue is the task queue for OpenIDNet workflows.
const (
	OpenIDNetTaskQueue              = "OpenIDNetTaskQueue"
	OpenIDNetStepCITemplatePathv1_0 = "pkg/workflowengine/workflows/openidnet_config/stepci_wallet_template_v1_0.yaml"
	OpenIDNetStepCITemplatePathDr24 = "pkg/workflowengine/workflows/openidnet_config/stepci_wallet_template_draft_24.yaml"
	OpenIDNetSubscription           = "openidnet-logs"
	OpenIDNetStartCheckSignal       = "start-openidnet-check-log-update"
	OpenIDNetStopCheckSignal        = "stop-openidnet-check-log-update"
)

// OpenIDNetWorkflow is a workflow that start a conformance checks on the OpenID certification site.
type OpenIDNetWorkflow struct{}

// OpenIDNetWorkflowPayload represents the payload for the OpenIDNetWorkflow.
type OpenIDNetWorkflowPayload struct {
	Variant  string `json:"variant"   yaml:"variant"   validate:"required"`
	Form     Form   `json:"form"      yaml:"form"      validate:"required"`
	UserMail string `json:"user_mail" yaml:"user_mail" validate:"required"`
	TestName string `json:"test"      yaml:"test"      validate:"required"`
}

type Form struct {
	Alias       string `json:"alias"       yaml:"alias"       validate:"required"`
	Description string `json:"description" yaml:"description"`
	Client      struct {
		PresentationDefinition string `json:"presentation_definition,omitempty" yaml:"presentation_definition,omitempty"`
		JWKS                   struct {
			Keys []map[string]any `json:"keys" yaml:"keys"`
		} `json:"jwks" yaml:"jwks"`
		DCQL struct {
			Credentials []map[string]any `json:"credentials" yaml:"credentials"`
		} `json:"dcql,omitempty" yaml:"dcql,omitempty"`
		ClientID                          string `json:"client_id,omitempty" yaml:"client_id,omitempty"`
		AuthorizationEncryptedResponseEnc string `json:"authorization_encrypted_response_enc,omitempty" yaml:"authorization_encrypted_response_enc,omitempty"` //nolint:lll
		AuthorizationEncryptedResponseAlg string `json:"authorization_encrypted_response_alg,omitempty" yaml:"authorization_encrypted_response_alg,omitempty"` //nolint:lll
	} `json:"client"      yaml:"client"      validate:"required"`
	Server struct {
		AuthorizationEndpoint string `json:"authorization_endpoint" yaml:"authorization_endpoint"`
	} `json:"server"      yaml:"server"`
}

// Name returns the name of the OpenIDNetWorkflow.
func (OpenIDNetWorkflow) Name() string {
	return "Conformance check on https://www.certification.openid.net"
}

// GetOptions Configure sets up the workflow with the necessary options.
func (OpenIDNetWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

// Workflow is the main workflow function for the OpenIDNetWorkflow. It orchestrates
// the execution of various activities and child workflows to perform conformance checks
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
//  1. Configure and execute the StepCIWorkflowActivity to perform initial checks.
//  2. Generate a URL with query parameters for the user to continue the process.
//  3. Configure and execute the SendMailActivity to notify the user via email.
//  4. Execute a child workflow (OpenIDNetLogsWorkflow) asynchronously to monitor logs.
//  5. Wait for either a signal ("openidnet-check-result-signal") or the completion of the child workflow.
//  6. Process the signal data to determine the success or failure of the workflow.
//
// Notes:
//   - The workflow uses a selector to wait for either a signal or the child workflow's completion.
//   - If the signal data indicates failure, the workflow terminates with a failure message.
//   - The workflow relies on environment variables (e.g., OPENIDNET_TOKEN) for configuration.
func (w *OpenIDNetWorkflow) Workflow(
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
	payload, err := workflowengine.DecodePayload[OpenIDNetWorkflowPayload](input.Payload)
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
	suite, ok := input.Config["memo"].(map[string]any)["author"].(string)
	if !ok {
		return workflowengine.WorkflowResult{},
			workflowengine.NewMissingConfigError(
				"author",
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
	stepCIPayload := activities.StepCIWorkflowActivityPayload{
		Data: map[string]any{
			"variant": payload.Variant,
			"form":    payload.Form,
			"test":    payload.TestName,
		},
		Secrets: map[string]string{
			"token": utils.GetEnvironmentVariable("OPENIDNET_TOKEN", nil, true),
		},
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
		RunMeta:       runMetadata,
		Suite:         suite,
	}

	result, err := RunStepCIAndSendMail(ctx, cfg)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	rid, ok := result.Captures["rid"].(string)
	if !ok {
		rid = ""
	}

	child := OpenIDNetLogsWorkflow{}
	ctx = child.Configure(ctx)

	logsWorkflow := workflow.ExecuteChildWorkflow(
		ctx,
		child.Name(),
		workflowengine.WorkflowInput{
			Payload: OpenIDNetLogsWorkflowPayload{
				Rid:   rid,
				Token: utils.GetEnvironmentVariable("OPENIDNET_TOKEN"),
			},
			Config: map[string]any{
				"app_url":  appURL,
				"interval": time.Second,
			},
		},
	)
	var subWorkflowResponse workflowengine.WorkflowResult
	err = logsWorkflow.Get(ctx, &subWorkflowResponse)
	if err != nil {
		if !temporal.IsCanceledError(err) {
			logger.Error("Child workflow failed", "error", err)
			subWorkflowResponse = workflowengine.WorkflowResult{
				Log: fmt.Sprintf("Child workflow failed: %s", err.Error()),
			}
		}
	}

	if subWorkflowResponse.Message == "Failed" {
		errCode := errorcodes.Codes[errorcodes.OpenIDnetCheckFailed]
		appErr := workflowengine.NewAppError(errCode, errCode.Description, subWorkflowResponse.Log)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	return workflowengine.WorkflowResult{
		Message: "Check completed successfully",
		Log:     subWorkflowResponse.Log,
	}, nil
}

// Start initializes and starts the OpenIDNetWorkflow execution.
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
func (w *OpenIDNetWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "OpenIDNetCheckWorkflow" + uuid.NewString(),
		TaskQueue:                OpenIDNetTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	namespace := DefaultNamespace
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	return workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

// OpenIDNetLogsWorkflow is a workflow that drains logs from the OpenID certification site.
type OpenIDNetLogsWorkflow struct{}

type OpenIDNetLogsWorkflowPayload struct {
	Rid   string `json:"rid"   yaml:"rid"   validate:"required"`
	Token string `json:"token" yaml:"token" validate:"required"`
}

// Name returns the name of the OpenIDNetLogsWorkflow.
func (OpenIDNetLogsWorkflow) Name() string {
	return "Drain logs from https://www.certification.openid.net"
}

// GetOptions returns the activity options for the OpenIDNetLogsWorkflow.
func (OpenIDNetLogsWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

// Workflow is the main workflow function for the OpenIDNetLogsWorkflow.
// It periodically fetches logs from a specified URL and processes them
// based on the provided input configuration. The workflow listens for
// signals to trigger additional activities and terminates when a specific
// condition in the logs is met.
//
// Parameters:
//   - ctx: The workflow context used to manage workflow execution.
//   - input: The input configuration and payload for the workflow.
//
// Returns:
//   - workflowengine.WorkflowResult: Contains the collected logs upon
//     successful completion of the workflow.
//   - error: An error if the workflow fails during execution.
//
// Behavior:
//   - Fetches logs from a remote API using the provided input configuration.
//   - Listens for a signal ("wallet-test-start-log-update") to trigger
//     additional activities.
//   - Uses a timer to periodically fetch logs at intervals specified in
//     the input configuration.
//   - Terminates when the logs contain a "result" field with a value of
//     "INTERRUPTED" or "FINISHED".
//   - Sends logs to a specified endpoint when triggered by a signal.
func (w *OpenIDNetLogsWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	subCtx := workflow.WithActivityOptions(ctx, w.GetOptions())
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(subCtx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(subCtx).Namespace,
		TemporalUI: utils.JoinURL(
			input.Config["app_url"].(string),
			"my", "tests", "runs",
			workflow.GetInfo(subCtx).WorkflowExecution.ID,
			workflow.GetInfo(subCtx).WorkflowExecution.RunID,
		),
	}

	payload, err := workflowengine.DecodePayload[OpenIDNetLogsWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			runMetadata,
		)
	}
	getLogsInput := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				"https://www.certification.openid.net/api/log",
				url.PathEscape(payload.Rid),
			),
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", payload.Token),
			},
			QueryParams: map[string]string{
				"public": "false",
			},
			ExpectedStatus: 200,
		},
	}
	var logs []map[string]any
	startSignalChan := workflow.GetSignalChannel(subCtx, OpenIDNetStartCheckSignal)
	stopSignalChan := workflow.GetSignalChannel(subCtx, OpenIDNetStopCheckSignal)

	selector := workflow.NewSelector(subCtx)

	var isPolling bool

	var timerFuture workflow.Future
	var startTimer func()
	startTimer = func() {
		timerCtx, _ := workflow.WithCancel(subCtx)
		timerFuture = workflow.NewTimer(
			timerCtx,
			time.Duration(input.Config["interval"].(float64)),
		)
		selector.AddFuture(timerFuture, func(_ workflow.Future) {
			if isPolling {
				startTimer()
			}
		})
	}

	for {
		// Always listen for pause/resume signals
		selector.AddReceive(startSignalChan, func(c workflow.ReceiveChannel, _ bool) {
			var signalVal any
			c.Receive(subCtx, &signalVal)
			if !isPolling {
				isPolling = true
				startTimer()
				logger.Info("Received start signal, unpausing workflow")
			}
		})
		selector.AddReceive(stopSignalChan, func(c workflow.ReceiveChannel, _ bool) {
			var signalVal any
			c.Receive(subCtx, &signalVal)
			isPolling = false
			logger.Info("Received stop signal, pausing workflow")
		})

		// Wait for a signal or timer
		selector.Select(subCtx)

		if !isPolling {
			continue
		}

		// Perform activity to fetch logs
		var HTTPActivity = activities.NewHTTPActivity()
		var HTTPResponse workflowengine.ActivityResult

		err := workflow.ExecuteActivity(subCtx, HTTPActivity.Name(), getLogsInput).
			Get(subCtx, &HTTPResponse)
		if err != nil {
			logger.Error("Failed to get logs", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}

		logs = workflowengine.AsSliceOfMaps(HTTPResponse.Output.(map[string]any)["body"])

		triggerLogsInput := workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					input.Config["app_url"].(string),
					"api", "compliance", "send-openidnet-log-update",
				),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"workflow_id": strings.TrimSuffix(
						workflow.GetInfo(subCtx).WorkflowExecution.ID,
						"-log",
					),
					"logs": logs,
				},
				ExpectedStatus: 200,
			},
		}

		err = workflow.ExecuteActivity(subCtx, HTTPActivity.Name(), triggerLogsInput).
			Get(subCtx, nil)
		if err != nil {
			logger.Error("Failed to send logs", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}

		// Stop if logs are done
		if len(logs) > 0 {
			lastResult := ""
			for i, logEntry := range logs {
				if result, ok := logEntry["result"].(string); ok {
					// Check for failure in any log
					if result == "FAILURE" {
						return workflowengine.WorkflowResult{
							Message: "Failed",
							Log:     logs,
						}, nil
					}
					// Save the last result for the final check
					if i == len(logs)-1 {
						lastResult = result
					}
				}
			}
			if lastResult == "INTERRUPTED" || lastResult == "FINISHED" {
				return workflowengine.WorkflowResult{
					Message: "Passed",
					Log:     logs,
				}, nil
			}
		}
	}
}

// Configure sets up the OpenIDNetLogsWorkflow with specific child workflow options.
// It configures the child workflow to have a unique WorkflowID by appending "-log"
// to the parent workflow's ID and sets the ParentClosePolicy to terminate the child
// workflow when the parent workflow is closed.
//
// Parameters:
//   - ctx: The workflow.Context for the current workflow execution.
//
// Returns:
//   - A new workflow.Context configured with the specified child workflow options.
func (w *OpenIDNetLogsWorkflow) Configure(ctx workflow.Context) workflow.Context {
	childOptions := workflow.ChildWorkflowOptions{
		WorkflowID:        workflow.GetInfo(ctx).WorkflowExecution.ID + "-log",
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
	}
	return workflow.WithChildOptions(ctx, childOptions)
}
