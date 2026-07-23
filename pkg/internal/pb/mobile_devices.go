// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

// RegisterMobileDeviceHooks enforces the immutable device identity and releases
// only the deleted device's scheduling resource.
func RegisterMobileDeviceHooks(app core.App) {
	app.OnRecordCreate("mobile_devices").BindFunc(func(e *core.RecordEvent) error {
		return validateMobileDeviceOwner(app, e.Record)
	})
	app.OnRecordUpdate("mobile_devices").BindFunc(func(e *core.RecordEvent) error {
		original := e.Record.Original()
		if original != nil {
			for _, field := range []string{"owner", "runner", "name", "canonified_name"} {
				if e.Record.GetString(field) != original.GetString(field) {
					return fmt.Errorf("mobile device %s is immutable", field)
				}
			}
		}
		return validateMobileDeviceOwner(app, e.Record)
	})
	app.OnRecordAfterDeleteSuccess("mobile_devices").BindFunc(func(e *core.RecordEvent) error {
		if err := shutdownMobileDeviceSemaphore(
			app,
			e.Record,
			"mobile device deleted",
		); err != nil {
			return err
		}
		return e.Next()
	})
}

func validateMobileDeviceOwner(app core.App, device *core.Record) error {
	runner, err := app.FindRecordById("mobile_runners", device.GetString("runner"))
	if err != nil {
		return fmt.Errorf("load mobile device runner: %w", err)
	}
	if device.GetString("owner") != runner.GetString("owner") {
		return fmt.Errorf("mobile device owner must match runner owner")
	}
	return nil
}

func mobileDeviceRecordIdentifier(app core.App, record *core.Record) (string, error) {
	deviceID, err := canonify.BuildPath(app, record, canonify.CanonifyPaths["mobile_devices"], "")
	if err != nil {
		return "", err
	}
	return canonify.NormalizePath(deviceID), nil
}

func shutdownMobileDeviceSemaphore(app core.App, device *core.Record, reason string) error {
	deviceID, err := mobileDeviceRecordIdentifier(app, device)
	if err != nil {
		return fmt.Errorf("build mobile device identifier: %w", err)
	}
	temporalClient, err := mobileRunnerShutdownTemporalClient(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return fmt.Errorf("create semaphore temporal client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), mobileRunnerShutdownAcceptedTimeout)
	defer cancel()
	_, err = temporalClient.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID: workflows.MobileRunnerSemaphoreWorkflowID(deviceID),
		UpdateName: workflows.MobileRunnerSemaphoreShutdownRunnerUpdate,
		UpdateID:   "shutdown/" + deviceID,
		Args: []interface{}{
			workflows.MobileRunnerSemaphoreShutdownRunnerRequest{Reason: reason},
		},
		WaitForStage: tclient.WorkflowUpdateStageAccepted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("request device semaphore workflow shutdown: %w", err)
	}
	return nil
}
