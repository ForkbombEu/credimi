// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

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
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const openIDConformancePollInterval = 5 * time.Second

// openIDNetLogsActivityChangeID gates fetching OpenID certification logs
// through the activity that reads the token on the worker.
const openIDNetLogsActivityChangeID = "credimi-2026-10-openidnet-logs-activity"

// openIDNetLogsRequest returns the activity name and input that poll the
// certification logs of rid. Call it once before the polling loop. Runs
// recorded before the change keep the HTTP activity with the bearer token,
// sent with the given HTTP timeout.
func openIDNetLogsRequest(
	ctx workflow.Context,
	rid string,
	token string,
	httpTimeout string,
) (string, workflowengine.ActivityInput) {
	if workflow.GetVersion(ctx, openIDNetLogsActivityChangeID, workflow.DefaultVersion, 1) == 1 {
		return activities.NewOpenIDNetLogsActivity().Name(), workflowengine.ActivityInput{
			Payload: activities.OpenIDNetLogsPayload{Rid: rid},
		}
	}

	return activities.NewHTTPActivity().Name(), workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				"https://www.certification.openid.net/api/log",
				url.PathEscape(rid),
			),
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", token),
			},
			QueryParams: map[string]string{
				"public": "false",
			},
			ExpectedStatus: 200,
			Timeout:        httpTimeout,
		},
	}
}

// openIDConformanceActivityOptions extends DefaultActivityOptions with longer
// timeouts to accommodate StepCI setup and follow-up log polling.
var openIDConformanceActivityOptions = workflow.ActivityOptions{
	ScheduleToCloseTimeout: time.Hour,
	StartToCloseTimeout:    time.Minute * 30,
	RetryPolicy:            retryPolicy,
}

var openIDConformancePollingActivityOptions = workflow.ActivityOptions{
	ScheduleToCloseTimeout: time.Minute,
	StartToCloseTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		MaximumAttempts: 1,
	},
}

// OpenIDConformanceWorkflowPayload is the input payload for automated OpenID
// certification suite workflows.
type OpenIDConformanceWorkflowPayload struct {
	Parameters map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	UserMail   string         `json:"user_mail"            yaml:"user_mail"            validate:"required"`
	TestName   string         `json:"test"                 yaml:"test"                 validate:"required"`
}

func runOpenIDConformanceWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	options workflow.ActivityOptions,
	notifyLogs bool,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, options)

	payload, err := workflowengine.DecodePayload[OpenIDConformanceWorkflowPayload](input.Payload)
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

	stepCIPayload := activities.StepCIWorkflowActivityPayload{
		Data: payload.Parameters,
	}
	stepCIPayload.Data["test"] = payload.TestName

	result, err := RunStepCIAndSendMail(ctx, StepCIAndEmailConfig{
		AppURL:        appURL,
		AppName:       workflowengine.AsString(input.Config["app_name"]),
		AppLogo:       workflowengine.AsString(input.Config["app_logo"]),
		UserName:      workflowengine.AsString(input.Config["user_name"]),
		UserMail:      payload.UserMail,
		Template:      template,
		StepCIPayload: stepCIPayload,
		Secrets: map[string]any{
			"token": utils.GetEnvironmentVariable("OPENIDNET_TOKEN", nil, true),
		},
		Namespace:   workflowengine.AsString(input.Config["namespace"]),
		RunMetadata: input.RunMetadata,
		Suite:       OpenIDConformanceSuite,
		SendMail:    false,
	})
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	rid := openIDConformanceDeviceID(result.Captures)
	if rid == "" {
		return workflowengine.WorkflowResult{},
			workflowengine.NewStepCIOutputError("rid", result.Captures, input.RunMetadata)
	}

	return pollOpenIDConformanceLogs(
		ctx,
		rid,
		utils.GetEnvironmentVariable("OPENIDNET_TOKEN"),
		notifyLogs,
		input.RunMetadata,
	)
}

func openIDConformanceDeviceID(captures map[string]any) string {
	if rid, ok := captures["rid"].(string); ok && rid != "" {
		return rid
	}
	deviceID, _ := captures["device_id"].(string)
	return deviceID
}

func pollOpenIDConformanceLogs(
	ctx workflow.Context,
	deviceID string,
	token string,
	notifyLogs bool,
	metadata *workflowengine.WorkflowRunMetadata,
) (workflowengine.WorkflowResult, error) {
	pollCtx := workflow.WithActivityOptions(ctx, openIDConformancePollingActivityOptions)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	activityName, request := openIDNetLogsRequest(ctx, deviceID, token, "30")
	var logsTracker realtimeLogsTracker

	for {
		var httpResponse workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(pollCtx, activityName, request).
			Get(pollCtx, &httpResponse); err != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, metadata)
		}

		logs := workflowengine.AsSliceOfMaps(workflowengine.AsMap(httpResponse.Output)["body"])
		if notifyLogs && logsTracker.changed(logs) {
			if err := notifyOpenIDConformanceLogs(pollCtx, workflowID, logs); err != nil {
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					err,
					metadata,
				)
			}
		}
		if len(logs) == 0 {
			if err := workflow.Sleep(ctx, openIDConformancePollInterval); err != nil {
				return workflowengine.WorkflowResult{}, err
			}
			continue
		}

		lastLog := logs[len(logs)-1]
		lastResult := workflowengine.AsString(lastLog["result"])
		if lastResult != openIDCertificationResultFinished &&
			lastResult != openIDCertificationResultInterrupted {
			if err := workflow.Sleep(ctx, openIDConformancePollInterval); err != nil {
				return workflowengine.WorkflowResult{}, err
			}
			continue
		}

		testModuleResult := workflowengine.AsString(lastLog["testmodule_result"])
		if lastResult == openIDCertificationResultInterrupted ||
			testModuleResult == openIDCertificationTestModuleFailed {
			errCode := errorcodes.Codes[errorcodes.OpenID4VCIIssuerCheckFailed]
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: errCode.Description,
					Details: map[string]any{"payload": logs},
				},
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				metadata,
			)
		}

		return workflowengine.WorkflowResult{
			Message: "Check completed successfully",
			Log:     logs,
		}, nil
	}
}

func notifyOpenIDConformanceLogs(
	ctx workflow.Context,
	workflowID string,
	logs []map[string]any,
) error {
	return sendRealtimeLogsUpdate(ctx, workflowID+OpenID4VPWalletSubscription, logs)
}
