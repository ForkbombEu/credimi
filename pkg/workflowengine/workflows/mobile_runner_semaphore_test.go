// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileRunnerSemaphoreWorkflowFIFO(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permitsCh := make(chan MobileRunnerSemaphorePermit, 3)
	errCh := make(chan error, 3)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitsCh,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-2", LeaseID: "lease-2"},
			permitsCh,
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-3", LeaseID: "lease-3"},
			permitsCh,
			errCh,
		)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-1"}, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-2"}, errCh)
	}, time.Second*5)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-3"}, errCh)
	}, time.Second*6)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*7)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	permits := map[string]MobileRunnerSemaphorePermit{}
	for len(permits) < 3 {
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case permit := <-permitsCh:
			permits[permit.LeaseID] = permit
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for permits")
		}
	}

	require.Empty(t, drainErrors(errCh))

	p1 := permits["lease-1"]
	p2 := permits["lease-2"]
	p3 := permits["lease-3"]

	require.True(t, p1.GrantedAt.Before(p2.GrantedAt))
	require.True(t, p2.GrantedAt.Before(p3.GrantedAt))
	require.Zero(t, p1.QueueWaitMs)
	require.Greater(t, p2.QueueWaitMs, int64(0))
	require.Greater(t, p3.QueueWaitMs, int64(0))
}

func TestMobileRunnerSemaphoreWorkflowAcquireIdempotent(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permitsCh := make(chan MobileRunnerSemaphorePermit, 2)
	errCh := make(chan error, 2)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitsCh,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitsCh,
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*3)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var first MobileRunnerSemaphorePermit
	var second MobileRunnerSemaphorePermit

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case permit := <-permitsCh:
			if i == 0 {
				first = permit
			} else {
				second = permit
			}
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for permits")
		}
	}

	require.Empty(t, drainErrors(errCh))
	require.Equal(t, first, second)
}

func TestMobileRunnerSemaphoreWorkflowReleaseUnknownLease(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 1)
	releaseCh := make(chan MobileRunnerSemaphoreReleaseResult, 1)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdateWithResult(
			env,
			MobileRunnerSemaphoreReleaseRequest{LeaseID: "missing-lease"},
			releaseCh,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*2)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case result := <-releaseCh:
		require.False(t, result.Released)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for release")
	}

	require.Empty(t, drainErrors(errCh))
}

func enqueueAcquireUpdate(
	env *testsuite.TestWorkflowEnvironment,
	req MobileRunnerSemaphoreAcquireRequest,
	permitsCh chan<- MobileRunnerSemaphorePermit,
	errCh chan<- error,
) {
	env.UpdateWorkflow(MobileRunnerSemaphoreAcquireUpdate, req.RequestID, &testsuite.TestUpdateCallback{
		OnReject: func(err error) {
			errCh <- err
		},
		OnComplete: func(response interface{}, err error) {
			if err != nil {
				errCh <- err
				return
			}
			permit, ok := response.(MobileRunnerSemaphorePermit)
			if !ok {
				errCh <- fmt.Errorf("unexpected permit type: %T", response)
				return
			}
			permitsCh <- permit
		},
	}, req)
}

func enqueueReleaseUpdate(
	env *testsuite.TestWorkflowEnvironment,
	req MobileRunnerSemaphoreReleaseRequest,
	errCh chan<- error,
) {
	env.UpdateWorkflow(MobileRunnerSemaphoreReleaseUpdate, req.LeaseID, &testsuite.TestUpdateCallback{
		OnReject: func(err error) {
			errCh <- err
		},
		OnComplete: func(_ interface{}, err error) {
			if err != nil {
				errCh <- err
			}
		},
	}, req)
}

func enqueueReleaseUpdateWithResult(
	env *testsuite.TestWorkflowEnvironment,
	req MobileRunnerSemaphoreReleaseRequest,
	releaseCh chan<- MobileRunnerSemaphoreReleaseResult,
	errCh chan<- error,
) {
	env.UpdateWorkflow(MobileRunnerSemaphoreReleaseUpdate, req.LeaseID, &testsuite.TestUpdateCallback{
		OnReject: func(err error) {
			errCh <- err
		},
		OnComplete: func(response interface{}, err error) {
			if err != nil {
				errCh <- err
				return
			}
			result, ok := response.(MobileRunnerSemaphoreReleaseResult)
			if !ok {
				errCh <- fmt.Errorf("unexpected release type: %T", response)
				return
			}
			releaseCh <- result
		},
	}, req)
}

func drainErrors(errCh <-chan error) []error {
	var errs []error
	for {
		select {
		case err := <-errCh:
			errs = append(errs, err)
		default:
			return errs
		}
	}
}
