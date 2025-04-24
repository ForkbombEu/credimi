// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for the OpenID certification site.
// It includes the OpenIDNetWorkflow for conformance checks and the OpenIDNetLogsWorkflow
// for draining logs from the OpenID certification site.
package workflows

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"

	"github.com/forkbombeu/didimo/pkg/internal/temporalclient"
	"github.com/forkbombeu/didimo/pkg/utils"
	"github.com/forkbombeu/didimo/pkg/workflowengine"
	"github.com/forkbombeu/didimo/pkg/workflowengine/activities"
)

// SignalData represents the data structure for signals used in the workflow.
type SignalData struct {
	Success bool
	Reason  string
}

// OpenIDNetTaskQueue is the task queue for OpenIDNet workflows.
const OpenIDNetTaskQueue = "OpenIDNetTaskQueue"

// OpenIDNetStepCITemplatePath points to the StepCI template for OpenIDNet workflows.
const OpenIDNetStepCITemplatePath = "pkg/workflowengine/workflows/openidnet_config/stepci_wallet_template.yaml"

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
//  5. Wait for either a signal ("wallet-test-signal") or the completion of the child workflow.
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

	stepCIWorkflowActivity := activities.StepCIWorkflowActivity{}
	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"variant": input.Payload["variant"],
			"form":    input.Payload["form"],
		},
		Config: map[string]string{
			"template": input.Config["template"].(string),
			"token":    utils.GetEnvironmentVariable("OPENIDNET_TOKEN", nil, true),
		},
	}
	var stepCIResult workflowengine.ActivityResult
	err := stepCIWorkflowActivity.Configure(context.Background(), &stepCIInput)
	if err != nil {
		logger.Error(" StepCI configure failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	err = workflow.ExecuteActivity(ctx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	result, ok := stepCIResult.Output.(map[string]any)["result"].(string)
	if !ok {
		result = ""
	}
	baseURL := input.Payload["app_url"].(string) + "/tests/wallet"
	u, err := url.Parse(baseURL)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("unexpected error parsing URL: %v", err)
	}
	query := u.Query()
	query.Set("workflow-id", workflow.GetInfo(ctx).WorkflowExecution.ID)
	query.Set("qr", result)
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
	err = emailActivity.Configure(context.Background(), &emailInput)
	if err != nil {
		logger.Error("Email activity configure failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}
	err = workflow.ExecuteActivity(ctx, emailActivity.Name(), emailInput).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to send mail to user ", "error", err)
		return workflowengine.WorkflowResult{}, err
	}

	rid, ok := stepCIResult.Output.(map[string]any)["rid"].(string)
	if !ok {
		rid = ""
	}

	childCtx, cancelHandler := workflow.WithCancel(ctx)
	defer cancelHandler()

	child := OpenIDNetLogsWorkflow{}
	childCtx = child.Configure(childCtx)
	// Execute child workflow asynchronously
	logsWorkflow := workflow.ExecuteChildWorkflow(
		childCtx,
		child.Name(),
		workflowengine.WorkflowInput{
			Payload: map[string]any{
				"rid":     rid,
				"token":   os.Getenv("OPENIDNET_TOKEN"),
				"app_url": input.Payload["app_url"].(string),
			},
			Config: map[string]any{
				"interval": time.Second,
			},
		},
	)

	// Wait for either signal or child completion
	selector := workflow.NewSelector(ctx)
	var subWorkflowResponse workflowengine.WorkflowResult
	var data SignalData

	selector.AddFuture(logsWorkflow, func(f workflow.Future) {
		f.Get(ctx, &subWorkflowResponse)
	})
	var signalSent bool
	signalChan := workflow.GetSignalChannel(ctx, "wallet-test-signal")
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, _ bool) {
		signalSent = true
		c.Receive(ctx, &data)
		cancelHandler()
		logsWorkflow.Get(ctx, &subWorkflowResponse)
	})
	for !signalSent {
		selector.Select(ctx)
	}

	// Process the signal data
	if !data.Success {
		return workflowengine.WorkflowResult{
			Message: fmt.Sprintf("Workflow terminated with a failure message: %s", data.Reason),
			Log:     subWorkflowResponse.Log,
		}, nil
	}

	return workflowengine.WorkflowResult{
		Message: "Workflow completed successfully",
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
	// Load environment variables.
	godotenv.Load()
	namespace := "default"
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	c, err := temporalclient.GetTemporalClientWithNamespace(
		namespace,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("unable to create client: %v", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:        "OpenIDTestWorkflow" + uuid.NewString(),
		TaskQueue: OpenIDNetTaskQueue,
	}
	if input.Config["memo"] != nil {
		workflowOptions.Memo = input.Config["memo"].(map[string]any)
	}

	// Start the workflow execution.
	_, err = c.ExecuteWorkflow(context.Background(), workflowOptions, w.Name(), input)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to start workflow: %v", err)
	}

	return workflowengine.WorkflowResult{}, nil
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

	GetLogsInput := workflowengine.ActivityInput{
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

	signalChan := workflow.GetSignalChannel(ctx, "wallet-test-start-log-update")
	selector := workflow.NewSelector(ctx)
	var triggerEnabled bool // Persistent flag to enable activity executions

	// Timer setup
	var timerFuture workflow.Future
	startTimer := func() {
		timerCtx, _ := workflow.WithCancel(ctx)
		timerFuture = workflow.NewTimer(timerCtx, time.Duration(input.Config["interval"].(float64))*time.Nanosecond)
	}

	// Initialize the timer
	startTimer()

	for {
		if ctx.Err() != nil {
			logger.Info("Workflow canceled, returning collected logs")
			return workflowengine.WorkflowResult{Log: logs}, nil
		}
		var HTTPActivity activities.HTTPActivity
		var HTTPResponse workflowengine.ActivityResult
		// Fetch logs
		err := workflow.ExecuteActivity(subCtx, HTTPActivity.Name(), GetLogsInput).
			Get(subCtx, &HTTPResponse)
		if err != nil {
			logger.Error("Failed to get logs", "error", err)
			return workflowengine.WorkflowResult{}, err
		}

		logs = asSliceOfMaps(HTTPResponse.Output.(map[string]any)["body"])
		// Check termination condition
		if len(logs) > 0 {
			if result, ok := logs[len(logs)-1]["result"].(string); ok {
				if result == "INTERRUPTED" || result == "FINISHED" {
					return workflowengine.WorkflowResult{Log: logs}, nil
				}
			}
		}

		// Register events in the selector
		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, _ bool) {
			var signalVal interface{}
			c.Receive(ctx, &signalVal)
			triggerEnabled = true // Enable activity execution
		})
		selector.AddFuture(timerFuture, func(_ workflow.Future) {
			// Timer expired; reset the timer for the next interval
			startTimer()
		})

		// Wait for either a signal or the timer to expire
		selector.Select(ctx)

		// Execute TriggerLogsUpdateActivity if enabled
		if triggerEnabled {
			triggerLogsInput := workflowengine.ActivityInput{
				Config: map[string]string{
					"method": "POST",
					"url": fmt.Sprintf(
						"%s/%s",
						input.Payload["app_url"].(string),
						"wallet-test/send-log-update",
					),
				},
				Payload: map[string]any{
					"headers": map[string]any{
						"Content-Type": "application/json",
					},
					"body": map[string]any{
						"workflow_id": strings.TrimSuffix(
							workflow.GetInfo(ctx).WorkflowExecution.ID,
							"-log",
						),
						"logs": logs,
					},
				},
			}

			err = workflow.ExecuteActivity(subCtx, HTTPActivity.Name(), triggerLogsInput).
				Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to send logs", "error", err)
				return workflowengine.WorkflowResult{}, err
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

func asSliceOfMaps(val any) []map[string]any {
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
