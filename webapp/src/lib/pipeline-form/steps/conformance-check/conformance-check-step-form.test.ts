// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('./conformance-check-step-form.svelte', () => ({ default: class {} }));
// Avoids uuid import failure when loading conformance-check-step-form module graph in Vitest.
vi.mock('@forkbombeu/temporal-ui', () => ({}));
vi.mock('runed', () => ({
	resource: () => ({
		loading: false,
		error: undefined,
		current: null
	})
}));
const i18nMocks = vi.hoisted(() => ({
	Pipeline_form_choose_wallet_before_openid4vci_wallet_check: ({
		category
	}: {
		category: string;
	}) => `choose-wallet:${category}`,
	Pipeline_form_wallet_missing_action_category: ({
		wallet,
		category
	}: {
		wallet: string;
		category: string;
	}) => `missing-action:${wallet}:${category}`
}));

vi.mock('@/i18n', () => ({
	m: new Proxy(i18nMocks, {
		get(target, prop) {
			if (typeof prop === 'string' && prop in target) {
				return target[prop as keyof typeof target];
			}

			return () => String(prop);
		}
	})
}));

import {
	ConformanceCheckStepForm,
	getWalletTestBlockReason,
	resolveWalletActionSelection,
	type WalletActionSelection
} from './conformance-check-step-form.svelte.js';

const action = (id: string) => ({ id, name: id, category: 'get-credential-generic' }) as never;

const wallet = { name: 'TestWallet' } as never;

describe('getWalletTestBlockReason', () => {
	it.each([
		{
			name: 'no wallet',
			wallet: undefined,
			walletActions: { loading: false, error: undefined, current: [] },
			expected: 'choose-wallet:get-credential-generic'
		},
		{
			name: 'loading',
			wallet,
			walletActions: { loading: true, error: undefined, current: [] },
			expected: null
		},
		{
			name: 'error',
			wallet,
			walletActions: {
				loading: false,
				error: new Error('wallet actions failed'),
				current: []
			},
			expected: 'wallet actions failed'
		},
		{
			name: 'current: []',
			wallet,
			walletActions: { loading: false, error: undefined, current: [] },
			expected: 'missing-action:TestWallet:get-credential-generic'
		},
		{
			name: 'current: [one action]',
			wallet,
			walletActions: {
				loading: false,
				error: undefined,
				current: [action('a1')]
			},
			expected: null
		},
		{
			name: 'current: [two actions]',
			wallet,
			walletActions: {
				loading: false,
				error: undefined,
				current: [action('a1'), action('a2')]
			},
			expected: null
		}
	])('$name', ({ wallet: testWallet, walletActions, expected }) => {
		expect(getWalletTestBlockReason(testWallet, walletActions)).toBe(expected);
	});
});

describe('ConformanceCheckStepForm discard cascade', () => {
	it('discardSuite clears test and action_id', () => {
		const form = new ConformanceCheckStepForm({
			initial: {
				standard: { uid: 's', name: 'S', versions: [] } as never,
				version: { uid: 'v', name: 'V', suites: [] } as never,
				suite: { uid: 'su', name: 'Su', paths: [] } as never,
				test: 'openid4vci_wallet/foo',
				action_id: 'owners/w/actions/a1'
			}
		});

		form.discardSuite();

		expect(form.data.suite).toBeUndefined();
		expect(form.data.test).toBeUndefined();
		expect(form.data.action_id).toBeUndefined();
	});
});

describe('ConformanceCheckStepForm edit intent', () => {
	it('selectWalletAction does not commit until commit()', () => {
		const onSubmit = vi.fn();
		const form = new ConformanceCheckStepForm({
			intent: 'edit',
			initial: {
				standard: { uid: 's', name: 'S', versions: [] } as never,
				version: { uid: 'v', name: 'V', suites: [] } as never,
				suite: { uid: 'su', name: 'Su', paths: [] } as never,
				test: 'openid4vci_wallet/foo',
				action_id: 'old/action/path'
			}
		});
		form.onSubmit(onSubmit);
		const newAction = { __canonified_path__: 'new/action/path' } as never;
		form.selectWalletAction(newAction);
		expect(onSubmit).not.toHaveBeenCalled();
		expect(form.data.action_id).toBe('new/action/path');
	});
});

describe('resolveWalletActionSelection', () => {
	it('returns blocked when no actions', () => {
		expect(resolveWalletActionSelection([])).toEqual({ kind: 'blocked' });
	});

	it('returns auto with single action', () => {
		const a = action('a1');
		expect(resolveWalletActionSelection([a])).toEqual({
			kind: 'auto',
			action: a
		} satisfies WalletActionSelection);
	});

	it('returns picker when multiple actions', () => {
		expect(resolveWalletActionSelection([action('a1'), action('a2')])).toEqual({
			kind: 'picker'
		});
	});
});
