// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('./wallet-action-step-form.svelte', () => ({ default: class {} }));

import { ExecutionTarget } from '$lib/pipeline-form/execution-target';

import {
	EXTERNAL_VERSION,
	GLOBAL_RUNNER,
	WalletActionStepForm
} from './wallet-action-step-form.svelte.js';

describe('WalletActionStepForm target lock', () => {
	afterEach(() => ExecutionTarget.clear());

	it('isTargetLocked is false for single step edit with global runner', () => {
		ExecutionTarget.state.locked = false;
		const form = new WalletActionStepForm({
			intent: 'edit',
			existingMobileCount: 1,
			initial: {
				wallet: { id: 'w1', name: 'W' } as never,
				version: EXTERNAL_VERSION,
				runner: GLOBAL_RUNNER,
				action: { id: 'a1' } as never
			}
		});
		expect(form.isTargetLocked).toBe(false);
	});

	it('isTargetLocked is true when ExecutionTarget locked', () => {
		ExecutionTarget.state.locked = true;
		const form = new WalletActionStepForm({
			intent: 'edit',
			existingMobileCount: 2,
			initial: {
				wallet: { id: 'w1', name: 'W' } as never,
				version: EXTERNAL_VERSION,
				runner: GLOBAL_RUNNER,
				action: { id: 'a1' } as never
			}
		});
		expect(form.isTargetLocked).toBe(true);
	});

	it('isTargetLocked is true when adding 3rd step', () => {
		ExecutionTarget.state.locked = false;
		const form = new WalletActionStepForm({ intent: 'add', existingMobileCount: 2 });
		expect(form.isTargetLocked).toBe(true);
	});
});

describe('WalletActionStepForm multi-wallet global runner', () => {
	it('canSave is false when distinct wallets and runner is global', () => {
		const form = new WalletActionStepForm({
			intent: 'add',
			existingMobileCount: 1,
			otherMobileWalletIds: ['w-other']
		});
		form.data = {
			wallet: { id: 'w-new', name: 'N' } as never,
			version: EXTERNAL_VERSION,
			runner: GLOBAL_RUNNER,
			action: { id: 'a1', name: 'A' } as never
		};
		expect(form.canSave()).toBe(false);
	});
});

describe('WalletActionStepForm edit intent', () => {
	it('selectAction does not commit until commit()', () => {
		const onSubmit = vi.fn();
		const form = new WalletActionStepForm({
			intent: 'edit',
			initial: {
				wallet: { id: 'w1', name: 'W' } as never,
				version: EXTERNAL_VERSION,
				runner: GLOBAL_RUNNER,
				action: { id: 'a1', name: 'Old' } as never
			}
		});
		form.onSubmit(onSubmit);
		const newAction = { id: 'a2', name: 'New' } as never;
		form.selectAction(newAction);
		expect(onSubmit).not.toHaveBeenCalled();
		form.commit({
			wallet: { id: 'w1', name: 'W' } as never,
			version: EXTERNAL_VERSION,
			runner: GLOBAL_RUNNER,
			action: newAction
		});
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0].action.name).toBe('New');
	});
});
