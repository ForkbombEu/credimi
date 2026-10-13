// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
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
		return result, recordLookupError(&a.BaseActivity, err)
	}
	result.Output = info
	return result, nil
}

// mobileDevice returns the mobile device at identifier. It returns
// mobilerunner.ErrDeviceNotFound when identifier is not a mobile device and
// mobilerunner.ErrDeviceRunnerNotFound when its runner is missing.
func mobileDevice(app core.App, identifier string) (MobileDeviceInfo, error) {
	deviceID := canonify.NormalizePath(identifier)
	device, runner, err := mobilerunner.ResolveDevice(app, deviceID)
	if err != nil {
		return MobileDeviceInfo{}, err
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
	org, err := pbutils.FindOrganizationByNamespace(a.app, payload.OwnerNamespace)
	if err != nil {
		return result, recordLookupError(&a.BaseActivity, err)
	}
	if err := mobilerunner.ValidateDeviceAccess(a.app, org.Id, payload.DeviceIDs); err != nil {
		return result, recordLookupError(&a.BaseActivity, err)
	}
	return result, nil
}
