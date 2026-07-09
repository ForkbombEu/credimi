// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
import type { FormIntent } from '$pipeline-form/steps/index.js';

import { GLOBAL_RUNNER, type ExecutionTarget } from '$pipeline-form/execution-target/types.js';

export function isExecutionTargetLocked(ctx: {
	intent: FormIntent;
	steps: EnrichedStep[];
	target: ExecutionTarget | undefined;
}): boolean {
	const mobileStepCount = ctx.steps.filter(([raw]) => raw.use === 'mobile-automation').length;
	if (ctx.intent === 'edit' && mobileStepCount === 1) return false;
	if (!ctx.target) return false;
	return ctx.target.runner === GLOBAL_RUNNER || ctx.target.runner === undefined;
}
