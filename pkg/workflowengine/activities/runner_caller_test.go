// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestCheckRunnerNamespaceOfCaller(t *testing.T) {
	a := &workflowengine.BaseActivity{Name: "probe"}
	tests := []struct {
		name    string
		caller  string
		target  string
		wantErr string
	}{
		{name: "organization caller", caller: "", target: "runner-a"},
		{name: "own runner", caller: "runner-a", target: "runner-a"},
		{
			name:    "other runner",
			caller:  "runner-a",
			target:  "runner-b",
			wantErr: "activity called from runner-a for runner b",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkRunnerNamespaceOfCaller(a, tc.caller, tc.target)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			var appErr *temporal.ApplicationError
			require.ErrorAs(t, err, &appErr)
			require.True(t, appErr.NonRetryable())
			require.Equal(
				t,
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				appErr.Type(),
			)
			require.Contains(t, appErr.Error(), tc.wantErr)
		})
	}
}

func TestRequireRunnerNamespaceOfCallerOutsideActivity(t *testing.T) {
	a := &workflowengine.BaseActivity{Name: "probe"}
	require.NoError(t, requireRunnerNamespaceOfCaller(context.Background(), a, "runner-b"))
}
