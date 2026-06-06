// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('./conformance-check-step-form.svelte', () => ({ default: class {} }));
vi.mock('@forkbombeu/temporal-ui', () => ({}));

import {
	resolveWalletActionSelection,
	type WalletActionSelection
} from './conformance-check-step-form.svelte.js';

const action = (id: string) =>
	({ id, name: id, category: 'get-credential-generic' }) as never;

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
