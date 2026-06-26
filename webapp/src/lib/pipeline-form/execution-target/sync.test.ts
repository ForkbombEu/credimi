// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it } from 'vitest';

import type { EnrichedStep } from '../steps-builder/types';
import type { WalletActionStepData } from '../steps/wallet-action/wallet-action-step-form.svelte.js';

import { clear } from './state.svelte';
import {
	establishFromStep,
	getAddFormPrefill,
	getCurrentWallet,
	shouldDefaultRunnerToGlobal,
	shouldOfferChooseRunnerLater,
	syncAfterStepsChange
} from './sync';

const GLOBAL_RUNNER = 'global' as const;
const EXTERNAL_VERSION = 'installed_from_external_source' as const;

const wallet = { id: 'w1', name: 'Wallet' } as WalletActionStepData['wallet'];
const version = EXTERNAL_VERSION;
const runner = GLOBAL_RUNNER;
const action = { id: 'a1', name: 'Act' } as WalletActionStepData['action'];

function mobileTuple(data: WalletActionStepData): EnrichedStep {
	return [
		{ use: 'mobile-automation', id: 's1', continue_on_error: false, with: {} },
		data as never
	];
}

describe('ExecutionTarget sync', () => {
	beforeEach(() => clear());

	it('getAddFormPrefill returns undefined when no target', () => {
		expect(getAddFormPrefill()).toBeUndefined();
	});

	it('establishFromStep sets prefill', () => {
		establishFromStep({ wallet, version, runner, action });
		expect(getAddFormPrefill()).toEqual({ wallet, version, runner });
	});

	it('syncAfterStepsChange clears when no mobile steps', () => {
		establishFromStep({ wallet, version, runner, action });
		syncAfterStepsChange([]);
		expect(getAddFormPrefill()).toBeUndefined();
	});

	it('syncAfterStepsChange updates from sole mobile step', () => {
		const data: WalletActionStepData = {
			wallet,
			version,
			runner: { name: 'R', path: 'org/r', isOwned: true, isPublished: true, isOnline: true },
			action
		};
		syncAfterStepsChange([mobileTuple(data)]);
		expect(getAddFormPrefill()?.runner).toEqual(data.runner);
	});

	it('getCurrentWallet returns wallet from target', () => {
		establishFromStep({ wallet, version, runner, action });
		expect(getCurrentWallet()).toEqual(wallet);
	});

	it('shouldDefaultRunnerToGlobal when runner is global or undefined', () => {
		establishFromStep({ wallet, version, runner: GLOBAL_RUNNER, action });
		expect(shouldDefaultRunnerToGlobal()).toBe(true);
	});

	it('shouldOfferChooseRunnerLater false when locked', () => {
		expect(shouldOfferChooseRunnerLater(true)).toBe(false);
	});
});
