// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import "fmt"

// ValidateStepIDs checks that main step IDs identify a single step each, so that a
// step_id recorded in the run's history maps to exactly one main step. Hook and finally
// step IDs may repeat among themselves but must not reuse a main step ID. Empty IDs are
// ignored.
func ValidateStepIDs(def *WorkflowDefinition) error {
	if def == nil {
		return nil
	}
	mainIDs := make(map[string]struct{}, len(def.Steps))
	for _, step := range def.Steps {
		if step.ID == "" {
			continue
		}
		if _, exists := mainIDs[step.ID]; exists {
			return fmt.Errorf("duplicate step id %q", step.ID)
		}
		mainIDs[step.ID] = struct{}{}
	}

	hookIDs := make([]string, 0)
	for _, step := range def.Steps {
		for _, hook := range step.OnError {
			if hook != nil {
				hookIDs = append(hookIDs, hook.ID)
			}
		}
		for _, hook := range step.OnSuccess {
			if hook != nil {
				hookIDs = append(hookIDs, hook.ID)
			}
		}
	}
	for _, step := range def.Finally.AllSteps() {
		hookIDs = append(hookIDs, step.ID)
	}
	for _, id := range hookIDs {
		if id == "" {
			continue
		}
		if _, exists := mainIDs[id]; exists {
			return fmt.Errorf("hook step id %q collides with a main step id", id)
		}
	}
	return nil
}
