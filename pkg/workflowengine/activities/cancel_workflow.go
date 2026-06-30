// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/api/serviceerror"
)

type temporalWorkflowCanceler interface {
	CancelWorkflow(ctx context.Context, workflowID string, runID string) error
}

type CancelWorkflowActivity struct {
	workflowengine.BaseActivity
	temporalClientFactory func(namespace string) (temporalWorkflowCanceler, error)
}

type CancelWorkflowActivityInput struct {
	WorkflowID        string `json:"workflow_id"`
	RunID             string `json:"run_id,omitempty"`
	WorkflowNamespace string `json:"workflow_namespace"`
	Reason            string `json:"reason,omitempty"`
}

type CancelWorkflowActivityOutput struct {
	Canceled bool   `json:"canceled"`
	Status   string `json:"status,omitempty"`
}

func NewCancelWorkflowActivity() *CancelWorkflowActivity {
	return &CancelWorkflowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Cancel workflow",
		},
		temporalClientFactory: func(namespace string) (temporalWorkflowCanceler, error) {
			return temporalclient.GetTemporalClientWithNamespace(namespace)
		},
	}
}

func (a *CancelWorkflowActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *CancelWorkflowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[CancelWorkflowActivityInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	workflowID := strings.TrimSpace(payload.WorkflowID)
	if workflowID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "workflow_id is required",
		})
	}

	workflowNamespace := strings.TrimSpace(payload.WorkflowNamespace)
	if workflowNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "workflow_namespace is required",
		})
	}

	client, err := a.temporalClientFactory(workflowNamespace)
	if err != nil {
		return result, err
	}

	err = client.CancelWorkflow(ctx, workflowID, strings.TrimSpace(payload.RunID))
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			result.Output = CancelWorkflowActivityOutput{
				Canceled: false,
				Status:   "NOT_FOUND",
			}
			return result, nil
		}
		return result, err
	}

	result.Output = CancelWorkflowActivityOutput{
		Canceled: true,
		Status:   "CANCELED",
	}
	return result, nil
}
