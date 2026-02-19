// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
)

func TestReleaseMobileRunnerPermitActivityDisabled(t *testing.T) {
	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "true")

	activity := NewReleaseMobileRunnerPermitActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: mobilerunnersemaphore.MobileRunnerSemaphorePermit{
			RunnerID: "runner-1",
			LeaseID:  "lease-1",
		},
	})
	require.NoError(t, err)
}

func TestReleaseMobileRunnerPermitActivityEarlyReturnAndDecodeError(t *testing.T) {
	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "false")
	activity := NewReleaseMobileRunnerPermitActivity()

	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-permit-payload",
	})
	require.Error(t, err)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: mobilerunnersemaphore.MobileRunnerSemaphorePermit{
			RunnerID: "runner-1",
			LeaseID:  "   ",
		},
	})
	require.NoError(t, err)
}

func TestMobileRunnerSemaphoreHelpers(t *testing.T) {
	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "true")
	require.True(t, isMobileRunnerSemaphoreDisabled())

	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "YES")
	require.True(t, isMobileRunnerSemaphoreDisabled())

	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "0")
	require.False(t, isMobileRunnerSemaphoreDisabled())
}

func TestIsNotFoundError(t *testing.T) {
	require.True(t, isNotFoundError(&serviceerror.NotFound{Message: "missing"}))
	require.False(t, isNotFoundError(errors.New("different error")))
}
