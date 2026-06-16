// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
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

func PrepareWorkflowOptions(rc pipeline.RuntimeConfig) pipeline.WorkflowOptions {
	// Set defaults for task queue and namespace
	taskQueue := PipelineTaskQueue
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

	return pipeline.WorkflowOptions{
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
	stepAO *pipeline.ActivityOptionsConfig,
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

func SetRunDataValue(runData *map[string]any, key string, val any) {
	if *runData == nil {
		*runData = make(map[string]any)
	}
	(*runData)[key] = val
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

func getPipelineRunIdentifier(namespace, workflowID, runID string) string {
	id := fmt.Sprintf("%s-%s", workflowID, runID)
	return fmt.Sprintf("%s/%s", namespace, canonify.CanonifyPlain(id))
}

func DecodePayload(s *pipeline.StepDefinition) (any, error) {
	desc, ok := registry.Registry[s.Use]
	if !ok {
		return nil, fmt.Errorf("unknown step type: %s", s.Use)
	}
	if desc.PayloadType == nil {
		return nil, fmt.Errorf("no input type registered for %s", s.Use)
	}

	data, err := json.Marshal(s.With.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload map: %w", err)
	}

	valPtr := reflect.New(desc.PayloadType).Interface()

	if err := json.Unmarshal(data, valPtr); err != nil {
		return nil, fmt.Errorf(
			"failed to decode payload for %s: %s",
			s.ID,
			formatPayloadDecodeError(s, err),
		)
	}

	return valPtr, nil
}

func formatPayloadDecodeError(s *pipeline.StepDefinition, err error) string {
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		return err.Error()
	}

	if typeErr.Field == "parameters" && typeErr.Type.Kind() == reflect.String {
		return formatStringParameterDecodeError(s)
	}

	if typeErr.Field == "" {
		return fmt.Sprintf(
			"invalid payload for %s: expected %s but got %s",
			s.Use,
			typeErr.Type,
			typeErr.Value,
		)
	}

	return fmt.Sprintf(
		"invalid payload for %s: with.payload.%s expected %s but got %s",
		s.Use,
		typeErr.Field,
		typeErr.Type,
		typeErr.Value,
	)
}

func formatStringParameterDecodeError(s *pipeline.StepDefinition) string {
	paramName, paramType := firstNonStringParameter(s.With.Payload["parameters"])
	if paramName == "" {
		return fmt.Sprintf(
			"invalid payload for %s: with.payload.parameters must contain only string values",
			s.Use,
		)
	}

	return fmt.Sprintf(
		"invalid payload for %s: with.payload.parameters must contain only string values; "+
			"parameter %q resolved to %s",
		s.Use,
		paramName,
		paramType,
	)
}

func firstNonStringParameter(parameters any) (string, string) {
	switch params := parameters.(type) {
	case map[string]any:
		for key, value := range params {
			if _, ok := value.(string); !ok {
				return key, payloadValueType(value)
			}
		}
	case map[string]string:
		return "", ""
	default:
		return "parameters", payloadValueType(parameters)
	}

	return "", ""
}

func payloadValueType(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case map[string]any, map[string]string:
		return "object"
	case []any, []string:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	default:
		return fmt.Sprintf("%T", value)
	}
}
