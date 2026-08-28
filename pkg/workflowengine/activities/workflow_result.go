// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type GetWorkflowResultActivity struct {
	workflowengine.BaseActivity
}

type GetWorkflowResultActivityInput struct {
	WorkflowID           string `json:"workflow_id"`
	RunID                string `json:"run_id,omitempty"`
	WorkflowNamespace    string `json:"workflow_namespace"`
	ReturnFailureDetails bool   `json:"return_failure_details,omitempty"`
}

func NewGetWorkflowResultActivity() *GetWorkflowResultActivity {
	return &GetWorkflowResultActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Get workflow result"},
	}
}

func (a *GetWorkflowResultActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *GetWorkflowResultActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[GetWorkflowResultActivityInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	workflowID := strings.TrimSpace(payload.WorkflowID)
	if workflowID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "workflow_id is required",
		})
	}
	workflowNamespace := strings.TrimSpace(payload.WorkflowNamespace)
	if workflowNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "workflow_namespace is required",
		})
	}

	client, err := temporalclient.GetTemporalClientWithNamespace(workflowNamespace)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	result, err := workflowengine.WaitForWorkflowResult(client, workflowID, payload.RunID)
	if err != nil {
		if payload.ReturnFailureDetails {
			failure := workflowengine.ParseWorkflowError(err)
			if failure.Details != nil {
				output, hasOutput := failure.Details["output"]
				errors, hasErrors := failure.Details["errors"]
				if hasOutput || hasErrors {
					return workflowengine.ActivityResult{Output: workflowengine.WorkflowResult{
						WorkflowID:    workflowID,
						WorkflowRunID: payload.RunID,
						Errors:        errors,
						Output:        output,
					}}, nil
				}
			}
		}
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: result}, nil
}
