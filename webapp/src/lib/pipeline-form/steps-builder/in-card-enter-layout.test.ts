// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const animateMock = vi.hoisted(() => vi.fn());

vi.mock('animejs', () => ({ animate: animateMock }));

import {
	inCardFormHostClass,
	playInCardEnterLayout
} from './in-card-enter-layout.js';

type AnimeParams = {
	height?: string[];
	opacity?: number;
	duration: number;
	ease: string;
	onComplete: () => void;
};

type FakeEl = {
	className: string;
	style: Record<string, string>;
	getBoundingClientRect: () => { height: number };
};

function fakeEl(height = 0, naturalHeight = 120, className = ''): HTMLElement {
	const el: FakeEl = {
		className,
		style: { height: `${height}px` },
		getBoundingClientRect: () => {
			if (el.style.height === 'auto') return { height: naturalHeight };
			const parsed = parseFloat(el.style.height || '0');
			if (Number.isFinite(parsed) && el.style.height !== '') return { height: parsed };
			return { height };
		}
	};
	return el as unknown as HTMLElement;
}

beforeEach(() => {
	animateMock.mockReset();
	animateMock.mockImplementation(() => ({ cancel: vi.fn() }));
});

function paramsAt(index: number): AnimeParams {
	return animateMock.mock.calls[index]![1] as AnimeParams;
}

/** Drive enter through display fade → form fade → grow complete. */
function completeEnterSteps() {
	paramsAt(0).onComplete();
	paramsAt(1).onComplete();
	paramsAt(2).onComplete();
}

describe('inCardFormHostClass', () => {
	it('keeps flex column during enter absolute fill so Save stays column-pinned', () => {
		const enter = inCardFormHostClass(false);
		expect(enter.split(/\s+/)).toEqual(
			expect.arrayContaining(['absolute', 'inset-0', 'flex', 'flex-col', 'min-h-0'])
		);
		const settled = inCardFormHostClass(true);
		expect(settled.split(/\s+/)).toEqual(
			expect.arrayContaining(['flex', 'flex-col', 'grow', 'min-h-0', 'overflow-hidden'])
		);
		expect(settled).not.toContain('absolute');
	});
});

describe('playInCardEnterLayout', () => {
	it('starts with absolute-fill host class, settles host before absolute clears', async () => {
		const lock = fakeEl(80, 80, 'relative min-h-0');
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onSettled = vi.fn(async () => {
			// Settled host class is applied; motion has not cleared absolute yet.
			expect(form.className.split(/\s+/)).toEqual(
				expect.arrayContaining(['flex', 'flex-col', 'grow', 'min-h-0'])
			);
			expect(form.className).not.toContain('absolute');
			expect(form.style.position).toBe('absolute');
			expect(form.style.height).toBe('100%');
			expect(lock.style.height).toBe('480px');
			expect(lock.className.split(/\s+/)).toEqual(
				expect.arrayContaining(['relative', 'min-h-0', 'flex', 'flex-col', 'grow'])
			);
		});

		const handle = playInCardEnterLayout({
			lock,
			display,
			form,
			bodyMaxPx: 480,
			onSettled
		});

		expect(form.className.split(/\s+/)).toEqual(
			expect.arrayContaining(['absolute', 'inset-0', 'flex', 'flex-col', 'min-h-0'])
		);

		completeEnterSteps();
		await handle.finished;

		expect(onSettled).toHaveBeenCalledOnce();
		expect(onSettled).toHaveBeenCalledWith({ bodyMaxPx: 480 });
		// After settleEnterFormHost (post onComplete): absolute fill cleared.
		expect(form.style.position).toBe('');
		expect(form.style.height).toBe('');
		expect(form.className).not.toContain('absolute');
	});

	it('computes body max from card + cardMaxHeightPx when bodyMaxPx omitted', async () => {
		const card = fakeEl(120, 120);
		const lock = fakeEl(80, 80, 'relative min-h-0');
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onSettled = vi.fn();

		const handle = playInCardEnterLayout({
			lock,
			display,
			form,
			card,
			cardMaxHeightPx: 480,
			onSettled
		});

		completeEnterSteps();
		await handle.finished;

		// chrome = card 120 - display 80 = 40 → body max 440
		expect(onSettled).toHaveBeenCalledWith({ bodyMaxPx: 440 });
		expect(lock.style.height).toBe('440px');
		expect(lock.style.maxHeight).toBe('440px');
	});

	it('calls onSettled before settle clears absolute fill', async () => {
		const lock = fakeEl(80, 80, 'relative min-h-0');
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const order: string[] = [];

		const handle = playInCardEnterLayout({
			lock,
			display,
			form,
			bodyMaxPx: 300,
			onSettled: async () => {
				order.push('onSettled');
				expect(form.style.position).toBe('absolute');
			}
		});

		completeEnterSteps();
		await handle.finished;

		order.push('finished');
		expect(order).toEqual(['onSettled', 'finished']);
		expect(form.style.position).toBe('');
	});
});
