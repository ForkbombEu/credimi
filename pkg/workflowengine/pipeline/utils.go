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

func SetPayloadValue(payload map[string]InputSource, key string, val any) {
	src := payload[key]
	src.Value = val
	payload[key] = src
}

func SetConfigValue(config *map[string]any, key string, val any) {
	if *config == nil {
		*config = make(map[string]any)
	}
	(*config)[key] = val
}

func MergePayload(dst map[string]InputSource, src map[string]any) {
	for k, v := range src {
		if vMap, ok := v.(map[string]any); ok {
			// If dst already has a map, recurse
			if existing, exists := dst[k]; exists {
				if existingMap, ok := existing.Value.(map[string]any); ok {
					mergeMaps(existingMap, vMap)
					dst[k] = InputSource{Value: existingMap}
					continue
				}
			}
			dst[k] = InputSource{Value: vMap}
		} else {
			if _, exists := dst[k]; !exists {
				dst[k] = InputSource{Value: v}
			}
		}
	}
}

func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		switch vTyped := v.(type) {
		case map[string]any:
			if existing, exists := dst[k]; exists {
				if existingMap, ok := existing.(map[string]any); ok {
					mergeMaps(existingMap, vTyped)
					dst[k] = existingMap
					continue
				}
			}
			dst[k] = vTyped
		case []any:
			if existing, exists := dst[k]; !exists || existing == nil {
				dst[k] = vTyped
			}
		default:
			if _, exists := dst[k]; !exists {
				dst[k] = v
			}
		}
	}
}
