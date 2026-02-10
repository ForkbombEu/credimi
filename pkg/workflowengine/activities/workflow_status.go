// SPDX-FileCopyrightText: 2025 Forkbomb BV
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
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
)

type CheckWorkflowClosedActivity struct {
	workflowengine.BaseActivity
}

type CheckWorkflowClosedActivityInput struct {
	WorkflowID        string `json:"workflow_id"`
	RunID             string `json:"run_id,omitempty"`
	WorkflowNamespace string `json:"workflow_namespace"`
}

type CheckWorkflowClosedActivityOutput struct {
	Closed bool   `json:"closed"`
	Status string `json:"status,omitempty"`
}

func NewCheckWorkflowClosedActivity() *CheckWorkflowClosedActivity {
	return &CheckWorkflowClosedActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Check workflow closed",
		},
	}
}

func (a *CheckWorkflowClosedActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *CheckWorkflowClosedActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[CheckWorkflowClosedActivityInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	workflowID := strings.TrimSpace(payload.WorkflowID)
	if workflowID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "workflow_id is required")
	}

	workflowNamespace := strings.TrimSpace(payload.WorkflowNamespace)
	if workflowNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "workflow_namespace is required")
	}

	client, err := temporalclient.GetTemporalClientWithNamespace(workflowNamespace)
	if err != nil {
		return result, err
	}

	describeResp, err := client.DescribeWorkflowExecution(ctx, workflowID, payload.RunID)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			result.Output = CheckWorkflowClosedActivityOutput{
				Closed: true,
				Status: "NOT_FOUND",
			}
			return result, nil
		}
		return result, err
	}

	status := describeResp.GetWorkflowExecutionInfo().GetStatus()
	result.Output = CheckWorkflowClosedActivityOutput{
		Closed: status != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		Status: status.String(),
	}
	return result, nil
}
