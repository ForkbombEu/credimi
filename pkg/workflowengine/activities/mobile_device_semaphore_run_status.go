// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"go.temporal.io/api/serviceerror"
)

type QueryMobileDeviceSemaphoreRunStatusInput struct {
	DeviceID       string `json:"runner_id"`
	OwnerNamespace string `json:"owner_namespace"`
	TicketID       string `json:"ticket_id"`
}

type QueryMobileDeviceSemaphoreRunStatusActivity struct {
	workflowengine.BaseActivity
}

func NewQueryMobileDeviceSemaphoreRunStatusActivity() *QueryMobileDeviceSemaphoreRunStatusActivity {
	return &QueryMobileDeviceSemaphoreRunStatusActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Query mobile device semaphore run status",
		},
	}
}

func (a *QueryMobileDeviceSemaphoreRunStatusActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *QueryMobileDeviceSemaphoreRunStatusActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[QueryMobileDeviceSemaphoreRunStatusInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	if payload.DeviceID == "" || payload.OwnerNamespace == "" || payload.TicketID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "runner_id, owner_namespace, and ticket_id are required",
			},
		)
	}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return result, err
	}

	workflowID := mobiledevicesemaphore.WorkflowID(payload.DeviceID)
	encoded, err := temporalClient.QueryWorkflow(
		ctx,
		workflowID,
		"",
		mobiledevicesemaphore.RunStatusQuery,
		payload.OwnerNamespace,
		payload.TicketID,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			result.Output = mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{
				TicketID: payload.TicketID,
				Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound,
			}
			return result, nil
		}
		return result, err
	}

	var status mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView
	if err := encoded.Get(&status); err != nil {
		return result, err
	}
	result.Output = status
	return result, nil
}
