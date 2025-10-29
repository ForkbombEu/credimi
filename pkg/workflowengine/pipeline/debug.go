// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"context"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type DebugActivity struct {
	workflowengine.BaseActivity
}

func NewDebugActivity() *DebugActivity {
	return &DebugActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "[DEBUG]: Show current outputs",
		},
	}
}

func (a *DebugActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *DebugActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {

	stepID, _ := input.Payload["step"].(string)
	outputs, _ := input.Payload["outputs"].(map[string]any)

	return workflowengine.ActivityResult{
		Output: map[string]any{
			"current_step": stepID,
			"outputs":      outputs,
		},
	}, nil
}

func runDebugActivity(ctx workflow.Context, logger log.Logger, stepID string, finalOutput map[string]any) {
	debugAO := workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	debugCtx := workflow.WithActivityOptions(ctx, debugAO)
	debugInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"step":    stepID,
			"outputs": finalOutput,
		},
	}

	err := workflow.ExecuteActivity(
		debugCtx,
		NewDebugActivity().Name(),
		debugInput,
	).Get(debugCtx, nil)

	if err != nil {
		logger.Error(stepID, "debug activity execution error", err)
	}
}
