// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"errors"
	"os"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func FetchIssuersWorkflow(ctx workflow.Context) error {
	retrypolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    100 * time.Second,
		MaximumAttempts:    500,
	}

	options := workflow.ActivityOptions{
		TaskQueue:           FetchIssuersTaskQueue,
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         retrypolicy,
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var response FetchIssuersActivityResponse

	err := workflow.ExecuteActivity(ctx, FetchIssuersActivity).Get(ctx, &response)

	if err != nil {
		return err
	}

	if len(response.Issuers) == 0 {
		return errors.New("no issuers found")
	}
	input := CreateCredentialIssuersInput{
		Issuers: response.Issuers,
		DBPath:  os.Getenv("DATA_DB_PATH"),
	}

	errCreate := workflow.ExecuteActivity(ctx, CreateCredentialIssuersActivity, input).Get(ctx, nil)

	if errCreate != nil {
		return errCreate
	}
	return nil
}
