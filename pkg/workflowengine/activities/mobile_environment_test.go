//go:build credimi_extra

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"sync"
	"testing"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestBuildMobileInputUsesDefaultEnvironmentGetter(t *testing.T) {
	t.Setenv("CREDIMI_MOBILE_ENV_TEST", "default-value")

	input := buildMobileInput(context.Background(), nil, nil, nil, false)

	require.Equal(
		t,
		utils.GetEnvironmentVariable("CREDIMI_MOBILE_ENV_TEST"),
		input.GetEnv("CREDIMI_MOBILE_ENV_TEST"),
	)
}

func TestBuildMobileInputUsesScopedEnvironmentGetter(t *testing.T) {
	getter := MobileEnvironmentGetter(func(name string, fallback ...any) string {
		if name == "BASE_NAME" {
			return "device-a"
		}
		return ""
	})
	ctx := WithMobileEnvironment(context.Background(), getter)

	input := buildMobileInput(ctx, nil, nil, nil, false)

	require.Equal(t, "device-a", input.GetEnv("BASE_NAME"))
}

func TestWithMobileEnvironmentNilGetterFallsBackToDefault(t *testing.T) {
	t.Setenv("CREDIMI_MOBILE_ENV_TEST", "default-value")
	ctx := WithMobileEnvironment(context.Background(), nil)

	input := buildMobileInput(ctx, nil, nil, nil, false)

	require.Equal(t, "default-value", input.GetEnv("CREDIMI_MOBILE_ENV_TEST"))
}

func TestMobileEnvironmentIsScopedToItsContext(t *testing.T) {
	t.Setenv("CREDIMI_MOBILE_ENV_TEST", "default-value")
	ctxA := WithMobileEnvironment(context.Background(), func(name string, fallback ...any) string {
		if name == "BASE_NAME" {
			return "device-a"
		}
		return ""
	})
	ctxB := context.Background()

	inputA := buildMobileInput(ctxA, nil, nil, nil, false)
	inputB := buildMobileInput(ctxB, nil, nil, nil, false)

	require.Equal(t, "device-a", inputA.GetEnv("BASE_NAME"))
	require.Equal(t, "default-value", inputB.GetEnv("CREDIMI_MOBILE_ENV_TEST"))
}

func TestMobileEnvironmentParallelIsolation(t *testing.T) {
	getterA := MobileEnvironmentGetter(func(name string, fallback ...any) string {
		if name == "BASE_NAME" {
			return "device-a"
		}
		return ""
	})
	getterB := MobileEnvironmentGetter(func(name string, fallback ...any) string {
		if name == "BASE_NAME" {
			return "device-b"
		}
		return ""
	})

	type result struct {
		name  string
		value string
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, test := range []struct {
		name   string
		getter MobileEnvironmentGetter
	}{
		{name: "device-a", getter: getterA},
		{name: "device-b", getter: getterB},
	} {
		test := test
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := WithMobileEnvironment(context.Background(), test.getter)
			input := buildMobileInput(ctx, nil, nil, nil, false)
			for range 100 {
				results <- result{name: test.name, value: input.GetEnv("BASE_NAME")}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		require.Equal(t, result.name, result.value)
	}
}

func TestRunMobileFlowActivityContract(t *testing.T) {
	activity := NewRunMobileFlowActivity()

	var _ workflowengine.ExecutableActivity = activity
	require.Equal(t, "Run a mobile test flow", activity.Name())
}
