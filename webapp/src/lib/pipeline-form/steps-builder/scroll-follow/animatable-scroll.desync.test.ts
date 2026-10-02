// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Real animejs regression: external DOM scrollTop writes leave Animatable's
 * internal `_number` stale. scrollTo must resync before tweening so the first
 * frame starts from the DOM position, not the stale value.
 */
import { describe, expect, it } from 'vitest';

import { createAnimatableScroll } from './animatable-scroll.js';

type ScrollTopTween = {
	_fromNumber: number;
	_toNumber: number;
};

type ScrollTopAnimation = {
	duration: number;
	_head: ScrollTopTween;
	seek(time: number): void;
};

describe('animatable-scroll desync (real animejs)', () => {
	it('scrollTo tweens from DOM scrollTop after an external write, not stale Animatable', () => {
		const scroller = { scrollTop: 100 } as HTMLElement;
		const handle = createAnimatableScroll(scroller, {
			durationMs: 200,
			ease: 'linear'
		});

		const jsAnim = handle.animatable.animations.scrollTop as unknown as ScrollTopAnimation;
		const tween = jsAnim._head;

		handle.scrollTo(800, 50);
		jsAnim.seek(jsAnim.duration);
		expect(scroller.scrollTop).toBe(800);

		// External write (restoreScrollTop / revealUnitNearest / native scroll)
		scroller.scrollTop = 200;

		handle.scrollTo(500, 100);

		expect(tween._fromNumber).toBeCloseTo(200, 0);
		expect(tween._fromNumber).not.toBe(800);
		expect(tween._toNumber).toBeCloseTo(500, 0);

		jsAnim.seek(0);
		expect(scroller.scrollTop).toBeCloseTo(200, 0);

		handle.dispose();
	});
});
