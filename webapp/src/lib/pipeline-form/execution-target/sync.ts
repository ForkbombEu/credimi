// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import type { EnrichedStep } from '../steps-builder/types';
import type {
	SelectedVersion,
	WalletActionStepData
} from '../steps/wallet-action/wallet-action-step-form.svelte.js';

import { countMobileSteps } from './rules';
import { clear, getConfigClone, setCurrentConfig } from './state.svelte';

const GLOBAL_RUNNER = 'global' as const;

export function getAddFormPrefill():
	| Pick<WalletActionStepData, 'wallet' | 'version' | 'runner'>
	| undefined {
	return getConfigClone();
}

export function getCurrentWallet(): HubItem | undefined {
	return getConfigClone()?.wallet;
}

export function establishFromStep(data: WalletActionStepData) {
	setCurrentConfig({
		wallet: data.wallet,
		version: data.version,
		runner: data.runner
	});
}

export function shouldDefaultRunnerToGlobal(): boolean {
	const runner = getConfigClone()?.runner;
	return runner === GLOBAL_RUNNER || runner === undefined;
}

export function shouldOfferChooseRunnerLater(lockExecutionTarget: boolean): boolean {
	if (lockExecutionTarget) return false;
	return getConfigClone()?.runner === undefined;
}

export function syncVersionIfSameWallet(walletId: string, version: SelectedVersion) {
	const current = getConfigClone();
	if (current?.wallet.id === walletId) {
		setCurrentConfig({ ...current, version });
	}
}

function walletActionDataFromStep(tuple: EnrichedStep): WalletActionStepData | undefined {
	const [raw, data] = tuple;
	if (raw.use !== 'mobile-automation') return undefined;
	if (data instanceof Error) return undefined;
	if (!data || typeof data !== 'object') return undefined;
	return data as unknown as WalletActionStepData;
}

export function syncAfterStepsChange(steps: EnrichedStep[]) {
	const count = countMobileSteps(steps);
	if (count === 0) {
		clear();
		return;
	}
	if (count === 1) {
		const tuple = steps.find(([raw]) => raw.use === 'mobile-automation');
		if (!tuple) return;
		const data = walletActionDataFromStep(tuple);
		if (!data) return;
		setCurrentConfig({
			wallet: data.wallet,
			version: data.version,
			runner: data.runner
		});
	}
}
