// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import internalpipeline "github.com/forkbombeu/credimi/pkg/internal/pipeline"

func walkSteps(
	steps []internalpipeline.StepDefinition,
	visit func(internalpipeline.StepSpec) error,
) error {
	for _, step := range steps {
		if err := visit(step.StepSpec); err != nil {
			return err
		}
		for _, onError := range step.OnError {
			if err := visit(onError.StepSpec); err != nil {
				return err
			}
		}
		for _, onSuccess := range step.OnSuccess {
			if err := visit(onSuccess.StepSpec); err != nil {
				return err
			}
		}
	}
	return nil
}
