// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
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
	// Set defaults for task queue and namespace
	taskQueue := PipelineTaskQueue
	if rc.Temporal.TaskQueue != "" {
		taskQueue = rc.Temporal.TaskQueue
	}
	namespace := DefaultNameSpace
	if rc.Temporal.Namespace != "" {
		namespace = rc.Temporal.Namespace
	}

	rp := &temporal.RetryPolicy{
		MaximumAttempts: DefaultRetryMaxAttempts,
		InitialInterval: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.RetryPolicy.InitialInterval,
			DefaultRetryInitialInterval,
		),
		MaximumInterval: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.RetryPolicy.MaximumInterval,
			DefaultRetryMaxInterval,
		),
		BackoffCoefficient: DefaultRetryBackoff,
	}

	if rc.Temporal.ActivityOptions.RetryPolicy.MaximumAttempts > 0 {
		rp.MaximumAttempts = rc.Temporal.ActivityOptions.RetryPolicy.MaximumAttempts
	}
	if rc.Temporal.ActivityOptions.RetryPolicy.BackoffCoefficient > 0 {
		rp.BackoffCoefficient = rc.Temporal.ActivityOptions.RetryPolicy.BackoffCoefficient
	}

	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.ScheduleToCloseTimeout,
			DefaultActivityScheduleTimeout,
		),
		StartToCloseTimeout: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.StartToCloseTimeout,
			DefaultActivityStartTimeout,
		),
		RetryPolicy: rp,
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
		ActivityOptions: ao,
	}
}

func PrepareActivityOptions(
	globalAO workflow.ActivityOptions,
	stepAO *ActivityOptionsConfig,
) workflow.ActivityOptions {
	rp := globalAO.RetryPolicy
	scheduleToClose := globalAO.ScheduleToCloseTimeout
	startToClose := globalAO.StartToCloseTimeout

	if stepAO != nil {
		if stepAO.RetryPolicy.MaximumAttempts > 0 {
			rp.MaximumAttempts = stepAO.RetryPolicy.MaximumAttempts
		}
		if stepAO.RetryPolicy.InitialInterval != "" {
			rp.InitialInterval = parseDurationOrDefault(
				stepAO.RetryPolicy.InitialInterval,
				rp.InitialInterval.String(),
			)
		}
		if stepAO.RetryPolicy.MaximumInterval != "" {
			rp.MaximumInterval = parseDurationOrDefault(
				stepAO.RetryPolicy.MaximumInterval,
				rp.MaximumInterval.String(),
			)
		}
		if stepAO.RetryPolicy.BackoffCoefficient > 0 {
			rp.BackoffCoefficient = stepAO.RetryPolicy.BackoffCoefficient
		}
		if stepAO.ScheduleToCloseTimeout != "" {
			scheduleToClose = parseDurationOrDefault(
				stepAO.ScheduleToCloseTimeout,
				scheduleToClose.String(),
			)
		}
		if stepAO.StartToCloseTimeout != "" {
			startToClose = parseDurationOrDefault(stepAO.StartToCloseTimeout, startToClose.String())
		}
	}

	return workflow.ActivityOptions{
		ScheduleToCloseTimeout: scheduleToClose,
		StartToCloseTimeout:    startToClose,
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
		Runtime: RuntimeConfig{},
		Checks:  map[string]WorkflowBlock{},
		Config:  s.Config,
		Steps:   s.Steps,
	}
}
func convertMapAnyToString(m map[string]any) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}

// SetPayloadValue sets a key/value pair in the given payload map.
// If the key exists, it overwrites it; otherwise, it adds it.
func SetPayloadValue(payload *map[string]any, key string, val any) error {
	if payload == nil {
		return fmt.Errorf("payload is nil")
	}
	if *payload == nil {
		*payload = make(map[string]any)
	}

	(*payload)[key] = val
	return nil
}

func SetConfigValue(config *map[string]any, key string, val any) {
	if *config == nil {
		*config = make(map[string]any)
	}
	(*config)[key] = val
}

// MergePayload merges all keys from src into dst recursively.
func MergePayload(dst, src *map[string]any) error {
	if dst == nil || src == nil {
		return nil
	}
	if *dst == nil {
		*dst = make(map[string]any)
	}
	mergeMaps(*dst, *src)
	return nil
}

func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		if existing, ok := dst[k]; ok {
			// recursive merge if both are maps
			dstMap, dstIsMap := existing.(map[string]any)
			srcMap, srcIsMap := v.(map[string]any)
			if dstIsMap && srcIsMap {
				mergeMaps(dstMap, srcMap)
				continue
			}
		}
		dst[k] = deepCopy(v)
	}
}

// deepCopy makes a deep copy of arbitrary JSON/YAML-compatible data.
func deepCopy(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		// fallback: return original reference if not serializable
		return v
	}
	var c any
	if err := json.Unmarshal(b, &c); err != nil {
		return v
	}
	return c
}
