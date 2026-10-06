// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
)

// ScoreboardSaveOutput is the output of SaveScoreboardResultsActivity.
type ScoreboardSaveOutput struct {
	RecordsCount int      `json:"records_count"`
	Errors       []string `json:"errors,omitempty"`
}

// ListScoreboardNamespacesActivity lists the namespaces of every organization.
type ListScoreboardNamespacesActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewListScoreboardNamespacesActivity(app core.App) *ListScoreboardNamespacesActivity {
	return &ListScoreboardNamespacesActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: workflows.ListScoreboardNamespacesActivityName,
		},
		app: app,
	}
}

func (a *ListScoreboardNamespacesActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ListScoreboardNamespacesActivity) Execute(
	_ context.Context,
	_ workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	namespaces, err := allOrganizationNamespaces(a.app)
	if err != nil {
		return workflowengine.ActivityResult{}, newScoreboardDatabaseError(&a.BaseActivity, err)
	}
	return workflowengine.ActivityResult{Output: namespaces}, nil
}

// GetNamespaceScoreboardActivity computes the pipeline scoreboard of a namespace.
type GetNamespaceScoreboardActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewGetNamespaceScoreboardActivity(app core.App) *GetNamespaceScoreboardActivity {
	return &GetNamespaceScoreboardActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: workflows.GetNamespaceScoreboardActivityName,
		},
		app: app,
	}
}

func (a *GetNamespaceScoreboardActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *GetNamespaceScoreboardActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[workflows.ScoreboardNamespaceInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	stats, err := namespaceScoreboard(ctx, a.app, payload.Namespace)
	if err != nil {
		return workflowengine.ActivityResult{}, newScoreboardDatabaseError(&a.BaseActivity, err)
	}
	return workflowengine.ActivityResult{Output: stats}, nil
}

// GetScoreboardExecutionDetailsActivity describes one pipeline execution for
// the scoreboard.
type GetScoreboardExecutionDetailsActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewGetScoreboardExecutionDetailsActivity(
	app core.App,
) *GetScoreboardExecutionDetailsActivity {
	return &GetScoreboardExecutionDetailsActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: workflows.GetScoreboardExecutionDetailsActivityName,
		},
		app: app,
	}
}

func (a *GetScoreboardExecutionDetailsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *GetScoreboardExecutionDetailsActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[workflows.ScoreboardExecutionInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	details, err := scoreboardExecutionDetails(
		a.app,
		payload.Namespace,
		payload.WorkflowID,
		payload.RunID,
	)
	if err != nil {
		return workflowengine.ActivityResult{}, newScoreboardDatabaseError(&a.BaseActivity, err)
	}
	return workflowengine.ActivityResult{Output: details}, nil
}

// SaveScoreboardResultsActivity replaces the scoreboard cache with the
// aggregated pipelines.
type SaveScoreboardResultsActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewSaveScoreboardResultsActivity(app core.App) *SaveScoreboardResultsActivity {
	return &SaveScoreboardResultsActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: workflows.SaveScoreboardResultsActivityName,
		},
		app: app,
	}
}

func (a *SaveScoreboardResultsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *SaveScoreboardResultsActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[workflows.AggregateScoreboardWorkflowOutput](
		input.Payload,
	)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	recordsCount, saveErrors, err := saveScoreboardResults(a.app, &payload)
	if errors.Is(err, errEmptyScoreboardResults) {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	if err != nil {
		return workflowengine.ActivityResult{}, newScoreboardDatabaseError(&a.BaseActivity, err)
	}

	output := ScoreboardSaveOutput{RecordsCount: recordsCount}
	for _, saveErr := range saveErrors {
		output.Errors = append(output.Errors, saveErr.Error())
	}
	return workflowengine.ActivityResult{Output: output}, nil
}

func newScoreboardDatabaseError(a *workflowengine.BaseActivity, err error) error {
	errCode := errorcodes.Codes[errorcodes.DatabaseOperationFailed]
	return a.NewActivityError(workflowengine.ActivityError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: err.Error(),
	})
}
