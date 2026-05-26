// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"net/url"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	OpenIDConformanceSuite    = "openid_conformance_suite"
	EWCSuite                  = "ewc"
	WebuildSuite              = "webuild"
	EudiwSuite                = "eudiw"
	VLEISuite                 = "vlei"
	OpenID4VPWalletStandard   = "openid4vp_wallet"
	OpenID4VCIWalletStandard  = "openid4vci_wallet"
	OpenID4VPVerifierStandard = "openid4vp_verifier"
	OpenID4VCIIssuerStandard  = "openid4vci_issuer"
	ConformanceCheckTaskQueue = "ConformanceCheckTaskQueue"
	PipelineCancelSignal      = "pipeline_cancel_signal"
)

type StepCIAndEmailConfig struct {
	AppURL        string
	AppName       string
	AppLogo       string
	UserName      string
	UserMail      string
	Namespace     string
	Template      string
	StepCIPayload activities.StepCIWorkflowActivityPayload
	Secrets       map[string]any
	RunMetadata   *workflowengine.WorkflowErrorMetadata
	Suite         string
	SendMail      bool
}

type StepCIAndEmailResult struct {
	Captures map[string]any
}

func RunStepCIAndSendMail(
	ctx workflow.Context,
	cfg StepCIAndEmailConfig,
) (StepCIAndEmailResult, error) {
	logger := workflow.GetLogger(ctx)

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	stepCIInput := workflowengine.ActivityInput{
		Payload: cfg.StepCIPayload,
		Config: map[string]string{
			"template": cfg.Template,
		},
	}
	if err := stepCIActivity.Configure(&stepCIInput); err != nil {
		logger.Error("StepCI configure failed", "error", err)
		return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMetadata)
	}

	var stepCIResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, stepCIActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult); err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMetadata)
	}

	captures, ok := stepCIResult.Output.(map[string]any)["captures"].(map[string]any)
	if !ok {
		return StepCIAndEmailResult{},
			workflowengine.NewStepCIOutputError(
				"StepCI unexpected output",
				stepCIResult.Output,
				cfg.RunMetadata,
			)
	}
	// Send mail only if SendMail is true
	if cfg.SendMail {
		deepLink, ok := captures["deeplink"].(string)
		if !ok {
			return StepCIAndEmailResult{},
				workflowengine.NewStepCIOutputError(
					"StepCI unexpected output: missing deeplink in captures",
					captures,
					cfg.RunMetadata,
				)
		}
		suite := cfg.Suite
		if suite == OpenIDConformanceSuite {
			suite = "openidnet"
		}
		baseURL := utils.JoinURL(cfg.AppURL, "tests", "wallet", suite)
		u, err := url.Parse(baseURL)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
			appErr := workflowengine.NewAppError(errCode, baseURL)
			return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(appErr, cfg.RunMetadata)
		}
		q := u.Query()
		q.Set("workflow-id", workflow.GetInfo(ctx).WorkflowExecution.ID)
		q.Set("qr", deepLink)
		q.Set("namespace", cfg.Namespace)
		u.RawQuery = q.Encode()

		emailActivity := activities.NewSendMailActivity()
		emailInput := workflowengine.ActivityInput{
			Payload: activities.SendMailActivityPayload{
				Recipient: cfg.UserMail,
				Subject:   "[CREDIMI] Action required to continue your conformance checks",
				Template:  activities.ContinueConformanceCheckEmailTemplate,
				Data: map[string]any{
					"AppName":          cfg.AppName,
					"AppLogo":          cfg.AppLogo,
					"UserName":         cfg.UserName,
					"VerificationLink": u.String(),
				},
			},
		}
		if err := emailActivity.Configure(&emailInput); err != nil {
			logger.Error("Email configure failed", "error", err)
			return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMetadata)
		}
		if err := workflow.ExecuteActivity(ctx, emailActivity.Name(), emailInput).
			Get(ctx, nil); err != nil {
			logger.Error("Failed to send mail", "error", err)
			return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMetadata)
		}
	}
	return StepCIAndEmailResult{
		Captures: captures,
	}, nil
}

type StartCheckWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

var startCheckWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

type StartCheckWorkflowPayload struct {
	Suite      string         `json:"suite"                yaml:"suite"`
	Standard   string         `json:"standard,omitempty"   yaml:"standard,omitempty"`
	CheckID    string         `json:"check_id"             yaml:"check_id"             validate:"required"`
	TestName   string         `json:"test,omitempty"       yaml:"test,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	UserMail   string         `json:"user_mail"            yaml:"user_mail"`
	SendMail   bool           `json:"send_mail"            yaml:"send_mail"`
}

type StartCheckWorkflowPipelinePayload struct {
	CheckID    string         `json:"check_id"             yaml:"check_id"             validate:"required"`
	TestName   string         `json:"test,omitempty"       yaml:"test,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

func conformanceCheckParameters(payload StartCheckWorkflowPayload) map[string]any {
	parameters := make(map[string]any, len(payload.Parameters))
	for k, v := range payload.Parameters {
		parameters[k] = v
	}
	return parameters
}

func conformanceCheckStandard(payload StartCheckWorkflowPayload, config map[string]any) string {
	if payload.Standard != "" {
		return payload.Standard
	}

	memo, ok := config["memo"].(map[string]any)
	if !ok {
		return ""
	}
	standard, _ := memo["standard"].(string)
	return standard
}

func conformanceCheckNeedsDeeplinkOutput(standard string) bool {
	return standard != OpenID4VPVerifierStandard && standard != OpenID4VCIIssuerStandard
}

func isOpenIDAutomatedConformanceStandard(standard string) bool {
	return standard == OpenID4VPVerifierStandard || standard == OpenID4VCIIssuerStandard
}

func conformanceCheckSessionID(payload StartCheckWorkflowPayload, captures map[string]any) string {
	if sessionID, ok := captures["session_id"].(string); ok && sessionID != "" {
		return sessionID
	}
	sessionID, _ := payload.Parameters["session_id"].(string)
	return sessionID
}

func NewStartCheckWorkflow() *StartCheckWorkflow {
	w := &StartCheckWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (StartCheckWorkflow) Name() string {
	return "Start conformance check"
}

func (StartCheckWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

// Workflow starts the conformance check workflow.
func (w *StartCheckWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// executeWorkflow starts the conformance check workflow.
//
// It takes the workflow input and starts the conformance check workflow.
// The workflow input should contain the suite, standard, check_id, test and parameters.
// The function first decodes the workflow input and checks if the required fields are present.
// If not, it returns an error.
// Then it runs the StepCIAndEmail function with the decoded input and returns the result.
// If the StepCIAndEmail function returns an error, it logs the error and returns the error.
// If the StepCIAndEmail function returns a successful result, it runs the child workflow depending on the suite and standard.
// If the child workflow returns an error, it logs the error and returns the error.
// If the child workflow returns a successful result, it returns the successful result.
func (w *StartCheckWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)
	logger := workflow.GetLogger(ctx)
	payload, err := workflowengine.DecodePayload[StartCheckWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}
	appURL := input.Config["app_url"].(string)
	if appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}
	var stepCIPayload activities.StepCIWorkflowActivityPayload
	var ewcSessionID string
	parameters := conformanceCheckParameters(payload)
	standard := conformanceCheckStandard(payload, input.Config)
	switch payload.Suite {
	case OpenIDConformanceSuite:
		if payload.TestName == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
				fmt.Errorf("test is required for suite %s", payload.Suite),
				input.RunMetadata,
			)
		}
		stepCIPayload.Data = parameters
		stepCIPayload.Data["test"] = payload.TestName
		stepCIPayload.Secrets = map[string]string{
			"token": utils.GetEnvironmentVariable("OPENIDNET_TOKEN", nil, true),
		}
	case EWCSuite, WebuildSuite:
		stepCIPayload.Data = parameters
	default:
		return workflowengine.WorkflowResult{}, fmt.Errorf("unsupported suite: %s", payload.Suite)
	}
	cfg := StepCIAndEmailConfig{
		Template:      input.Config["template"].(string),
		StepCIPayload: stepCIPayload,
		Namespace:     input.Config["namespace"].(string),
		RunMetadata:   input.RunMetadata,
		Suite:         payload.Suite,
		SendMail:      payload.SendMail,
	}

	if payload.SendMail {
		cfg.AppURL = appURL
		cfg.AppName = input.Config["app_name"].(string)
		cfg.AppLogo = input.Config["app_logo"].(string)
		cfg.UserName = input.Config["user_name"].(string)
		cfg.UserMail = payload.UserMail
	}

	setupResult, err := RunStepCIAndSendMail(ctx, cfg)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	var childID string
	switch payload.Suite {
	case OpenIDConformanceSuite:
		rid := openIDConformanceRunnerID(setupResult.Captures)
		if rid == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
				"rid",
				setupResult.Captures,
				input.RunMetadata,
			)
		}

		if isOpenIDAutomatedConformanceStandard(standard) {
			return pollOpenIDConformanceLogs(
				ctx,
				rid,
				appURL,
				utils.GetEnvironmentVariable("OPENIDNET_TOKEN"),
				false,
				input.RunMetadata,
			)
		}

		deeplink, ok := setupResult.Captures["deeplink"].(string)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
				"deeplink",
				setupResult.Captures,
				input.RunMetadata,
			)
		}

		child := NewOpenIDNetLogsWorkflow()
		childID = workflow.GetInfo(ctx).WorkflowExecution.ID + "-log"
		ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:        childID,
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
		})

		childCtx, _ := workflow.NewDisconnectedContext(ctx)
		err = workflow.ExecuteChildWorkflow(
			childCtx,
			child.Name(),
			workflowengine.WorkflowInput{
				Payload: OpenIDNetLogsWorkflowPayload{
					Rid:   rid,
					Token: utils.GetEnvironmentVariable("OPENIDNET_TOKEN"),
				},
				Config: workflowengine.MergeTelemetryConfig(ctx, map[string]any{
					"app_url":  appURL,
					"interval": time.Second,
				}),
			}).GetChildWorkflowExecution().Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to execute child workflow", "error", err)
			errCode := errorcodes.Codes[errorcodes.ChildWorkflowExecutionError]
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				workflowengine.NewAppError(errCode, err.Error(), nil),
				cfg.RunMetadata,
			)
		}
		return workflowengine.WorkflowResult{
			Message: "Check completed successfully",
			Output: map[string]any{
				"deeplink": deeplink,
				"child_id": childID,
			},
		}, nil
	case EWCSuite, WebuildSuite:
		if standard == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				"standard",
				input.RunMetadata,
			)
		}
		checkEndpoint, err := ResolveEWCLikeCheckEndpoint(payload.Suite, standard)
		if err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				err.Error(),
				input.RunMetadata,
			)
		}
		logsEndpoint, err := ResolveEWCLikeLogsEndpoint(payload.Suite, standard)
		if err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				err.Error(),
				input.RunMetadata,
			)
		}

		deeplink, hasDeeplink := setupResult.Captures["deeplink"].(string)
		if conformanceCheckNeedsDeeplinkOutput(standard) && !hasDeeplink {
			return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
				"deeplink",
				setupResult.Captures,
				input.RunMetadata,
			)
		}
		ewcSessionID = conformanceCheckSessionID(payload, setupResult.Captures)
		if ewcSessionID == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewStepCIOutputError(
				"session_id",
				setupResult.Captures,
				input.RunMetadata,
			)
		}

		var child workflowengine.Workflow
		switch payload.Suite {
		case EWCSuite:
			child = NewEWCStatusWorkflow()
		case WebuildSuite:
			child = NewWebuildStatusWorkflow()
		default:
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				fmt.Sprintf("unsupported suite %s", payload.Suite),
				input.RunMetadata,
			)
		}
		childID = workflow.GetInfo(ctx).WorkflowExecution.ID + "-status"
		ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:        childID,
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
		})

		childInput := workflowengine.WorkflowInput{
			Payload: EWCStatusWorkflowPayload{
				SessionID: ewcSessionID,
			},
			Config: workflowengine.MergeTelemetryConfig(ctx, map[string]any{
				"app_url":        appURL,
				"interval":       time.Second * 5,
				"check_endpoint": checkEndpoint,
				"logs_endpoint":  logsEndpoint,
			}),
		}

		if !conformanceCheckNeedsDeeplinkOutput(standard) {
			var statusResult workflowengine.WorkflowResult
			err = workflow.ExecuteChildWorkflow(ctx, child.Name(), childInput).
				Get(ctx, &statusResult)
			if err != nil {
				logger.Error("Failed to execute child workflow", "error", err)
				errCode := errorcodes.Codes[errorcodes.ChildWorkflowExecutionError]
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					workflowengine.NewAppError(errCode, err.Error(), nil),
					cfg.RunMetadata,
				)
			}
			return statusResult, nil
		}

		childCtx, _ := workflow.NewDisconnectedContext(ctx)
		err = workflow.ExecuteChildWorkflow(
			childCtx,
			child.Name(),
			childInput,
		).GetChildWorkflowExecution().Get(ctx, nil)

		if err != nil {
			logger.Error("Failed to execute child workflow", "error", err)
			errCode := errorcodes.Codes[errorcodes.ChildWorkflowExecutionError]
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				workflowengine.NewAppError(errCode, err.Error(), nil),
				cfg.RunMetadata,
			)
		}
		return workflowengine.WorkflowResult{
			Message: "Check completed successfully",
			Output: map[string]any{
				"deeplink": deeplink,
				"child_id": childID,
			},
		}, nil
	default:
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			fmt.Sprintf("unsupported suite %s", payload.Suite),
			input.RunMetadata,
		)
	}
}

func (w *StartCheckWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "conformance-check" + "-" + uuid.NewString(),
		TaskQueue: ConformanceCheckTaskQueue,
	}
	return startCheckWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
