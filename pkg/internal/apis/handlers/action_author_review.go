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

// HandleActionAuthorReview signs the caller's acceptance of a candidate and
// delivers the decision to the agent's pending save review. Only the user
// who started the session can review it.
func HandleActionAuthorReview() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorReviewRequest](e)
		if err != nil {
			return authoringAPIError(e, actionauthor.ErrInvalid)
		}
		if e.Auth == nil {
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		actor := authoringActor(e.Auth)
		organization, err := e.App.FindFirstRecordByData(
			"organizations",
			"canonified_name",
			input.Namespace,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return actionAuthorSessionNotFound()
		}
		if err != nil {
			return authoringAPIError(e, err)
		}
		owns, err := actionauthor.OwnsOrganization(e.App, actor, organization.Id)
		if errors.Is(err, actionauthor.ErrForbidden) || (err == nil && !owns) {
			return actionAuthorSessionNotFound()
		}
		if err != nil {
			return authoringAPIError(e, err)
		}
		c, err := queueTemporalClient(input.Namespace)
		if err != nil {
			return temporalUnavailableError()
		}
		ctx := e.Request.Context()
		ref, err := lookupActionAuthorSession(ctx, c, input.AuthorWorkflowID, "")
		if errors.Is(err, actionauthor.ErrForbidden) {
			return actionAuthorSessionNotFound()
		}
		if err != nil {
			return temporalUnavailableError()
		}
		if ref.Session.ActorID != actor.ActorID ||
			ref.Session.ActorCollection != actor.ActorCollection {
			return apierror.New(
				http.StatusForbidden,
				"action_author",
				"forbidden",
				"only the user who started this session can review its actions",
			)
		}
		result := map[string]any{"accepted": false, "approval": ""}
		if *input.Accepted {
			approval, err := actionauthor.SignApproval(e.App, actionauthor.Approval{
				AuthorWorkflowID:   input.AuthorWorkflowID,
				CandidateSHA256:    input.CandidateSHA256,
				ApproverID:         actor.ActorID,
				ApproverCollection: actor.ActorCollection,
				ApprovedAt:         time.Now().Unix(),
			})
			if err != nil {
				return authoringAPIError(e, err)
			}
			result = map[string]any{"accepted": true, "approval": approval}
		}
		err = updateActionAuthorWorkflow(ctx, c, input.AuthorWorkflowID, input.ToolID, result)
		if err != nil {
			return actionAuthorReviewUpdateError(err)
		}
		return e.JSON(http.StatusOK, ActionAuthorReviewResponse{
			Accepted: *input.Accepted,
			ToolID:   input.ToolID,
		})
	}
}

// actionAuthorReviewUpdateError answers a rejected update (an unknown or
// settled tool id, an invalid result, an ended workflow) as no longer
// pending, and other failures as Temporal being unavailable.
func actionAuthorReviewUpdateError(err error) *apierror.APIError {
	var applicationErr *temporal.ApplicationError
	var notFound *serviceerror.NotFound
	var invalid *serviceerror.InvalidArgument
	var precondition *serviceerror.FailedPrecondition
	if errors.As(err, &applicationErr) || errors.As(err, &notFound) ||
		errors.As(err, &invalid) || errors.As(err, &precondition) {
		return apierror.New(
			http.StatusConflict,
			"action_author",
			"review_not_pending",
			"this save review is no longer pending; reload the chat",
		)
	}
	return temporalUnavailableError()
}
