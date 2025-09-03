// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	PipelineTaskQueue              = "PipelineTaskQueue"
	DefaultNameSpace               = "default"
	DefaultExecutionTimeout        = "24h"
	DefaultActivityScheduleTimeout = "10m"
	DefaultActivityStartTimeout    = "5m"
	DefaultRetryMaxAttempts        = int32(5)
	DefaultRetryInitialInterval    = "5s"
	DefaultRetryMaxInterval        = "1m"
	DefaultRetryBackoff            = 2.0
)

// Convert runtime config to Temporal SDK types
func PrepareWorkflowOptions(rc RuntimeConfig) WorkflowOptions {
	// Parse timeouts
	taskQueue := PipelineTaskQueue
	if rc.Temporal.TaskQueue != "" {
		taskQueue = rc.Temporal.TaskQueue
	}
	namespace := DefaultNameSpace
	if rc.Temporal.Namespace != "" {
		namespace = rc.Temporal.Namespace
	}

	rp := temporal.RetryPolicy{
		MaximumAttempts: DefaultRetryMaxAttempts,
		InitialInterval: parseDurationOrDefault(
			rc.Temporal.RetryPolicy.InitialInterval,
			DefaultRetryInitialInterval,
		),
		BackoffCoefficient: DefaultRetryBackoff,
		MaximumInterval: parseDurationOrDefault(
			rc.Temporal.RetryPolicy.MaximumInterval,
			DefaultRetryMaxInterval,
		),
	}
	if rc.Temporal.RetryPolicy.MaximumAttempts > 0 {
		rp.MaximumAttempts = rc.Temporal.RetryPolicy.MaximumAttempts
	}
	if rc.Temporal.RetryPolicy.BackoffCoefficient > 0 {
		rp.BackoffCoefficient = rc.Temporal.RetryPolicy.BackoffCoefficient
	}

	return WorkflowOptions{
		Namespace: namespace,
		Options: client.StartWorkflowOptions{

			TaskQueue: taskQueue,
			WorkflowExecutionTimeout: parseDurationOrDefault(
				rc.Temporal.ExecutionTimeout,
				DefaultExecutionTimeout,
			),
		},
		ActivityOptions: workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute * 10,
			StartToCloseTimeout:    time.Minute * 5,
			RetryPolicy:            &rp,
		},
	}
}

func PrepareActivityOptions(
	rp *temporal.RetryPolicy,
	retry map[string]any,
	timeout string,
) workflow.ActivityOptions {
	initialInterval, ok := retry["initialInterval"].(string)
	if ok {
		rp.InitialInterval = parseDurationOrDefault(initialInterval, rp.InitialInterval.String())
	}

	maxInterval, ok := retry["maximumInterval"].(string)
	if !ok {
		rp.MaximumInterval = parseDurationOrDefault(maxInterval, rp.MaximumInterval.String())
	}

	if retry["maximumAttempts"] != nil {
		rp.MaximumAttempts = int32(retry["maximumAttempts"].(float64))
	}
	if retry["backoffCoefficient"] != nil {
		rp.BackoffCoefficient = retry["backoffCoefficient"].(float64)
	}
	return workflow.ActivityOptions{
		ScheduleToCloseTimeout: parseDurationOrDefault(timeout, DefaultActivityScheduleTimeout),
		StartToCloseTimeout:    parseDurationOrDefault(timeout, DefaultActivityStartTimeout),
		RetryPolicy:            rp,
	}
}

func parseDurationOrDefault(s, def string) time.Duration {
	if s == "" {
		s = def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Minute * 5 // fallback
	}
	return d
}

// Adapter: turn SubWorkflowDefinition into a WorkflowDefinition
func (s *WorkflowBlock) ToWorkflowDefinition(name string) *WorkflowDefinition {
	return &WorkflowDefinition{

		Name:    name,
		Env:     map[string]string{},
		Runtime: RuntimeConfig{},
		Checks:  map[string]WorkflowBlock{},
		Config:  s.Config,
		Steps:   s.Steps,
	}
}
