// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
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
