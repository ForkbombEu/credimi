// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"os"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type CheckFileExistsActivity struct {
	workflowengine.BaseActivity
	Input CheckFileExistsActivityPayload
}

// CheckFileExistsActivityPayload is the input payload for the CheckFileExistsActivity.
type CheckFileExistsActivityPayload struct {
	Path string `json:"path" yaml:"path" validate:"required"`
}

func NewCheckFileExistsActivity() *CheckFileExistsActivity {
	return &CheckFileExistsActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Check if a file exists",
		},
	}
}

func (a *CheckFileExistsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *CheckFileExistsActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[CheckFileExistsActivityPayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	_, err = os.Stat(payload.Path)
	exists := err == nil
	return workflowengine.ActivityResult{
		Output: exists,
	}, nil
}
