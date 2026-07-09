// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { FormIntent } from '../steps/types.js';
import type { ExecutionTargetConfig } from './types.js';

import { GLOBAL_RUNNER } from '../shared/mobile-target.js';

export function isExecutionTargetLocked(ctx: {
	intent: FormIntent;
	mobileStepCount: number;
	target: ExecutionTargetConfig | undefined;
}): boolean {
	if (ctx.intent === 'edit' && ctx.mobileStepCount === 1) return false;
	if (!ctx.target) return false;
	return ctx.target.runner === GLOBAL_RUNNER || ctx.target.runner === undefined;
}
