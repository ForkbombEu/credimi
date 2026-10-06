// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// MobileDeviceInfo describes a mobile device and the runner that hosts it.
type MobileDeviceInfo struct {
	DeviceID  string `json:"device_id"`
	RunnerID  string `json:"runner_id"`
	Type      string `json:"type"`
	Serial    string `json:"serial"`
	RunnerURL string `json:"runner_url"`
}

type GetMobileDeviceInput struct {
	DeviceIdentifier string `json:"device_identifier" validate:"required"`
}

// GetMobileDeviceActivity returns a mobile device and its runner URL.
type GetMobileDeviceActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewGetMobileDeviceActivity(app core.App) *GetMobileDeviceActivity {
	return &GetMobileDeviceActivity{
		BaseActivity: workflowengine.BaseActivity{Name: GetMobileDeviceActivityName},
		app:          app,
	}
}

func (a *GetMobileDeviceActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *GetMobileDeviceActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[GetMobileDeviceInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	info, err := mobileDevice(a.app, payload.DeviceIdentifier)
	if err != nil {
		if errors.Is(err, mobilerunner.ErrDeviceNotFound) ||
			errors.Is(err, mobilerunner.ErrDeviceRunnerNotFound) {
			return result, credimiActivityError(
				&a.BaseActivity,
				errorcodes.RecordNotFound,
				false,
				err,
			)
		}
		return result, credimiActivityError(
			&a.BaseActivity,
			errorcodes.DatabaseOperationFailed,
			true,
			err,
		)
	}
	result.Output = info
	return result, nil
}

// mobileDevice returns the mobile device at identifier. It returns
// mobilerunner.ErrDeviceNotFound when identifier is not a mobile device and
// mobilerunner.ErrDeviceRunnerNotFound when its runner is missing.
func mobileDevice(app core.App, identifier string) (MobileDeviceInfo, error) {
	deviceID := canonify.NormalizePath(identifier)
	device, err := canonify.Resolve(app, deviceID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return MobileDeviceInfo{}, fmt.Errorf("resolve mobile device %s: %w", deviceID, err)
	}
	if err != nil || device.Collection().Name != "mobile_devices" {
		return MobileDeviceInfo{}, fmt.Errorf(
			"%w: mobile device %s was not found",
			mobilerunner.ErrDeviceNotFound,
			deviceID,
		)
	}
	runnerID := device.GetString("runner")
	runner, err := app.FindRecordById("mobile_runners", runnerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MobileDeviceInfo{}, fmt.Errorf(
				"%w: runner %s of mobile device %s",
				mobilerunner.ErrDeviceRunnerNotFound,
				runnerID,
				deviceID,
			)
		}
		return MobileDeviceInfo{}, fmt.Errorf("find mobile runner %s: %w", runnerID, err)
	}
	runnerIdentifier, err := mobilerunner.RunnerIdentifier(app, runner)
	if err != nil {
		return MobileDeviceInfo{}, fmt.Errorf("build mobile runner identifier: %w", err)
	}
	return MobileDeviceInfo{
		DeviceID:  deviceID,
		RunnerID:  runnerIdentifier,
		Type:      device.GetString("type"),
		Serial:    device.GetString("serial"),
		RunnerURL: mobilerunner.RunnerURL(runner),
	}, nil
}

type ValidateDeviceAccessInput struct {
	OwnerNamespace string   `json:"owner_namespace" validate:"required"`
	DeviceIDs      []string `json:"device_ids"`
}

// ValidateDeviceAccessActivity checks that an organization may use mobile
// devices.
type ValidateDeviceAccessActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewValidateDeviceAccessActivity(app core.App) *ValidateDeviceAccessActivity {
	return &ValidateDeviceAccessActivity{
		BaseActivity: workflowengine.BaseActivity{Name: ValidateDeviceAccessActivityName},
		app:          app,
	}
}

func (a *ValidateDeviceAccessActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ValidateDeviceAccessActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ValidateDeviceAccessInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	org, err := a.app.FindFirstRecordByFilter(
		"organizations",
		"canonified_name = {:namespace}",
		map[string]any{"namespace": payload.OwnerNamespace},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, credimiActivityError(
				&a.BaseActivity,
				errorcodes.RecordNotFound,
				false,
				fmt.Errorf("organization %s not found", payload.OwnerNamespace),
			)
		}
		return result, credimiActivityError(
			&a.BaseActivity,
			errorcodes.DatabaseOperationFailed,
			true,
			fmt.Errorf("find organization %s: %w", payload.OwnerNamespace, err),
		)
	}
	err = mobilerunner.ValidateDeviceAccess(a.app, org.Id, payload.DeviceIDs)
	switch {
	case err == nil:
		return result, nil
	case errors.Is(err, mobilerunner.ErrDeviceNotFound),
		errors.Is(err, mobilerunner.ErrDeviceRunnerNotFound):
		return result, credimiActivityError(&a.BaseActivity, errorcodes.RecordNotFound, false, err)
	case errors.Is(err, mobilerunner.ErrDeviceNotAccessible):
		return result, credimiActivityError(
			&a.BaseActivity,
			errorcodes.RecordNotAccessible,
			false,
			err,
		)
	default:
		return result, credimiActivityError(
			&a.BaseActivity,
			errorcodes.DatabaseOperationFailed,
			true,
			err,
		)
	}
}
