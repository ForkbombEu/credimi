// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
)

// ActionAuthorPlanRequest is a human decision on a plan the agent proposed.
type ActionAuthorPlanRequest struct {
	Namespace        string            `json:"namespace"          validate:"required"`
	AuthorWorkflowID string            `json:"author_workflow_id" validate:"required"`
	ToolID           string            `json:"tool_id"            validate:"required"`
	Plan             actionauthor.Spec `json:"plan"`
	Accepted         *bool             `json:"accepted"           validate:"required"`
}

// ActionAuthorPlanResponse confirms the decision the agent received.
type ActionAuthorPlanResponse struct {
	Accepted bool   `json:"accepted"`
	ToolID   string `json:"tool_id"`
}

// HandleActionAuthorPlan validates and signs the caller's confirmation of a
// plan and delivers the decision to the agent's pending plan card. Only the
// user who started a session without a start spec can confirm its plans.
func HandleActionAuthorPlan() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorPlanRequest](e)
		if err != nil {
			return authoringAPIError(e, actionauthor.ErrInvalid)
		}
		c, ref, actor, err := actionAuthorCallerSession(
			e,
			input.Namespace,
			input.AuthorWorkflowID,
			"only the user who started this session can confirm its plans",
		)
		if err != nil {
			return err
		}
		session := ref.Session
		if !session.IsEmpty() {
			return apierror.New(
				http.StatusConflict,
				"action_author",
				"plan_fixed",
				"this session's plan was fixed when it started",
			)
		}
		result := map[string]any{
			"accepted":      false,
			"plan_token":    "",
			"plan":          nil,
			"prerequisites": []actionauthor.ResolvedPrerequisite{},
			"parameters":    []actionauthor.ParameterInfo{},
		}
		if *input.Accepted {
			selection, err := actionauthor.Select(e.App, input.Namespace, session.BuildRequest)
			if err != nil {
				return authoringAPIError(e, err)
			}
			plan := input.Plan
			if err := plan.Normalize(session.Platform); err != nil {
				return authoringAPIError(e, err)
			}
			resolved, err := actionauthor.ResolvePrerequisites(e.App, selection, &plan)
			if err != nil {
				return authoringAPIError(e, err)
			}
			token, err := actionauthor.SignPlan(e.App, actionauthor.PlanGrant{
				AuthorWorkflowID:   input.AuthorWorkflowID,
				Plan:               plan,
				ApproverID:         actor.ActorID,
				ApproverCollection: actor.ActorCollection,
				ConfirmedAt:        time.Now().Unix(),
			})
			if err != nil {
				return authoringAPIError(e, err)
			}
			result = map[string]any{
				"accepted":      true,
				"plan_token":    token,
				"plan":          plan,
				"prerequisites": resolved,
				"parameters":    actionauthor.ParameterInfos(plan),
			}
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
				"plan_not_pending",
				"this plan is no longer pending; reload the chat",
			)
		}
		return e.JSON(http.StatusOK, ActionAuthorPlanResponse{
			Accepted: *input.Accepted,
			ToolID:   input.ToolID,
		})
	}
}
