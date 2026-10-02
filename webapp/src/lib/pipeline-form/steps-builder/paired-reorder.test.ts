// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { runPairedReorder } from './paired-reorder.js';

function fakeScroller(scrollTop: number): HTMLElement {
	return { scrollTop } as HTMLElement;
}

describe('runPairedReorder', () => {
	it('without layout mutates, syncs, then restores both scroll tops after tick', async () => {
		const cards = fakeScroller(120);
		const yaml = fakeScroller(80);
		const order: string[] = [];

		await runPairedReorder({
			cardsScroller: cards,
			yamlScroller: yaml,
			layout: null,
			mutate: () => {
				order.push('mutate');
				cards.scrollTop = 999;
				yaml.scrollTop = 888;
			},
			syncBoth: () => {
				order.push('sync');
			},
			tick: async () => {
				order.push('tick');
			},
			restoreScrollTop: (el, top) => {
				order.push(`restore:${top}`);
				if (el) el.scrollTop = top;
			}
		});

		expect(order).toEqual(['mutate', 'sync', 'tick', 'restore:120', 'restore:80']);
		expect(cards.scrollTop).toBe(120);
		expect(yaml.scrollTop).toBe(80);
	});

	it('with layout runs mutate, sync, tick, and restore inside layout.run', async () => {
		const cards = fakeScroller(50);
		const yaml = fakeScroller(25);
		const order: string[] = [];
		let insideRun = false;

		await runPairedReorder({
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
				cards.scrollTop = 0;
				yaml.scrollTop = 0;
			},
			syncBoth: () => {
				expect(insideRun).toBe(true);
				order.push('sync');
			},
			tick: async () => {
				expect(insideRun).toBe(true);
				order.push('tick');
			},
			restoreScrollTop: (el, top) => {
				expect(insideRun).toBe(true);
				order.push(`restore:${top}`);
				if (el) el.scrollTop = top;
			}
		});

		expect(order).toEqual([
			'layout-enter',
			'mutate',
			'sync',
			'tick',
			'restore:50',
			'restore:25',
			'layout-exit'
		]);
		expect(cards.scrollTop).toBe(50);
		expect(yaml.scrollTop).toBe(25);
	});

	it('calls syncBoth after mutate so rememo sees the new step order', async () => {
		const order: string[] = [];

		await runPairedReorder({
			cardsScroller: null,
			yamlScroller: undefined,
			layout: null,
			mutate: () => order.push('mutate'),
			syncBoth: () => order.push('sync'),
			tick: async () => {},
			restoreScrollTop: () => {}
		});

		expect(order.indexOf('mutate')).toBeLessThan(order.indexOf('sync'));
		expect(order).toEqual(['mutate', 'sync']);
	});
});
