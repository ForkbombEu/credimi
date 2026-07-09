// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { PipelineStepByType } from '../../pipeline/types.js';
import type { EnrichedStep } from '../shared/enriched-step.js';
import type { WalletActionStepData } from '../steps/wallet-action/types.js';

import { EXTERNAL_VERSION, GLOBAL_RUNNER } from './types.js';
import { resolveExecutionTarget } from './resolve.js';

function mobileStep(data: WalletActionStepData): EnrichedStep {
	return [
		{
			use: 'mobile-automation',
			with: { action_id: 'a1', version_id: 'v1' }
		} as PipelineStepByType<'mobile-automation'>,
		data as never
	];
}

const walletA = { id: 'w-a', name: 'Wallet A' } as never;
const walletB = { id: 'w-b', name: 'Wallet B' } as never;
const action = {
	id: 'a1',
	name: 'Action',
	canonified_name: 'action',
	wallet: 'w-a',
	code: ''
} as never;

describe('resolveExecutionTarget', () => {
	it('returns undefined for empty steps', () => {
		expect(resolveExecutionTarget([])).toBeUndefined();
	});

	it('returns config from one valid mobile-automation step', () => {
		const data: WalletActionStepData = {
			wallet: walletA,
			version: EXTERNAL_VERSION,
			runner: GLOBAL_RUNNER,
			action
		};
		const result = resolveExecutionTarget([mobileStep(data)]);
		expect(result).toEqual({
			wallet: walletA,
			version: EXTERNAL_VERSION,
			runner: GLOBAL_RUNNER
		});
	});

	it('uses the last mobile-automation step when multiple exist', () => {
		const first: WalletActionStepData = {
			wallet: walletA,
			version: EXTERNAL_VERSION,
			runner: GLOBAL_RUNNER,
			action
		};
		const last: WalletActionStepData = {
			wallet: walletB,
			version: EXTERNAL_VERSION,
			runner: {
				name: 'Runner',
				path: 'org/runner',
				isOwned: true,
				isPublished: true,
				isOnline: true
			},
			action
		};
		const steps: EnrichedStep[] = [
			mobileStep(first),
			[{ use: 'debug' }, {} as never],
			mobileStep(last)
		];
		const result = resolveExecutionTarget(steps);
		expect(result?.wallet).toEqual(walletB);
		expect(result?.runner).toEqual(last.runner);
	});

	it('returns undefined when the latest mobile step is error-enriched', () => {
		const valid: WalletActionStepData = {
			wallet: walletA,
			version: EXTERNAL_VERSION,
			runner: GLOBAL_RUNNER,
			action
		};
		const steps: EnrichedStep[] = [
			mobileStep(valid),
			[
				{
					use: 'mobile-automation',
					with: { action_id: 'a1', version_id: 'v1' }
				} as PipelineStepByType<'mobile-automation'>,
				new Error('enrich failed')
			]
		];
		expect(resolveExecutionTarget(steps)).toBeUndefined();
	});
});
