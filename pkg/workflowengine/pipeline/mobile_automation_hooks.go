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
	input workflowengine.WorkflowInput,
) error {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

	httpActivity := activities.NewHTTPActivity()
	startEmuActivity := activities.NewStartEmulatorActivity()
	installActivity := activities.NewApkInstallActivity()
	// unlockActivity := activities.NewUnlockEmulatorActivity()
	mobileServerURL := utils.GetEnvironmentVariable("MAESTRO_WORKER", "http://localhost:8050")

	startedEmulators := make(map[string]string)
	for i := range *steps {
		step := &(*steps)[i]

		if step.Use != "mobile-automation" {
			continue
		}

		logger.Info("MobileAutomationSetupHook: processing step", "id", step.ID)

		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		payload, err := workflowengine.DecodePayload[workflows.MobileAutomationWorkflowPipelinePayload](
			step.With.Payload,
		)
		if err != nil {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error decoding payload for step %s: %s", step.ID, err.Error()),
			)
		}
		// If action_code is present, version_id is REQUIRED
		if payload.ActionCode != "" {
			if payload.VersionID == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("missing or invalid version_id for step %s", step.ID))
			}
		}
		// If action_code is NOT present -> action_id is REQUIRED
		if payload.ActionCode == "" {
			if payload.ActionID == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("missing or invalid action_id for step %s", step.ID),
				)
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
					"version_identifier": payload.VersionID,
					"action_identifier":  payload.ActionID,
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
				fmt.Sprintf(
					"%s: missing apk_path in response for step %s",
					errCode.Description,
					step.ID,
				),
				body,
			)
		}
		versionIdentifier, ok := body["version_id"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing version_id in response for step %s",
					errCode.Description,
					step.ID,
				),
				body,
			)
		}
		actionCode := payload.ActionCode
		if actionCode == "" {
			actionCode, ok = body["code"].(string)
			if !ok || actionCode == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf(
						"%s: missing action_code in response for step %s",
						errCode.Description,
						step.ID,
					),
					body,
				)
			}
			SetPayloadValue(&step.With.Payload, "action_code", actionCode)
			SetPayloadValue(&step.With.Payload, "stored_action_code", true)
		}
		SetPayloadValue(&step.With.Payload, "version_id", versionIdentifier)

		if serial, ok := startedEmulators[versionIdentifier]; ok {
			logger.Info(
				"Emulator already started, skipping start",
				"version",
				versionIdentifier,
				"serial",
				serial,
			)
			SetPayloadValue(&step.With.Payload, "emulator_serial", serial)
			continue
		}

		mobileAo := *input.ActivityOptions
		mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
		mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)
		startResult := workflowengine.ActivityResult{}
		startEmuInput := workflowengine.ActivityInput{
			Payload: map[string]any{"version_id": versionIdentifier},
		}
		err = workflow.ExecuteActivity(mobileCtx, startEmuActivity.Name(), startEmuInput).
			Get(ctx, &startResult)
		if err != nil {
			return err
		}
		serial, ok := startResult.Output.(map[string]any)["serial"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing serial in response for step %s",
					errCode.Description,
					step.ID,
				),
				startResult.Output,
			)
		}
		SetPayloadValue(&step.With.Payload, "emulator_serial", serial)

		installInput := workflowengine.ActivityInput{
			Payload: map[string]any{"apk": apkPath, "emulator_serial": serial},
		}
		if err := workflow.ExecuteActivity(mobileCtx, installActivity.Name(), installInput).Get(ctx, nil); err != nil {
			return err
		}
		startedEmulators[versionIdentifier] = serial
		// unlockInput := workflowengine.ActivityInput{Payload: map[string]any{"emulator_serial": serial}}
		// if err := workflow.ExecuteActivity(mobileCtx, unlockActivity.Name(), unlockInput).Get(ctx, nil); err != nil {
		// 	return err
		// }
	}

	return nil
}

func MobileAutomationCleanupHook(
	ctx workflow.Context,
	steps []StepDefinition,
	input workflowengine.WorkflowInput,
) error {
	logger := workflow.GetLogger(ctx)
	mobileAo := *input.ActivityOptions
	mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
	mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)

	stopActivity := activities.NewStopEmulatorActivity()

	stoppedEmulators := make(map[string]struct{})
	for _, step := range steps {
		if step.Use != "mobile-automation" {
			continue
		}

		payload, err := workflowengine.DecodePayload[workflows.MobileAutomationWorkflowPayload](
			step.With.Payload,
		)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error decoding payload for step %s: %s", step.ID, err.Error()),
			)
		}

		if payload.EmulatorSerial == "" {
			logger.Error("MobileAutomationCleanupHook: no emulator serial found.", "step", step.ID)
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				fmt.Sprintf("missing or invalid emulator serial	 for step %s", step.ID),
			)
		}

		logger.Info(
			"MobileAutomationCleanupHook: stopping emulator",
			"emulator",
			payload.EmulatorSerial,
			"step",
			step.ID,
		)
		if _, ok := stoppedEmulators[payload.EmulatorSerial]; ok {
			logger.Info("Emulator already stopped", "emulator", payload.EmulatorSerial)
			continue
		}

		stopInput := workflowengine.ActivityInput{
			Payload: map[string]any{"emulator_serial": payload.EmulatorSerial},
		}
		if err := workflow.ExecuteActivity(mobileCtx, stopActivity.Name(), stopInput).Get(ctx, nil); err != nil {
			logger.Error(
				"MobileAutomationCleanupHook: error stopping emulator",
				payload.EmulatorSerial,
				"error",
				err,
			)
			return err
		}

		stoppedEmulators[payload.EmulatorSerial] = struct{}{}
	}

	return nil
}
