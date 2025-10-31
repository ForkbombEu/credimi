// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"net/url"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type StepCIAndEmailConfig struct {
	AppURL        string
	AppName       string
	AppLogo       string
	UserName      string
	UserMail      string
	Namespace     string
	Template      string
	StepCIPayload map[string]any
	Secrets       map[string]any
	RunMeta       workflowengine.WorkflowErrorMetadata
	Suite         string
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
		return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMeta)
	}

	var stepCIResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, stepCIActivity.Name(), stepCIInput).Get(ctx, &stepCIResult); err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMeta)
	}

	captures, ok := stepCIResult.Output.(map[string]any)["captures"].(map[string]any)
	if !ok {
		return StepCIAndEmailResult{},
			workflowengine.NewStepCIOutputError(
				"StepCI unexpected output",
				stepCIResult.Output,
				cfg.RunMeta,
			)
	}

	deepLink, ok := captures["deeplink"].(string)
	if !ok {
		return StepCIAndEmailResult{},
			workflowengine.NewStepCIOutputError(
				"StepCI unexpected output: missing deeplink in captures",
				captures,
				cfg.RunMeta,
			)
	}
	suite := cfg.Suite
	if suite == "openid_conformance_suite" {
		suite = "openidnet"
	}
	baseURL := fmt.Sprintf("%s/tests/wallet/%s", cfg.AppURL, suite)
	u, err := url.Parse(baseURL)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
		appErr := workflowengine.NewAppError(errCode, baseURL)
		return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(appErr, cfg.RunMeta)
	}
	q := u.Query()
	q.Set("workflow-id", workflow.GetInfo(ctx).WorkflowExecution.ID)
	q.Set("qr", deepLink)
	q.Set("namespace", cfg.Namespace)
	u.RawQuery = q.Encode()

	emailActivity := activities.NewSendMailActivity()
	emailInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"recipient": cfg.UserMail,
			"subject":   "[CREDIMI] Action required to continue your conformance checks",
			"template":  activities.ContinueConformanceCheckEmailTemplate,
			"data": map[string]any{
				"AppName":          cfg.AppName,
				"AppLogo":          cfg.AppLogo,
				"UserName":         cfg.UserName,
				"VerificationLink": u.String(),
			},
		},
	}
	if err := emailActivity.Configure(&emailInput); err != nil {
		logger.Error("Email configure failed", "error", err)
		return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMeta)
	}
	if err := workflow.ExecuteActivity(ctx, emailActivity.Name(), emailInput).Get(ctx, nil); err != nil {
		logger.Error("Failed to send mail", "error", err)
		return StepCIAndEmailResult{}, workflowengine.NewWorkflowError(err, cfg.RunMeta)
	}

	return StepCIAndEmailResult{
		Captures: captures,
	}, nil
}

type StartCheckWorkflow struct{}

func (StartCheckWorkflow) Name() string {
	return "Start conformance check"
}

func (StartCheckWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *StartCheckWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)
	logger := workflow.GetLogger(ctx)
	runMeta := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
	}

	suite, ok := input.Payload["suite"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError("suite", runMeta)
	}

	var stepCIPayload map[string]any
	var ewcSessionID string
	switch suite {
	case "openid_conformance_suite":
		variant, ok := input.Payload["variant"].(string)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError("variant", runMeta)
		}
		form, ok := input.Payload["form"].(map[string]any)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError("form", runMeta)
		}
		test, ok := input.Payload["test"].(string)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError("test", runMeta)
		}
		stepCIPayload = map[string]any{
			"variant": variant,
			"form":    form,
			"test":    test,
			"secrets": map[string]any{
				"token": utils.GetEnvironmentVariable("OPENIDNET_TOKEN", nil, true),
			},
		}
	case "ewc":
		ewcSessionID, ok = input.Payload["session_id"].(string)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError("session_id", runMeta)
		}
		stepCIPayload = map[string]any{
			"session_id": ewcSessionID,
		}
	default:
		return workflowengine.WorkflowResult{}, fmt.Errorf("unsupported suite: %s", suite)
	}
	cfg := StepCIAndEmailConfig{
		AppURL:        input.Config["app_url"].(string),
		AppName:       input.Config["app_name"].(string),
		AppLogo:       input.Config["app_logo"].(string),
		UserName:      input.Config["user_name"].(string),
		UserMail:      input.Payload["user_mail"].(string),
		Template:      input.Config["template"].(string),
		StepCIPayload: stepCIPayload,
		Namespace:     input.Config["namespace"].(string),
		RunMeta:       runMeta,
		Suite:         suite,
	}

	setupResult, err := RunStepCIAndSendMail(ctx, cfg)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	switch suite {
	case "openid_conformance_suite":
		rid, ok := setupResult.Captures["rid"].(string)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
				"rid",
				runMeta,
			)
		}
		child := OpenIDNetLogsWorkflow{}
		ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:        workflow.GetInfo(ctx).WorkflowExecution.ID + "-log",
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
		})

		childCtx, _ := workflow.NewDisconnectedContext(ctx)
		err = workflow.ExecuteChildWorkflow(
			childCtx,
			child.Name(),
			workflowengine.WorkflowInput{
				Payload: map[string]any{
					"rid":   rid,
					"token": utils.GetEnvironmentVariable("OPENIDNET_TOKEN"),
				},
				Config: map[string]any{
					"app_url":  cfg.AppURL,
					"interval": 1.0,
				},
			}).GetChildWorkflowExecution().Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to execute child workflow", "error", err)
			errCode := errorcodes.Codes[errorcodes.ChildWorkflowExecutionError]
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				workflowengine.NewAppError(errCode, err.Error(), nil),
				cfg.RunMeta,
			)
		}
		return workflowengine.WorkflowResult{
			Message: "Check completed successfully",
			Output: map[string]any{
				"deeplink": setupResult.Captures["deeplink"],
			},
		}, nil
	case "ewc":
		standard, ok := input.Config["memo"].(map[string]any)["standard"].(string)
		if !ok {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				"standard",
				runMeta,
			)
		}
		var checkEndpoint string
		switch standard {
		case "openid4vp_wallet":
			checkEndpoint = "https://ewc.api.forkbomb.eu/verificationStatus"
		case "openid4vci_wallet":
			checkEndpoint = "https://ewc.api.forkbomb.eu/issueStatus"
		default:
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				fmt.Sprintf("unsupported standard %s", standard),
				runMeta,
			)
		}
		child := EWCStatusWorkflow{}
		ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:        workflow.GetInfo(ctx).WorkflowExecution.ID + "-status",
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
		})
		childCtx, _ := workflow.NewDisconnectedContext(ctx)
		err = workflow.ExecuteChildWorkflow(
			childCtx,
			child.Name(),
			workflowengine.WorkflowInput{
				Payload: map[string]any{
					"session_id": ewcSessionID,
				},
				Config: map[string]any{
					"app_url":        cfg.AppURL,
					"interval":       1.0,
					"check_endpoint": checkEndpoint,
				},
			}).GetChildWorkflowExecution().Get(ctx, nil)

		if err != nil {
			logger.Error("Failed to execute child workflow", "error", err)
			errCode := errorcodes.Codes[errorcodes.ChildWorkflowExecutionError]
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				workflowengine.NewAppError(errCode, err.Error(), nil),
				cfg.RunMeta,
			)
		}
		return workflowengine.WorkflowResult{
			Message: "Check completed successfully",
			Output: map[string]any{
				"deeplink": setupResult.Captures["deeplink"],
			},
		}, nil
	default:
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			fmt.Sprintf("unsupported suite %s", suite),
			runMeta,
		)
	}
}
