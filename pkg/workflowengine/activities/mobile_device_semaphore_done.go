// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

type ReportMobileDeviceSemaphoreDoneInput struct {
	OwnerNamespace string `json:"owner_namespace"`
	LeaderDeviceID string `json:"leader_device_id"`
	TicketID       string `json:"ticket_id"`
	WorkflowID     string `json:"workflow_id"`
	RunID          string `json:"run_id"`
	WorkflowResult string `json:"workflow_result,omitempty"`
}

type ReportMobileDeviceSemaphoreDoneActivity struct {
	workflowengine.BaseActivity
	app core.App
	// sessionLookup returns the running, Credimi-started action author session
	// run sessionRunID of sessionID in namespace.
	sessionLookup func(
		ctx context.Context,
		namespace, sessionID, sessionRunID string,
	) (ActionAuthorSessionRef, error)
}

// ReportMobileDeviceSemaphoreDoneActivityName reports a finished run to the
// leader device's semaphore.
const ReportMobileDeviceSemaphoreDoneActivityName = "Report mobile device semaphore done"

// NewReportMobileDeviceSemaphoreDoneActivity returns the activity; callers
// that only need its Name may pass a nil app.
func NewReportMobileDeviceSemaphoreDoneActivity(
	app core.App,
) *ReportMobileDeviceSemaphoreDoneActivity {
	return &ReportMobileDeviceSemaphoreDoneActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: ReportMobileDeviceSemaphoreDoneActivityName,
		},
		app: app,
		sessionLookup: func(
			ctx context.Context,
			namespace, sessionID, sessionRunID string,
		) (ActionAuthorSessionRef, error) {
			c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
			if err != nil {
				return ActionAuthorSessionRef{}, fmt.Errorf(
					"look up action author session: %w",
					err,
				)
			}
			return lookupSessionStart(ctx, app, c, namespace, sessionID, sessionRunID)
		},
	}
}

// authorizeRunnerNamespaceReport accepts a report from a runner namespace only
// from the trusted action author session run of the reported ticket, for its
// own device and organization.
func (a *ReportMobileDeviceSemaphoreDoneActivity) authorizeRunnerNamespaceReport(
	ctx context.Context,
	caller authorCaller,
	payload ReportMobileDeviceSemaphoreDoneInput,
) error {
	const notSession = "only an action author session may report a run from a runner namespace"
	if caller.WorkflowType != ActionAuthorSessionWorkflowName {
		return actionAuthorError(&a.BaseActivity, sessionForbidden(notSession))
	}
	namespace, ok := actionauthor.SessionNamespace(caller.WorkflowID)
	if !ok || namespace != caller.Namespace {
		return actionAuthorError(&a.BaseActivity, sessionForbidden(notSession))
	}
	runnerID, _ := actionauthor.SessionRunnerID(caller.WorkflowID)
	ticketID := strings.TrimSpace(payload.TicketID)
	if caller.WorkflowID != actionauthor.SessionWorkflowID(runnerID, ticketID) {
		return actionAuthorError(&a.BaseActivity, sessionForbidden(
			fmt.Sprintf("ticket %s is not this session's ticket", payload.TicketID),
		))
	}
	if strings.TrimSpace(payload.WorkflowID) != caller.WorkflowID ||
		strings.TrimSpace(payload.RunID) != caller.RunID {
		return actionAuthorError(&a.BaseActivity, sessionForbidden("the report names another run"))
	}
	ref, err := a.sessionLookup(ctx, caller.Namespace, caller.WorkflowID, caller.RunID)
	if err != nil {
		if errors.Is(err, actionauthor.ErrForbidden) {
			return actionAuthorError(&a.BaseActivity, err)
		}
		return lookupError(&a.BaseActivity, err)
	}
	if canonify.NormalizePath(payload.LeaderDeviceID) != canonify.NormalizePath(ref.DeviceID) {
		return actionAuthorError(&a.BaseActivity, sessionForbidden(
			"the leader device is not the session's device",
		))
	}
	if strings.TrimSpace(payload.OwnerNamespace) != ref.Organization {
		return actionAuthorError(&a.BaseActivity, sessionForbidden(
			"the owner namespace is not the session's organization",
		))
	}
	return nil
}

func (a *ReportMobileDeviceSemaphoreDoneActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ReportMobileDeviceSemaphoreDoneActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ReportMobileDeviceSemaphoreDoneInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	ticketID := strings.TrimSpace(payload.TicketID)
	leaderDeviceID := strings.TrimSpace(payload.LeaderDeviceID)
	ownerNamespace := strings.TrimSpace(payload.OwnerNamespace)
	if ticketID == "" || leaderDeviceID == "" || ownerNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "ticket_id, leader_device_id, and owner_namespace are required",
		})
	}
	// From a runner namespace only an action author session may report, and
	// only its own run: the runner owner can start anything there.
	if callerRunnerNamespace(ctx) != "" {
		if err := a.authorizeRunnerNamespaceReport(ctx, activityCaller(ctx), payload); err != nil {
			return result, err
		}
	}

	if isMobileDeviceSemaphoreDisabled() {
		return result, nil
	}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return result, err
	}

	updateReq := mobiledevicesemaphore.MobileDeviceSemaphoreRunDoneRequest{
		TicketID:       ticketID,
		OwnerNamespace: ownerNamespace,
		WorkflowID:     strings.TrimSpace(payload.WorkflowID),
		RunID:          strings.TrimSpace(payload.RunID),
		WorkflowResult: strings.TrimSpace(payload.WorkflowResult),
	}

	handle, err := temporalClient.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   mobiledevicesemaphore.WorkflowID(leaderDeviceID),
		UpdateName:   mobiledevicesemaphore.RunDoneUpdate,
		UpdateID:     fmt.Sprintf("run-done/%s", ticketID),
		Args:         []interface{}{updateReq},
		WaitForStage: tclient.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		if isNotFoundError(err) {
			return result, nil
		}
		return result, err
	}

	var status mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView
	if err := handle.Get(ctx, &status); err != nil {
		if isNotFoundError(err) {
			return result, nil
		}
		return result, err
	}

	return result, nil
}

func isMobileDeviceSemaphoreDisabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("MOBILE_DEVICE_SEMAPHORE_DISABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func isNotFoundError(err error) bool {
	var notFound *serviceerror.NotFound
	return errors.As(err, &notFound)
}
