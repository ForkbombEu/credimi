// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestAcquireMobileRunnerPermitActivityMapAcquireError(t *testing.T) {
	activity := NewAcquireMobileRunnerPermitActivity()

	testCases := []struct {
		name        string
		err         error
		waitTimeout time.Duration
		code        string
		queueLen    *int
		assertDetails bool
	}{
		{
			name:        "timeout maps to busy",
			err:         temporal.NewApplicationError("timeout", workflows.MobileRunnerSemaphoreErrTimeout),
			waitTimeout: time.Minute,
			code:        errorcodes.Codes[errorcodes.MobileRunnerBusy].Code,
			queueLen:    intPtr(3),
			assertDetails: true,
		},
		{
			name:        "generic error maps to pipeline execution error",
			err:         errors.New("network down"),
			waitTimeout: 0,
			code:        errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapped := activity.mapAcquireError(tc.err, "runner-1", tc.waitTimeout, tc.queueLen)
			var appErr *temporal.ApplicationError
			require.True(t, errors.As(mapped, &appErr))
			require.Equal(t, tc.code, appErr.Type())
			if tc.assertDetails {
				var details map[string]any
				require.NoError(t, appErr.Details(&details))
				require.Equal(t, "runner-1", details["runner_id"])
				require.Equal(t, tc.waitTimeout.Milliseconds(), details["waited_ms"])
				require.Equal(t, 3, details["queue_len"])
			}
		})
	}
}

func intPtr(value int) *int {
	return &value
}
