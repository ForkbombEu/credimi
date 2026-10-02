// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import { runPairedShift } from './paired-shift.js';

function fakeScroller(scrollTop: number): HTMLElement {
	return { scrollTop } as HTMLElement;
}

describe('runPairedShift', () => {
	it('runs mute → mount → mutate/sync/FLIP restore → frame cards → delay → frame yaml', async () => {
		const order: string[] = [];
		const cards = fakeScroller(0);
		const yaml = fakeScroller(0);
		let insideRun = false;
		// Lower fully below port 100–500; pair fits → min scrollTop 85 (see frame-pair.test).
		const pairBelowFold = {
			clientHeight: 400,
			scrollHeight: 2000,
			upper: { top: 300, bottom: 380 },
			lower: { top: 505, bottom: 585 },
			scrollerRect: { top: 100, bottom: 500 }
		};

		await runPairedShift({
			fromIndex: 2,
			toIndex: 3,
			notePairedReorder: () => order.push('mute'),
			ensureSwapMounted: async (from, to) => {
				order.push(`mount:${from},${to}`);
			},
			cardsScroller: cards,
			yamlScroller: yaml,
			layout: {
				run: async (fn) => {
					order.push('layout-enter');
					insideRun = true;
					await fn();
					insideRun = false;
					order.push('layout-exit');
				}
			},
			mutate: () => {
				expect(insideRun).toBe(true);
				order.push('mutate');
				cards.scrollTop = 999;
				yaml.scrollTop = 888;
			},
			rememoAfterMutate: () => {
				expect(insideRun).toBe(true);
				order.push('sync');
			},
			tick: async () => {
				expect(insideRun).toBe(true);
				order.push('tick');
			},
			pinScrollports: (cardsTop, yamlTop) => {
				expect(insideRun).toBe(true);
				order.push(`restore:${cardsTop}`);
				order.push(`restore:${yamlTop}`);
				cards.scrollTop = cardsTop;
				yaml.scrollTop = yamlTop;
			},
			measurePair: (scroller, upper, lower) => {
				const pane = scroller === cards ? 'cards' : 'yaml';
				order.push(`measure:${pane}:${upper},${lower}`);
				return { scrollTop: scroller.scrollTop, ...pairBelowFold };
			},
			scrollTo: (scroller, top) => {
				const pane = scroller === cards ? 'cards' : 'yaml';
				order.push(`scroll:${pane}:${top}`);
				scroller.scrollTop = top;
			},
			yamlFrameDelayMs: 50,
			delay: async (ms) => {
				order.push(`delay:${ms}`);
			}
		});

		expect(order).toEqual([
			'mute',
			'mount:2,3',
			'layout-enter',
			'mutate',
			'sync',
			'tick',
			'restore:0',
			'restore:0',
			'layout-exit',
			'measure:cards:2,3',
			'scroll:cards:85',
			'delay:50',
			'measure:yaml:2,3',
			'scroll:yaml:85'
		]);
	});
	it('frames min/max indices regardless of swap direction', async () => {
		const measured: string[] = [];

		await runPairedShift({
			fromIndex: 5,
			toIndex: 4,
			notePairedReorder: () => {},
			ensureSwapMounted: async () => {},
			cardsScroller: fakeScroller(0),
			yamlScroller: fakeScroller(0),
			layout: null,
			mutate: () => {},
			rememoAfterMutate: () => {},
			tick: async () => {},
			pinScrollports: () => {},
			measurePair: (_scroller, upper, lower) => {
				measured.push(`${upper},${lower}`);
				return null;
			},
			scrollTo: () => {},
			yamlFrameDelayMs: 0,
			delay: async () => {}
		});

		expect(measured).toEqual(['4,5', '4,5']);
	});

	it('skips scrollTo when measurePair returns null or frame needs no move', async () => {
		const scrollTo = vi.fn();

		await runPairedShift({
			fromIndex: 0,
			toIndex: 1,
			notePairedReorder: () => {},
			ensureSwapMounted: async () => {},
			cardsScroller: fakeScroller(0),
			yamlScroller: fakeScroller(0),
			layout: null,
			mutate: () => {},
			rememoAfterMutate: () => {},
			tick: async () => {},
			pinScrollports: () => {},
			measurePair: () => ({
				scrollTop: 120,
				clientHeight: 400,
				scrollHeight: 2000,
				// both already fully visible in port 100–500
				upper: { top: 150, bottom: 250 },
				lower: { top: 280, bottom: 360 },
				scrollerRect: { top: 100, bottom: 500 }
			}),
			scrollTo,
			yamlFrameDelayMs: 0,
			delay: async () => {}
		});

		expect(scrollTo).not.toHaveBeenCalled();
	});

	it('frames yaml when both blocks intersect but the lower is clipped (pair fits)', async () => {
		const scrollTo = vi.fn();
		// Smoke geometry: header above steps; both peek in; lower clips → min Δ = 20.
		const clippedFittingPair = {
			scrollTop: 0,
			clientHeight: 400,
			scrollHeight: 2000,
			upper: { top: 200, bottom: 350 },
			lower: { top: 360, bottom: 520 },
			scrollerRect: { top: 100, bottom: 500 }
		};

		await runPairedShift({
			fromIndex: 0,
			toIndex: 1,
			notePairedReorder: () => {},
			ensureSwapMounted: async () => {},
			cardsScroller: fakeScroller(0),
			yamlScroller: fakeScroller(0),
			layout: null,
			mutate: () => {},
			rememoAfterMutate: () => {},
			tick: async () => {},
			pinScrollports: () => {},
			measurePair: () => clippedFittingPair,
			scrollTo,
			yamlFrameDelayMs: 0,
			delay: async () => {}
		});

		expect(scrollTo).toHaveBeenCalledTimes(2);
		expect(scrollTo.mock.calls[0]?.[1]).toBe(20);
		expect(scrollTo.mock.calls[1]?.[1]).toBe(20);
	});

	it('does not scroll cards when only row-gap padding clips; still scrolls clipped YAML', async () => {
		const cards = fakeScroller(0);
		const yaml = fakeScroller(0);
		const scrollTo = vi.fn();
		// Cards: p-4 + pb-3; both card bodies in view; 6px of lower gap past the fold.
		const cardsGapOnlyClip = {
			scrollTop: 0,
			clientHeight: 600,
			scrollHeight: 2000,
			upper: { top: 116, bottom: 386 },
			lower: { top: 386, bottom: 706 },
			scrollerRect: { top: 100, bottom: 700 }
		};
		// YAML: real lower block content clipped → must still frame.
		const yamlContentClipped = {
			scrollTop: 0,
			clientHeight: 400,
			scrollHeight: 2000,
			upper: { top: 200, bottom: 350 },
			lower: { top: 360, bottom: 520 },
			scrollerRect: { top: 100, bottom: 500 }
		};

		await runPairedShift({
			fromIndex: 0,
			toIndex: 1,
			notePairedReorder: () => {},
			ensureSwapMounted: async () => {},
			cardsScroller: cards,
			yamlScroller: yaml,
			layout: null,
			mutate: () => {},
			rememoAfterMutate: () => {},
			tick: async () => {},
			pinScrollports: () => {},
			measurePair: (scroller) => (scroller === cards ? cardsGapOnlyClip : yamlContentClipped),
			scrollTo,
			yamlFrameDelayMs: 0,
			delay: async () => {}
		});

		expect(scrollTo).toHaveBeenCalledTimes(1);
		expect(scrollTo.mock.calls[0]?.[0]).toBe(yaml);
		expect(scrollTo.mock.calls[0]?.[1]).toBe(20);
	});

	it('without layout mutates, syncs, then restores both scroll tops after tick', async () => {
		const cards = fakeScroller(120);
		const yaml = fakeScroller(80);
		const order: string[] = [];

		await runPairedShift({
			fromIndex: 0,
			toIndex: 1,
			notePairedReorder: () => {},
			ensureSwapMounted: async () => {},
			cardsScroller: cards,
			yamlScroller: yaml,
			layout: null,
			mutate: () => {
				order.push('mutate');
				cards.scrollTop = 999;
				yaml.scrollTop = 888;
			},
			rememoAfterMutate: () => {
				order.push('sync');
			},
			tick: async () => {
				order.push('tick');
			},
			pinScrollports: (cardsTop, yamlTop) => {
				order.push(`restore:${cardsTop}`);
				order.push(`restore:${yamlTop}`);
				cards.scrollTop = cardsTop;
				yaml.scrollTop = yamlTop;
			},
			measurePair: () => null,
			scrollTo: () => {},
			yamlFrameDelayMs: 0,
			delay: async () => {}
		});

		expect(order).toEqual(['mutate', 'sync', 'tick', 'restore:120', 'restore:80']);
		expect(cards.scrollTop).toBe(120);
		expect(yaml.scrollTop).toBe(80);
	});

	it('calls rememoAfterMutate after mutate so rememo sees the new step order', async () => {
		const order: string[] = [];

		await runPairedShift({
			fromIndex: 0,
			toIndex: 1,
			notePairedReorder: () => {},
			ensureSwapMounted: async () => {},
			cardsScroller: null,
			yamlScroller: undefined,
			layout: null,
			mutate: () => order.push('mutate'),
			rememoAfterMutate: () => order.push('sync'),
			tick: async () => {},
			pinScrollports: () => {},
			measurePair: () => null,
			scrollTo: () => {},
			yamlFrameDelayMs: 0,
			delay: async () => {}
		});

		expect(order.indexOf('mutate')).toBeLessThan(order.indexOf('sync'));
		expect(order).toEqual(['mutate', 'sync']);
	});

	it('still delays before yaml framing when cards scroller is missing', async () => {
		const order: string[] = [];

		await runPairedShift({
			fromIndex: 1,
			toIndex: 2,
			notePairedReorder: () => order.push('mute'),
			ensureSwapMounted: async () => order.push('mount'),
			cardsScroller: null,
			yamlScroller: fakeScroller(0),
			layout: null,
			mutate: () => order.push('mutate'),
			rememoAfterMutate: () => order.push('sync'),
			tick: async () => order.push('tick'),
			pinScrollports: () => order.push('pin'),
			measurePair: (scroller) => {
				order.push(scroller ? 'measure-yaml' : 'measure-missing');
				return null;
			},
			scrollTo: () => {},
			yamlFrameDelayMs: 80,
			delay: async (ms) => {
				order.push(`delay:${ms}`);
			}
		});

		expect(order).toEqual([
			'mute',
			'mount',
			'mutate',
			'sync',
			'tick',
			'pin',
			'delay:80',
			'measure-yaml'
		]);
	});
});
