// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/require"
)

func TestCESRParsingActivityErrors(t *testing.T) {
	activity := NewCESRParsingActivity()

	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{})
	require.Error(t, err)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CESRParsingActivityPayload{RawCESR: `{"v":"KERI10JSON000001_"`},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.CESRParsingError].Code)
}

func TestCESRValidateActivitySuccessAndFailure(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "et-tu-cesr")

	script := []byte("#!/bin/sh\nif [ \"$1\" = \"validate-parsed-credentials\" ]; then echo OK; echo ERR 1>&2; exit 0; fi\nexit 1\n")
	require.NoError(t, os.WriteFile(binPath, script, 0o755))
	t.Setenv("BIN", dir)

	activity := NewCESRValidateActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CesrValidateActivityPayload{Events: "events"},
	})
	require.NoError(t, err)
	require.Equal(t, "OK\n", result.Output)
	require.Equal(t, []string{"ERR\n"}, result.Log)

	// make script fail
	require.NoError(t, os.WriteFile(binPath, []byte("#!/bin/sh\nexit 2\n"), 0o755))
	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CesrValidateActivityPayload{Events: "events"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.CommandExecutionFailed].Code)
}

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

func TestReportMobileRunnerSemaphoreDoneActivityMissingFields(t *testing.T) {
	activity := NewReportMobileRunnerSemaphoreDoneActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReportMobileRunnerSemaphoreDoneInput{
			TicketID: "ticket-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

func TestQueryMobileRunnerSemaphoreRunStatusActivityMissingFields(t *testing.T) {
	activity := NewQueryMobileRunnerSemaphoreRunStatusActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileRunnerSemaphoreRunStatusInput{
			RunnerID: "runner-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

func TestCheckWorkflowClosedActivityMissingFields(t *testing.T) {
	activity := NewCheckWorkflowClosedActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CheckWorkflowClosedActivityInput{
			WorkflowNamespace: "ns",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

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
			require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code)
		})
	}
}
