// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// RecordActionAuthorUsageActivityName is the organization worker's model usage recorder.
const RecordActionAuthorUsageActivityName = "Record action author usage"

// ReadActionAuthorUsageActivityName reads what is left of the session's daily allowance.
const ReadActionAuthorUsageActivityName = "Read action author usage"

// RecordActionAuthorUsageActivity adds an agent turn's model usage to the
// daily allowance of the session runner's pool, once per report ID.
type RecordActionAuthorUsageActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewRecordActionAuthorUsageActivity binds usage recording to the application.
func NewRecordActionAuthorUsageActivity(app core.App) *RecordActionAuthorUsageActivity {
	return &RecordActionAuthorUsageActivity{
		BaseActivity:        workflowengine.BaseActivity{Name: RecordActionAuthorUsageActivityName},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *RecordActionAuthorUsageActivity) Name() string { return a.BaseActivity.Name }

// Execute records the usage of the activity's actual caller.
func (a *RecordActionAuthorUsageActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[actionauthor.UsageReport](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	state, err := a.record(ctx, activityCaller(ctx), payload, time.Now())
	if err != nil {
		return result, err
	}
	result.Output = state
	return result, nil
}

func (a *RecordActionAuthorUsageActivity) record(
	ctx context.Context,
	caller authorCaller,
	report actionauthor.UsageReport,
	now time.Time,
) (actionauthor.UsageState, error) {
	if report.ReportID == "" {
		return actionauthor.UsageState{}, actionAuthorError(&a.BaseActivity, fmt.Errorf(
			"%w: report_id is required", actionauthor.ErrInvalid,
		))
	}
	allowance, err := sessionAllowance(ctx, a.app, &a.BaseActivity, a.authorSessionLookup, caller)
	if err != nil {
		return actionauthor.UsageState{}, err
	}
	state, err := actionauthor.RecordUsage(a.app, allowance, now, report)
	if err != nil {
		return actionauthor.UsageState{}, actionAuthorError(&a.BaseActivity, err)
	}
	return state, nil
}

// ReadActionAuthorUsageActivity reports what is left of the daily allowance
// of the session runner's pool, so the agent can cap a turn before it runs.
type ReadActionAuthorUsageActivity struct {
	workflowengine.BaseActivity
	authorSessionLookup
	app core.App
}

// NewReadActionAuthorUsageActivity binds usage reading to the application.
func NewReadActionAuthorUsageActivity(app core.App) *ReadActionAuthorUsageActivity {
	return &ReadActionAuthorUsageActivity{
		BaseActivity:        workflowengine.BaseActivity{Name: ReadActionAuthorUsageActivityName},
		authorSessionLookup: newAuthorSessionLookup(),
		app:                 app,
	}
}

// Name returns the registered Temporal activity name.
func (a *ReadActionAuthorUsageActivity) Name() string { return a.BaseActivity.Name }

// Execute reads the usage state of the activity's actual caller; the payload is empty.
func (a *ReadActionAuthorUsageActivity) Execute(
	ctx context.Context,
	_ workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	state, err := a.read(ctx, activityCaller(ctx), time.Now())
	if err != nil {
		return result, err
	}
	result.Output = state
	return result, nil
}

func (a *ReadActionAuthorUsageActivity) read(
	ctx context.Context,
	caller authorCaller,
	now time.Time,
) (actionauthor.UsageState, error) {
	allowance, err := sessionAllowance(ctx, a.app, &a.BaseActivity, a.authorSessionLookup, caller)
	if err != nil {
		return actionauthor.UsageState{}, err
	}
	state, err := actionauthor.ReadUsage(a.app, allowance, now)
	if err != nil {
		return actionauthor.UsageState{}, actionAuthorError(&a.BaseActivity, err)
	}
	return state, nil
}

// sessionAllowance returns the allowance of the caller's session: the pool of
// the session's runner that the session's organization counts against.
func sessionAllowance(
	ctx context.Context,
	app core.App,
	base *workflowengine.BaseActivity,
	lookup authorSessionLookup,
	caller authorCaller,
) (actionauthor.Allowance, error) {
	ref, err := lookup.callerSession(ctx, base, caller)
	if err != nil {
		return actionauthor.Allowance{}, err
	}
	selection, err := actionauthor.Select(app, caller.Namespace, ref.Session)
	if err != nil {
		return actionauthor.Allowance{}, actionAuthorError(base, err)
	}
	_, runner, err := mobilerunner.ResolveDevice(app, ref.DeviceID)
	if err != nil {
		return actionauthor.Allowance{}, actionAuthorError(base, fmt.Errorf(
			"%w: session device: %w", actionauthor.ErrForbidden, err,
		))
	}
	return actionauthor.AllowanceFor(runner, selection.Organization.Id), nil
}
