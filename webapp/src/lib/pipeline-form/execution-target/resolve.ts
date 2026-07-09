// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { isError } from 'effect/Predicate';

import type { EnrichedStep } from '../shared/enriched-step.js';
import { isExecutionTarget, type ExecutionTarget } from './types.js';

export function resolveExecutionTarget(steps: EnrichedStep[]): ExecutionTarget | undefined {
	const mobile = steps.filter(([raw]) => raw.use === 'mobile-automation');
	const last = mobile.at(-1);
	if (!last) return undefined;
	const [, data] = last;
	if (isError(data) || !isExecutionTarget(data)) return undefined;
	const { wallet, version, runner } = data;
	return { wallet, version, runner };
}
