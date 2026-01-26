// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package avdpool

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type responseCollector struct {
	mu        sync.Mutex
	responses []PoolSlotResponse
}

func (collector *responseCollector) Add(resp PoolSlotResponse) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.responses = append(collector.responses, resp)
}

func (collector *responseCollector) Responses() []PoolSlotResponse {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	copyOfResponses := make([]PoolSlotResponse, len(collector.responses))
	copy(copyOfResponses, collector.responses)
	return copyOfResponses
}

func setupPoolTestEnv(t *testing.T) (*testsuite.TestWorkflowEnvironment, *responseCollector) {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	collector := &responseCollector{}
	env.OnSignalExternalWorkflow(
		mock.Anything,
		mock.Anything,
		mock.Anything,
		PoolResponseSignal,
		mock.Anything,
	).Return(func(_ string, _ string, _ string, _ string, arg interface{}) error {
		resp, ok := arg.(PoolSlotResponse)
		require.True(t, ok)
		collector.Add(resp)
		return nil
	})

	return env, collector
}

func TestPoolManagerWorkflow_FIFOQueue(t *testing.T) {
	env, collector := setupPoolTestEnv(t)
	workflow := NewPoolManagerWorkflow()
	env.RegisterWorkflow(workflow.Workflow)

	config := PoolConfig{
		MaxConcurrentEmulators: 3,
		MaxQueueDepth:          10,
		LeaseTimeout:           5 * time.Minute,
		HeartbeatInterval:      time.Second,
	}
	input := workflowengine.WorkflowInput{
		Payload: PoolManagerWorkflowInput{Config: config},
		Config:  map[string]any{},
	}

	env.RegisterDelayedCallback(func() {
		for index := 1; index <= 5; index++ {
			request := PoolRequest{
				WorkflowID:  fmt.Sprintf("wf-%d", index),
				RunID:       "run",
				RequestID:   fmt.Sprintf("req-%d", index),
				RequestTime: env.Now(),
				Timeout:     time.Minute,
			}
			env.SignalWorkflow(PoolAcquireSignal, request)
		}
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(PoolReleaseSignal, PoolRelease{WorkflowID: "wf-1", RunID: "run"})
		env.SignalWorkflow(PoolReleaseSignal, PoolRelease{WorkflowID: "wf-2", RunID: "run"})
	}, 2*time.Second)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, 4*time.Second)

	env.ExecuteWorkflow(workflow.Workflow, input)
	require.Error(t, env.GetWorkflowError())

	responses := collector.Responses()
	require.Len(t, responses, 5)
	for i := 0; i < 5; i++ {
		require.True(t, responses[i].Granted)
		require.Equal(t, fmt.Sprintf("wf-%d", i+1), responses[i].WorkflowID)
	}
}

func TestPoolManagerWorkflow_Timeout(t *testing.T) {
	env, collector := setupPoolTestEnv(t)
	workflow := NewPoolManagerWorkflow()
	env.RegisterWorkflow(workflow.Workflow)

	config := PoolConfig{
		MaxConcurrentEmulators: 1,
		MaxQueueDepth:          5,
		LeaseTimeout:           5 * time.Minute,
		HeartbeatInterval:      time.Second,
	}
	input := workflowengine.WorkflowInput{
		Payload: PoolManagerWorkflowInput{Config: config},
		Config:  map[string]any{},
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(PoolAcquireSignal, PoolRequest{
			WorkflowID:  "wf-1",
			RunID:       "run",
			RequestID:   "req-1",
			RequestTime: env.Now(),
			Timeout:     time.Minute,
		})
		env.SignalWorkflow(PoolAcquireSignal, PoolRequest{
			WorkflowID:  "wf-2",
			RunID:       "run",
			RequestID:   "req-2",
			RequestTime: env.Now(),
			Timeout:     time.Minute,
		})
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, 2*time.Minute)

	env.ExecuteWorkflow(workflow.Workflow, input)
	require.Error(t, env.GetWorkflowError())

	responses := collector.Responses()
	require.Len(t, responses, 2)
	require.True(t, responses[0].Granted)
	require.False(t, responses[1].Granted)
	require.Equal(t, PoolTimeoutErrorCode, responses[1].ErrorCode)
}

func TestPoolManagerWorkflow_LeaseExpiryReleasesSlot(t *testing.T) {
	env, collector := setupPoolTestEnv(t)
	workflow := NewPoolManagerWorkflow()
	env.RegisterWorkflow(workflow.Workflow)

	config := PoolConfig{
		MaxConcurrentEmulators: 1,
		MaxQueueDepth:          5,
		LeaseTimeout:           2 * time.Minute,
		HeartbeatInterval:      time.Second,
	}
	input := workflowengine.WorkflowInput{
		Payload: PoolManagerWorkflowInput{Config: config},
		Config:  map[string]any{},
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(PoolAcquireSignal, PoolRequest{
			WorkflowID:  "wf-1",
			RunID:       "run",
			RequestID:   "req-1",
			RequestTime: env.Now(),
			Timeout:     5 * time.Minute,
		})
		env.SignalWorkflow(PoolAcquireSignal, PoolRequest{
			WorkflowID:  "wf-2",
			RunID:       "run",
			RequestID:   "req-2",
			RequestTime: env.Now(),
			Timeout:     5 * time.Minute,
		})
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, 3*time.Minute)

	env.ExecuteWorkflow(workflow.Workflow, input)
	require.Error(t, env.GetWorkflowError())

	responses := collector.Responses()
	require.Len(t, responses, 2)
	require.True(t, responses[0].Granted)
	require.True(t, responses[1].Granted)
}

func TestPoolManagerWorkflow_UpdateCapacity(t *testing.T) {
	env, collector := setupPoolTestEnv(t)
	workflow := NewPoolManagerWorkflow()
	env.RegisterWorkflow(workflow.Workflow)

	config := PoolConfig{
		MaxConcurrentEmulators: 3,
		MaxQueueDepth:          10,
		LeaseTimeout:           5 * time.Minute,
		HeartbeatInterval:      time.Second,
	}
	input := workflowengine.WorkflowInput{
		Payload: PoolManagerWorkflowInput{Config: config},
		Config:  map[string]any{},
	}

	env.RegisterDelayedCallback(func() {
		for index := 1; index <= 5; index++ {
			env.SignalWorkflow(PoolAcquireSignal, PoolRequest{
				WorkflowID:  fmt.Sprintf("wf-%d", index),
				RunID:       "run",
				RequestID:   fmt.Sprintf("req-%d", index),
				RequestTime: env.Now(),
				Timeout:     5 * time.Minute,
			})
		}
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(PoolUpdateCapacitySignal, PoolCapacityUpdate{MaxConcurrentEmulators: 5})
	}, 2*time.Second)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, 3*time.Second)

	env.ExecuteWorkflow(workflow.Workflow, input)
	require.Error(t, env.GetWorkflowError())

	responses := collector.Responses()
	require.Len(t, responses, 5)
}

func TestPoolManagerWorkflow_IntegrationConcurrency(t *testing.T) {
	t.Skip("integration test requires running emulators and avdctl tooling")
}

func TestPoolManagerWorkflow_ChaosRestart(t *testing.T) {
	t.Skip("chaos test requires Temporal cluster and worker orchestration")
}
