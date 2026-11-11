// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"slices"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

const pipelineResultScreenshotsField = "maestro_screenshots"

// DeletePipelineStepScreenshotsInput names the screenshot files to remove from a pipeline
// result, such as those of a linked block attempt that Temporal retries.
type DeletePipelineStepScreenshotsInput struct {
	RunIdentifier string   `json:"run_identifier"`
	FileNames     []string `json:"file_names"`
}

// DeletePipelineStepScreenshotsActivity removes screenshot files from the
// maestro_screenshots field of a pipeline result.
type DeletePipelineStepScreenshotsActivity struct {
	workflowengine.BaseActivity
	app core.App
}

// NewDeletePipelineStepScreenshotsActivity returns the activity bound to app.
func NewDeletePipelineStepScreenshotsActivity(app core.App) *DeletePipelineStepScreenshotsActivity {
	return &DeletePipelineStepScreenshotsActivity{
		BaseActivity: workflowengine.BaseActivity{Name: DeletePipelineStepScreenshotsActivityName},
		app:          app,
	}
}

// Name returns the activity name.
func (a *DeletePipelineStepScreenshotsActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute removes the named files that the pipeline result still lists.
func (a *DeletePipelineStepScreenshotsActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[DeletePipelineStepScreenshotsInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if payload.RunIdentifier == "" || len(payload.FileNames) == 0 {
		return result, a.NewMissingOrInvalidPayloadError(
			errors.New("run_identifier and file_names are required"),
		)
	}
	record, err := canonify.Resolve(a.app, payload.RunIdentifier)
	if err != nil {
		return result, a.NewCodedError(errorcodes.RecordNotFound, false, err)
	}
	stored := record.GetStringSlice(pipelineResultScreenshotsField)
	names := make([]string, 0, len(payload.FileNames))
	for _, name := range payload.FileNames {
		if slices.Contains(stored, name) && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		result.Output = map[string]any{"deleted": []string{}}
		return result, nil
	}
	record.Set(pipelineResultScreenshotsField+"-", names)
	if err := a.app.Save(record); err != nil {
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
	result.Output = map[string]any{"deleted": names}
	return result, nil
}
