// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package activities is a package that provides activities for the workflow engine.
// This file contains the CesrParsingActivity and CesrValidateActivity structs and their methods.
package activities

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/ForkbombEu/et-tu-cesr/cesr"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// CESRParsingActivity is an activity that parses a CESR string
type CESRParsingActivity struct {
	workflowengine.BaseActivity
}

type CESRParsingActivityPayload struct {
	RawCESR string `json:"rawCESR" yaml:"rawCESR" validate:"required"`
}

func NewCESRParsingActivity() *CESRParsingActivity {
	return &CESRParsingActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Parse a CESR string",
		},
	}
}

// Name returns the name of the activity.
func (a *CESRParsingActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute parses a JSON string from the input payload and validates it against a registered struct type.
func (a *CESRParsingActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}

	payload, err := workflowengine.DecodePayload[CESRParsingActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	events, err := cesr.ParseCESR(payload.RawCESR)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CESRParsingError]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.RawCESR,
		)
	}
	result.Output = events
	return result, nil
}

// CESRValidateActivity is an activity that validates CESR credential events
type CESRValidateActivity struct {
	workflowengine.BaseActivity
}

type CesrValidateActivityPayload struct {
	Events string `json:"events" yaml:"events" validate:"required"`
}

func NewCESRValidateActivity() *CESRValidateActivity {
	return &CESRValidateActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Validate CESR credential events",
		},
	}
}

// Name returns the name of the activity.
func (a *CESRValidateActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute parses a JSON string from the input payload and validates it against a registered struct type.
func (a *CESRValidateActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}

	payload, err := workflowengine.DecodePayload[CesrValidateActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	binDir := utils.GetEnvironmentVariable("BIN", ".bin")
	binName := "et-tu-cesr"
	binPath := fmt.Sprintf("%s/%s", binDir, binName)
	command := "validate-parsed-credentials"
	args := []string{command, payload.Events}
	cmd := exec.CommandContext(ctx, binPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
			stderrStr,
		)
	}

	result.Output = stdoutStr
	result.Log = []string{stderrStr}
	return result, nil
}
