//go:build !credimi_extra

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
)

func TestMobileFlowStubActivities(t *testing.T) {
	tests := []workflowengine.ExecutableActivity{
		NewStartEmulatorActivity(),
		NewApkInstallActivity(),
		NewUnlockEmulatorActivity(),
		NewCleanupDeviceActivity(),
		NewStartRecordingActivity(),
		NewStopRecordingActivity(),
		NewRunMobileFlowActivity(),
	}

	for _, activity := range tests {
		activity := activity
		t.Run(activity.Name(), func(t *testing.T) {
			t.Helper()
			_, err := activity.Execute(
				context.Background(),
				workflowengine.ActivityInput{},
			)
			require.Error(t, err)
			require.Contains(
				t,
				err.Error(),
				errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
			)
		})
	}
}
