//go:build credimi_extra

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/forkbombeu/credimi/pkg/utils"
)

// MobileEnvironmentGetter resolves an environment value for a mobile activity.
type MobileEnvironmentGetter func(name string, fallback ...any) string

type mobileEnvironmentKey struct{}

// These are the environment variables read by the avdctl and Redroid command
// paths used by mobile activities. Keep this list explicit so unrelated device
// configuration never reaches a child process.
func mobileCommandEnvironmentKeys() []string {
	return []string{
		"AVDCTL_SSH_TARGET",
		"AVDCTL_SSH_PASSWORD",
		"AVDCTL_SSH_ARGS",
		"AVDCTL_SUDO",
		"AVDCTL_SUDO_PASSWORD",
		"AVDCTL_SUDO_BIN",
		"AVDCTL_CORRELATION_ID",
		"DOCKER_HOST",
	}
}

// WithMobileEnvironment scopes a mobile environment getter to activity calls
// that use the returned context.
func WithMobileEnvironment(ctx context.Context, getter MobileEnvironmentGetter) context.Context {
	if getter == nil {
		return ctx
	}

	return context.WithValue(ctx, mobileEnvironmentKey{}, getter)
}

func mobileEnvironmentFromContext(ctx context.Context) MobileEnvironmentGetter {
	if getter, ok := ctx.Value(mobileEnvironmentKey{}).(MobileEnvironmentGetter); ok &&
		getter != nil {
		return getter
	}

	return utils.GetEnvironmentVariable
}

func deviceScopedCommandEnvironment(ctx context.Context, getter MobileEnvironmentGetter) []string {
	if scopedGetter, ok := ctx.Value(mobileEnvironmentKey{}).(MobileEnvironmentGetter); !ok ||
		scopedGetter == nil {
		return nil
	}

	keys := mobileCommandEnvironmentKeys()
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+getter(key, ""))
	}
	return env
}

func mergeEnvironment(base, overrides []string) []string {
	overridesByKey := make(map[string]string, len(overrides))
	for _, entry := range overrides {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			overridesByKey[key] = entry
		}
	}

	merged := make([]string, 0, len(base)+len(overrides))
	seen := make(map[string]struct{}, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			merged = append(merged, entry)
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if override, exists := overridesByKey[key]; exists {
			merged = append(merged, override)
			continue
		}
		merged = append(merged, entry)
	}

	for _, entry := range overrides {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, overridesByKey[key])
	}

	return merged
}

func mobileCommandContext(ctx context.Context) func(context.Context, string, ...string) *exec.Cmd {
	getter := mobileEnvironmentFromContext(ctx)
	return func(commandCtx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(commandCtx, name, args...)
		if scoped := deviceScopedCommandEnvironment(ctx, getter); len(scoped) > 0 {
			cmd.Env = mergeEnvironment(os.Environ(), scoped)
		}
		return cmd
	}
}
