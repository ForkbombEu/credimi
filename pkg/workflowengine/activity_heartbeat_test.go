// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
)

func TestHeartbeatHelpersOutsideActivity(t *testing.T) {
	require.NotPanics(t, func() {
		RecordHeartbeat(context.Background(), "details")
		stop := StartHeartbeat(context.Background(), time.Millisecond)
		stop()
		stop()
	})
}

// heartbeatCounter counts RecordHeartbeat calls before the SDK batches them;
// the test environment reports at most one heartbeat per 30s throttle window.
type heartbeatCounter struct {
	interceptor.WorkerInterceptorBase
	calls      atomic.Int32
	badDetails atomic.Int32
}

func (c *heartbeatCounter) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &heartbeatCounterInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		counter:                        c,
	}
}

type heartbeatCounterInbound struct {
	interceptor.ActivityInboundInterceptorBase
	counter *heartbeatCounter
}

func (i *heartbeatCounterInbound) Init(outbound interceptor.ActivityOutboundInterceptor) error {
	return i.Next.Init(&heartbeatCounterOutbound{
		ActivityOutboundInterceptorBase: interceptor.ActivityOutboundInterceptorBase{
			Next: outbound,
		},
		counter: i.counter,
	})
}

type heartbeatCounterOutbound struct {
	interceptor.ActivityOutboundInterceptorBase
	counter *heartbeatCounter
}

func (o *heartbeatCounterOutbound) RecordHeartbeat(ctx context.Context, details ...any) {
	if len(details) != 1 || details[0] != "working" {
		o.counter.badDetails.Add(1)
	}
	o.counter.calls.Add(1)
	o.Next.RecordHeartbeat(ctx, details...)
}

func TestStartHeartbeatRecordsUntilStopped(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	counter := &heartbeatCounter{}
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{counter},
	})
	heartbeats := &counter.calls

	var afterStop atomic.Int32
	env.RegisterActivityWithOptions(
		func(ctx context.Context) error {
			stop := StartHeartbeat(ctx, 5*time.Millisecond, "working")
			deadline := time.Now().Add(5 * time.Second)
			for heartbeats.Load() < 3 {
				if time.Now().After(deadline) {
					stop()
					return errors.New("no heartbeats recorded")
				}
				time.Sleep(time.Millisecond)
			}
			stop()
			// Let a tick that raced with stop finish, then expect no more.
			time.Sleep(20 * time.Millisecond)
			stoppedAt := heartbeats.Load()
			time.Sleep(50 * time.Millisecond)
			afterStop.Store(heartbeats.Load() - stoppedAt)
			return nil
		},
		activity.RegisterOptions{Name: "heartbeat-test-activity"},
	)

	_, err := env.ExecuteActivity("heartbeat-test-activity")
	require.NoError(t, err)
	require.GreaterOrEqual(t, heartbeats.Load(), int32(3))
	require.Zero(t, counter.badDetails.Load())
	require.Zero(t, afterStop.Load())
}
