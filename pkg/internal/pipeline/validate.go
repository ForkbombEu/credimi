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
	return ValidateLinkedSteps(def)
}

// unlinkableStepUses lists the step types that cannot be part of a linked block.
var unlinkableStepUses = map[string]struct{}{
	"child-pipeline":    {},
	"debug":             {},
	"fcaf-validation":   {},
	"conformance-check": {},
}

// ValidateLinkedSteps checks the linked_to fields of the main steps: each names the
// immediately previous step, and the steps of a block share continue_on_error, carry no
// hooks, and only the last one sets a retry policy, which drives the block's attempts.
func ValidateLinkedSteps(def *WorkflowDefinition) error {
	if def == nil {
		return nil
	}
	steps := def.Steps
	for i, step := range steps {
		if step.LinkedTo == "" {
			continue
		}
		if i == 0 {
			return fmt.Errorf("step %q: linked_to cannot be set on the first step", step.ID)
		}
		previous := steps[i-1]
		if previous.ID == "" {
			return fmt.Errorf("step %q: linked_to needs the previous step to have an id", step.ID)
		}
		if step.LinkedTo != previous.ID {
			return fmt.Errorf(
				"step %q: linked_to %q must name the previous step %q",
				step.ID,
				step.LinkedTo,
				previous.ID,
			)
		}
	}
	for _, block := range LinkedStepBlocks(steps) {
		first := steps[block[0]]
		last := steps[block[1]]
		for k := block[0]; k <= block[1]; k++ {
			step := steps[k]
			if _, ok := unlinkableStepUses[step.Use]; ok {
				return fmt.Errorf("step %q: %s steps cannot be linked", step.ID, step.Use)
			}
			if len(step.OnError) > 0 || len(step.OnSuccess) > 0 {
				return fmt.Errorf(
					"step %q: linked steps cannot have on_error or on_success steps",
					step.ID,
				)
			}
			if k != block[1] && step.ActivityOptions != nil &&
				step.ActivityOptions.RetryPolicy != (RetryPolicy{}) {
				return fmt.Errorf(
					"step %q: only the last linked step %q can set activity_options.retry_policy",
					step.ID,
					last.ID,
				)
			}
			if step.ContinueOnError != first.ContinueOnError {
				return fmt.Errorf(
					"step %q: linked steps must share the same continue_on_error",
					step.ID,
				)
			}
		}
	}
	return nil
}
