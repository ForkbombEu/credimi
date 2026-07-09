// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import type { PipelineStepByType } from '../../pipeline/types.js';
import type { EnrichedStep } from '../shared/enriched-step.js';
import type { WalletActionStepData } from '../steps/wallet-action/types.js';

import { EXTERNAL_VERSION, GLOBAL_RUNNER } from '../shared/mobile-target.js';

type MobileAutomationStep = PipelineStepByType<'mobile-automation'>;

vi.mock('../steps/wallet-action/index.js', () => ({
	walletActionStepConfig: {
		serialize: (data: WalletActionStepData) => ({
			action_id: 'org/w-a/action',
			version_id:
				data.version === EXTERNAL_VERSION
					? EXTERNAL_VERSION
					: (data.version as unknown as { __canonified_path__: string })
							.__canonified_path__
		})
	}
}));

import { syncMobileStepVersionsIfSameWallet } from './sync-mobile-versions.js';

const walletA = { id: 'w-a', name: 'Wallet A' } as never;
const walletB = { id: 'w-b', name: 'Wallet B' } as never;
const action = {
	id: 'a1',
	name: 'Action',
	canonified_name: 'action',
	wallet: 'w-a',
	code: ''
} as never;

function mobileStep(
	data: WalletActionStepData,
	withPayload: PipelineStepByType<'mobile-automation'>['with']
): EnrichedStep {
	return [
		{ use: 'mobile-automation', with: withPayload } as PipelineStepByType<'mobile-automation'>,
		data as never
	];
}

function stepData(
	wallet: typeof walletA,
	version: WalletActionStepData['version']
): WalletActionStepData {
	return { wallet, version, runner: GLOBAL_RUNNER, action };
}

describe('syncMobileStepVersionsIfSameWallet', () => {
	it('updates all mobile-automation steps with the same wallet and re-serializes with', () => {
		const oldVersion = EXTERNAL_VERSION;
		const newVersion = {
			id: 'v2',
			tag: '2.0',
			__canonified_path__: 'org/w-a/v2'
		} as unknown as WalletActionStepData['version'];

		const step1 = mobileStep(stepData(walletA, oldVersion), {
			action_id: 'org/w-a/action',
			version_id: EXTERNAL_VERSION
		});
		const step2 = mobileStep(stepData(walletA, oldVersion), {
			action_id: 'org/w-a/action',
			version_id: EXTERNAL_VERSION
		});
		const otherWallet = mobileStep(stepData(walletB, oldVersion), {
			action_id: 'org/w-b/action',
			version_id: EXTERNAL_VERSION
		});
		const debugStep: EnrichedStep = [{ use: 'debug' }, {} as never];

		const original = [step1, step2, otherWallet, debugStep];
		const result = syncMobileStepVersionsIfSameWallet(original, 'w-a', newVersion);

		expect(result).not.toBe(original);
		expect(result[0][1]).toMatchObject({ version: newVersion });
		expect(result[1][1]).toMatchObject({ version: newVersion });
		expect((result[0][0] as MobileAutomationStep).with).toEqual({
			action_id: 'org/w-a/action',
			version_id: 'org/w-a/v2'
		});
		expect((result[1][0] as MobileAutomationStep).with).toEqual({
			action_id: 'org/w-a/action',
			version_id: 'org/w-a/v2'
		});

		expect(result[2][1]).toMatchObject({ version: oldVersion });
		expect((result[2][0] as MobileAutomationStep).with).toEqual(
			(otherWallet[0] as MobileAutomationStep).with
		);
		expect(result[3]).toBe(debugStep);
	});
});
