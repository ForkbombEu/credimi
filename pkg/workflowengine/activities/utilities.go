// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"fmt"
	"os"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type CheckFileExistsActivity struct {
	workflowengine.BaseActivity
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

func (a *CheckFileExistsActivity) Execute(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
	path, ok := input.Payload["path"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'path'", errCode.Description),
		)
	}
	_, err := os.Stat(path)
	exists := err == nil
	return workflowengine.ActivityResult{
		Output: exists,
	}, nil
}
