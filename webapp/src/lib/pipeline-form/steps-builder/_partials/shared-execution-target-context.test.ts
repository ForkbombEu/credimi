// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('../../steps/wallet-action/wallet-action-step-form.svelte', () => ({ default: class {} }));

import { GLOBAL_RUNNER } from '../../steps/wallet-action/wallet-action-step-form.svelte.js';
import {
	countMobileSteps,
	getSharedExecutionTargetContext,
	hasDistinctMobileWallets
} from './shared-execution-target-context.js';

const walletA = { id: 'w-a', name: 'A' } as never;
const walletB = { id: 'w-b', name: 'B' } as never;
const version = { id: 'v1', tag: '1.0' } as never;
const runner = { path: 'org/runner', name: 'R' } as never;

function mobileStep(data: object) {
	return [{ use: 'mobile-automation', id: 's1', continue_on_error: false, with: {} }, data] as const;
}

describe('getSharedExecutionTargetContext', () => {
	it('returns null for zero mobile steps', () => {
		expect(getSharedExecutionTargetContext([])).toBeNull();
	});

	it('returns context when two steps share wallet, version, runner', () => {
		const data = { wallet: walletA, version, runner, action: {} };
		const steps = [mobileStep(data), mobileStep(data)];
		const ctx = getSharedExecutionTargetContext(steps);
		expect(ctx?.wallet.id).toBe('w-a');
		expect(ctx?.mobileIndices).toEqual([0, 1]);
	});

	it('returns null when wallets differ', () => {
		const steps = [
			mobileStep({ wallet: walletA, version, runner: GLOBAL_RUNNER, action: {} }),
			mobileStep({ wallet: walletB, version, runner: GLOBAL_RUNNER, action: {} })
		];
		expect(getSharedExecutionTargetContext(steps)).toBeNull();
	});
});

describe('hasDistinctMobileWallets', () => {
	it('is false for single wallet', () => {
		const steps = [mobileStep({ wallet: walletA, version, runner: GLOBAL_RUNNER, action: {} })];
		expect(hasDistinctMobileWallets(steps)).toBe(false);
	});

	it('is true for two wallets', () => {
		const steps = [
			mobileStep({ wallet: walletA, version, runner: GLOBAL_RUNNER, action: {} }),
			mobileStep({ wallet: walletB, version, runner: GLOBAL_RUNNER, action: {} })
		];
		expect(hasDistinctMobileWallets(steps)).toBe(true);
	});
});

describe('countMobileSteps', () => {
	it('counts only mobile-automation steps', () => {
		const steps = [
			mobileStep({ wallet: walletA, version, runner, action: {} }),
			[{ use: 'debug' }, {}]
		];
		expect(countMobileSteps(steps)).toBe(1);
	});
});
