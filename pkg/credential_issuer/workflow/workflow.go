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

// FetchIssuersWorkflow is a Temporal workflow that orchestrates the process of fetching
// credential issuers and creating them in the database. It performs the following steps:
//
// 1. Configures retry policies and activity options for the workflow context.
// 2. Executes the FetchIssuersActivity to retrieve a list of issuers.
// 3. Validates the response to ensure issuers are found.
// 4. Executes the CreateCredentialIssuersActivity to store the issuers in the database.
//
// Parameters:
// - ctx: The workflow context provided by Temporal.
//
// Returns:
// - error: An error if any of the activities fail or if no issuers are found.
//
// Activities:
// - FetchIssuersActivity: Fetches a list of credential issuers.
// - CreateCredentialIssuersActivity: Stores the fetched issuers in the database.
//
// Notes:
// - The workflow uses a retry policy with exponential backoff for activity retries.
// - The database path is retrieved from the "DATA_DB_PATH" environment variable.
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
