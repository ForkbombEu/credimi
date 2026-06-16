// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestBaseActivityNewActivityError(t *testing.T) {
	activity := BaseActivity{Name: "Example"}
	err := activity.NewActivityError(
		ActivityError{
			Code:     errorcodes.Codes[errorcodes.CommandExecutionFailed].Code,
			Summary:  "Maestro flow failed",
			Category: "external_command",
			Details:  map[string]any{"exit_code": 1},
		},
	)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, errorcodes.Codes[errorcodes.CommandExecutionFailed].Code, appErr.Type())
	require.Equal(t, "Maestro flow failed", appErr.Message())

	var failure ActivityError
	require.NoError(t, appErr.Details(&failure))
	require.Equal(t, errorcodes.Codes[errorcodes.CommandExecutionFailed].Code, failure.Code)
	require.Equal(t, "Example", failure.ActivityName)
	require.Equal(t, "external_command", failure.Category)
}

func TestBaseActivityNewNonRetryableActivityError(t *testing.T) {
	activity := BaseActivity{Name: "Example"}
	err := activity.NewNonRetryableActivityError(ActivityError{
		Code:    "E456",
		Summary: "nope",
		Details: map[string]any{
			"value": "x",
		},
	})

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "E456", appErr.Type())
	require.Equal(t, "nope", appErr.Message())
	require.True(t, appErr.NonRetryable())

	var details ActivityError
	require.NoError(t, appErr.Details(&details))
	require.Equal(t, "Example", details.ActivityName)
	require.Equal(t, "x", details.Details["value"])
}
