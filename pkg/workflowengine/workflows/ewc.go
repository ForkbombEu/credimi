// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"encoding/json"
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

type ewcStartWithOptionsFn func(
	namespace string,
	options client.StartWorkflowOptions,
	name string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error)

var (
	ewcStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions
)

type EWCWorkflowPayload struct {
	Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	UserMail   string         `json:"user_mail"            yaml:"user_mail"            validate:"required"`
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
	Status    string         `json:"status"`
	Error     string         `json:"error,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty"`
	Reason    string         `json:"reason,omitempty"`
	SessionID string         `json:"sessionId"`
	Claims    map[string]any `json:"claims,omitempty"`
}

const (
	ewcAPIBaseURL             = "https://ewc.api.forkbomb.eu"
	webuildAPIBaseURL         = "https://webuild.api.forkbomb.eu"
	webuildWalletClientAPIURL = "https://webuild.wallet-client.forkbomb.eu"
	ewcLikeSessionIDTemplate  = "{{ sessionId }}"
	ewcVerificationStatusPath = "/verificationStatus"
	ewcIssueStatusPath        = "/issueStatus"
	ewcSessionStatusPath      = "/session-status/"
	ewcLogsPath               = "/logs/"
	EWCSubscription           = "ewc-logs"
)

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
	return executeEWCLikeWorkflow(ctx, input, w.GetOptions())
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
	return startEWCLikeWorkflow(
		input,
		w.Name(),
		"EWCWorkflow",
		ewcStartWorkflowWithOptions,
	)
}

func startEWCLikeWorkflow(
	input workflowengine.WorkflowInput,
	name string,
	workflowPrefix string,
	startFn ewcStartWithOptionsFn,
) (result workflowengine.WorkflowResult, err error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       workflowPrefix + uuid.NewString(),
		TaskQueue:                EWCTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	namespace := DefaultNamespace
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	return startFn(namespace, workflowOptions, name, input)
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
	return executeEWCLikeStatusWorkflow(ctx, input, w.GetOptions())
}

func executeEWCLikeWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	options workflow.ActivityOptions,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, options)

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
	logsEndpoint, ok := input.Config["logs_endpoint"].(string)
	if !ok || logsEndpoint == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"logs_endpoint",
			input.RunMetadata,
		)
	}
	parameters := make(map[string]any, len(payload.Parameters))
	for k, v := range payload.Parameters {
		parameters[k] = v
	}
	stepCIPayload := activities.StepCIWorkflowActivityPayload{
		Data: parameters,
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
		appURL,
		checkEndpoint,
		logsEndpoint,
		sessionID,
		input.RunMetadata,
		false,
	)
	return workflowResult, err
}

func executeEWCLikeStatusWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	options workflow.ActivityOptions,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, options)

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
	logsEndpoint, ok := input.Config["logs_endpoint"].(string)
	if !ok || logsEndpoint == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"logs_endpoint",
			input.RunMetadata,
		)
	}

	result, err := pollEWCCheck(
		ctx,
		interval,
		appURL,
		checkEndpoint,
		logsEndpoint,
		payload.SessionID,
		input.RunMetadata,
		true,
	)
	return result, err
}

func ResolveEWCLikeCheckEndpoint(suite string, standard string) (string, error) {
	switch standard {
	case OpenID4VPWalletStandard:
		baseURL, err := resolveEWCLikeBaseURL(suite, standard)
		if err != nil {
			return "", err
		}
		return baseURL + ewcVerificationStatusPath, nil
	case OpenID4VCIWalletStandard:
		baseURL, err := resolveEWCLikeBaseURL(suite, standard)
		if err != nil {
			return "", err
		}
		return baseURL + ewcIssueStatusPath, nil
	case OpenID4VPVerifierStandard, OpenID4VCIIssuerStandard:
		if suite != WebuildSuite {
			return "", fmt.Errorf("unsupported standard %s for suite %s", standard, suite)
		}
		baseURL, err := resolveEWCLikeBaseURL(suite, standard)
		if err != nil {
			return "", err
		}
		return baseURL + ewcSessionStatusPath + ewcLikeSessionIDTemplate, nil
	default:
		return "", fmt.Errorf("unsupported standard %s for suite %s", standard, suite)
	}
}

func ResolveEWCLikeLogsEndpoint(suite string, standard string) (string, error) {
	baseURL, err := resolveEWCLikeBaseURL(suite, standard)
	if err != nil {
		return "", err
	}

	return baseURL + ewcLogsPath + ewcLikeSessionIDTemplate, nil
}

func resolveEWCLikeBaseURL(suite string, standard string) (string, error) {
	switch suite {
	case EWCSuite:
		return ewcAPIBaseURL, nil
	case WebuildSuite:
		if standard == OpenID4VPVerifierStandard || standard == OpenID4VCIIssuerStandard {
			return webuildWalletClientAPIURL, nil
		}
		return webuildAPIBaseURL, nil
	default:
		return "", fmt.Errorf("unsupported suite %s", suite)
	}
}

func pollEWCCheck(
	ctx workflow.Context,
	interval time.Duration,
	appURL string,
	checkEndpoint string,
	logsEndpoint string,
	sessionID string,
	runMetadata *workflowengine.WorkflowRunMetadata,
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

	httpPayload := activities.HTTPActivityPayload{
		Method:         http.MethodGet,
		URL:            resolveEWCLikeEndpoint(checkEndpoint, sessionID),
		ExpectedStatus: 200,
	}
	if !strings.Contains(checkEndpoint, ewcLikeSessionIDTemplate) {
		httpPayload.QueryParams = map[string]string{
			"sessionId": sessionID,
		}
	}
	httpInput := workflowengine.ActivityInput{Payload: httpPayload}
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

		statusResponse := response.Output.(map[string]any)["body"]
		bodyJSON, err := json.Marshal(statusResponse)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: err.Error(),
					Details: map[string]any{"payload": response.Output},
				},
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}

		var parsed EWCResponseBody
		if err := json.Unmarshal(bodyJSON, &parsed); err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: err.Error(),
					Details: map[string]any{"payload": bodyJSON},
				},
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}

		errCode := errorcodes.Codes[errorcodes.EWCCheckFailed]
		logs, logsResponse, err := fetchEWCLikeLogs(ctx, logsEndpoint, sessionID, runMetadata)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
		if len(logs) > 0 {
			if err := notifyEWCLikeLogs(
				ctx,
				appURL,
				workflow.GetInfo(ctx).WorkflowExecution.ID,
				logs,
				runMetadata,
			); err != nil {
				logger.Error("Failed to send EWC logs", "error", err)
				return workflowengine.WorkflowResult{}, err
			}
		}
		resultPayload := map[string]any{
			"status_response": statusResponse,
			"logs_response":   logsResponse,
			"logs":            logs,
		}
		failedErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: parsed.Reason,
				Details: map[string]any{"payload": resultPayload},
			},
		)

		switch parsed.Status {
		case "success":
			var resultLog any = logs
			if len(logs) == 0 {
				resultLog = parsed.Claims
			}
			return workflowengine.WorkflowResult{
				Message: "EWC check completed successfully",
				Log:     resultLog,
				Output:  resultPayload,
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
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf(
						"unexpected status from '%s': %s",
						checkEndpoint,
						parsed.Status,
					),
					Details: map[string]any{"payload": parsed},
				},
			)

			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				failedErr,
				runMetadata,
			)
		}
	}
}

func resolveEWCLikeEndpoint(endpoint string, sessionID string) string {
	return strings.ReplaceAll(endpoint, ewcLikeSessionIDTemplate, url.PathEscape(sessionID))
}

func fetchEWCLikeLogs(
	ctx workflow.Context,
	logsEndpoint string,
	sessionID string,
	runMetadata *workflowengine.WorkflowRunMetadata,
) ([]map[string]any, any, error) {
	if logsEndpoint == "" {
		return nil, nil, nil
	}

	httpActivity := activities.NewHTTPActivity()
	httpInput := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method:         http.MethodGet,
			URL:            resolveEWCLikeEndpoint(logsEndpoint, sessionID),
			ExpectedStatus: 200,
		},
	}
	var response workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), httpInput).
		Get(ctx, &response); err != nil {
		return nil, nil, workflowengine.NewWorkflowError(err, runMetadata)
	}

	logsResponse := workflowengine.AsMap(response.Output)["body"]
	return extractEWCLikeLogs(logsResponse), logsResponse, nil
}

func extractEWCLikeLogs(logsResponse any) []map[string]any {
	if body := workflowengine.AsMap(logsResponse); body != nil {
		if logs := workflowengine.AsSliceOfMaps(body["logs"]); len(logs) > 0 {
			return logs
		}
	}

	return workflowengine.AsSliceOfMaps(logsResponse)
}

func notifyEWCLikeLogs(
	ctx workflow.Context,
	appURL string,
	workflowID string,
	logs []map[string]any,
	runMetadata *workflowengine.WorkflowRunMetadata,
) error {
	if appURL == "" {
		return nil
	}

	httpActivity := activities.NewHTTPActivity()
	triggerLogsInput := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodPost,
			URL: utils.JoinURL(
				appURL,
				"api",
				"compliance",
				"send-ewc-log-update",
			),
			Headers: map[string]string{
				workflowengine.HTTPHeaderContentType: workflowengine.MIMEApplicationJSON,
			},
			Body: map[string]any{
				"workflow_id": strings.TrimSuffix(workflowID, "-status"),
				"logs":        logs,
			},
			ExpectedStatus: 200,
		},
	}

	if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), triggerLogsInput).
		Get(ctx, nil); err != nil {
		return workflowengine.NewWorkflowError(err, runMetadata)
	}

	return nil
}
