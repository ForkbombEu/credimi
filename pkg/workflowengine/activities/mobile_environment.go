//go:build credimi_extra

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/utils"
)

// MobileEnvironmentGetter resolves an environment value for a mobile activity.
type MobileEnvironmentGetter func(name string, fallback ...any) string

type mobileEnvironmentKey struct{}

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
