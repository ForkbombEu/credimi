// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import { completeInCardExit, shouldCompleteExitOnDestroy } from './use-card-shell.svelte.js';

describe('completeInCardExit', () => {
	it('clears held then unlocks', () => {
		const order: string[] = [];
		completeInCardExit(
			() => order.push('clear'),
			() => order.push('unlock')
		);
		expect(order).toEqual(['clear', 'unlock']);
	});

	it('is safe to call twice (session unlock is idempotent)', () => {
		const clear = vi.fn();
		const unlock = vi.fn();
		completeInCardExit(clear, unlock);
		completeInCardExit(clear, unlock);
		expect(clear).toHaveBeenCalledTimes(2);
		expect(unlock).toHaveBeenCalledTimes(2);
	});
});

describe('shouldCompleteExitOnDestroy', () => {
	it('unlocks when holding form after edit ended (cancelled exit / remount)', () => {
		expect(shouldCompleteExitOnDestroy(true, false)).toBe(true);
	});

	it('does not unlock while still editing (virtualizer remount of the open card)', () => {
		expect(shouldCompleteExitOnDestroy(true, true)).toBe(false);
	});

	it('does not unlock idle cards', () => {
		expect(shouldCompleteExitOnDestroy(false, false)).toBe(false);
		expect(shouldCompleteExitOnDestroy(false, true)).toBe(false);
	});
});
