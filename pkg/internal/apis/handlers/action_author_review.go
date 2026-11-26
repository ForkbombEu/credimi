// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
// pending review.
const actionAuthorReviewUpdate = "provide_callback_result"

// ActionAuthorReviewRequest holds the human decisions on the actions of a
// pending review, one per action key.
type ActionAuthorReviewRequest struct {
	AuthorWorkflowID string                       `json:"author_workflow_id" validate:"required"`
	ToolID           string                       `json:"tool_id"            validate:"required"`
	Decisions        []ActionAuthorReviewDecision `json:"decisions"          validate:"required,min=1,max=40,dive"`
}

// ActionAuthorReviewDecision is the decision on one action of the review:
// accept, refuse or change (with a comment saying what to change). Plan is
// the action's plan as the review shows it, with its candidate prerequisites
// unresolved; Credimi checks it again and binds an accept's approval to it.
// Profile is the FCAF profile the review showed, which an accept also binds.
// FCAFResult is the agent's verdict on an FCAF action the review showed, and
// FCAFReplays its replays on fixtures of the role's authoring set, which an
// accept also binds.
type ActionAuthorReviewDecision struct {
	Key             string                    `json:"key"                    validate:"required"`
	Name            string                    `json:"name"                   validate:"max=200"`
	CandidateSHA256 string                    `json:"candidate_sha256"       validate:"required,len=64,hexadecimal"`
	Decision        string                    `json:"decision"               validate:"required,oneof=accept refuse change"`
	Comment         string                    `json:"comment"                validate:"max=2000"`
	Plan            actionauthor.Spec         `json:"plan"`
	Profile         map[string]any            `json:"profile,omitempty"`
	FCAFResult      *actionauthor.FCAFResult  `json:"fcaf_result,omitempty"`
	FCAFReplays     []actionauthor.FCAFReplay `json:"fcaf_replays,omitempty"`
}

// ActionAuthorReviewResponse confirms the decisions the agent received.
type ActionAuthorReviewResponse struct {
	ToolID    string                     `json:"tool_id"`
	Decisions []ActionAuthorReviewResult `json:"decisions"`
}

// ActionAuthorReviewResult is one decision the agent received.
type ActionAuthorReviewResult struct {
	Key      string `json:"key"`
	Decision string `json:"decision"`
}

// Review decisions.
const (
	reviewAccept = "accept"
	reviewChange = "change"
)

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

// actionAuthorCallerSession authorizes the caller for the running session of
// an agent workflow: the session's trusted run must have been started for the
// caller's organization by the caller (404 session_not_found otherwise). It
// looks the session up in the runner namespace its ID names and returns that
// namespace's client.
func actionAuthorCallerSession(
	e *core.RequestEvent,
	authorWorkflowID string,
) (client.Client, activities.ActionAuthorSessionRef, actionauthor.Actor, error) {
	var ref activities.ActionAuthorSessionRef
	if e.Auth == nil {
		return nil, ref, actionauthor.Actor{}, authoringAPIError(e, actionauthor.ErrForbidden)
	}
	if apiErr := requireAuthoringUser(e.Auth); apiErr != nil {
		return nil, ref, actionauthor.Actor{}, apiErr
	}
	actor := authoringActor(e.Auth)
	organization, err := pbutils.GetUserOrganizationCanonifiedName(e.App, actor.ActorID)
	if err != nil {
		return nil, ref, actor, actionAuthorSessionMissing()
	}
	sessionID := strings.TrimSuffix(authorWorkflowID, activities.ActionAuthorWorkflowIDSuffix)
	namespace, ok := actionauthor.SessionNamespace(sessionID)
	if !ok {
		return nil, ref, actor, actionAuthorSessionMissing()
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
	if errors.Is(err, actionauthor.ErrForbidden) ||
		errors.Is(err, actionauthor.ErrSessionNotVisible) {
		return nil, ref, actor, actionAuthorSessionMissing()
	}
	if err != nil {
		return nil, ref, actor, temporalUnavailableError()
	}
	if ref.Organization != organization || ref.Session.ActorID != actor.ActorID {
		return nil, ref, actor, actionAuthorSessionMissing()
	}
	return c, ref, actor, nil
}

// HandleActionAuthorReview checks the caller's decisions on the actions of a
// pending review, signs an approval of each accepted action and its plan,
// checked again for the session, and delivers all decisions to the agent's
// pending review at once. Only the user who started the session can review
// it.
func HandleActionAuthorReview() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorReviewRequest](e)
		if err != nil {
			return authoringAPIError(e, actionauthor.ErrInvalid)
		}
		c, ref, actor, err := actionAuthorCallerSession(e, input.AuthorWorkflowID)
		if err != nil {
			return err
		}
		if err := checkReviewDecisions(input.Decisions); err != nil {
			return authoringAPIError(e, err)
		}
		decisions := make([]map[string]any, 0, len(input.Decisions))
		results := make([]ActionAuthorReviewResult, 0, len(input.Decisions))
		for _, decision := range input.Decisions {
			// A refusal or a change request signs nothing, so it never needs
			// the plan: it must reach the agent even with an invalid plan.
			approval := ""
			if decision.Decision == reviewAccept {
				approval, err = signReviewApproval(e, ref, actor, input.AuthorWorkflowID, decision)
				if err != nil {
					return authoringAPIError(e, err)
				}
			}
			comment := ""
			if decision.Decision == reviewChange {
				comment = strings.TrimSpace(decision.Comment)
			}
			decisions = append(decisions, map[string]any{
				"key":      decision.Key,
				"name":     decision.Name,
				"decision": decision.Decision,
				"approval": approval,
				"comment":  comment,
			})
			results = append(results, ActionAuthorReviewResult{
				Key:      decision.Key,
				Decision: decision.Decision,
			})
		}
		err = updateActionAuthorWorkflow(
			e.Request.Context(),
			c,
			input.AuthorWorkflowID,
			input.ToolID,
			map[string]any{"decisions": decisions},
		)
		if err != nil {
			return actionAuthorUpdateError(
				err,
				"review_not_pending",
				"this review is no longer pending; reload the chat",
			)
		}
		return e.JSON(http.StatusOK, ActionAuthorReviewResponse{
			ToolID:    input.ToolID,
			Decisions: results,
		})
	}
}

// checkReviewDecisions refuses a key given twice, a change request without a
// comment, and an accepted action starting from an action of the same review
// that is not accepted. A candidate missing from the decisions was saved by
// an earlier review; the save checks it.
func checkReviewDecisions(decisions []ActionAuthorReviewDecision) error {
	given := make(map[string]string, len(decisions))
	for _, decision := range decisions {
		if !actionauthor.IsCandidateKey(decision.Key) {
			return fmt.Errorf(
				"%w: decision key %s: must match a<number>",
				actionauthor.ErrInvalid,
				decision.Key,
			)
		}
		if _, ok := given[decision.Key]; ok {
			return fmt.Errorf(
				"%w: decision %s is given twice",
				actionauthor.ErrInvalid,
				decision.Key,
			)
		}
		given[decision.Key] = decision.Decision
		if decision.Decision == reviewChange && strings.TrimSpace(decision.Comment) == "" {
			return fmt.Errorf(
				"%w: request changes for %s: say what to change",
				actionauthor.ErrInvalid,
				decision.Key,
			)
		}
	}
	for _, decision := range decisions {
		if decision.Decision != reviewAccept {
			continue
		}
		for _, prerequisite := range decision.Plan.Prerequisites {
			start, ok := given[prerequisite.Candidate]
			if prerequisite.Candidate != "" && ok && start != reviewAccept {
				return fmt.Errorf(
					"%w: %s starts from %s, which you did not accept",
					actionauthor.ErrInvalid,
					decision.Key,
					prerequisite.Candidate,
				)
			}
		}
	}
	return nil
}

// signReviewApproval checks an accepted action's plan, FCAF profile, FCAF
// result and FCAF replays again for the session and signs the caller's
// approval of its code, plan, profile, result and replays.
func signReviewApproval(
	e *core.RequestEvent,
	ref activities.ActionAuthorSessionRef,
	actor actionauthor.Actor,
	authorWorkflowID string,
	decision ActionAuthorReviewDecision,
) (string, error) {
	if len(decision.Profile) > 0 && decision.Plan.FCAFRole == "" {
		return "", fmt.Errorf(
			"%w: profile: only FCAF actions carry a profile", actionauthor.ErrInvalid,
		)
	}
	if err := actionauthor.CheckFCAFProfile(decision.Profile, decision.Plan.FCAFRole); err != nil {
		return "", err
	}
	if err := actionauthor.CheckFCAFResult(
		decision.FCAFResult,
		decision.Plan.FCAFRole,
	); err != nil {
		return "", err
	}
	selection, err := actionauthor.Select(e.App, ref.Organization, ref.Session)
	if err != nil {
		return "", err
	}
	scope, err := activities.FCAFScope(ref)
	if err != nil {
		return "", err
	}
	// A fix session approves only actions of the roles it fixes, whose
	// replays may also cover the fixtures each failed on.
	if ref.FCAFFix {
		scope, err = activities.FCAFFixScope(
			e.App,
			ref,
			selection,
			scope,
			fcafRunStateReader(e, ref.Organization),
		)
		if err != nil {
			return "", err
		}
	}
	plan, err := activities.SessionPlan(e.App, selection, decision.Plan, scope)
	if err != nil {
		return "", err
	}
	if err := actionauthor.CheckFCAFFixRole(scope, plan.FCAFRole); err != nil {
		return "", err
	}
	if err := actionauthor.CheckFCAFReplays(
		scope,
		decision.FCAFReplays,
		plan.FCAFRole,
	); err != nil {
		return "", err
	}
	return actionauthor.SignApproval(e.App, actionauthor.Approval{
		AuthorWorkflowID: authorWorkflowID,
		CandidateSHA256:  decision.CandidateSHA256,
		ApproverID:       actor.ActorID,
		ApprovedAt:       time.Now().Unix(),
		PlanSHA256:       actionauthor.PlanSHA256(plan),
		Key:              decision.Key,
		ProfileSHA256:    actionauthor.ProfileSHA256(decision.Profile),
		ResultSHA256:     actionauthor.FCAFResultSHA256(decision.FCAFResult),
		ReplaysSHA256:    actionauthor.FCAFReplaysSHA256(decision.FCAFReplays),
	})
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
