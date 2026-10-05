// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const animateMock = vi.hoisted(() => vi.fn());

vi.mock('animejs', () => ({ animate: animateMock }));

import {
	bodyMaxHeightWithinCard,
	inCardFormHostClass,
	playInCardEnterLayout,
	playInCardExitLayout
} from './in-card-layout.js';

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

describe('bodyMaxHeightWithinCard', () => {
	it('subtracts header chrome from the card fill max', () => {
		const card = fakeEl(120, 120);
		const display = fakeEl(80, 80);
		expect(bodyMaxHeightWithinCard(card, display, 480)).toBe(440);
		expect(bodyMaxHeightWithinCard(card, display, 100)).toBe(60);
	});
});

describe('playInCardEnterLayout', () => {
	it('starts with absolute-fill host class, settles host before absolute clears', async () => {
		const lock = fakeEl(80, 80, 'relative min-h-0');
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onSettled = vi.fn(async () => {
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
		expect(form.style.position).toBe('');
		expect(form.style.height).toBe('');
		expect(form.className).not.toContain('absolute');
	});

	it('computes body max from card + cardFillMaxPx when bodyMaxPx omitted', async () => {
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
			cardFillMaxPx: 480,
			onSettled
		});

		completeEnterSteps();
		await handle.finished;

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

describe('playInCardExitLayout', () => {
	it('completes without anime when enter never settled', async () => {
		const onComplete = vi.fn();
		const handle = playInCardExitLayout({
			lock: fakeEl(80, 80),
			display: fakeEl(80, 80),
			form: fakeEl(80, 80),
			enterSettled: false,
			onComplete
		});

		expect(animateMock).not.toHaveBeenCalled();
		expect(onComplete).not.toHaveBeenCalled();
		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
	});

	it('shrinks via anime when enter settled', () => {
		const lock = fakeEl(200, 200);
		lock.style.height = '200px';
		const display = fakeEl(0, 80);
		const form = fakeEl(200, 200);

		playInCardExitLayout({
			lock,
			display,
			form,
			enterSettled: true
		});

		expect(animateMock).toHaveBeenCalled();
		expect(paramsAt(0).height).toEqual(['200px', '80px']);
	});
});
