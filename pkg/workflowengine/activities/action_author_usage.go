// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// RecordActionAuthorUsageActivityName is the organization worker's model usage recorder.
const RecordActionAuthorUsageActivityName = "Record action author usage"

// RecordActionAuthorUsageActivity adds an agent turn's model usage to its
// organization's daily quota.
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
	payload, err := workflowengine.DecodePayload[actionauthor.UsageDelta](in.Payload)
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
	delta actionauthor.UsageDelta,
	now time.Time,
) (actionauthor.UsageState, error) {
	session, err := a.callerSession(ctx, &a.BaseActivity, caller)
	if err != nil {
		return actionauthor.UsageState{}, err
	}
	selection, err := actionauthor.Select(a.app, caller.Namespace, session.BuildRequest)
	if err != nil {
		return actionauthor.UsageState{}, actionAuthorError(&a.BaseActivity, err)
	}
	state, err := actionauthor.RecordUsage(
		a.app,
		selection.Organization.Id,
		now,
		delta,
		actionauthor.QuotaFromEnv(),
	)
	if err != nil {
		return actionauthor.UsageState{}, actionAuthorError(&a.BaseActivity, err)
	}
	return state, nil
}
