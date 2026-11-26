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
	"sync"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
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
) (actionauthor.Selection, error) {
	ref, err := l.callerSession(ctx, base, caller)
	if err != nil {
		return actionauthor.Selection{}, err
	}
	selection, err := actionauthor.Select(app, ref.Organization, ref.Session)
	if err != nil {
		return actionauthor.Selection{}, actionAuthorError(base, err)
	}
	return selection, nil
}

// sessionScope reauthorizes the build of the caller's session and returns
// its selection and its FCAF scope, nil for a chat session.
func (l authorSessionLookup) sessionScope(
	ctx context.Context,
	base *workflowengine.BaseActivity,
	app core.App,
	caller authorCaller,
) (actionauthor.Selection, *actionauthor.FCAFScope, error) {
	_, selection, scope, err := l.sessionRefScope(ctx, base, app, caller)
	return selection, scope, err
}

// sessionRefScope is sessionScope that also returns the session.
func (l authorSessionLookup) sessionRefScope(
	ctx context.Context,
	base *workflowengine.BaseActivity,
	app core.App,
	caller authorCaller,
) (ActionAuthorSessionRef, actionauthor.Selection, *actionauthor.FCAFScope, error) {
	ref, err := l.callerSession(ctx, base, caller)
	if err != nil {
		return ActionAuthorSessionRef{}, actionauthor.Selection{}, nil, err
	}
	selection, err := actionauthor.Select(app, ref.Organization, ref.Session)
	if err != nil {
		return ActionAuthorSessionRef{}, actionauthor.Selection{}, nil, actionAuthorError(base, err)
	}
	scope, err := FCAFScope(ref)
	if err != nil {
		return ActionAuthorSessionRef{}, actionauthor.Selection{}, nil, actionAuthorError(base, err)
	}
	return ref, selection, scope, nil
}

// FCAFScope returns the role contract and suite of an FCAF session, nil for
// a chat session. A contract that fails to load is a transient error.
func FCAFScope(ref ActionAuthorSessionRef) (*actionauthor.FCAFScope, error) {
	if ref.Mode != actionauthor.SessionModeFCAF {
		return nil, nil
	}
	return LoadFCAFScope(ref.FCAFSuite)
}

// FCAFFixScope returns scope with the roles a fix session (ref.FCAFFix)
// fixes and the fixtures each failed on (actionauthor.FCAFFixCoverage since
// the session started, reading the Temporal state of runs without a report
// through state, which may be nil); scope itself for any other session.
func FCAFFixScope(
	app core.App,
	ref ActionAuthorSessionRef,
	selection actionauthor.Selection,
	scope *actionauthor.FCAFScope,
	state actionauthor.FCAFRunStateFunc,
) (*actionauthor.FCAFScope, error) {
	if scope == nil || !ref.FCAFFix {
		return scope, nil
	}
	coverage, _, err := actionauthor.FCAFFixCoverage(app, selection, scope, state, ref.Started)
	if err != nil {
		return nil, err
	}
	fixed := *scope
	fixed.Fix = actionauthor.FCAFFixRoles(coverage)
	return &fixed, nil
}

// fcafRunStateTimeout bounds each Temporal read of a pipeline run's state.
const fcafRunStateTimeout = 5 * time.Second

// fcafRunState reads the Temporal state of the pipeline runs of the
// organization (a canonified_name) in its namespace, each read within
// fcafRunStateTimeout. It gets the namespace's client on its first read, so
// a session that reads no run without a report never asks for it; without
// a client every such run is left out, as unfinished.
func (l authorSessionLookup) fcafRunState(
	ctx context.Context,
	organization string,
) actionauthor.FCAFRunStateFunc {
	return func(workflowID, runID string) (actionauthor.FCAFRunState, error) {
		c, err := l.temporalClient(organization)
		if err != nil {
			return actionauthor.FCAFRunState{}, err
		}
		ctx, cancel := context.WithTimeout(ctx, fcafRunStateTimeout)
		defer cancel()
		return FCAFRunState(ctx, c, workflowID, runID)
	}
}

// LoadFCAFScope returns the scope of the suite over the role contract of
// LoadFCAFContract and its templates root.
func LoadFCAFScope(suite string) (*actionauthor.FCAFScope, error) {
	contract, err := LoadFCAFContract()
	if err != nil {
		return nil, err
	}
	return &actionauthor.FCAFScope{Contract: contract, Root: fcafTemplatesRoot(), Suite: suite}, nil
}

// fcafContract is the role contract, loaded once per process; it also keeps
// the scenarios its Fixtures read.
var fcafContract struct {
	mu       sync.Mutex
	contract *walletroles.Contract
}

// LoadFCAFContract returns the FCAF role contract, loading it once per
// process; a failed load is retried on the next call.
func LoadFCAFContract() (*walletroles.Contract, error) {
	fcafContract.mu.Lock()
	defer fcafContract.mu.Unlock()
	if fcafContract.contract == nil {
		contract, err := walletroles.Load(fcafTemplatesRoot())
		if err != nil {
			return nil, err
		}
		fcafContract.contract = contract
	}
	return fcafContract.contract, nil
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
		authorSessionLookup: newAuthorSessionLookup(app),
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
	selection, err := a.sessionSelection(ctx, &a.BaseActivity, a.app, caller)
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
// none, the normalized plan with its summary, resolved prerequisites,
// parameters and the action it replaces, with its current code. Its lists
// are never null; with problems the plan and its replaced action are null.
type CheckActionAuthorPlanOutput struct {
	Problems      []string                            `json:"problems"`
	Plan          *actionauthor.Spec                  `json:"plan"`
	Summary       []string                            `json:"summary"`
	Prerequisites []actionauthor.ResolvedPrerequisite `json:"prerequisites"`
	Parameters    []actionauthor.ParameterInfo        `json:"parameters"`
	Replaces      *actionauthor.ReplaceTarget         `json:"replaces"` // null without a replacement or with problems
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
		authorSessionLookup: newAuthorSessionLookup(app),
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
	selection, scope, err := a.sessionScope(ctx, &a.BaseActivity, a.app, caller)
	if err != nil {
		return CheckActionAuthorPlanOutput{}, err
	}
	plan := payload.Plan
	checked, err := checkPlan(a.app, selection, &plan, scope)
	if errors.Is(err, actionauthor.ErrInvalid) {
		return checkPlanProblems(
			strings.TrimPrefix(err.Error(), actionauthor.ErrInvalid.Error()+": "),
		), nil
	}
	if err != nil {
		return CheckActionAuthorPlanOutput{}, actionAuthorError(&a.BaseActivity, err)
	}
	var replaces *actionauthor.ReplaceTarget
	if checked.target != nil {
		replaces = &actionauthor.ReplaceTarget{
			Action:   plan.Replaces.Action,
			Name:     checked.target.GetString("name"),
			Category: checked.target.GetString("category"),
			Code:     checked.target.GetString("code"),
		}
	}
	return CheckActionAuthorPlanOutput{
		Problems:      []string{},
		Plan:          &plan,
		Summary:       actionauthor.PlanSummary(plan, checked.prerequisites, replaces),
		Prerequisites: checked.prerequisites,
		Parameters:    actionauthor.ParameterInfos(plan),
		Replaces:      replaces,
	}, nil
}

// checkedPlan is a plan's resolved prerequisites and the action it replaces,
// nil without one.
type checkedPlan struct {
	prerequisites []actionauthor.ResolvedPrerequisite
	target        *core.Record
}

// checkPlan normalizes the plan, checks it against the FCAF contract when it
// is bound to a role, resolves its prerequisites and checks the action it
// replaces, filling its code hash; an invalid plan wraps
// actionauthor.ErrInvalid. fcaf is the session's FCAF scope, nil for chat.
func checkPlan(
	app core.App,
	selection actionauthor.Selection,
	plan *actionauthor.Spec,
	fcaf *actionauthor.FCAFScope,
) (checkedPlan, error) {
	if err := plan.Normalize(); err != nil {
		return checkedPlan{}, err
	}
	if plan.FCAFRole != "" {
		if _, err := actionauthor.CheckFCAFPlan(fcaf, plan); err != nil {
			return checkedPlan{}, err
		}
	}
	target, err := actionauthor.CheckReplaceTarget(app, selection, plan)
	if err != nil {
		return checkedPlan{}, err
	}
	prerequisites, err := actionauthor.ResolvePrerequisites(app, selection, plan)
	if err != nil {
		return checkedPlan{}, err
	}
	return checkedPlan{prerequisites: prerequisites, target: target}, nil
}

// SessionPlan checks a plan the agent sends for its session exactly as
// Check action author plan does: normalized, checked against the FCAF
// contract (fcaf is the session's scope, nil for chat), its prerequisites
// resolved and its replaced action checked for the session's build. An
// empty plan is ErrInvalid.
func SessionPlan(
	app core.App,
	selection actionauthor.Selection,
	plan actionauthor.Spec,
	fcaf *actionauthor.FCAFScope,
) (actionauthor.Spec, error) {
	if plan.Slot == "" {
		return actionauthor.Spec{}, fmt.Errorf(
			"%w: no plan: describe the action with describe_action", actionauthor.ErrInvalid,
		)
	}
	if _, err := checkPlan(app, selection, &plan, fcaf); err != nil {
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
		authorSessionLookup: newAuthorSessionLookup(app),
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
	selection, err := a.sessionSelection(ctx, &a.BaseActivity, a.app, caller)
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
