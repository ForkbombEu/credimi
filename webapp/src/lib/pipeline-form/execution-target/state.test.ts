// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('../steps/wallet-action/wallet-action-step-form.svelte', () => ({ default: class {} }));

import * as ExecutionTarget from './state.svelte.js';
import type { Config } from './state.svelte.js';
import { GLOBAL_RUNNER } from '../steps/wallet-action/wallet-action-step-form.svelte.js';
import type { EnrichedStep } from '../steps-builder/types.js';

const wallet = { id: 'w1', name: 'W' } as never;
const version = { id: 'v1', tag: '1' } as never;
const action = { id: 'a1', name: 'A' } as never;

function mobileTuple(overrides?: Partial<{ wallet: typeof wallet }>): EnrichedStep {
	const data = { wallet, version, runner: GLOBAL_RUNNER, action, ...overrides };
	return [{ use: 'mobile-automation', id: 's', continue_on_error: false, with: {} }, data] as unknown as EnrichedStep;
}

afterEach(() => {
	ExecutionTarget.clear();
});

describe('ExecutionTarget.syncFromSteps', () => {
	it('clears when no mobile steps', () => {
		ExecutionTarget.state.current = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.syncFromSteps([[{ use: 'debug' }, {}]]);
		expect(ExecutionTarget.state.current).toBeUndefined();
		expect(ExecutionTarget.state.locked).toBe(false);
	});

	it('sets current and unlocked for one mobile step', () => {
		ExecutionTarget.syncFromSteps([mobileTuple()]);
		expect(ExecutionTarget.state.current?.wallet.id).toBe('w1');
		expect(ExecutionTarget.state.locked).toBe(false);
	});

	it('locks when two mobile steps share target', () => {
		ExecutionTarget.syncFromSteps([mobileTuple(), mobileTuple()]);
		expect(ExecutionTarget.state.locked).toBe(true);
	});

	it('stays unlocked when two mobile steps have different wallets', () => {
		const otherWallet = { id: 'w2', name: 'W2' } as never;
		ExecutionTarget.syncFromSteps([mobileTuple(), mobileTuple({ wallet: otherWallet })]);
		expect(ExecutionTarget.state.locked).toBe(false);
	});

	it('clears secondStepPrefillSnapshot when no mobile steps remain', () => {
		const config: Config = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.state.current = config;
		ExecutionTarget.beginSecondStepAdd();
		expect(ExecutionTarget.state.secondStepPrefillSnapshot).toEqual(config);

		ExecutionTarget.syncFromSteps([[{ use: 'debug' }, {}]]);
		expect(ExecutionTarget.state.secondStepPrefillSnapshot).toBeUndefined();
	});
});

describe('ExecutionTarget.finishSecondStepAdd', () => {
	it('locks when submitted target matches snapshot', () => {
		const config: Config = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.state.current = config;
		ExecutionTarget.beginSecondStepAdd();
		ExecutionTarget.finishSecondStepAdd(config);
		expect(ExecutionTarget.state.locked).toBe(true);
		expect(ExecutionTarget.state.secondStepPrefillSnapshot).toBeUndefined();
	});

	it('stays unlocked when submitted target differs', () => {
		const config: Config = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.state.current = config;
		ExecutionTarget.beginSecondStepAdd();
		const otherWallet = { id: 'w2', name: 'W2' } as never;
		ExecutionTarget.finishSecondStepAdd({ wallet: otherWallet, version, runner: GLOBAL_RUNNER });
		expect(ExecutionTarget.state.locked).toBe(false);
	});

	it('unlocks when submitted target differs after prior lock', () => {
		const config: Config = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.state.current = config;
		ExecutionTarget.state.locked = true;
		ExecutionTarget.beginSecondStepAdd();
		const otherWallet = { id: 'w2', name: 'W2' } as never;
		ExecutionTarget.finishSecondStepAdd({ wallet: otherWallet, version, runner: GLOBAL_RUNNER });
		expect(ExecutionTarget.state.locked).toBe(false);
	});
});
