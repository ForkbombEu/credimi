// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// actionAuthorReviewUpdate is the agent workflow's update that settles a
// pending save review.
const actionAuthorReviewUpdate = "provide_callback_result"

// ActionAuthorReviewRequest is a human decision on a pending save review.
type ActionAuthorReviewRequest struct {
	Namespace        string `json:"namespace"          validate:"required"`
	AuthorWorkflowID string `json:"author_workflow_id" validate:"required"`
	ToolID           string `json:"tool_id"            validate:"required"`
	CandidateSHA256  string `json:"candidate_sha256"   validate:"required,len=64,hexadecimal"`
	Accepted         *bool  `json:"accepted"           validate:"required"`
	// Plan is the plan the save card shows; Credimi checks it again and binds
	// the approval to it.
	Plan actionauthor.Spec `json:"plan"`
}

// ActionAuthorReviewResponse confirms the decision the agent received.
type ActionAuthorReviewResponse struct {
	Accepted bool   `json:"accepted"`
	ToolID   string `json:"tool_id"`
}

// updateActionAuthorWorkflow settles the agent's pending review and is
// stubbed in unit tests.
var updateActionAuthorWorkflow = func(
	ctx context.Context,
	c client.Client,
	authorWorkflowID, toolID string,
	result map[string]any,
) error {
	handle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID: authorWorkflowID,
		UpdateName: actionAuthorReviewUpdate,
		Args: []any{map[string]any{
			"tool_id": toolID,
			"result":  result,
			"error":   nil,
		}},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return err
	}
	var ack any
	return handle.Get(ctx, &ack)
}

// lookupActionAuthorSession finds the running session of an agent workflow
// and is stubbed in unit tests.
var lookupActionAuthorSession = activities.LookupActionAuthorSession

func actionAuthorSessionNotFound() *apierror.APIError {
	return apierror.New(
		http.StatusNotFound,
		"action_author",
		"session_not_found",
		"action author session not found",
	)
}

// actionAuthorCallerSession authorizes the caller for the running session of
// an agent workflow: the caller must be a member of the namespace's
// organization and the user who started the session (404 session_not_found
// otherwise).
func actionAuthorCallerSession(
	e *core.RequestEvent,
	namespace, authorWorkflowID string,
) (client.Client, activities.ActionAuthorSessionRef, actionauthor.Actor, error) {
	var ref activities.ActionAuthorSessionRef
	if e.Auth == nil {
		return nil, ref, actionauthor.Actor{}, authoringAPIError(e, actionauthor.ErrForbidden)
	}
	if apiErr := requireAuthoringUser(e.Auth); apiErr != nil {
		return nil, ref, actionauthor.Actor{}, apiErr
	}
	actor := authoringActor(e.Auth)
	organization, err := e.App.FindFirstRecordByData(
		"organizations",
		"canonified_name",
		namespace,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ref, actor, actionAuthorSessionNotFound()
	}
	if err != nil {
		return nil, ref, actor, authoringAPIError(e, err)
	}
	orgID, err := pbutils.GetUserOrganizationID(e.App, actor.ActorID)
	if err != nil || orgID != organization.Id {
		return nil, ref, actor, actionAuthorSessionNotFound()
	}
	c, err := queueTemporalClient(namespace)
	if err != nil {
		return nil, ref, actor, temporalUnavailableError()
	}
	ref, err = lookupActionAuthorSession(
		e.Request.Context(),
		e.App,
		c,
		namespace,
		authorWorkflowID,
		"",
	)
	if errors.Is(err, actionauthor.ErrForbidden) {
		return nil, ref, actor, actionAuthorSessionNotFound()
	}
	if err != nil {
		return nil, ref, actor, temporalUnavailableError()
	}
	if ref.Session.ActorID != actor.ActorID {
		return nil, ref, actor, actionAuthorSessionNotFound()
	}
	return c, ref, actor, nil
}

// HandleActionAuthorReview signs the caller's acceptance of a candidate of
// the plan the save card shows, checked again for the session, and delivers
// the decision to the agent's pending save review. Only the user who started
// the session can review it.
func HandleActionAuthorReview() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorReviewRequest](e)
		if err != nil {
			return authoringAPIError(e, actionauthor.ErrInvalid)
		}
		c, ref, actor, err := actionAuthorCallerSession(
			e,
			input.Namespace,
			input.AuthorWorkflowID,
		)
		if err != nil {
			return err
		}
		// A refusal signs nothing, so it never needs the plan: a refused
		// candidate with a missing or invalid plan must still reach the agent.
		result := map[string]any{"accepted": false, "approval": ""}
		if *input.Accepted {
			selection, err := actionauthor.Select(e.App, input.Namespace, ref.Session)
			if err != nil {
				return authoringAPIError(e, err)
			}
			plan, err := activities.SessionPlan(e.App, selection, input.Plan)
			if err != nil {
				return authoringAPIError(e, err)
			}
			approval, err := actionauthor.SignApproval(e.App, actionauthor.Approval{
				AuthorWorkflowID: input.AuthorWorkflowID,
				CandidateSHA256:  input.CandidateSHA256,
				ApproverID:       actor.ActorID,
				ApprovedAt:       time.Now().Unix(),
				PlanSHA256:       actionauthor.PlanSHA256(plan),
			})
			if err != nil {
				return authoringAPIError(e, err)
			}
			result = map[string]any{"accepted": true, "approval": approval}
		}
		err = updateActionAuthorWorkflow(
			e.Request.Context(),
			c,
			input.AuthorWorkflowID,
			input.ToolID,
			result,
		)
		if err != nil {
			return actionAuthorUpdateError(
				err,
				"review_not_pending",
				"this save review is no longer pending; reload the chat",
			)
		}
		return e.JSON(http.StatusOK, ActionAuthorReviewResponse{
			Accepted: *input.Accepted,
			ToolID:   input.ToolID,
		})
	}
}

// actionAuthorUpdateError answers a rejected update (an unknown or settled
// tool id, an invalid result, an ended workflow) as 409 reason with message,
// and other failures as Temporal being unavailable.
func actionAuthorUpdateError(err error, reason, message string) *apierror.APIError {
	var applicationErr *temporal.ApplicationError
	var notFound *serviceerror.NotFound
	var invalid *serviceerror.InvalidArgument
	var precondition *serviceerror.FailedPrecondition
	if errors.As(err, &applicationErr) || errors.As(err, &notFound) ||
		errors.As(err, &invalid) || errors.As(err, &precondition) {
		return apierror.New(http.StatusConflict, "action_author", reason, message)
	}
	return temporalUnavailableError()
}
