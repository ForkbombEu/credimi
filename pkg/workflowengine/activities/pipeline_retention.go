// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// DeletePipelineResultFilesActivity clears the files of old pipeline results.
type DeletePipelineResultFilesActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewDeletePipelineResultFilesActivity(app core.App) *DeletePipelineResultFilesActivity {
	return &DeletePipelineResultFilesActivity{
		BaseActivity: workflowengine.BaseActivity{Name: DeletePipelineResultFilesActivityName},
		app:          app,
	}
}

func (a *DeletePipelineResultFilesActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *DeletePipelineResultFilesActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[pipelineresults.DeleteFilesOptions](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	output, err := pipelineresults.DeleteFilesOlderThan(a.app, payload)
	if err != nil {
		return result, credimiActivityError(
			&a.BaseActivity,
			errorcodes.DatabaseOperationFailed,
			true,
			err,
		)
	}
	result.Output = output
	return result, nil
}
