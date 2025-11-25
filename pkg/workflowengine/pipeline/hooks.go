// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/workflow"
)

type SetupFunc func(ctx workflow.Context, steps *[]StepDefinition, input workflowengine.WorkflowInput) error

type CleanupFunc func(ctx workflow.Context, steps []StepDefinition, input workflowengine.WorkflowInput) error

var (
	setupHooks = []SetupFunc{
		MobileAutomationSetupHook,
		ConformanceCheckHook,
	}

	cleanupHooks = []CleanupFunc{
		MobileAutomationCleanupHook,
	}
)
