// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var retryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second * 5,
	BackoffCoefficient: 2.0,
	MaximumInterval:    time.Minute,
	MaximumAttempts:    5,
}

// DefaultActivityOptions defines the configuration options for a workflow activity.
// It includes timeouts and retry policies to control the execution behavior of activities.
//
// Fields:
// - ScheduleToCloseTimeout: The maximum time allowed from scheduling to the completion of the activity.
// - StartToCloseTimeout: The maximum time allowed from the start to the completion of the activity.
// - RetryPolicy: The policy that defines how retries should be handled in case of activity failures.
var DefaultActivityOptions = workflow.ActivityOptions{
	ScheduleToCloseTimeout: time.Minute * 10,
	StartToCloseTimeout:    time.Minute * 5,
	RetryPolicy:            retryPolicy,
}
