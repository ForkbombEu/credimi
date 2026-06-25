// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import type { WalletActionStepData } from '../../steps/wallet-action/wallet-action-step-form.svelte.js';
import {
	EXTERNAL_VERSION,
	GLOBAL_RUNNER
} from '../../steps/wallet-action/wallet-action-step-form.svelte.js';

import { Enrich404Error, type EnrichedStep } from '../types';

export type SharedExecutionTargetContext = {
	wallet: HubItem;
	version: WalletActionStepData['version'];
	runner: WalletActionStepData['runner'];
	mobileIndices: number[];
};

function isWalletActionData(value: unknown): value is WalletActionStepData {
	if (!value || typeof value !== 'object') return false;
	const o = value as Record<string, unknown>;
	return typeof o.wallet === 'object' && o.wallet !== null && 'version' in o && 'runner' in o;
}

export function versionKey(version: WalletActionStepData['version']): string {
	return version === EXTERNAL_VERSION ? EXTERNAL_VERSION : version.id;
}

export function runnerKey(runner: WalletActionStepData['runner']): string {
	return runner === GLOBAL_RUNNER ? GLOBAL_RUNNER : runner.path;
}

export function targetsEqual(
	a: Pick<WalletActionStepData, 'wallet' | 'version' | 'runner'>,
	b: Pick<WalletActionStepData, 'wallet' | 'version' | 'runner'>
): boolean {
	return (
		a.wallet.id === b.wallet.id &&
		versionKey(a.version) === versionKey(b.version) &&
		runnerKey(a.runner) === runnerKey(b.runner)
	);
}

export function countMobileSteps(steps: EnrichedStep[]): number {
	return steps.filter(([raw]) => raw.use === 'mobile-automation').length;
}

export function mobileWalletIds(steps: EnrichedStep[]): Set<string> {
	const ids = new Set<string>();
	for (const [raw, data] of steps) {
		if (raw.use !== 'mobile-automation') continue;
		if (data instanceof Enrich404Error || data instanceof Error) continue;
		if (!isWalletActionData(data)) continue;
		ids.add(data.wallet.id);
	}
	return ids;
}

export function hasDistinctMobileWallets(steps: EnrichedStep[]): boolean {
	return mobileWalletIds(steps).size > 1;
}

export function getSharedExecutionTargetContext(
	steps: EnrichedStep[]
): SharedExecutionTargetContext | null {
	const mobileIndices: number[] = [];
	for (let i = 0; i < steps.length; i++) {
		if (steps[i]![0].use === 'mobile-automation') mobileIndices.push(i);
	}
	if (mobileIndices.length === 0) return null;

	let wallet: HubItem | undefined;
	let version: WalletActionStepData['version'] | undefined;
	let runner: WalletActionStepData['runner'] | undefined;

	for (const i of mobileIndices) {
		const [, data] = steps[i]!;
		if (data instanceof Enrich404Error || data instanceof Error) return null;
		if (!isWalletActionData(data)) return null;

		if (!wallet) {
			wallet = data.wallet;
			version = data.version;
			runner = data.runner;
		} else if (!targetsEqual({ wallet, version: version!, runner: runner! }, data)) {
			return null;
		}
	}

	if (!wallet || version === undefined || runner === undefined) return null;
	return { wallet, version, runner, mobileIndices };
}
