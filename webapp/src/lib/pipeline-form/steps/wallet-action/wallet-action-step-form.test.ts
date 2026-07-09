// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('./wallet-action-step-form.svelte', () => ({ default: class {} }));

import { isExecutionTargetLocked } from '../../execution-target/lock.js';
import { EXTERNAL_VERSION, GLOBAL_RUNNER } from '../../execution-target/types.js';
import { WalletActionStepForm } from './wallet-action-step-form.svelte.js';

const executionTarget = {
	wallet: { id: 'w1', name: 'W' } as never,
	version: EXTERNAL_VERSION,
	runner: GLOBAL_RUNNER
};

describe('WalletActionStepForm execution target', () => {
	it('locked add with target starts at select-action', () => {
		const form = new WalletActionStepForm({
			intent: 'add',
			getExecutionTarget: () => executionTarget,
			isExecutionTargetLocked: () => true
		});

		expect(form.state).toBe('select-action');
		expect(form.data.wallet).toEqual(executionTarget.wallet);
		expect(form.data.version).toBe(EXTERNAL_VERSION);
		expect(form.data.runner).toBe(GLOBAL_RUNNER);
		expect(form.data.action).toBeUndefined();
	});

	it('unlocked add with target can return to select-wallet after removeWallet', () => {
		const form = new WalletActionStepForm({
			intent: 'add',
			getExecutionTarget: () => executionTarget,
			isExecutionTargetLocked: () => false
		});

		expect(form.state).toBe('select-action');

		form.removeWallet();

		expect(form.state).toBe('select-wallet');
		expect(form.data.wallet).toBeUndefined();
	});

	it('edit sole step with global runner is not locked', () => {
		const form = new WalletActionStepForm({
			intent: 'edit',
			initial: {
				wallet: { id: 'w1', name: 'W' } as never,
				version: EXTERNAL_VERSION,
				runner: GLOBAL_RUNNER,
				action: { id: 'a1', name: 'Old' } as never
			},
			isExecutionTargetLocked: () =>
				isExecutionTargetLocked({
					intent: 'edit',
					steps: [[{ use: 'mobile-automation' } as never, {}]],
					target: executionTarget
				})
		});

		expect(form.isExecutionTargetLocked()).toBe(false);
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
	});
});
