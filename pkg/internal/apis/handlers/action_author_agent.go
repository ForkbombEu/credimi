// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine/authorsession"
	"github.com/pocketbase/pocketbase/core"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

const (
	// actionAuthorPollerFreshness is how recently a runner's action author
	// must have polled its task queue to count as running.
	actionAuthorPollerFreshness = 2 * time.Minute
	actionAuthorDescribeTimeout = 5 * time.Second
)

// describeActionAuthorTaskQueue describes a runner's action author workflow
// task queue and is stubbed in unit tests.
var describeActionAuthorTaskQueue = func(
	ctx context.Context,
	c client.Client,
	taskQueue string,
) (*workflowservice.DescribeTaskQueueResponse, error) {
	return c.DescribeTaskQueue(ctx, taskQueue, enumspb.TASK_QUEUE_TYPE_WORKFLOW)
}

// actionAuthorAgentAvailable reports whether the runner's action author
// agent polled <runnerID>-ActionAuthor within actionAuthorPollerFreshness.
// c must be a client of the runner organization's namespace.
func actionAuthorAgentAvailable(
	ctx context.Context,
	c client.Client,
	runnerID string,
) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, actionAuthorDescribeTimeout)
	defer cancel()
	taskQueue := runnerID + authorsession.AuthorTaskQueueSuffix
	resp, err := describeActionAuthorTaskQueue(ctx, c, taskQueue)
	if err != nil {
		return false, fmt.Errorf("describe task queue %s: %w", taskQueue, err)
	}
	cutoff := time.Now().Add(-actionAuthorPollerFreshness)
	for _, poller := range resp.GetPollers() {
		if last := poller.GetLastAccessTime(); last != nil && last.AsTime().After(cutoff) {
			return true, nil
		}
	}
	return false, nil
}

// requireActionAuthorAgent refuses runners whose action author agent does
// not poll its task queue in the runner organization's namespace.
func requireActionAuthorAgent(
	e *core.RequestEvent,
	runner *core.Record,
	namespace string,
) *apierror.APIError {
	runnerID, err := mobilerunner.RunnerIdentifier(e.App, runner)
	if err != nil {
		return authoringAPIError(fmt.Errorf("build runner identifier: %w", err))
	}
	c, err := queueTemporalClient(namespace)
	if err != nil {
		return temporalUnavailableError()
	}
	available, err := actionAuthorAgentAvailable(e.Request.Context(), c, runnerID)
	if err != nil {
		e.App.Logger().Warn(fmt.Sprintf(
			"cannot check the action author of runner %s: %v",
			runnerID,
			err,
		))
	}
	if !available {
		return actionAuthorAgentUnavailableError()
	}
	return nil
}

// runnerActionAuthorAvailable reports a runner's agent for the device list;
// any error, including an unreachable Temporal, reports it unavailable.
func runnerActionAuthorAvailable(e *core.RequestEvent, runner *core.Record, namespace string) bool {
	runnerID, err := mobilerunner.RunnerIdentifier(e.App, runner)
	if err != nil {
		e.App.Logger().Warn(fmt.Sprintf("cannot identify runner %s: %v", runner.Id, err))
		return false
	}
	c, err := queueTemporalClient(namespace)
	if err != nil {
		e.App.Logger().Warn(fmt.Sprintf(
			"cannot check the action author of runner %s: %v", runnerID, err,
		))
		return false
	}
	available, err := actionAuthorAgentAvailable(e.Request.Context(), c, runnerID)
	if err != nil {
		e.App.Logger().Warn(fmt.Sprintf(
			"cannot check the action author of runner %s: %v", runnerID, err,
		))
	}
	return available
}

func actionAuthorAgentUnavailableError() *apierror.APIError {
	return apierror.New(
		http.StatusServiceUnavailable,
		"action_author",
		"agent_unavailable",
		"the runner's action author is not enabled or not running",
	)
}
