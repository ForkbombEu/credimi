// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { computeAlignedScrollTop, computeNearestScrollTop } from './active-unit.js';

describe('computeNearestScrollTop', () => {
	const scroller = { top: 100, bottom: 500 }; // 400px tall

	it('returns null when fully visible', () => {
		expect(computeNearestScrollTop(50, 400, { top: 150, bottom: 200 }, scroller)).toBeNull();
	});

	it('scrolls up when above the viewport', () => {
		expect(computeNearestScrollTop(80, 400, { top: 60, bottom: 90 }, scroller)).toBe(40);
	});

	it('scrolls down when below the viewport', () => {
		expect(computeNearestScrollTop(0, 400, { top: 480, bottom: 560 }, scroller)).toBe(60);
	});
});

describe('computeAlignedScrollTop', () => {
	const scroller = { top: 100, bottom: 500 }; // 400px tall
	const scrollHeight = 2000;

	it('nearest returns null when fully visible', () => {
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 150, bottom: 200 },
				scroller,
				'nearest'
			)
		).toBeNull();
	});

	it('start aligns element top with padding even when already visible', () => {
		// el top at 200, scroller top 100 → delta 100; padding 16 → target scrollTop + 84
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 200, bottom: 220 },
				scroller,
				'start'
			)
		).toBe(134);
	});

	it('center aligns element mid to viewport mid', () => {
		// el mid 210, port mid 300 → delta -90 → target 50 - 90 = -40 → clamped 0
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 200, bottom: 220 },
				scroller,
				'center'
			)
		).toBe(0);
	});

	it('clamps to max scroll', () => {
		expect(
			computeAlignedScrollTop(0, 400, 500, { top: 800, bottom: 820 }, scroller, 'start')
		).toBe(100);
	});

	it('returns null when already within epsilon of start align', () => {
		// delta = 16, padding 16 → target = scrollTop
		expect(
			computeAlignedScrollTop(
				100,
				400,
				scrollHeight,
				{ top: 116, bottom: 136 },
				scroller,
				'start'
			)
		).toBeNull();
	});

	it('start-band skips when line already sits in the upper band', () => {
		// offset 80px into a 400px port → within 35% band (140px), after padding floor
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 180, bottom: 200 },
				scroller,
				'start-band'
			)
		).toBeNull();
	});

	it('start-band scrolls when line is below the band', () => {
		// offset 200px > 140px band
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 300, bottom: 320 },
				scroller,
				'start-band'
			)
		).toBe(234);
	});
});

it('start barely moves a near-top card while center re-homes it', () => {
	// Repro insight: with card N centered, card N-1 often already sits near the
	// scroller top — start/nearest barely move (or no-op), so the viewport still
	// "belongs" to N; the next off-screen unit then leaps.
	const scroller = { top: 100, bottom: 500 };
	const nearTop = { top: 120, bottom: 280 }; // fully visible, near top
	const startTop = computeAlignedScrollTop(200, 400, 2000, nearTop, scroller, 'start');
	const centerTop = computeAlignedScrollTop(200, 400, 2000, nearTop, scroller, 'center');
	expect(computeAlignedScrollTop(200, 400, 2000, nearTop, scroller, 'nearest')).toBeNull();
	expect(startTop).toBe(204); // only +4px
	expect(centerTop).toBe(100); // -100px — enough to change which card owns center
});
