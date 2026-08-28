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
	if len(def.Evidence) == 0 {
		errs.add("evidence is required")
	}
	if len(def.Assertions) == 0 {
		errs.add("assertions is required")
	}
	for name, binding := range def.Evidence {
		if strings.TrimSpace(name) == "" {
			errs.add("evidence keys must not be empty")
		}
		if !validPipelineOutputBinding(binding.From) {
			errs.add(fmt.Sprintf(
				"evidence.%s.from must match pipeline.<id>.outputs.<name>",
				name,
			))
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

func validPipelineOutputBinding(binding string) bool {
	const marker = ".outputs."
	binding = strings.TrimSpace(binding)
	return strings.HasPrefix(binding, "pipeline.") &&
		strings.Contains(binding, marker) &&
		!strings.HasSuffix(binding, marker)
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
