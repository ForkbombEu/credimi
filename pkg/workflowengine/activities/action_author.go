// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/activity"
)

// ResolveActionAuthorBuildActivityName is the organization worker's build-resolution activity.
const ResolveActionAuthorBuildActivityName = "Resolve action author build"

// SaveAuthoredWalletActionActivityName is the organization worker's approved-action writer.
const SaveAuthoredWalletActionActivityName = "Save authored wallet action"

// ResolveActionAuthorBuildActivity authorizes and resolves the workflow's selected build.
type ResolveActionAuthorBuildActivity struct {
	workflowengine.BaseActivity
	app core.App
}

// NewResolveActionAuthorBuildActivity binds build resolution to the application's database.
func NewResolveActionAuthorBuildActivity(app core.App) *ResolveActionAuthorBuildActivity {
	return &ResolveActionAuthorBuildActivity{
		BaseActivity: workflowengine.BaseActivity{Name: ResolveActionAuthorBuildActivityName},
		app:          app,
	}
}

// Name returns the registered Temporal activity name.
func (a *ResolveActionAuthorBuildActivity) Name() string { return a.BaseActivity.Name }

// Execute resolves the selected build in the activity's actual organization namespace.
func (a *ResolveActionAuthorBuildActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[actionauthor.BuildRequest](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	info := activity.GetInfo(ctx)
	if info.Namespace == "" {
		return result, a.NewMissingOrInvalidPayloadError(actionauthor.ErrForbidden)
	}
	build, err := actionauthor.Resolve(a.app, info.Namespace, payload)
	if err != nil {
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, false, err)
	}
	result.Output = build
	return result, nil
}

// SaveAuthoredWalletActionActivity persists a candidate approved through the workflow.
type SaveAuthoredWalletActionActivity struct {
	workflowengine.BaseActivity
	app core.App
}

// NewSaveAuthoredWalletActionActivity binds approved-action persistence to the application.
func NewSaveAuthoredWalletActionActivity(app core.App) *SaveAuthoredWalletActionActivity {
	return &SaveAuthoredWalletActionActivity{
		BaseActivity: workflowengine.BaseActivity{Name: SaveAuthoredWalletActionActivityName},
		app:          app,
	}
}

// Name returns the registered Temporal activity name.
func (a *SaveAuthoredWalletActionActivity) Name() string { return a.BaseActivity.Name }

// Execute saves the candidate with the activity's actual namespace and workflow identity.
func (a *SaveAuthoredWalletActionActivity) Execute(
	ctx context.Context,
	in workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[actionauthor.SaveRequest](in.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	info := activity.GetInfo(ctx)
	if info.Namespace == "" {
		return result, a.NewMissingOrInvalidPayloadError(actionauthor.ErrForbidden)
	}
	receipt, err := actionauthor.Save(
		a.app,
		info.Namespace,
		info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID,
		payload,
	)
	if err != nil {
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, false, err)
	}
	result.Output = receipt
	return result, nil
}
