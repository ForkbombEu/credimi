// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	enumspb "go.temporal.io/api/enums/v1"
)

// MobileRunnerSessionPoolRequest names a running action author session of
// the runner.
type MobileRunnerSessionPoolRequest struct {
	RunnerID  string `json:"runner_id"  validate:"required"`
	SessionID string `json:"session_id" validate:"required"`
}

// MobileRunnerSessionPoolResponse names the daily token pool the session
// spends: "own" for a session of the runner's organization, "others" for any
// other organization's.
type MobileRunnerSessionPoolResponse struct {
	Pool string `json:"pool"`
}

// HandleMobileRunnerSessionPool tells a runner's model proxy which of the
// runner's daily token pools a running action author session of the runner
// spends; the proxy meters and enforces the limits itself.
func HandleMobileRunnerSessionPool() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerSessionPoolRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			)
		}
		record, _, apiErr := authorizeRunnerKey(e, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}
		if !record.GetBool("agent_enabled") {
			return agentNotEnabledError()
		}
		if runnerID, ok := actionauthor.SessionRunnerID(input.SessionID); !ok ||
			runnerID != record.Id {
			return runnerSessionNotFound()
		}
		ctx := e.Request.Context()
		ref, err := activities.TrustedActionAuthorSession(
			ctx,
			e.App,
			actionAuthorSessionTemporalClient,
			input.SessionID,
		)
		switch {
		case errors.Is(err, actionauthor.ErrForbidden),
			errors.Is(err, actionauthor.ErrSessionNotVisible):
			return runnerSessionNotFound()
		case err != nil:
			e.App.Logger().Warn(
				"action author session pool lookup",
				"session", input.SessionID,
				"error", err,
			)
			return temporalUnavailableError()
		}
		c, err := actionAuthorSessionTemporalClient(ref.Namespace)
		if err != nil {
			return temporalUnavailableError()
		}
		described, err := c.DescribeWorkflowExecution(ctx, ref.SessionWorkflowID, ref.SessionRunID)
		if err != nil {
			return temporalUnavailableError()
		}
		if described.GetWorkflowExecutionInfo().GetStatus() !=
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
			return runnerSessionNotFound()
		}
		org, err := e.App.FindFirstRecordByFilter(
			"organizations",
			"canonified_name = {:name}",
			dbx.Params{"name": ref.Organization},
		)
		if errors.Is(err, sql.ErrNoRows) {
			return runnerSessionNotFound()
		}
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed_to_find_organization",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, MobileRunnerSessionPoolResponse{
			Pool: string(actionauthor.AllowanceFor(record, org.Id).Pool),
		})
	}
}

// runnerSessionNotFound answers a session that is not a running action
// author session of the calling runner.
func runnerSessionNotFound() *apierror.APIError {
	return apierror.New(
		http.StatusNotFound,
		"action_author",
		"session_not_found",
		"no running action author session of this runner",
	)
}
