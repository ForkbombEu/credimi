// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"go.temporal.io/api/serviceerror"
)

type QueryMobileRunnerSemaphoreRunStatusInput struct {
	RunnerID       string `json:"runner_id"`
	OwnerNamespace string `json:"owner_namespace"`
	TicketID       string `json:"ticket_id"`
}

type QueryMobileRunnerSemaphoreRunStatusActivity struct {
	workflowengine.BaseActivity
}

func NewQueryMobileRunnerSemaphoreRunStatusActivity() *QueryMobileRunnerSemaphoreRunStatusActivity {
	return &QueryMobileRunnerSemaphoreRunStatusActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Query mobile runner semaphore run status",
		},
	}
}

func (a *QueryMobileRunnerSemaphoreRunStatusActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *QueryMobileRunnerSemaphoreRunStatusActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[QueryMobileRunnerSemaphoreRunStatusInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	if payload.RunnerID == "" || payload.OwnerNamespace == "" || payload.TicketID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(
				"%s: runner_id, owner_namespace, and ticket_id are required",
				errCode.Description,
			),
		)
	}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return result, err
	}

	workflowID := mobilerunnersemaphore.WorkflowID(payload.RunnerID)
	encoded, err := temporalClient.QueryWorkflow(
		ctx,
		workflowID,
		"",
		mobilerunnersemaphore.RunStatusQuery,
		payload.OwnerNamespace,
		payload.TicketID,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			result.Output = mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{
				TicketID: payload.TicketID,
				Status:   mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound,
			}
			return result, nil
		}
		return result, err
	}

	var status mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView
	if err := encoded.Get(&status); err != nil {
		return result, err
	}
	result.Output = status
	return result, nil
}
