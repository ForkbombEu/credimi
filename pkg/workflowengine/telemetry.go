// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"fmt"

	"go.temporal.io/sdk/workflow"
)

const (
	TelemetryRootWorkflowIDKey    = "root_workflow_id"
	TelemetryRootRunIDKey         = "root_run_id"
	TelemetryParentWorkflowIDKey  = "parent_workflow_id"
	TelemetryParentRunIDKey       = "parent_run_id"
	TelemetryCurrentWorkflowIDKey = "current_workflow_id"
	TelemetryCurrentRunIDKey      = "current_run_id"
)

func MergeTelemetryConfig(ctx workflow.Context, cfg map[string]any) map[string]any {
	merged := copyAnyMap(cfg)
	info := workflow.GetInfo(ctx)

	rootWorkflowID := stringConfigValue(merged, TelemetryRootWorkflowIDKey, info.WorkflowExecution.ID)
	rootRunID := stringConfigValue(merged, TelemetryRootRunIDKey, info.WorkflowExecution.RunID)
	parentWorkflowID := stringConfigValue(merged, TelemetryParentWorkflowIDKey, info.WorkflowExecution.ID)
	parentRunID := stringConfigValue(merged, TelemetryParentRunIDKey, info.WorkflowExecution.RunID)

	merged[TelemetryRootWorkflowIDKey] = rootWorkflowID
	merged[TelemetryRootRunIDKey] = rootRunID
	merged[TelemetryParentWorkflowIDKey] = parentWorkflowID
	merged[TelemetryParentRunIDKey] = parentRunID
	merged[TelemetryCurrentWorkflowIDKey] = info.WorkflowExecution.ID
	merged[TelemetryCurrentRunIDKey] = info.WorkflowExecution.RunID

	return merged
}

func ActivityTelemetryConfig(ctx workflow.Context, cfg map[string]any) map[string]string {
	merged := MergeTelemetryConfig(ctx, cfg)
	result := make(map[string]string, len(merged))
	for key, value := range merged {
		if value == nil {
			continue
		}
		result[key] = fmt.Sprint(value)
	}
	return result
}

func stringConfigValue(cfg map[string]any, key, fallback string) string {
	if cfg == nil {
		return fallback
	}
	value, ok := cfg[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case string:
		if v != "" {
			return v
		}
	case fmt.Stringer:
		if s := v.String(); s != "" {
			return s
		}
	default:
		s := fmt.Sprint(v)
		if s != "" {
			return s
		}
	}
	return fallback
}

func copyAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
