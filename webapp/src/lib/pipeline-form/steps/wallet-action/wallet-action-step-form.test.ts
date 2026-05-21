// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('./wallet-action-step-form.svelte', () => ({ default: class {} }));

import {
	EXTERNAL_VERSION,
	GLOBAL_RUNNER,
	WalletActionStepForm
} from './wallet-action-step-form.svelte.js';

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
