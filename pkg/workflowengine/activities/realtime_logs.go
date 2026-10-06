// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/realtimelogs"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

type SendRealtimeLogsInput struct {
	Subscription string           `json:"subscription" validate:"required"`
	Logs         []map[string]any `json:"logs"`
}

// SendRealtimeLogsActivity pushes workflow logs to realtime subscribers.
type SendRealtimeLogsActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewSendRealtimeLogsActivity(app core.App) *SendRealtimeLogsActivity {
	return &SendRealtimeLogsActivity{
		BaseActivity: workflowengine.BaseActivity{Name: SendRealtimeLogsActivityName},
		app:          app,
	}
}

func (a *SendRealtimeLogsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *SendRealtimeLogsActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[SendRealtimeLogsInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if err := realtimelogs.Notify(a.app, payload.Subscription, payload.Logs); err != nil {
		return result, credimiActivityError(
			&a.BaseActivity,
			errorcodes.JSONMarshalFailed,
			false,
			err,
		)
	}
	return result, nil
}
