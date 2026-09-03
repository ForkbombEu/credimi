//go:build credimi_extra

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"os"
	"strings"
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

func TestBuildMobileInputScopesCommandEnvironment(t *testing.T) {
	t.Setenv("AVDCTL_SSH_TARGET", "process-host")
	t.Setenv("PATH", "/process/path")

	const sshPassword = "ssh-password-must-not-leak"
	const sudoPassword = "sudo-password-must-not-leak"
	getter := MobileEnvironmentGetter(func(name string, fallback ...any) string {
		values := map[string]string{
			"AVDCTL_SSH_TARGET":     "device-host",
			"AVDCTL_SSH_PASSWORD":   sshPassword,
			"AVDCTL_SSH_ARGS":       "-p 2222 -o BatchMode=yes",
			"AVDCTL_SUDO":           "false",
			"AVDCTL_SUDO_PASSWORD":  sudoPassword,
			"AVDCTL_SUDO_BIN":       "/usr/bin/sudo",
			"AVDCTL_CORRELATION_ID": "activity-123",
			"DOCKER_HOST":           "ssh://device-host",
			"DEVICE_ONLY_VALUE":     "must-not-be-exported",
		}
		if value, ok := values[name]; ok {
			return value
		}
		return ""
	})
	ctx := WithMobileEnvironment(context.Background(), getter)

	input := buildMobileInput(ctx, nil, nil, nil, true)
	cmd := input.CommandContext(context.Background(), "mobile-command")
	env := environmentMap(cmd.Env)

	requireEnvironmentValue(t, env, "AVDCTL_SSH_TARGET", "device-host")
	requireEnvironmentValue(t, env, "AVDCTL_SSH_PASSWORD", sshPassword)
	requireEnvironmentValue(t, env, "AVDCTL_SSH_ARGS", "-p 2222 -o BatchMode=yes")
	requireEnvironmentValue(t, env, "AVDCTL_SUDO", "false")
	requireEnvironmentValue(t, env, "AVDCTL_SUDO_PASSWORD", sudoPassword)
	requireEnvironmentValue(t, env, "AVDCTL_SUDO_BIN", "/usr/bin/sudo")
	requireEnvironmentValue(t, env, "AVDCTL_CORRELATION_ID", "activity-123")
	requireEnvironmentValue(t, env, "DOCKER_HOST", "ssh://device-host")
	requireEnvironmentValue(t, env, "PATH", "/process/path")
	requireNoDuplicateEnvironmentKeys(t, cmd.Env)
	requireAbsentEnvironmentKey(t, env, "DEVICE_ONLY_VALUE")

	err := cmd.Run()
	require.Error(t, err)
	if strings.Contains(err.Error(), sshPassword) || strings.Contains(err.Error(), sudoPassword) {
		t.Fatal("command error contained a password")
	}
}

func TestBuildMobileInputEmptyScopedValuesOverrideProcessEnvironment(t *testing.T) {
	t.Setenv("AVDCTL_SSH_TARGET", "stale-host")
	t.Setenv("AVDCTL_SUDO", "true")
	getter := MobileEnvironmentGetter(func(name string, fallback ...any) string {
		switch name {
		case "AVDCTL_SSH_TARGET":
			return ""
		case "AVDCTL_SUDO":
			return "false"
		default:
			return ""
		}
	})

	input := buildMobileInput(
		WithMobileEnvironment(context.Background(), getter),
		nil,
		nil,
		nil,
		true,
	)
	env := environmentMap(input.CommandContext(context.Background(), "mobile-command").Env)

	requireEnvironmentValue(t, env, "AVDCTL_SSH_TARGET", "")
	requireEnvironmentValue(t, env, "AVDCTL_SUDO", "false")
}

func TestBuildMobileInputCommandEnvironmentIsConcurrentAndScoped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		sudo   string
	}{
		{name: "device-a", target: "alice@10.0.0.1", sudo: "false"},
		{name: "device-b", target: "bob@10.0.0.2", sudo: "true"},
	}

	var wg sync.WaitGroup
	for _, test := range tests {
		test := test
		wg.Add(1)
		go func() {
			defer wg.Done()
			getter := MobileEnvironmentGetter(func(name string, fallback ...any) string {
				switch name {
				case "AVDCTL_SSH_TARGET":
					return test.target
				case "AVDCTL_SUDO":
					return test.sudo
				default:
					return ""
				}
			})
			input := buildMobileInput(
				WithMobileEnvironment(context.Background(), getter),
				nil,
				nil,
				nil,
				true,
			)
			for range 100 {
				env := environmentMap(
					input.CommandContext(context.Background(), "mobile-command").Env,
				)
				requireEnvironmentValue(t, env, "AVDCTL_SSH_TARGET", test.target)
				requireEnvironmentValue(t, env, "AVDCTL_SUDO", test.sudo)
			}
		}()
	}
	wg.Wait()
}

func TestBuildMobileInputWithoutScopedEnvironmentPreservesProcessEnvironment(t *testing.T) {
	t.Setenv("PATH", "/process/path")

	input := buildMobileInput(context.Background(), nil, nil, nil, true)
	cmd := input.CommandContext(context.Background(), "mobile-command")

	require.Nil(t, cmd.Env)
	require.Equal(t, "/process/path", os.Getenv("PATH"))
}

func environmentMap(entries []string) map[string]string {
	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return values
}

func requireEnvironmentValue(t *testing.T, env map[string]string, key, expected string) {
	t.Helper()
	if got, ok := env[key]; !ok || got != expected {
		t.Fatalf("%s was not set to the expected scoped value", key)
	}
}

func requireNoDuplicateEnvironmentKeys(t *testing.T, entries []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, exists := seen[key]; exists {
			t.Fatalf("environment key %s was duplicated", key)
		}
		seen[key] = struct{}{}
	}
}

func requireAbsentEnvironmentKey(t *testing.T, env map[string]string, key string) {
	t.Helper()
	if _, ok := env[key]; ok {
		t.Fatalf("unexpected environment key %s was exported", key)
	}
}

func TestRunMobileFlowActivityContract(t *testing.T) {
	activity := NewRunMobileFlowActivity()

	var _ workflowengine.ExecutableActivity = activity
	require.Equal(t, "Run a mobile test flow", activity.Name())
}
