// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestAuthorizeRunnerNamespaceReport(t *testing.T) {
	const (
		runnerID = "abcdefghijklmno"
		ticketID = "11111111-2222-3333-4444-555555555555"
	)
	sessionID := actionauthor.SessionWorkflowID(runnerID, ticketID)
	session := authorCaller{
		Namespace:    "runner-" + runnerID,
		WorkflowType: ActionAuthorSessionWorkflowName,
		WorkflowID:   sessionID,
		RunID:        "run-1",
	}
	report := ReportMobileDeviceSemaphoreDoneInput{
		OwnerNamespace: "org",
		LeaderDeviceID: "org/runner/pixel",
		TicketID:       ticketID,
		WorkflowID:     sessionID,
		RunID:          "run-1",
	}

	tests := []struct {
		name         string
		caller       func(*authorCaller)
		payload      func(*ReportMobileDeviceSemaphoreDoneInput)
		lookupErr    error
		wantErr      string
		wantRetrying bool
	}{
		{name: "valid"},
		{
			name:    "pipeline workflow type",
			caller:  func(c *authorCaller) { c.WorkflowType = "Run a mobile automation workflow" },
			wantErr: "only an action author session may report a run from a runner namespace",
		},
		{
			name:    "standalone activity",
			caller:  func(c *authorCaller) { c.WorkflowType = "" },
			wantErr: "only an action author session may report a run from a runner namespace",
		},
		{
			name: "session of another runner",
			caller: func(c *authorCaller) {
				c.WorkflowID = actionauthor.SessionWorkflowID("zzzzzzzzzzzzzzz", ticketID)
			},
			wantErr: "only an action author session may report a run from a runner namespace",
		},
		{
			name: "other ticket",
			payload: func(p *ReportMobileDeviceSemaphoreDoneInput) {
				p.TicketID = "99999999-2222-3333-4444-555555555555"
			},
			wantErr: "is not this session's ticket",
		},
		{
			name: "ticket with a path prefix",
			payload: func(p *ReportMobileDeviceSemaphoreDoneInput) {
				p.TicketID = "x/" + ticketID
			},
			wantErr: "is not this session's ticket",
		},
		{
			name:    "other run",
			payload: func(p *ReportMobileDeviceSemaphoreDoneInput) { p.RunID = "run-2" },
			wantErr: "the report names another run",
		},
		{
			name:    "other leader",
			payload: func(p *ReportMobileDeviceSemaphoreDoneInput) { p.LeaderDeviceID = "org/other/phone" },
			wantErr: "the leader device is not the session's device",
		},
		{
			name:    "other owner",
			payload: func(p *ReportMobileDeviceSemaphoreDoneInput) { p.OwnerNamespace = "other-org" },
			wantErr: "the owner namespace is not the session's organization",
		},
		{
			name:      "lookup forbidden",
			lookupErr: sessionForbidden("action author session is not running"),
			wantErr:   "action author session is not running",
		},
		{
			name:         "lookup not visible yet",
			lookupErr:    actionauthor.ErrSessionNotVisible,
			wantErr:      actionauthor.ErrSessionNotVisible.Error(),
			wantRetrying: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caller, payload := session, report
			if tc.caller != nil {
				tc.caller(&caller)
			}
			if tc.payload != nil {
				tc.payload(&payload)
			}
			act := NewReportMobileDeviceSemaphoreDoneActivity(nil)
			act.sessionLookup = func(
				_ context.Context,
				namespace, sessionID, sessionRunID string,
			) (ActionAuthorSessionRef, error) {
				assert.Equal(t, caller.Namespace, namespace)
				assert.Equal(t, caller.WorkflowID, sessionID)
				assert.Equal(t, caller.RunID, sessionRunID)
				if tc.lookupErr != nil {
					return ActionAuthorSessionRef{}, tc.lookupErr
				}
				return ActionAuthorSessionRef{
					DeviceID:     "org/runner/pixel",
					Organization: "org",
				}, nil
			}

			err := act.authorizeRunnerNamespaceReport(context.Background(), caller, payload)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			var appErr *temporal.ApplicationError
			require.True(t, errors.As(err, &appErr))
			assert.Equal(t, !tc.wantRetrying, appErr.NonRetryable())
		})
	}
}
