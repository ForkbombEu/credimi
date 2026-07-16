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

type temporalWorkflowSignaler interface {
	SignalWorkflow(
		ctx context.Context,
		workflowID string,
		runID string,
		signalName string,
		arg interface{},
	) error
}

type SignalWorkflowActivity struct {
	workflowengine.BaseActivity
	temporalClientFactory func(namespace string) (temporalWorkflowSignaler, error)
}

type SignalWorkflowActivityInput struct {
	WorkflowID        string `json:"workflow_id"`
	RunID             string `json:"run_id,omitempty"`
	WorkflowNamespace string `json:"workflow_namespace"`
	SignalName        string `json:"signal_name"`
	Payload           any    `json:"payload,omitempty"`
}

type SignalWorkflowActivityOutput struct {
	Signaled bool   `json:"signaled"`
	Status   string `json:"status,omitempty"`
}

func NewSignalWorkflowActivity() *SignalWorkflowActivity {
	return &SignalWorkflowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Signal workflow",
		},
		temporalClientFactory: func(namespace string) (temporalWorkflowSignaler, error) {
			return temporalclient.GetTemporalClientWithNamespace(namespace)
		},
	}
}

func (a *SignalWorkflowActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *SignalWorkflowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[SignalWorkflowActivityInput](input.Payload)
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

	signalName := strings.TrimSpace(payload.SignalName)
	if signalName == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "signal_name is required",
		})
	}

	client, err := a.temporalClientFactory(workflowNamespace)
	if err != nil {
		return result, err
	}

	err = client.SignalWorkflow(
		ctx,
		workflowID,
		strings.TrimSpace(payload.RunID),
		signalName,
		payload.Payload,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			result.Output = SignalWorkflowActivityOutput{
				Signaled: false,
				Status:   "NOT_FOUND",
			}
			return result, nil
		}
		return result, err
	}

	result.Output = SignalWorkflowActivityOutput{
		Signaled: true,
		Status:   "SIGNALED",
	}
	return result, nil
}
