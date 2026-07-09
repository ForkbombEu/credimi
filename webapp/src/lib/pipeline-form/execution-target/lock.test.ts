// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { EnrichedStep } from '../steps-builder/types.js';
import type { FormIntent } from '../steps/types.js';
import type { ExecutionTargetConfig } from './types.js';

import { GLOBAL_RUNNER } from '../shared/mobile-target.js';
import { isExecutionTargetLocked } from './lock.js';

function mobileSteps(count: number): EnrichedStep[] {
	return Array.from({ length: count }, () => [{ use: 'mobile-automation' } as never, {}]);
}

const wallet = { id: 'w1', name: 'Wallet' } as never;
const version = 'installed_from_external_source' as const;
const specificRunner = {
	name: 'Runner',
	path: 'org/runner',
	isOwned: true,
	isPublished: true,
	isOnline: true
};

function target(runner: ExecutionTargetConfig['runner']): ExecutionTargetConfig {
	return { wallet, version, runner };
}

describe('isExecutionTargetLocked', () => {
	it.each([
		{
			intent: 'edit' as FormIntent,
			mobileStepCount: 1,
			runner: GLOBAL_RUNNER,
			expected: false
		},
		{ intent: 'add' as FormIntent, mobileStepCount: 1, runner: GLOBAL_RUNNER, expected: true },
		{
			intent: 'add' as FormIntent,
			mobileStepCount: 1,
			runner: specificRunner,
			expected: false
		},
		{ intent: 'edit' as FormIntent, mobileStepCount: 2, runner: GLOBAL_RUNNER, expected: true },
		{
			intent: 'edit' as FormIntent,
			mobileStepCount: 2,
			runner: specificRunner,
			expected: false
		},
		{ intent: 'add' as FormIntent, mobileStepCount: 0, runner: undefined, expected: false }
	])(
		'intent=$intent mobileStepCount=$mobileStepCount runner=$runner → $expected',
		({ intent, mobileStepCount, runner, expected }) => {
			expect(
				isExecutionTargetLocked({
					intent,
					steps: mobileSteps(mobileStepCount),
					target: runner === undefined ? undefined : target(runner)
				})
			).toBe(expected);
		}
	);
});
