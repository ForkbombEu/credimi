// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestBaseActivityNewActivityError(t *testing.T) {
	activity := BaseActivity{Name: "Example"}
	err := activity.NewActivityError("E123", "boom", []any{"a", "b"}, "c")

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "E123", appErr.Type())
	require.Equal(t, "[Example]: boom", appErr.Message())
	require.False(t, appErr.NonRetryable())

	var details []any
	require.NoError(t, appErr.Details(&details))
	require.Equal(t, []any{"a", "b", "c"}, details)
}

func TestBaseActivityNewNonRetryableActivityError(t *testing.T) {
	activity := BaseActivity{Name: "Example"}
	err := activity.NewNonRetryableActivityError("E456", "nope", "x")

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "E456", appErr.Type())
	require.Equal(t, "[Example]: nope", appErr.Message())
	require.True(t, appErr.NonRetryable())

	var details []any
	require.NoError(t, appErr.Details(&details))
	require.Equal(t, []any{"x"}, details)
}
