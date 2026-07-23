// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ExecutionTarget } from '$pipeline-form/execution-target/types.js';
import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
import type { FormIntent } from '$pipeline-form/steps/types.js';

import { GLOBAL_DEVICE } from '$pipeline-form/execution-target/types.js';
import { describe, expect, it } from 'vitest';

import { isExecutionTargetLocked } from './execution-target-lock.js';

//

function mobileSteps(count: number): EnrichedStep[] {
	return Array.from({ length: count }, () => [{ use: 'mobile-automation' } as never, {}]);
}

const wallet = { id: 'w1', name: 'Wallet' } as never;
const version = 'installed_from_external_source' as const;
const specificDevice = {
	name: 'Device',
	path: 'org/device',
	isOwned: true,
	isPublished: true,
	isOnline: true
};

function target(device: ExecutionTarget['device']): ExecutionTarget {
	return { wallet, version, device };
}

describe('isExecutionTargetLocked', () => {
	it.each([
		{
			intent: 'edit' as FormIntent,
			mobileStepCount: 1,
			device: GLOBAL_DEVICE,
			expected: false
		},
		{ intent: 'add' as FormIntent, mobileStepCount: 1, device: GLOBAL_DEVICE, expected: true },
		{
			intent: 'add' as FormIntent,
			mobileStepCount: 1,
			device: specificDevice,
			expected: false
		},
		{ intent: 'edit' as FormIntent, mobileStepCount: 2, device: GLOBAL_DEVICE, expected: true },
		{
			intent: 'edit' as FormIntent,
			mobileStepCount: 2,
			device: specificDevice,
			expected: false
		},
		{ intent: 'add' as FormIntent, mobileStepCount: 0, device: undefined, expected: false }
	])(
		'intent=$intent mobileStepCount=$mobileStepCount device=$device → $expected',
		({ intent, mobileStepCount, device, expected }) => {
			expect(
				isExecutionTargetLocked({
					intent,
					steps: mobileSteps(mobileStepCount),
					target: device === undefined ? undefined : target(device)
				})
			).toBe(expected);
		}
	);
});
