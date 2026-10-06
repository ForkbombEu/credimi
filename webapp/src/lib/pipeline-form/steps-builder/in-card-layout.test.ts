// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const animateMock = vi.hoisted(() => vi.fn());

vi.mock('animejs', () => ({ animate: animateMock }));

import {
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

describe('playInCardEnterLayout', () => {
	it('prepares absolute overlay and settles lock before absolute clears', async () => {
		const lock = fakeEl(80, 80, 'relative min-h-0');
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onSettled = vi.fn(async () => {
			// Host owns form-host className; layout keeps absolute fill until after onSettled.
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

		expect(form.style.position).toBe('absolute');
		expect(form.style.opacity).toBe('0');
		expect(lock.style.height).toBe('80px');
		expect(lock.style.overflow).toBe('hidden');

		completeEnterSteps();
		await handle.finished;

		expect(onSettled).toHaveBeenCalledOnce();
		expect(onSettled).toHaveBeenCalledWith({ bodyMaxPx: 480 });
		expect(form.style.position).toBe('');
		expect(form.style.height).toBe('');
		expect(display.style.visibility).toBe('hidden');
		expect(lock.style.height).toBe('480px');
	});

	it('settleEnterFormHost clears overlay styles without touching the lock', async () => {
		const lock = fakeEl(80, 80, 'relative min-h-0');
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);

		const handle = playInCardEnterLayout({
			lock,
			display,
			form,
			bodyMaxPx: 480
		});

		completeEnterSteps();
		await handle.finished;

		expect(form.style.position).toBe('');
		expect(form.style.height).toBe('');
		expect(form.style.opacity).toBe('');
		expect(lock.style.height).toBe('480px');
	});

	it('grows to form natural height even when inset pins the form', async () => {
		const lock = fakeEl(80, 80, 'relative min-h-0');
		const display = fakeEl(80, 80);
		const form = fakeEl(80, 240);
		form.style.inset = '0';
		form.style.top = '0';
		form.style.bottom = '0';
		form.style.height = '100%';

		const handle = playInCardEnterLayout({ lock, display, form });

		paramsAt(0).onComplete();
		paramsAt(1).onComplete();
		expect(paramsAt(2).height).toEqual(['80px', '240px']);
		paramsAt(2).onComplete();
		await handle.finished;

		expect(lock.style.height).toBe('240px');
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

		// card 120 − display 80 = 40 chrome → body max 480 − 40 = 440
		expect(onSettled).toHaveBeenCalledWith({ bodyMaxPx: 440 });
		expect(lock.style.height).toBe('440px');
		expect(lock.style.maxHeight).toBe('440px');
	});

	it('clamps body max at zero when chrome exceeds card fill', async () => {
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
			cardFillMaxPx: 100,
			onSettled
		});

		completeEnterSteps();
		await handle.finished;

		// chrome 40 → body max max(0, 100 − 40) = 60
		expect(onSettled).toHaveBeenCalledWith({ bodyMaxPx: 60 });
		expect(lock.style.height).toBe('60px');
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

		expect(form.style.position).toBe('absolute');
		expect(display.style.position).toBe('absolute');
		expect(animateMock).toHaveBeenCalled();
		expect(paramsAt(0).height).toEqual(['200px', '80px']);
	});

	it('tears down exit overlay after shrink and crossfade', async () => {
		const lock = fakeEl(200, 200);
		lock.style.height = '200px';
		const display = fakeEl(0, 80);
		const form = fakeEl(200, 200);
		const onComplete = vi.fn();

		const handle = playInCardExitLayout({
			lock,
			display,
			form,
			enterSettled: true,
			onComplete
		});

		paramsAt(0).onComplete();
		paramsAt(1).onComplete();
		paramsAt(2).onComplete();
		await handle.finished;

		expect(onComplete).toHaveBeenCalledOnce();
		expect(display.style.position).toBe('');
		expect(form.style.position).toBe('');
		expect(form.style.opacity).toBe('0');
		expect(form.style.visibility).toBe('hidden');
		expect(lock.style.height).toBe('');
	});
});
