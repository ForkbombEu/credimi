// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// Action author activities the Python agent calls to plan an action.
const (
	ListActionAuthorWalletActionsActivityName = "List action author wallet actions"
	CheckActionAuthorPlanActivityName         = "Check action author plan"
	ResolveActionAuthorReferenceActivityName  = "Resolve action author reference"
)

// sessionSelection reauthorizes the build of the caller's session.
func (l authorSessionLookup) sessionSelection(
	ctx context.Context,
	base *workflowengine.BaseActivity,
	app core.App,
	caller authorCaller,
) (ActionAuthorSessionRef, actionauthor.Selection, error) {
	ref, err := l.callerSession(ctx, base, caller)
	if err != nil {
		return ActionAuthorSessionRef{}, actionauthor.Selection{}, err
	}
	selection, err := actionauthor.Select(app, caller.Namespace, ref.Session)
	if err != nil {
		return ActionAuthorSessionRef{}, actionauthor.Selection{}, actionAuthorError(base, err)
	}
	return ref, selection, nil
}

// ListActionAuthorWalletActionsInput optionally filters by category.
type ListActionAuthorWalletActionsInput struct {
	Category string `json:"category,omitempty"`
}

// ListActionAuthorWalletActionsOutput lists the session wallet's actions.
type ListActionAuthorWalletActionsOutput struct {
	Actions []actionauthor.WalletAction `json:"actions"`
}

// ListActionAuthorWalletActionsActivity lists the actions of the caller
// session's wallet that a plan can start from.
type ListActionAuthorWalletActionsActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewListActionAuthorWalletActionsActivity binds the listing to the application.
func NewListActionAuthorWalletActionsActivity(app core.App) *ListActionAuthorWalletActionsActivity {
	return &ListActionAuthorWalletActionsActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: ListActionAuthorWalletActionsActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *ListActionAuthorWalletActionsActivity) Name() string { return a.BaseActivity.Name }

// Execute lists the wallet actions for the activity's actual caller.
func (a *ListActionAuthorWalletActionsActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ListActionAuthorWalletActionsInput](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.list(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

func (a *ListActionAuthorWalletActionsActivity) list(
	ctx context.Context,
	caller authorCaller,
	payload ListActionAuthorWalletActionsInput,
) (ListActionAuthorWalletActionsOutput, error) {
	_, selection, err := a.sessionSelection(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return ListActionAuthorWalletActionsOutput{}, err
	}
	actions, err := actionauthor.ListWalletActions(a.app, selection, payload.Category)
	if err != nil {
		return ListActionAuthorWalletActionsOutput{}, actionAuthorError(&a.BaseActivity, err)
	}
	return ListActionAuthorWalletActionsOutput{Actions: actions}, nil
}

// CheckActionAuthorPlanInput is the plan the agent sets.
type CheckActionAuthorPlanInput struct {
	Plan actionauthor.Spec `json:"plan"`
}

// CheckActionAuthorPlanOutput lists the plan's problems or, when it has
// none, the normalized plan with its summary, resolved prerequisites and
// parameters. Its lists are never null; with problems the plan is null.
type CheckActionAuthorPlanOutput struct {
	Problems      []string                            `json:"problems"`
	Plan          *actionauthor.Spec                  `json:"plan"`
	Summary       []string                            `json:"summary"`
	Prerequisites []actionauthor.ResolvedPrerequisite `json:"prerequisites"`
	Parameters    []actionauthor.ParameterInfo        `json:"parameters"`
}

// checkPlanProblems is the output of a plan with problems.
func checkPlanProblems(problems ...string) CheckActionAuthorPlanOutput {
	return CheckActionAuthorPlanOutput{
		Problems:      problems,
		Summary:       []string{},
		Prerequisites: []actionauthor.ResolvedPrerequisite{},
		Parameters:    []actionauthor.ParameterInfo{},
	}
}

// CheckActionAuthorPlanActivity checks the plan the agent sets for the
// caller's session.
type CheckActionAuthorPlanActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewCheckActionAuthorPlanActivity binds the plan check to the application.
func NewCheckActionAuthorPlanActivity(app core.App) *CheckActionAuthorPlanActivity {
	return &CheckActionAuthorPlanActivity{
		BaseActivity:        workflowengine.BaseActivity{Name: CheckActionAuthorPlanActivityName},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *CheckActionAuthorPlanActivity) Name() string { return a.BaseActivity.Name }

// Execute checks the plan for the activity's actual caller.
func (a *CheckActionAuthorPlanActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[CheckActionAuthorPlanInput](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.check(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

// check answers a plan with its problems, or with the normalized plan, its
// summary, resolved prerequisites and parameters.
func (a *CheckActionAuthorPlanActivity) check(
	ctx context.Context,
	caller authorCaller,
	payload CheckActionAuthorPlanInput,
) (CheckActionAuthorPlanOutput, error) {
	_, selection, err := a.sessionSelection(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return CheckActionAuthorPlanOutput{}, err
	}
	plan := payload.Plan
	resolved, err := checkPlan(a.app, selection, &plan)
	if errors.Is(err, actionauthor.ErrInvalid) {
		return checkPlanProblems(
			strings.TrimPrefix(err.Error(), actionauthor.ErrInvalid.Error()+": "),
		), nil
	}
	if err != nil {
		return CheckActionAuthorPlanOutput{}, actionAuthorError(&a.BaseActivity, err)
	}
	return CheckActionAuthorPlanOutput{
		Problems:      []string{},
		Plan:          &plan,
		Summary:       actionauthor.PlanSummary(plan, resolved),
		Prerequisites: resolved,
		Parameters:    actionauthor.ParameterInfos(plan),
	}, nil
}

// checkPlan normalizes the plan and resolves its prerequisites; an invalid
// plan wraps actionauthor.ErrInvalid.
func checkPlan(
	app core.App,
	selection actionauthor.Selection,
	plan *actionauthor.Spec,
) ([]actionauthor.ResolvedPrerequisite, error) {
	if err := plan.Normalize(); err != nil {
		return nil, err
	}
	return actionauthor.ResolvePrerequisites(app, selection, plan)
}

// SessionPlan checks a plan the agent sends for its session exactly as
// Check action author plan does: normalized, its prerequisites resolved
// for the session's build. An empty plan is ErrInvalid.
func SessionPlan(
	app core.App,
	selection actionauthor.Selection,
	plan actionauthor.Spec,
) (actionauthor.Spec, error) {
	if plan.Slot == "" {
		return actionauthor.Spec{}, fmt.Errorf(
			"%w: no plan: set one with set_plan", actionauthor.ErrInvalid,
		)
	}
	if _, err := checkPlan(app, selection, &plan); err != nil {
		return actionauthor.Spec{}, err
	}
	return plan, nil
}

// ResolveActionAuthorReferenceInput is a reference the user gave the agent.
type ResolveActionAuthorReferenceInput struct {
	Text string `json:"text"`
}

// ResolveActionAuthorReferenceActivity turns a Credimi URL, identifier or
// deeplink into parameter sources the session's organization can use.
type ResolveActionAuthorReferenceActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
	// transport fetches credential_offer_uri, like the http sources.
	transport http.RoundTripper
}

// NewResolveActionAuthorReferenceActivity binds reference resolution to the application.
func NewResolveActionAuthorReferenceActivity(app core.App) *ResolveActionAuthorReferenceActivity {
	return &ResolveActionAuthorReferenceActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: ResolveActionAuthorReferenceActivityName,
		},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
		transport:           actionAuthorTransport(),
	}
}

// Name returns the registered Temporal activity name.
func (a *ResolveActionAuthorReferenceActivity) Name() string { return a.BaseActivity.Name }

// Execute resolves the reference for the activity's actual caller.
func (a *ResolveActionAuthorReferenceActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ResolveActionAuthorReferenceInput](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := a.resolve(ctx, activityCaller(ctx), payload)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

func (a *ResolveActionAuthorReferenceActivity) resolve(
	ctx context.Context,
	caller authorCaller,
	payload ResolveActionAuthorReferenceInput,
) (actionauthor.ReferenceResolution, error) {
	_, selection, err := a.sessionSelection(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return actionauthor.ReferenceResolution{}, err
	}
	resolution, err := actionauthor.ResolveReference(
		ctx,
		a.app,
		selection.Organization.Id,
		payload.Text,
		&http.Client{
			Timeout:   actionAuthorHTTPTimeout,
			Transport: sourceRedirectPolicy{next: a.transport},
		},
	)
	if err != nil {
		return actionauthor.ReferenceResolution{}, actionAuthorError(&a.BaseActivity, err)
	}
	return resolution, nil
}
