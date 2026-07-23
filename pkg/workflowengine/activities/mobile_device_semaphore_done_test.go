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

func TestReportMobileDeviceSemaphoreDoneActivityMissingFields(t *testing.T) {
	activity := NewReportMobileDeviceSemaphoreDoneActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReportMobileDeviceSemaphoreDoneInput{
			TicketID: "ticket-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

func TestReportMobileDeviceSemaphoreDoneActivityDecodeAndDisabled(t *testing.T) {
	activity := NewReportMobileDeviceSemaphoreDoneActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-report-payload",
	})
	require.Error(t, err)

	t.Setenv("MOBILE_DEVICE_SEMAPHORE_DISABLED", "true")
	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReportMobileDeviceSemaphoreDoneInput{
			TicketID:       "ticket-1",
			LeaderDeviceID: "runner-1",
			OwnerNamespace: "org",
			WorkflowID:     "wf-1",
			RunID:          "run-1",
		},
	})
	require.NoError(t, err)
}

func TestMobileDeviceSemaphoreDoneActivityName(t *testing.T) {
	require.Equal(
		t,
		"Report mobile device semaphore done",
		NewReportMobileDeviceSemaphoreDoneActivity().Name(),
	)
}
