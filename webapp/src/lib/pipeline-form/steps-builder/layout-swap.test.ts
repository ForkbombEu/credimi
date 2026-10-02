// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
	createMultiListLayout,
	DEFAULT_YAML_LAYOUT_CHILDREN,
	flipInverts,
	LAYOUT_SWAP_DURATION_MS,
	LAYOUT_SWAP_EASE,
	type CardListLayout,
	type CreateCardListLayoutOptions
} from './layout-swap.js';

function el(id: string): HTMLElement {
	return { id } as HTMLElement;
}

/** Test-local single-root FLIP helper (production uses createMultiListLayout). */
function createCardListLayout(
	root: HTMLElement,
	options?: CreateCardListLayoutOptions
): CardListLayout {
	return createMultiListLayout([{ root, children: options?.children }], options);
}

describe('flipInverts', () => {
	it('returns first-minus-last for nodes in both snapshots', () => {
		const a = el('a');
		const b = el('b');
		const first = new Map<HTMLElement, number>([
			[a, 0],
			[b, 140]
		]);
		const last = new Map<HTMLElement, number>([
			[a, 140],
			[b, 0]
		]);

		expect(flipInverts(first, last)).toEqual([
			{ el: a, y: -140 },
			{ el: b, y: 140 }
		]);
	});

	it('skips nodes missing after the mutation and ~0 deltas', () => {
		const a = el('a');
		const gone = el('gone');
		const still = el('still');
		expect(
			flipInverts(
				new Map([
					[a, 10],
					[gone, 20],
					[still, 30]
				]),
				new Map([
					[a, 10.2],
					[still, 80]
				])
			)
		).toEqual([{ el: still, y: -50 }]);
	});
});

describe('createCardListLayout', () => {
	const a = el('a');
	const b = el('b');
	let firstCall = true;
	const measureTops = vi.fn(() => {
		if (firstCall) {
			firstCall = false;
			return new Map<HTMLElement, number>([
				[a, 0],
				[b, 140]
			]);
		}
		return new Map<HTMLElement, number>([
			[a, 140],
			[b, 0]
		]);
	});
	const animateFn = vi.fn();
	const cancelFn = vi.fn();

	beforeEach(() => {
		firstCall = true;
		measureTops.mockClear();
		animateFn.mockClear();
		cancelFn.mockClear();
		animateFn.mockImplementation(() => ({ cancel: cancelFn }));
	});

	it('measures, mutates, then animates invert y without touching top', async () => {
		const root = {} as HTMLElement;
		const handle = createCardListLayout(root, {
			measureTops,
			animate: animateFn
		});

		const order: string[] = [];
		measureTops.mockImplementation(() => {
			order.push('measure');
			if (order.filter((s) => s === 'measure').length <= 2) {
				return new Map<HTMLElement, number>([
					[a, 0],
					[b, 140]
				]);
			}
			return new Map<HTMLElement, number>([
				[a, 140],
				[b, 0]
			]);
		});

		await handle.run(async () => {
			order.push('mutate');
		});

		expect(order).toEqual(['measure', 'measure', 'mutate', 'measure']);
		expect(animateFn).toHaveBeenCalledWith(
			[a, b],
			expect.objectContaining({
				y: [-140, 140],
				duration: LAYOUT_SWAP_DURATION_MS,
				ease: LAYOUT_SWAP_EASE
			})
		);
	});

	it('skips animate after dispose and still runs mutate', async () => {
		const root = {} as HTMLElement;
		const handle = createCardListLayout(root, {
			measureTops,
			animate: animateFn
		});
		handle.dispose();

		const mutate = vi.fn();
		await handle.run(mutate);

		expect(mutate).toHaveBeenCalled();
		expect(animateFn).not.toHaveBeenCalled();
	});

	it('ignores stale animate when a newer run started', async () => {
		const placed = new Map<HTMLElement, number>([
			[a, 0],
			[b, 140]
		]);
		const swapped = new Map<HTMLElement, number>([
			[a, 140],
			[b, 0]
		]);
		const queue = [placed, placed, placed, placed, swapped];
		const handle = createCardListLayout({} as HTMLElement, {
			measureTops: () => queue.shift() ?? swapped,
			animate: animateFn
		});

		let releaseFirst!: () => void;
		const firstGate = new Promise<void>((resolve) => {
			releaseFirst = resolve;
		});

		const first = handle.run(() => firstGate);
		const second = handle.run(() => undefined);

		releaseFirst();
		await Promise.all([first, second]);

		expect(animateFn).toHaveBeenCalledTimes(1);
	});
});

describe('createMultiListLayout', () => {
	it('measures both roots, mutates once, animates invert ys from both', async () => {
		const cardA = el('card-a');
		const cardB = el('card-b');
		const yamlA = el('yaml-a');
		const yamlB = el('yaml-b');
		const cardsRoot = {} as HTMLElement;
		const yamlRoot = {} as HTMLElement;
		const animateFn = vi.fn(() => ({ cancel: vi.fn() }));

		const measureTops = vi.fn((root: HTMLElement, _selector: string) => {
			const call = measureTops.mock.calls.length;
			// clear + first pass: calls 1–4; after mutate last pass: 5–6
			const swapped = call > 4;
			if (root === cardsRoot) {
				return swapped
					? new Map<HTMLElement, number>([
							[cardA, 140],
							[cardB, 0]
						])
					: new Map<HTMLElement, number>([
							[cardA, 0],
							[cardB, 140]
						]);
			}
			return swapped
				? new Map<HTMLElement, number>([
						[yamlA, 200],
						[yamlB, 0]
					])
				: new Map<HTMLElement, number>([
						[yamlA, 0],
						[yamlB, 200]
					]);
		});

		const handle = createMultiListLayout(
			[
				{ root: cardsRoot },
				{ root: yamlRoot, children: DEFAULT_YAML_LAYOUT_CHILDREN }
			],
			{ measureTops, animate: animateFn }
		);

		const mutate = vi.fn();
		await handle.run(mutate);

		expect(mutate).toHaveBeenCalledTimes(1);
		expect(animateFn).toHaveBeenCalledWith(
			[cardA, cardB, yamlA, yamlB],
			expect.objectContaining({
				y: [-140, 140, -200, 200],
				duration: LAYOUT_SWAP_DURATION_MS,
				ease: LAYOUT_SWAP_EASE
			})
		);
	});
});
