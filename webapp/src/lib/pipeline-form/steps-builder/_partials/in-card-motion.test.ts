// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

const animateMock = vi.hoisted(() => vi.fn());

vi.mock('animejs', () => ({ animate: animateMock }));

import {
	IN_CARD_MOTION_EASE,
	IN_CARD_MOTION_MS,
	bodyMaxHeightWithinCard,
	cancelMotion,
	collapseRegion,
	expandRegion,
	fadeTo,
	playInCardEnter
} from './in-card-motion.js';

type AnimeParams = {
	height?: string[];
	opacity?: number;
	duration: number;
	ease: string;
	onComplete: () => void;
};

type FakeEl = {
	style: Record<string, string>;
	getBoundingClientRect: () => { height: number };
};

function fakeEl(height = 0, naturalHeight = 120): HTMLElement {
	const el: FakeEl = {
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

let cancels: ReturnType<typeof vi.fn>[];

beforeEach(() => {
	cancels = [];
	animateMock.mockReset();
	animateMock.mockImplementation(() => {
		const cancel = vi.fn();
		cancels.push(cancel);
		return { cancel };
	});
});

function lastParams(): AnimeParams {
	return animateMock.mock.calls.at(-1)![1] as AnimeParams;
}

function paramsAt(index: number): AnimeParams {
	return animateMock.mock.calls[index]![1] as AnimeParams;
}

describe('in-card-motion', () => {
	it('expands to natural height with ~200ms ease-out and clears inline styles on complete', async () => {
		const el = fakeEl();
		const onComplete = vi.fn();
		const handle = expandRegion(el, { startCollapsed: true, onComplete });

		const params = lastParams();
		expect(params.height).toEqual(['0px', '120px']);
		expect(params.opacity).toBe(1);
		expect(params.duration).toBe(IN_CARD_MOTION_MS);
		expect(params.ease).toBe(IN_CARD_MOTION_EASE);
		expect(el.style.overflow).toBe('hidden');

		params.onComplete();
		await handle.finished;

		expect(onComplete).toHaveBeenCalledOnce();
		expect(el.style.height).toBe('');
		expect(el.style.overflow).toBe('');
		expect(el.style.opacity).toBe('');
	});

	it('collapses from current height to 0 and fades out', () => {
		const el = fakeEl(80);
		collapseRegion(el);

		const params = lastParams();
		expect(params.height).toEqual(['80px', '0px']);
		expect(params.opacity).toBe(0);
	});

	it('cancels the previous animation on the same element and resolves finished', async () => {
		const el = fakeEl(80);
		const first = collapseRegion(el);
		expandRegion(el);

		expect(cancels[0]).toHaveBeenCalledOnce();
		await first.finished;
	});

	it('cancel is idempotent and does not run onComplete', async () => {
		const el = fakeEl(80);
		const onComplete = vi.fn();
		const handle = collapseRegion(el, { onComplete });

		handle.cancel();
		handle.cancel();
		cancelMotion(el);

		await handle.finished;
		expect(onComplete).not.toHaveBeenCalled();
		expect(() => cancelMotion(null)).not.toThrow();
	});

	it('fadeTo animates each element and completes once', async () => {
		const a = fakeEl();
		const b = fakeEl();
		const onComplete = vi.fn();
		const handle = fadeTo([a, b], 0.4, { onComplete });

		expect(animateMock).toHaveBeenCalledTimes(2);
		expect(lastParams().opacity).toBe(0.4);

		for (const call of animateMock.mock.calls) (call[1] as AnimeParams).onComplete();
		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
	});

	it('fadeTo with no targets completes immediately', () => {
		const onComplete = vi.fn();
		fadeTo([], 1, { onComplete });
		expect(onComplete).toHaveBeenCalledOnce();
		expect(animateMock).not.toHaveBeenCalled();
	});

	it('playInCardEnter fades display, fades form, then grows lock height', async () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onComplete = vi.fn();

		const handle = playInCardEnter({ lock, display, form, maxHeightPx: 480, onComplete });

		expect(lock.style.height).toBe('80px');
		expect(lock.style.overflow).toBe('hidden');
		expect(form.style.opacity).toBe('0');
		expect(form.style.position).toBe('absolute');

		// 1) display fade out
		expect(animateMock).toHaveBeenCalledTimes(1);
		expect(paramsAt(0).opacity).toBe(0);
		paramsAt(0).onComplete();

		// 2) form fade in
		expect(animateMock).toHaveBeenCalledTimes(2);
		expect(paramsAt(1).opacity).toBe(1);
		paramsAt(1).onComplete();

		// 3) grow lock — measures form with bottom:auto so height is natural, not lock-sized
		expect(animateMock).toHaveBeenCalledTimes(3);
		expect(paramsAt(2).height).toEqual(['80px', '200px']);
		paramsAt(2).onComplete();

		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
		expect(lock.style.height).toBe('');
		expect(form.style.position).toBe('');
		expect(display.style.visibility).toBe('hidden');
	});

	it('playInCardEnter grow measures natural height even when inset pins the form', () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(80, 240);
		// Simulate inset:0 pin: height:auto would still report lock height without bottom:auto.
		form.style.inset = '0';
		form.style.top = '0';
		form.style.bottom = '0';
		form.style.height = '100%';

		playInCardEnter({ lock, display, form });
		paramsAt(0).onComplete(); // display faded
		paramsAt(1).onComplete(); // form faded

		expect(paramsAt(2).height).toEqual(['80px', '240px']);
	});

	it('bodyMaxHeightWithinCard subtracts header chrome from the card cap', () => {
		const card = fakeEl(120, 120);
		const display = fakeEl(80, 80);
		expect(bodyMaxHeightWithinCard(card, display, 480)).toBe(440);
		expect(bodyMaxHeightWithinCard(card, display, 100)).toBe(60);
	});
});
