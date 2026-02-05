// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestAcquireMobileRunnerPermitActivityMapAcquireError(t *testing.T) {
	activity := NewAcquireMobileRunnerPermitActivity()

	testCases := []struct {
		name          string
		err           error
		waitTimeout   time.Duration
		code          string
		queueLen      *int
		assertDetails bool
	}{
		{
			name: "timeout maps to busy",
			err: temporal.NewApplicationError(
				"timeout",
				mobilerunnersemaphore.ErrTimeout,
			),
			waitTimeout:   time.Minute,
			code:          errorcodes.Codes[errorcodes.MobileRunnerBusy].Code,
			queueLen:      intPtr(3),
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
				var detailsRaw []any
				require.NoError(t, appErr.Details(&detailsRaw))
				require.NotEmpty(t, detailsRaw)

				details, ok := detailsRaw[0].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "runner-1", details["runner_id"])
				require.Equal(t, tc.waitTimeout.Milliseconds(), details["waited_ms"])
				require.Equal(t, 3, asInt(details["queue_len"]))
			}
		})
	}
}

func intPtr(value int) *int {
	return &value
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
