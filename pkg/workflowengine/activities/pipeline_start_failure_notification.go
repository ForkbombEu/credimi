// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/webpush"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// NotifyPipelineStartFailureInput identifies a queued pipeline run that the
// device semaphore could not start.
type NotifyPipelineStartFailureInput struct {
	OwnerNamespace     string `json:"owner_namespace"     validate:"required"`
	PipelineIdentifier string `json:"pipeline_identifier" validate:"required"`
	TicketID           string `json:"ticket_id"           validate:"required"`
	ErrorMessage       string `json:"error_message"`
}

// NotifyPipelineStartFailureOutput reports how many Web Push notifications
// were delivered.
type NotifyPipelineStartFailureOutput struct {
	Sent int `json:"sent"`
}

// NotifyPipelineStartFailureActivity tells the members of the owner
// organization, through Web Push, that a queued pipeline run could not start.
// Nothing is stored.
type NotifyPipelineStartFailureActivity struct {
	workflowengine.BaseActivity
	app core.App
}

// NewNotifyPipelineStartFailureActivity returns the activity bound to app.
func NewNotifyPipelineStartFailureActivity(app core.App) *NotifyPipelineStartFailureActivity {
	return &NotifyPipelineStartFailureActivity{
		BaseActivity: workflowengine.BaseActivity{Name: NotifyPipelineStartFailureActivityName},
		app:          app,
	}
}

// Name returns the registered activity name.
func (a *NotifyPipelineStartFailureActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute sends the start failure as a Web Push to the owner organization.
func (a *NotifyPipelineStartFailureActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[NotifyPipelineStartFailureInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	organization, err := pbutils.FindOrganizationByNamespace(a.app, payload.OwnerNamespace)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, a.NewCodedError(errorcodes.RecordNotFound, false, err)
		}
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
	pipelineName := payload.PipelineIdentifier
	if pipeline, err := canonify.Resolve(a.app, payload.PipelineIdentifier); err == nil {
		if name := pipeline.GetString("name"); name != "" {
			pipelineName = name
		}
	}

	sent, err := webpush.NotifyPipelineStartFailure(ctx, a.app, webpush.StartFailureRequest{
		OrgID:        organization.Id,
		PipelineName: pipelineName,
		PipelinePath: payload.PipelineIdentifier,
		Organization: organization.GetString("name"),
		Error:        payload.ErrorMessage,
		AppURL:       workflowengine.AppURL(a.app),
	})
	if err != nil {
		return result, a.NewCodedError(
			errorcodes.ExecuteHTTPRequestFailed,
			true,
			fmt.Errorf("send pipeline start failure notifications: %w", err),
		)
	}
	result.Output = NotifyPipelineStartFailureOutput{Sent: sent}
	return result, nil
}
