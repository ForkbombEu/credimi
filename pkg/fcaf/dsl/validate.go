// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package dsl

import (
	"fmt"
	"strings"
)

func ValidateTestDefinition(def TestDefinition) error {
	var errs validationErrors
	errs.require("id", def.ID)
	errs.require("suite.sut", def.Suite.SUT)
	errs.require("suite.role", def.Suite.Role)
	if len(def.NormativeReferences) == 0 {
		errs.add("normative_references is required")
	}
	if len(def.Preconditions) == 0 {
		errs.add("preconditions is required")
	}
	if len(def.Assertions) == 0 {
		errs.add("assertions is required")
	}
	for i, precondition := range def.Preconditions {
		if !validReference(precondition.Ref) {
			errs.add(
				fmt.Sprintf(
					"preconditions[%d].ref must start with pipeline., assertion., or test.",
					i,
				),
			)
		}
	}
	for name, binding := range def.Evidence {
		if strings.TrimSpace(name) == "" {
			errs.add("evidence keys must not be empty")
		}
		if strings.TrimSpace(binding.From) == "" {
			errs.add(fmt.Sprintf("evidence.%s.from is required", name))
		}
	}
	validateAssertions(def.Assertions, &errs)
	if def.Verdict.PassWhen == "" {
		errs.add("verdict.pass_when is required")
	} else if def.Verdict.PassWhen != "all_assertions_pass" {
		errs.add("verdict.pass_when must be all_assertions_pass")
	}
	return errs.err()
}

func ValidatePreconditionDefinition(def PreconditionDefinition) error {
	var errs validationErrors
	errs.require("id", def.ID)
	if !validReference(def.ID) {
		errs.add("id must start with pipeline., assertion., or test.")
	}
	switch def.Kind {
	case "pipeline":
		errs.require("pipeline_id", def.PipelineID)
		seenSteps := map[string]struct{}{}
		for index, stepID := range def.RequiredSteps {
			if strings.TrimSpace(stepID) == "" {
				errs.add(fmt.Sprintf("required_steps[%d] must not be empty", index))
				continue
			}
			if _, exists := seenSteps[stepID]; exists {
				errs.add(fmt.Sprintf("duplicate required step %q", stepID))
			}
			seenSteps[stepID] = struct{}{}
		}
		if len(def.Outputs) == 0 {
			errs.add("outputs is required for pipeline preconditions")
		}
		for name, output := range def.Outputs {
			errs.require("outputs."+name+".path", output.Path)
		}
	case "assertion":
		errs.require("validator", def.Validator)
		if def.Input == nil || strings.TrimSpace(def.Input.From) == "" {
			errs.add("input.from is required for assertion preconditions")
		}
	case "test":
		errs.require("test_id", def.TestID)
	default:
		errs.add("kind must be pipeline, assertion, or test")
	}
	switch def.FailurePolicy {
	case "",
		"fail_dependent_tests",
		"block_dependent_tests",
		"fail_dependent_preconditions_and_tests":
	default:
		errs.add("unsupported failure_policy")
	}
	return errs.err()
}

func validateAssertions(assertions []AssertionDefinition, errs *validationErrors) {
	seen := map[string]struct{}{}
	for i, assertion := range assertions {
		prefix := fmt.Sprintf("assertions[%d]", i)
		errs.require(prefix+".id", assertion.ID)
		errs.require(prefix+".validator", assertion.Validator)
		errs.require(prefix+".input", assertion.Input)
		if assertion.ID != "" {
			if _, ok := seen[assertion.ID]; ok {
				errs.add(fmt.Sprintf("duplicate assertion id %q", assertion.ID))
			}
			seen[assertion.ID] = struct{}{}
		}
	}
}

func validReference(ref string) bool {
	return strings.HasPrefix(ref, "pipeline.") ||
		strings.HasPrefix(ref, "assertion.") ||
		strings.HasPrefix(ref, "test.")
}

type validationErrors []string

func (e *validationErrors) require(field string, value string) {
	if strings.TrimSpace(value) == "" {
		e.add(field + " is required")
	}
}

func (e *validationErrors) add(message string) {
	*e = append(*e, message)
}

func (e validationErrors) err() error {
	if len(e) == 0 {
		return nil
	}
	return fmt.Errorf("invalid fcaf definition: %s", strings.Join(e, "; "))
}
