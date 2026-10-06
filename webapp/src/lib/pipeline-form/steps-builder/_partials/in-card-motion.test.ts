// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const animateMock = vi.hoisted(() => vi.fn());

vi.mock('animejs', () => ({ animate: animateMock }));

import {
	IN_CARD_MOTION_EASE,
	IN_CARD_MOTION_MS,
	cancelMotion,
	playInCardEnter,
	playInCardExit
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

function stubReducedMotion(matches: boolean) {
	vi.stubGlobal('window', {
		matchMedia: (query: string) => ({
			matches: matches && query.includes('prefers-reduced-motion'),
			media: query,
			addEventListener: () => {},
			removeEventListener: () => {},
			addListener: () => {},
			removeListener: () => {},
			dispatchEvent: () => false,
			onchange: null
		})
	});
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

afterEach(() => {
	vi.unstubAllGlobals();
});

function paramsAt(index: number): AnimeParams {
	return animateMock.mock.calls[index]![1] as AnimeParams;
}

describe('in-card-motion', () => {
	it('starting a new enter cancels the previous step animation', () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		playInCardEnter({ lock, display, form, toHeight: 480 });
		playInCardEnter({ lock, display, form, toHeight: 480 });

		expect(cancels[0]).toHaveBeenCalledOnce();
	});

	it('cancel is idempotent and does not run onComplete', async () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onComplete = vi.fn();
		const handle = playInCardEnter({ lock, display, form, toHeight: 480, onComplete });

		handle.cancel();
		handle.cancel();
		cancelMotion(lock);

		await handle.finished;
		expect(onComplete).not.toHaveBeenCalled();
		expect(() => cancelMotion(null)).not.toThrow();
	});

	it('playInCardEnter fades display, fades form, then grows lock to toHeight', async () => {
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onComplete = vi.fn();

		const handle = playInCardEnter({ lock, display, form, toHeight: 480, onComplete });

		expect(animateMock).toHaveBeenCalledTimes(1);
		expect(paramsAt(0).opacity).toBe(0);
		expect(paramsAt(0).duration).toBe(IN_CARD_MOTION_MS);
		expect(paramsAt(0).ease).toBe(IN_CARD_MOTION_EASE);
		paramsAt(0).onComplete();

		expect(animateMock).toHaveBeenCalledTimes(2);
		expect(paramsAt(1).opacity).toBe(1);
		paramsAt(1).onComplete();

		expect(animateMock).toHaveBeenCalledTimes(3);
		expect(paramsAt(2).height).toEqual(['80px', '480px']);
		paramsAt(2).onComplete();

		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
		expect(lock.style.height).toBe('480px');
		expect(display.style.opacity).toBe('0');
		expect(form.style.opacity).toBe('1');
	});

	it('playInCardExit shrinks lock to toHeight then crossfades', async () => {
		const lock = fakeEl(200, 200);
		lock.style.height = '200px';
		const display = fakeEl(0, 80);
		const form = fakeEl(200, 200);
		const onComplete = vi.fn();

		const handle = playInCardExit({ lock, display, form, toHeight: 80, onComplete });

		expect(animateMock).toHaveBeenCalledTimes(1);
		expect(paramsAt(0).height).toEqual(['200px', '80px']);
		paramsAt(0).onComplete();

		expect(animateMock).toHaveBeenCalledTimes(2);
		expect(paramsAt(1).opacity).toBe(0);
		paramsAt(1).onComplete();

		expect(animateMock).toHaveBeenCalledTimes(3);
		expect(paramsAt(2).opacity).toBe(1);
		paramsAt(2).onComplete();

		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
		expect(lock.style.height).toBe('80px');
		expect(form.style.opacity).toBe('0');
		expect(display.style.opacity).toBe('1');
	});

	it('prefers-reduced-motion skips anime on enter and still reaches onComplete', async () => {
		stubReducedMotion(true);
		const lock = fakeEl(80, 80);
		const display = fakeEl(80, 80);
		const form = fakeEl(0, 200);
		const onComplete = vi.fn();

		const handle = playInCardEnter({ lock, display, form, toHeight: 480, onComplete });

		expect(animateMock).not.toHaveBeenCalled();
		expect(lock.style.height).toBe('480px');
		expect(display.style.opacity).toBe('0');
		expect(form.style.opacity).toBe('1');
		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
	});

	it('prefers-reduced-motion skips anime on exit and still reaches onComplete', async () => {
		stubReducedMotion(true);
		const lock = fakeEl(200, 200);
		lock.style.height = '200px';
		const display = fakeEl(0, 80);
		const form = fakeEl(200, 200);
		const onComplete = vi.fn();

		const handle = playInCardExit({ lock, display, form, toHeight: 80, onComplete });

		expect(animateMock).not.toHaveBeenCalled();
		expect(lock.style.height).toBe('80px');
		expect(form.style.opacity).toBe('0');
		expect(display.style.opacity).toBe('1');
		await handle.finished;
		expect(onComplete).toHaveBeenCalledOnce();
	});
});
