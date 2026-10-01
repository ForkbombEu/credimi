// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
	animatableDrivenIdleMs,
	ANIMATABLE_DRIVEN_IDLE_PAD_MS,
	createAnimatableScroll,
	DISCRETE_SCROLL_DURATION_MS,
	FOLLOW_SCROLL_DURATION_MS,
	SCROLL_EASE
} from './animatable-scroll.js';

const scrollTopFn = vi.fn();
const revertFn = vi.fn();
const createAnimatableMock = vi.fn(
	(_target: unknown, params: { scrollTop: number; ease: string }) => {
		scrollTopFn.mockImplementation((to?: number, duration?: number) => {
			if (to === undefined) return 42;
			return { to, duration: duration ?? params.scrollTop };
		});
		return {
			scrollTop: scrollTopFn,
			animations: {},
			callbacks: null,
			revert: revertFn
		};
	}
);

vi.mock('animejs', () => ({
	createAnimatable: (...args: unknown[]) => createAnimatableMock(...args)
}));

describe('animatable-scroll', () => {
	beforeEach(() => {
		scrollTopFn.mockClear();
		revertFn.mockClear();
		createAnimatableMock.mockClear();
	});

	it('creates a persistent Animatable with follow defaults', () => {
		const scroller = { scrollTop: 0 } as HTMLElement;
		const handle = createAnimatableScroll(scroller);

		expect(createAnimatableMock).toHaveBeenCalledWith(
			scroller,
			expect.objectContaining({
				scrollTop: FOLLOW_SCROLL_DURATION_MS,
				ease: SCROLL_EASE
			})
		);

		handle.scrollTo(120);
		expect(scrollTopFn).toHaveBeenCalledWith(120);

		handle.scrollTo(200, DISCRETE_SCROLL_DURATION_MS);
		expect(scrollTopFn).toHaveBeenCalledWith(200, DISCRETE_SCROLL_DURATION_MS);

		expect(handle.getScrollTop()).toBe(42);

		handle.dispose();
		expect(revertFn).toHaveBeenCalled();
	});

	it('pads driven idle timeout past the tween duration', () => {
		expect(animatableDrivenIdleMs(FOLLOW_SCROLL_DURATION_MS)).toBe(
			FOLLOW_SCROLL_DURATION_MS + ANIMATABLE_DRIVEN_IDLE_PAD_MS
		);
		expect(animatableDrivenIdleMs(0)).toBe(ANIMATABLE_DRIVEN_IDLE_PAD_MS);
	});
});
