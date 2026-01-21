// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

type SetupFunc func(
	ctx workflow.Context,
	steps *[]StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
) error

type CleanupFunc func(
	ctx workflow.Context,
	steps []StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData map[string]any,
	output *map[string]any,
) error

var (
	setupHooks = []SetupFunc{
		MobileAutomationSetupHook,
		ConformanceCheckSetupHook,
	}

	cleanupHooks = []CleanupFunc{
		MobileAutomationCleanupHook,
		ConformanceCheckCleanupHook,
	}
)

func runSetupHooks(
	ctx workflow.Context,
	steps *[]StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
) error {
	for _, hook := range setupHooks {
		if err := hook(ctx, steps, ao, config, runData); err != nil {
			return err
		}
	}
	return nil
}

func runCleanupHooks(
	ctx workflow.Context,
	steps []StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData map[string]any,
	finalOutput *map[string]any,
	logger log.Logger,
	cleanupErrors *[]error,
) {
	for _, hook := range cleanupHooks {
		if err := hook(ctx, steps, ao, config, runData, finalOutput); err != nil {
			logger.Error("cleanup hook error", "error", err)
			*cleanupErrors = append(*cleanupErrors, err)
		}
	}
}
