// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import internalpipeline "github.com/forkbombeu/credimi/pkg/internal/pipeline"

// walkMutableStepSpecs visits the actual executable StepSpecs in a workflow.
func walkMutableStepSpecs(
	steps []internalpipeline.StepDefinition,
	visit func(*internalpipeline.StepSpec) error,
) error {
	for i := range steps {
		step := &steps[i]
		if err := visit(&step.StepSpec); err != nil {
			return err
		}
		for _, onError := range step.OnError {
			if onError == nil {
				continue
			}
			if err := visit(&onError.StepSpec); err != nil {
				return err
			}
		}
		for _, onSuccess := range step.OnSuccess {
			if onSuccess == nil {
				continue
			}
			if err := visit(&onSuccess.StepSpec); err != nil {
				return err
			}
		}
	}
	return nil
}
