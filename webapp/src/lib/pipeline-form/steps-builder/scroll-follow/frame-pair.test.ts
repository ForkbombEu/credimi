// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { framePairScrollTop } from './frame-pair.js';

describe('framePairScrollTop', () => {
	const port = { top: 100, bottom: 500 }; // 400px tall
	const clientHeight = 400;
	const scrollHeight = 2000;

	it('returns null when both units are already fully visible', () => {
		expect(
			framePairScrollTop(
				120,
				clientHeight,
				scrollHeight,
				{ top: 150, bottom: 250 },
				{ top: 280, bottom: 360 },
				port
			)
		).toBeNull();
	});

	it('returns null when card-sized units sit at scrollTop≈0 and only row-gap padding clips', () => {
		// Smoke regression after fullyVisible early-exit: cards list has p-4 (16px) and each
		// virtual row has pb-3 (12px). Two cards look fully in view; only the lower row's
		// gap padding straddles the fold by a few px. Must NOT scroll (contrast YAML below).
		const cardsPort = { top: 100, bottom: 700 }; // 600px
		expect(
			framePairScrollTop(
				0,
				600,
				scrollHeight,
				{ top: 116, bottom: 386 }, // upperTop=16 (p-4)
				{ top: 386, bottom: 706 }, // 6px of pb-3 past fold; content ends ~694
				cardsPort
			)
		).toBeNull();
	});

	it('returns null when an oversized pair already has both units intersecting', () => {
		// Each unit taller than half the viewport; both still peek into the port.
		expect(
			framePairScrollTop(
				0,
				clientHeight,
				scrollHeight,
				{ top: 50, bottom: 320 },
				{ top: 330, bottom: 620 },
				port
			)
		).toBeNull();
	});

	it('scrolls when a fitting pair both intersect but the lower unit is clipped', () => {
		// Smoke: YAML header above steps; scrollTop≈0; both blocks peek in; lower clips.
		// Content: upperTop=100, lowerBottom=420, pairHeight=320 < 400 → min scrollTop=20.
		expect(
			framePairScrollTop(
				0,
				clientHeight,
				scrollHeight,
				{ top: 200, bottom: 350 },
				{ top: 360, bottom: 520 },
				port
			)
		).toBe(20);
	});

	it('scrolls the minimum amount so a fitting pair is fully visible', () => {
		// Upper in view; lower fully below. Pair height 285 < 400.
		// Content: upperTop=200, lowerBottom=485 → min scrollTop = 85.
		expect(
			framePairScrollTop(
				0,
				clientHeight,
				scrollHeight,
				{ top: 300, bottom: 380 },
				{ top: 505, bottom: 585 },
				port
			)
		).toBe(85);
	});

	it('gap-centers when the pair is taller than the viewport', () => {
		// Lower fully below. upperBottom=250, lowerTop=420 → gap center=335 → target=135.
		expect(
			framePairScrollTop(
				0,
				clientHeight,
				scrollHeight,
				{ top: 100, bottom: 350 },
				{ top: 520, bottom: 900 },
				port
			)
		).toBe(135);
	});

	it('clamps gap-center to max scroll', () => {
		// Same geometry wants 135; scrollHeight 450 → maxScroll 50.
		expect(
			framePairScrollTop(
				0,
				clientHeight,
				450,
				{ top: 100, bottom: 350 },
				{ top: 520, bottom: 900 },
				port
			)
		).toBe(50);
	});
});
