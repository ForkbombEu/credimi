// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/workflow"
)

func MobileAutomationSetupHook(
	ctx workflow.Context,
	steps *[]StepDefinition,
	ao workflow.ActivityOptions,
) error {
	logger := workflow.GetLogger(ctx)

	// Use the activity options from the pipeline input
	ctx = workflow.WithActivityOptions(ctx, ao)

	httpActivity := activities.NewHTTPActivity()
	installActivity := activities.NewApkInstallActivity()
	mobileServerURL := utils.GetEnvironmentVariable("MAESTRO_WORKER", "http://localhost:8050")

	// iterate by index and mutate in-place
	for i := range *steps {
		step := &(*steps)[i]

		if step.Use != "mobile-automation" {
			continue
		}

		logger.Info("MobileAutomationSetupHook: processing step", "id", step.ID)

		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		var actionID string
		actionCode, actionCodeOk := step.With.Payload["action_code"].Value.(string)
		versionID, versionIDOk := step.With.Payload["version_id"].Value.(string)
		// If action_code is present, version_id is REQUIRED
		if actionCodeOk && actionCode != "" {
			if !versionIDOk || versionID == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("missing or invalid version_id for step %s", step.ID))
			}
		}
		// If action_code is NOT present -> action_id is REQUIRED
		if !actionCodeOk || actionCode == "" {
			actionIDValue, actionIDOk := step.With.Payload["action_id"].Value.(string)
			if !actionIDOk || actionIDValue == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("missing or invalid action_id for step %s", step.ID),
				)
			}
			actionID = actionIDValue

			if !versionIDOk {
				versionID = ""
			}
		}

		req := workflowengine.ActivityInput{
			Payload: map[string]any{
				"method": "POST",
				"url":    fmt.Sprintf("%s/%s", mobileServerURL, "fetch-apk-and-action"),
				"headers": map[string]any{
					"Content-Type": "application/json",
				},
				"body": map[string]any{
					"version_identifier": versionID,
					"action_identifier":  actionID,
				},
				"expected_status": 200,
			},
		}

		var res workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), req).Get(ctx, &res); err != nil {
			return err
		}
		errCode = errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		body, ok := res.Output.(map[string]any)["body"].(map[string]any)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("invalid HTTP response format for step %s", step.ID),
				res.Output,
			)
		}

		apkPath, ok := body["apk_path"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: missing apk_path in response for step %s", errCode.Description, step.ID),
				body,
			)
		}
		packageID, ok := body["package_id"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: missing package_id in response for step %s", errCode.Description, step.ID),
				body,
			)
		}

		if actionCode == "" {
			actionCode, ok = body["code"].(string)
			if !ok || actionCode == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("%s: missing action_code in response for step %s", errCode.Description, step.ID),
					body,
				)
			}
			SetPayloadValue(step.With.Payload, "action_code", actionCode)
			SetPayloadValue(step.With.Payload, "stored_action_code", true)
		}
		SetPayloadValue(step.With.Payload, "package_id", packageID)

		mobileAo := ao
		mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
		mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)
		installInput := workflowengine.ActivityInput{Payload: map[string]any{"apk": apkPath}}
		if err := workflow.ExecuteActivity(mobileCtx, installActivity.Name(), installInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	return nil
}

func MobileAutomationCleanupHook(
	ctx workflow.Context,
	steps []StepDefinition,
	ao workflow.ActivityOptions,
) error {
	logger := workflow.GetLogger(ctx)
	mobileAo := ao
	mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
	mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)

	uninstallActivity := activities.NewApkUninstallActivity()

	for _, step := range steps {

		if step.Use != "mobile-automation" {
			continue
		}

		packageID, ok := step.With.Payload["package_id"].Value.(string)
		if packageID == "" || !ok {
			logger.Error("MobileAutomationCleanupHook: no package_id found.", "step", step.ID)
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				fmt.Sprintf("missing or invalid package_id for step %s", step.ID),
			)
		}

		logger.Info("MobileAutomationCleanupHook: uninstalling package", "package", packageID, "step", step.ID)

		uninstallInput := workflowengine.ActivityInput{Payload: map[string]any{"package": packageID}}
		if err := workflow.ExecuteActivity(mobileCtx, uninstallActivity.Name(), uninstallInput).Get(ctx, nil); err != nil {
			logger.Error("MobileAutomationCleanupHook: uninstall failed", "package", packageID, "error", err)
			return err
		}
	}

	return nil
}
