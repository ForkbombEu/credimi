// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import { composeAttachments, endPadAttach } from './scrollport-attachments.js';

describe('composeAttachments', () => {
	it('returns undefined when all parts are undefined', () => {
		expect(composeAttachments(undefined, undefined)).toBeUndefined();
	});

	it('returns the sole active attachment unchanged', () => {
		const sole = vi.fn(() => undefined);
		expect(composeAttachments(undefined, sole)).toBe(sole);
	});

	it('runs each attachment and composes cleanups', () => {
		const cleanA = vi.fn();
		const cleanB = vi.fn();
		const a = vi.fn(() => cleanA);
		const b = vi.fn(() => cleanB);
		const composed = composeAttachments(a, b);
		expect(composed).toBeTypeOf('function');
		const node = {} as Element;
		const cleanup = composed!(node);
		expect(a).toHaveBeenCalledWith(node);
		expect(b).toHaveBeenCalledWith(node);
		cleanup?.();
		expect(cleanA).toHaveBeenCalledOnce();
		expect(cleanB).toHaveBeenCalledOnce();
	});
});

describe('endPadAttach', () => {
	it('reports 30% of clientHeight and resets on cleanup', () => {
		const setPx = vi.fn();
		const el = {
			clientHeight: 1000,
			observe: undefined as unknown
		} as unknown as HTMLElement;

		const observed: Element[] = [];
		const disconnect = vi.fn();
		vi.stubGlobal(
			'ResizeObserver',
			class {
				observe(target: Element) {
					observed.push(target);
				}
				disconnect = disconnect;
			}
		);

		const cleanup = endPadAttach(setPx)(el);
		expect(setPx).toHaveBeenCalledWith(300);
		expect(observed).toEqual([el]);
		cleanup?.();
		expect(disconnect).toHaveBeenCalledOnce();
		expect(setPx).toHaveBeenLastCalledWith(0);
		vi.unstubAllGlobals();
	});

	it('reports clientHeight on the optional viewport setter', () => {
		const setPx = vi.fn();
		const setViewportPx = vi.fn();
		const el = { clientHeight: 1000 } as unknown as HTMLElement;
		vi.stubGlobal(
			'ResizeObserver',
			class {
				observe() {}
				disconnect() {}
			}
		);

		const cleanup = endPadAttach(setPx, setViewportPx)(el);
		expect(setViewportPx).toHaveBeenCalledWith(1000);
		cleanup?.();
		expect(setViewportPx).toHaveBeenLastCalledWith(0);
		vi.unstubAllGlobals();
	});
});
