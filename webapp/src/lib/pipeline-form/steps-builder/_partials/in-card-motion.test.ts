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
	playInCardEnter,
	playInCardExit,
	settleEnterFormHost
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

function paramsAt(index: number): AnimeParams {
	return animateMock.mock.calls[index]![1] as AnimeParams;
}

describe('in-card-motion', () => {
	it('starting a new enter cancels the previous step animation', () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		playInCardEnter({ lock, display, form, maxHeightPx: 480 });
		playInCardEnter({ lock, display, form, maxHeightPx: 480 });

		expect(cancels[0]).toHaveBeenCalledOnce();
	});

	it('cancel is idempotent and does not run onComplete', async () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onComplete = vi.fn();
		const handle = playInCardEnter({ lock, display, form, maxHeightPx: 480, onComplete });

		handle.cancel();
		handle.cancel();
		cancelMotion(lock);

		await handle.finished;
		expect(onComplete).not.toHaveBeenCalled();
		expect(() => cancelMotion(null)).not.toThrow();
	});

	it('playInCardEnter fades display, fades form, then grows lock to the column body max', async () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onComplete = vi.fn(async () => {
			// Caller applies settled flex classes before settleEnterFormHost runs.
			expect(form.style.position).toBe('absolute');
			expect(form.style.height).toBe('100%');
			expect(lock.style.height).toBe('480px');
		});

		const handle = playInCardEnter({ lock, display, form, maxHeightPx: 480, onComplete });

		expect(lock.style.height).toBe('80px');
		expect(lock.style.overflow).toBe('hidden');
		expect(form.style.opacity).toBe('0');
		expect(form.style.position).toBe('absolute');

		// 1) display fade out
		expect(animateMock).toHaveBeenCalledTimes(1);
		expect(paramsAt(0).opacity).toBe(0);
		expect(paramsAt(0).duration).toBe(IN_CARD_MOTION_MS);
		expect(paramsAt(0).ease).toBe(IN_CARD_MOTION_EASE);
		paramsAt(0).onComplete();

		// 2) form fade in
		expect(animateMock).toHaveBeenCalledTimes(2);
		expect(paramsAt(1).opacity).toBe(1);
		paramsAt(1).onComplete();

		// 3) grow lock to the provided body max (fill column), not form natural height
		expect(animateMock).toHaveBeenCalledTimes(3);
		expect(paramsAt(2).height).toEqual(['80px', '480px']);
		paramsAt(2).onComplete();

		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
		expect(lock.style.height).toBe('480px');
		// Absolute fill is cleared only after onComplete resolves (post-tick in UI).
		expect(form.style.position).toBe('');
		expect(form.style.height).toBe('');
		expect(display.style.visibility).toBe('hidden');
	});

	it('settleEnterFormHost clears overlay styles without touching the lock', () => {
		const form = fakeEl(100, 100);
		form.style.position = 'absolute';
		form.style.inset = '0';
		form.style.height = '100%';
		form.style.opacity = '1';
		settleEnterFormHost(form);
		expect(form.style.position).toBe('');
		expect(form.style.height).toBe('');
		expect(form.style.opacity).toBe('');
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

	it('playInCardExit shrinks lock to summary height then crossfades', async () => {
		const lock = fakeEl(200, 200);
		lock.style.height = '200px';
		const display = fakeEl(0, 80);
		const form = fakeEl(200, 200);
		const onComplete = vi.fn();

		const handle = playInCardExit({ lock, display, form, onComplete });

		// 1) shrink to summary natural height (not zero)
		expect(animateMock).toHaveBeenCalledTimes(1);
		expect(paramsAt(0).height).toEqual(['200px', '80px']);
		paramsAt(0).onComplete();

		// 2) form fade out
		expect(animateMock).toHaveBeenCalledTimes(2);
		expect(paramsAt(1).opacity).toBe(0);
		paramsAt(1).onComplete();

		// 3) display fade in
		expect(animateMock).toHaveBeenCalledTimes(3);
		expect(paramsAt(2).opacity).toBe(1);
		paramsAt(2).onComplete();

		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
		expect(lock.style.height).toBe('');
	});
});
