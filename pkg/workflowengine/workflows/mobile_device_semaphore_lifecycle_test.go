// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"fmt"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileDeviceSemaphoreWorkflowPauseAndResume(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	pauseCh := make(chan MobileDeviceSemaphorePauseDeviceResponse, 1)
	resumeCh := make(chan MobileDeviceSemaphoreResumeDeviceResponse, 1)
	stateCh := make(chan MobileDeviceSemaphoreStateView, 2)
	errCh := make(chan error, 4)

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			MobileDeviceSemaphorePauseDeviceUpdate,
			"pause-runner",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(result interface{}, err error) {
					if err != nil {
						errCh <- err
						return
					}
					if resp, ok := result.(MobileDeviceSemaphorePauseDeviceResponse); ok {
						pauseCh <- resp
					}
				},
			},
			MobileDeviceSemaphorePauseDeviceRequest{
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
			MobileDeviceSemaphoreResumeDeviceUpdate,
			"resume-runner",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(result interface{}, err error) {
					if err != nil {
						errCh <- err
						return
					}
					if resp, ok := result.(MobileDeviceSemaphoreResumeDeviceResponse); ok {
						resumeCh <- resp
					}
				},
			},
			MobileDeviceSemaphoreResumeDeviceRequest{Reason: "runner_startup"},
		)
	}, 3*time.Second)

	env.RegisterDelayedCallback(func() {
		querySemaphoreState(env, stateCh, errCh)
	}, 4*time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, 5*time.Second)

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileDeviceSemaphoreWorkflowInput{
			DeviceID: "runner-1",
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

func TestMobileDeviceSemaphoreWorkflowResumeBeforePauseTimeoutPreventsShutdown(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	stateCh := make(chan MobileDeviceSemaphoreStateView, 1)
	errCh := make(chan error, 2)

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			MobileDeviceSemaphorePauseDeviceUpdate,
			"pause-runner-timeout",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(_ interface{}, err error) {
					if err != nil {
						errCh <- err
					}
				},
			},
			MobileDeviceSemaphorePauseDeviceRequest{
				Reason:               "maintenance",
				CancelRunning:        true,
				ShutdownAfterSeconds: 3,
			},
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			MobileDeviceSemaphoreResumeDeviceUpdate,
			"resume-runner-before-timeout",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { errCh <- err },
				OnComplete: func(_ interface{}, err error) {
					if err != nil {
						errCh <- err
					}
				},
			},
			MobileDeviceSemaphoreResumeDeviceRequest{Reason: "runner_startup"},
		)
	}, 2*time.Second)

	env.RegisterDelayedCallback(func() {
		querySemaphoreState(env, stateCh, errCh)
	}, 5*time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, 6*time.Second)

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileDeviceSemaphoreWorkflowInput{
			DeviceID: "runner-1",
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

// An offline device is paused once and then receives a no-op pause on every
// runner heartbeat. Each one is a distinct Temporal update, so the run must
// continue-as-new before the server's per-run update limit, keeping its state.
func TestMobileDeviceSemaphoreWorkflowNoopPausesContinueAsNew(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, mobileDeviceSemaphoreMaxUpdateBatches)
	env.RegisterDelayedCallback(func() {
		for i := range mobileDeviceSemaphoreMaxUpdateBatches {
			env.UpdateWorkflow(
				MobileDeviceSemaphorePauseDeviceUpdate,
				fmt.Sprintf("pause/runner-1/heartbeat-%d", i),
				&testsuite.TestUpdateCallback{
					OnReject:   func(err error) { errCh <- err },
					OnComplete: func(_ interface{}, err error) { errCh <- err },
				},
				MobileDeviceSemaphorePauseDeviceRequest{
					Reason:               "device offline",
					CancelRunning:        true,
					ShutdownAfterSeconds: int((7 * 24 * time.Hour) / time.Second),
				},
			)
		}
	}, time.Second)

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileDeviceSemaphoreWorkflowInput{
			DeviceID: "runner-1",
			Capacity: 1,
			State: &MobileDeviceSemaphoreWorkflowState{
				Paused:               true,
				PauseReason:          "device offline",
				PauseGeneration:      3,
				ShutdownAfterSeconds: int((7 * 24 * time.Hour) / time.Second),
			},
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)
	var canErr *workflow.ContinueAsNewError
	require.ErrorAs(t, err, &canErr)

	var next workflowengine.WorkflowInput
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(canErr.Input, &next))
	payload, err := workflowengine.DecodePayload[MobileDeviceSemaphoreWorkflowInput](next.Payload)
	require.NoError(t, err)
	require.NotNil(t, payload.State)
	require.True(t, payload.State.Paused)
	require.Equal(t, "device offline", payload.State.PauseReason)
	require.Equal(t, 3, payload.State.PauseGeneration)
	require.Zero(t, payload.State.UpdateCount)

	for range mobileDeviceSemaphoreMaxUpdateBatches {
		require.NoError(t, <-errCh)
	}
}

func querySemaphoreState(
	env *testsuite.TestWorkflowEnvironment,
	stateCh chan<- MobileDeviceSemaphoreStateView,
	errCh chan<- error,
) {
	encoded, err := env.QueryWorkflow(MobileDeviceSemaphoreStateQuery)
	if err != nil {
		errCh <- err
		return
	}

	var state MobileDeviceSemaphoreStateView
	if err := encoded.Get(&state); err != nil {
		errCh <- err
		return
	}
	stateCh <- state
}
