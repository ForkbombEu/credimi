// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestQueryMobileRunnerSemaphoreRunStatusActivityMissingFields(t *testing.T) {
	activity := NewQueryMobileRunnerSemaphoreRunStatusActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileRunnerSemaphoreRunStatusInput{
			RunnerID: "runner-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

func TestQueryMobileRunnerSemaphoreRunStatusActivityDecodeError(t *testing.T) {
	activity := NewQueryMobileRunnerSemaphoreRunStatusActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-query-payload",
	})
	require.Error(t, err)
}

func TestQueryMobileRunnerSemaphoreRunStatusActivityName(t *testing.T) {
	require.Equal(
		t,
		"Query mobile runner semaphore run status",
		NewQueryMobileRunnerSemaphoreRunStatusActivity().Name(),
	)
}
