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
	OpenIDNetTaskQueue          = "OpenIDNetTaskQueue"
	OpenIDNetStepCITemplatePath = "pkg/workflowengine/workflows/openidnet_config/stepci_wallet_template.yaml"
	OpenIDNetSubscription       = "openidnet-logs"
	OpenIDNetStartCheckSignal   = "start-openidnet-check-log-update"
	OpenIDNetStopCheckSignal    = "stop-openidnet-check-log-update"
)

// OpenIDNetWorkflow is a workflow that performs conformance checks on the OpenID certification site.
type OpenIDNetWorkflow struct{}

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
		TemporalUI: fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			input.Config["app_url"],
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}
	stepCIWorkflowActivity := activities.NewStepCIWorkflowActivity()
	variant, ok := input.Payload["variant"].(string)
	if !ok || variant == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"variant",
			runMetadata,
		)
	}
	form, ok := input.Payload["form"]
	if !ok || form == nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"form",
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
	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"variant": variant,
			"form":    form,
		},
		Config: map[string]string{
			"template": template,
			"token":    utils.GetEnvironmentVariable("OPENIDNET_TOKEN", nil, true),
		},
	}
	var stepCIResult workflowengine.ActivityResult
	err := stepCIWorkflowActivity.Configure(&stepCIInput)
	if err != nil {
		logger.Error(" StepCI configure failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	err = workflow.ExecuteActivity(ctx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	result, ok := stepCIResult.Output.(map[string]any)["captures"].(map[string]any)["result"].(string)
	if !ok {
		result = ""
	}
	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"app_url",
			runMetadata,
		)
	}
	baseURL := appURL + "/tests/wallet/openidnet"
	u, err := url.Parse(baseURL)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
		appErr := workflowengine.NewAppError(errCode, baseURL)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	query := u.Query()
	query.Set("workflow-id", workflow.GetInfo(ctx).WorkflowExecution.ID)
	query.Set("qr", result)
	namespace, ok := input.Config["namespace"].(string)
	if !ok || namespace == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"namespace",
			runMetadata,
		)
	}
	query.Set("namespace", input.Config["namespace"].(string))
	u.RawQuery = query.Encode()

	emailActivity := activities.NewSendMailActivity()
	userMail, ok := input.Payload["user_mail"].(string)
	if !ok || userMail == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"user_mail",
			runMetadata,
		)
	}

	emailInput := workflowengine.ActivityInput{
		Config: map[string]string{
			"recipient": userMail,
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
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	err = workflow.ExecuteActivity(ctx, emailActivity.Name(), emailInput).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to send mail to user ", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	rid, ok := stepCIResult.Output.(map[string]any)["captures"].(map[string]any)["rid"].(string)
	if !ok {
		rid = ""
	}

	child := OpenIDNetLogsWorkflow{}
	ctx = child.Configure(ctx)

	logsWorkflow := workflow.ExecuteChildWorkflow(
		ctx,
		child.Name(),
		workflowengine.WorkflowInput{
			Payload: map[string]any{
				"rid":   rid,
				"token": utils.GetEnvironmentVariable("OPENIDNET_TOKEN"),
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

	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}

// OpenIDNetLogsWorkflow is a workflow that drains logs from the OpenID certification site.
type OpenIDNetLogsWorkflow struct{}

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
		TemporalUI: fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			input.Config["app_url"],
			workflow.GetInfo(subCtx).WorkflowExecution.ID,
			workflow.GetInfo(subCtx).WorkflowExecution.RunID,
		),
	}

	getLogsInput := workflowengine.ActivityInput{
		Config: map[string]string{
			"method": "GET",
			"url": fmt.Sprintf(
				"%s/%s",
				"https://www.certification.openid.net/api/log/",
				url.PathEscape(input.Payload["rid"].(string)),
			),
		},
		Payload: map[string]any{
			"headers": map[string]any{
				"Authorization": fmt.Sprintf("Bearer %s", input.Payload["token"].(string)),
			},
			"query_params": map[string]any{
				"public": "false",
			},
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

		logs = AsSliceOfMaps(HTTPResponse.Output.(map[string]any)["body"])

		triggerLogsInput := workflowengine.ActivityInput{
			Config: map[string]string{
				"method": "POST",
				"url": fmt.Sprintf(
					"%s/%s",
					input.Config["app_url"].(string),
					"api/compliance/send-openidnet-log-update",
				),
			},
			Payload: map[string]any{
				"headers": map[string]any{
					"Content-Type": "application/json",
				},
				"body": map[string]any{
					"workflow_id": strings.TrimSuffix(
						workflow.GetInfo(subCtx).WorkflowExecution.ID,
						"-log",
					),
					"logs": logs,
				},
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

func AsSliceOfMaps(val any) []map[string]any {
	if v, ok := val.([]map[string]any); ok {
		return v
	}
	if arr, ok := val.([]any); ok {
		res := make([]map[string]any, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				res = append(res, m)
			}
		}
		return res
	}
	return nil
}
