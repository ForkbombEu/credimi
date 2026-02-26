// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...interface{}) {}
func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Warn(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}

func TestRunSetupHooksStopsOnError(t *testing.T) {
	origHooks := setupHooks
	t.Cleanup(func() {
		setupHooks = origHooks
	})

	setupHooks = []SetupFunc{
		func(
			_ workflow.Context,
			_ *[]StepDefinition,
			_ *workflow.ActivityOptions,
			_ map[string]any,
			_ *map[string]any,
		) error {
			return nil
		},
		func(
			_ workflow.Context,
			_ *[]StepDefinition,
			_ *workflow.ActivityOptions,
			_ map[string]any,
			_ *map[string]any,
		) error {
			return errors.New("boom")
		},
	}

	var ctx workflow.Context
	var steps []StepDefinition
	var ao workflow.ActivityOptions
	config := map[string]any{}
	runData := map[string]any{}

	err := runSetupHooks(ctx, &steps, &ao, config, &runData)
	require.Error(t, err)
}

func TestRunCleanupHooksCollectsErrors(t *testing.T) {
	origHooks := cleanupHooks
	t.Cleanup(func() {
		cleanupHooks = origHooks
	})

	cleanupHooks = []CleanupFunc{
		func(
			_ workflow.Context,
			_ []StepDefinition,
			_ *workflow.ActivityOptions,
			_ map[string]any,
			_ map[string]any,
			_ *map[string]any,
		) error {
			return errors.New("cleanup")
		},
		func(
			_ workflow.Context,
			_ []StepDefinition,
			_ *workflow.ActivityOptions,
			_ map[string]any,
			_ map[string]any,
			_ *map[string]any,
		) error {
			return nil
		},
	}

	var ctx workflow.Context
	var steps []StepDefinition
	var ao workflow.ActivityOptions
	config := map[string]any{}
	runData := map[string]any{}
	finalOutput := map[string]any{}
	var cleanupErrors []error

	runCleanupHooks(ctx, steps, &ao, config, runData, &finalOutput, log.Logger(noopLogger{}), &cleanupErrors)

	require.Len(t, cleanupErrors, 1)
}
