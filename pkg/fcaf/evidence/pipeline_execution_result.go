// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"encoding/json"
	"fmt"
)

type PipelineExecutionResult struct {
	Output        any                   `json:"output,omitempty"`
	WorkflowID    string                `json:"workflowId,omitempty"`
	WorkflowRunID string                `json:"workflowRunId,omitempty"`
	StepFailures  []PipelineStepFailure `json:"stepFailures,omitempty"`
}

type PipelineStepFailure struct {
	StepID  string `json:"step_id"`
	Code    string `json:"code,omitempty"`
	Summary string `json:"summary,omitempty"`
	Message string `json:"message,omitempty"`
}

func (r PipelineExecutionResult) LegacyMap() map[string]any {
	return map[string]any{
		"output":        r.Output,
		"workflowId":    r.WorkflowID,
		"workflowRunId": r.WorkflowRunID,
		"stepFailures":  r.StepFailures,
	}
}

func DecodePipelineExecutionResult(raw any) (PipelineExecutionResult, error) {
	if result, ok := raw.(PipelineExecutionResult); ok {
		return result, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return PipelineExecutionResult{}, fmt.Errorf("marshal pipeline execution result: %w", err)
	}

	var result PipelineExecutionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return PipelineExecutionResult{}, fmt.Errorf("decode pipeline execution result: %w", err)
	}
	return result, nil
}

func DecodePipelineStepFailures(raw any) ([]PipelineStepFailure, error) {
	if raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal pipeline step failures: %w", err)
	}
	var failures []struct {
		StepID  string         `json:"step_id"`
		Code    string         `json:"code"`
		Summary string         `json:"summary"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(data, &failures); err != nil {
		return nil, fmt.Errorf("decode pipeline step failures: %w", err)
	}
	out := make([]PipelineStepFailure, 0, len(failures))
	for _, failure := range failures {
		stepID := failure.StepID
		if stepID == "" {
			stepID, _ = failure.Details["step_id"].(string)
		}
		out = append(out, PipelineStepFailure{
			StepID:  stepID,
			Code:    failure.Code,
			Summary: failure.Summary,
			Message: failure.Message,
		})
	}
	return out, nil
}
