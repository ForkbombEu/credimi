// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { isError } from 'effect/Predicate';

import type { EnrichedStep } from '../shared/enriched-step.js';
import type { ExecutionTargetConfig } from './types.js';

import { isMobileTargetFields } from '../shared/guards.js';

export function resolveExecutionTarget(steps: EnrichedStep[]): ExecutionTargetConfig | undefined {
	const mobile = steps.filter(([raw]) => raw.use === 'mobile-automation');
	const last = mobile.at(-1);
	if (!last) return undefined;
	const [, data] = last;
	if (isError(data) || !isMobileTargetFields(data)) return undefined;
	const { wallet, version, runner } = data;
	return { wallet, version, runner };
}
