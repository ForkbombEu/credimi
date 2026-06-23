// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileRunnerSemaphoreWorkflowPauseAndResume(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	pauseCh := make(chan MobileRunnerSemaphorePauseRunnerResponse, 1)
	resumeCh := make(chan MobileRunnerSemaphoreResumeRunnerResponse, 1)
	stateCh := make(chan MobileRunnerSemaphoreStateView, 2)
	errCh := make(chan error, 4)

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			MobileRunnerSemaphorePauseRunnerUpdate,
			"pause-runner",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(result interface{}, err error) {
					if err != nil {
						errCh <- err
						return
					}
					if resp, ok := result.(MobileRunnerSemaphorePauseRunnerResponse); ok {
						pauseCh <- resp
					}
				},
			},
			MobileRunnerSemaphorePauseRunnerRequest{
				Reason:               "maintenance",
				CancelRunning:        true,
				ShutdownAfterSeconds: 30,
			},
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		querySemaphoreState(env, stateCh, errCh)
	}, 2*time.Second)

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			MobileRunnerSemaphoreResumeRunnerUpdate,
			"resume-runner",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(result interface{}, err error) {
					if err != nil {
						errCh <- err
						return
					}
					if resp, ok := result.(MobileRunnerSemaphoreResumeRunnerResponse); ok {
						resumeCh <- resp
					}
				},
			},
			MobileRunnerSemaphoreResumeRunnerRequest{Reason: "runner_startup"},
		)
	}, 3*time.Second)

	env.RegisterDelayedCallback(func() {
		querySemaphoreState(env, stateCh, errCh)
	}, 4*time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, 5*time.Second)

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileRunnerSemaphoreWorkflowInput{
			RunnerID: "runner-1",
			Capacity: 1,
		},
	})

	require.True(t, env.IsWorkflowCompleted())

	select {
	case resp := <-pauseCh:
		require.True(t, resp.Paused)
		require.Equal(t, 30, resp.ShutdownAfterSeconds)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for pause update")
	}

	select {
	case state := <-stateCh:
		require.True(t, state.Paused)
		require.Equal(t, "maintenance", state.PauseReason)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for paused state")
	}

	select {
	case resp := <-resumeCh:
		require.False(t, resp.Paused)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for resume update")
	}

	select {
	case state := <-stateCh:
		require.False(t, state.Paused)
		require.Empty(t, state.PauseReason)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for resumed state")
	}
}

func TestMobileRunnerSemaphoreWorkflowResumeBeforePauseTimeoutPreventsShutdown(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	stateCh := make(chan MobileRunnerSemaphoreStateView, 1)
	errCh := make(chan error, 2)

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			MobileRunnerSemaphorePauseRunnerUpdate,
			"pause-runner-timeout",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(_ interface{}, err error) {
					if err != nil {
						errCh <- err
					}
				},
			},
			MobileRunnerSemaphorePauseRunnerRequest{
				Reason:               "maintenance",
				CancelRunning:        true,
				ShutdownAfterSeconds: 3,
			},
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			MobileRunnerSemaphoreResumeRunnerUpdate,
			"resume-runner-before-timeout",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(_ interface{}, err error) {
					if err != nil {
						errCh <- err
					}
				},
			},
			MobileRunnerSemaphoreResumeRunnerRequest{Reason: "runner_startup"},
		)
	}, 2*time.Second)

	env.RegisterDelayedCallback(func() {
		querySemaphoreState(env, stateCh, errCh)
	}, 5*time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, 6*time.Second)

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileRunnerSemaphoreWorkflowInput{
			RunnerID: "runner-1",
			Capacity: 1,
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())

	select {
	case state := <-stateCh:
		require.False(t, state.Paused)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for resumed state after timeout window")
	}
}

func querySemaphoreState(
	env *testsuite.TestWorkflowEnvironment,
	stateCh chan<- MobileRunnerSemaphoreStateView,
	errCh chan<- error,
) {
	encoded, err := env.QueryWorkflow(MobileRunnerSemaphoreStateQuery)
	if err != nil {
		errCh <- err
		return
	}

	var state MobileRunnerSemaphoreStateView
	if err := encoded.Get(&state); err != nil {
		errCh <- err
		return
	}
	stateCh <- state
}
