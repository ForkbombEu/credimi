// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SvelteVirtualizer, VirtualItem } from '@tanstack/svelte-virtual';

import { readable, type Readable } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import {
	createComposerVirtualizer,
	DEFAULT_OVERSCAN,
	DEFAULT_STEP_ESTIMATE_SIZE,
	DEFAULT_YAML_STEP_ESTIMATE_SIZE,
	pinBothScrollports,
	stableItemKey,
	stepCardSelector,
	waitForSelectorInScroller,
	yamlStepBlockSelector
} from './composer-virtualizer.svelte.js';
import { createFakeClock } from './test-support/fake-clock.js';

type ElementStub = {
	getAttribute(name: string): string | null;
	setAttribute(name: string, value: string): void;
	appendChild(child: ElementStub): ElementStub;
	querySelector(selector: string): ElementStub | null;
	querySelectorAll(selector: string): ElementStub[];
	readonly _children: ElementStub[];
};

function createElementStub(attrs: Record<string, string> = {}): ElementStub {
	const children: ElementStub[] = [];
	const store = { ...attrs };

	const el: ElementStub = {
		getAttribute(name) {
			return store[name] ?? null;
		},
		setAttribute(name, value) {
			store[name] = value;
		},
		appendChild(child) {
			children.push(child);
			return child;
		},
		querySelector(selector) {
			return el.querySelectorAll(selector)[0] ?? null;
		},
		querySelectorAll(selector) {
			const pairs = [...selector.matchAll(/\[([^=\]]+)="([^"]*)"\]/g)].map(
				(m) => [m[1]!, m[2]!] as const
			);
			if (pairs.length === 0) return [];
			const out: ElementStub[] = [];
			const walk = (node: ElementStub) => {
				if (pairs.every(([name, value]) => node.getAttribute(name) === value)) {
					out.push(node);
				}
				for (const child of node._children) walk(child);
			};
			walk(el);
			return out;
		},
		get _children() {
			return children;
		}
	};

	return el;
}

function createFakeVirtualizerStore(overrides?: {
	scrollToIndex?: ReturnType<typeof vi.fn>;
	getVirtualItems?: () => VirtualItem[];
	getTotalSize?: () => number;
	measureElement?: ReturnType<typeof vi.fn>;
	setOptions?: ReturnType<typeof vi.fn>;
}): {
	store: Readable<SvelteVirtualizer<HTMLElement, Element>>;
	scrollToIndex: ReturnType<typeof vi.fn>;
	setOptions: ReturnType<typeof vi.fn>;
	measureElement: ReturnType<typeof vi.fn>;
} {
	const scrollToIndex = overrides?.scrollToIndex ?? vi.fn();
	const setOptions = overrides?.setOptions ?? vi.fn();
	const measureElement = overrides?.measureElement ?? vi.fn();
	const instance = {
		scrollToIndex,
		setOptions,
		measureElement,
		getVirtualItems: overrides?.getVirtualItems ?? (() => []),
		getTotalSize: overrides?.getTotalSize ?? (() => 0)
	} as unknown as SvelteVirtualizer<HTMLElement, Element>;

	return {
		store: readable(instance),
		scrollToIndex,
		setOptions,
		measureElement
	};
}

describe('pinBothScrollports', () => {
	it('pins DOM scrollTop and TanStack scrollOffset on both panes', () => {
		const cards = { scrollTop: 999 } as HTMLElement;
		const yaml = { scrollTop: 888 } as HTMLElement;
		const cardsVirt = { scrollOffset: 999 as number | null };
		const yamlVirt = { scrollOffset: 888 as number | null };

		pinBothScrollports(
			{ scroller: cards, top: 40, virtualizer: cardsVirt },
			{ scroller: yaml, top: 50, virtualizer: yamlVirt }
		);

		expect(cards.scrollTop).toBe(40);
		expect(yaml.scrollTop).toBe(50);
		expect(cardsVirt.scrollOffset).toBe(40);
		expect(yamlVirt.scrollOffset).toBe(50);
	});

	it('forwards both panes through an injected restore', () => {
		const calls: Array<{ top: number; hasVirt: boolean }> = [];
		pinBothScrollports(
			{ scroller: { scrollTop: 0 } as HTMLElement, top: 10, virtualizer: { scrollOffset: 0 } },
			{ scroller: { scrollTop: 0 } as HTMLElement, top: 20, virtualizer: { scrollOffset: 0 } },
			(_el, top, virtualizer) => {
				calls.push({ top, hasVirt: virtualizer != null });
			}
		);
		expect(calls).toEqual([
			{ top: 10, hasVirt: true },
			{ top: 20, hasVirt: true }
		]);
	});
});

describe('item selectors', () => {
	it.each([
		{
			name: 'step card',
			selector: stepCardSelector,
			expected: '[data-card-section="steps"][data-card-index="12"]'
		},
		{
			name: 'yaml step block',
			selector: yamlStepBlockSelector,
			expected: '[data-yaml-section="steps"][data-yaml-index="12"]'
		}
	])('targets $name by index', ({ selector, expected }) => {
		expect(selector(12)).toBe(expected);
	});
});

describe('waitForSelectorInScroller', () => {
	it('resolves true when the card appears before timeout', async () => {
		const clock = createFakeClock();
		const scroller = createElementStub();
		const pending = waitForSelectorInScroller(
			scroller as unknown as ParentNode,
			stepCardSelector(3),
			1000,
			clock
		);

		clock.flushRaf();
		scroller.appendChild(
			createElementStub({ 'data-card-section': 'steps', 'data-card-index': '3' })
		);
		clock.flushRaf();

		await expect(pending).resolves.toBe(true);
	});

	it('resolves false when the deadline elapses without a match', async () => {
		const clock = createFakeClock();
		const scroller = createElementStub();
		const pending = waitForSelectorInScroller(
			scroller as unknown as ParentNode,
			stepCardSelector(0),
			50,
			clock
		);

		clock.flushRaf();
		clock.advance(50);

		await expect(pending).resolves.toBe(false);
	});
});

describe('createComposerVirtualizer', () => {
	const disposers: Array<() => void> = [];

	afterEach(() => {
		for (const dispose of disposers.splice(0)) dispose();
	});

	it.each([
		{
			name: 'steps cards',
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			expectedEstimate: DEFAULT_STEP_ESTIMATE_SIZE,
			getScrollMargin: undefined as (() => number) | undefined,
			expectedScrollMargin: undefined as number | undefined,
			mountAttrs: { 'data-card-section': 'steps', 'data-card-index': '42' }
		},
		{
			name: 'yaml steps',
			itemSelector: yamlStepBlockSelector,
			estimateSize: () => DEFAULT_YAML_STEP_ESTIMATE_SIZE,
			expectedEstimate: DEFAULT_YAML_STEP_ESTIMATE_SIZE,
			getScrollMargin: () => 48,
			expectedScrollMargin: 48,
			mountAttrs: { 'data-yaml-section': 'steps', 'data-yaml-index': '42' }
		}
	])(
		'$name: defaults, scrollMargin, and ensureStepVisible selector',
		async ({
			itemSelector,
			estimateSize,
			expectedEstimate,
			getScrollMargin,
			expectedScrollMargin,
			mountAttrs
		}) => {
			const fake = createFakeVirtualizerStore();
			const createVirtualizer = vi.fn(() => fake.store);
			const clock = createFakeClock();
			const scroller = createElementStub();

			const list = createComposerVirtualizer({
				getCount: () => 10,
				getScrollElement: () => scroller as unknown as HTMLElement,
				itemSelector,
				estimateSize,
				getScrollMargin,
				clock,
				createVirtualizer: createVirtualizer as never
			});
			disposers.push(() => list.dispose());

			expect(createVirtualizer).toHaveBeenCalledOnce();
			const opts = createVirtualizer.mock.calls[0]?.[0] as unknown as {
				count: number;
				estimateSize: (index: number) => number;
				overscan: number;
				scrollMargin?: number;
			};
			expect(opts.count).toBe(10);
			expect(opts.overscan).toBe(DEFAULT_OVERSCAN);
			expect(opts.estimateSize(0)).toBe(expectedEstimate);
			expect(opts.scrollMargin).toBe(expectedScrollMargin);

			const last = fake.setOptions.mock.calls.at(-1)?.[0] as {
				count: number;
				scrollMargin?: number;
			};
			expect(last.count).toBe(10);
			expect(last.scrollMargin).toBe(expectedScrollMargin);

			const pending = list.ensureStepVisible(42, { align: 'start', behavior: 'auto' });
			expect(fake.scrollToIndex).toHaveBeenCalledWith(42, {
				align: 'start',
				behavior: 'auto'
			});

			clock.flushRaf();
			scroller.appendChild(createElementStub(mountAttrs));
			clock.flushRaf();

			await expect(pending).resolves.toBe(true);
		}
	);

	it('delegates getVirtualItems, getTotalSize, and measureElement to the store instance', () => {
		const items: VirtualItem[] = [{ key: 0, index: 0, start: 0, end: 140, size: 140, lane: 0 }];
		const fake = createFakeVirtualizerStore({
			getVirtualItems: () => items,
			getTotalSize: () => 1400
		});

		const list = createComposerVirtualizer({
			getCount: () => 10,
			getScrollElement: () => null,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => list.dispose());

		expect(list.getVirtualItems()).toBe(items);
		expect(list.getTotalSize()).toBe(1400);

		const node = {} as Element;
		list.measureElement(node);
		expect(fake.measureElement).toHaveBeenCalledWith(node);
	});

	it('returns false from ensureStepVisible when the item never mounts', async () => {
		const clock = createFakeClock();
		const scroller = createElementStub();
		const fake = createFakeVirtualizerStore();

		const list = createComposerVirtualizer({
			getCount: () => 100,
			getScrollElement: () => scroller as unknown as HTMLElement,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			clock,
			ensureVisibleTimeoutMs: 40,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => list.dispose());

		const pending = list.ensureStepVisible(7);
		clock.flushRaf();
		clock.advance(40);

		await expect(pending).resolves.toBe(false);
		expect(fake.scrollToIndex).toHaveBeenCalledWith(7, {
			align: 'center',
			behavior: 'auto'
		});
	});

	it('returns false from ensureStepVisible when there is no scroll element', async () => {
		const fake = createFakeVirtualizerStore();
		const list = createComposerVirtualizer({
			getCount: () => 1,
			getScrollElement: () => null,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => list.dispose());

		await expect(list.ensureStepVisible(0)).resolves.toBe(false);
		expect(fake.scrollToIndex).not.toHaveBeenCalled();
	});

	it('mountOnly skips scrollToIndex when the item is already mounted', async () => {
		const fake = createFakeVirtualizerStore();
		const scroller = createElementStub();
		scroller.appendChild(
			createElementStub({ 'data-card-section': 'steps', 'data-card-index': '2' })
		);

		const list = createComposerVirtualizer({
			getCount: () => 10,
			getScrollElement: () => scroller as unknown as HTMLElement,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => list.dispose());

		await expect(list.ensureStepVisible(2, { mountOnly: true, align: 'start' })).resolves.toBe(
			true
		);
		expect(fake.scrollToIndex).not.toHaveBeenCalled();
	});

	it('mountOnly still scrollToIndex when the item is not mounted', async () => {
		const clock = createFakeClock();
		const fake = createFakeVirtualizerStore();
		const scroller = createElementStub();

		const list = createComposerVirtualizer({
			getCount: () => 10,
			getScrollElement: () => scroller as unknown as HTMLElement,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			clock,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => list.dispose());

		const pending = list.ensureStepVisible(2, {
			mountOnly: true,
			align: 'start',
			behavior: 'auto'
		});
		expect(fake.scrollToIndex).toHaveBeenCalledWith(2, {
			align: 'start',
			behavior: 'auto'
		});

		clock.flushRaf();
		scroller.appendChild(
			createElementStub({ 'data-card-section': 'steps', 'data-card-index': '2' })
		);
		clock.flushRaf();

		await expect(pending).resolves.toBe(true);
	});

	it('without mountOnly, already-mounted ensure still scrollToIndex (reconcile race source)', async () => {
		/**
		 * Smoke diagnosis: before twin-pane `mountSwapIndices` used mountOnly, swap mount
		 * called ensureStepVisible(align:start) even when both swap cards were already in
		 * the DOM. TanStack scrollToIndex schedules reconcileScroll via rAF; restoreScrollTop
		 * then loses to that reconcile → jump toward align-start (often scrollTop≈0 for index 0).
		 */
		let scrollTop = 200;
		let deferredReconcile: (() => void) | null = null;
		const scroller = createElementStub();
		scroller.appendChild(
			createElementStub({ 'data-card-section': 'steps', 'data-card-index': '0' })
		);
		Object.defineProperty(scroller, 'scrollTop', {
			configurable: true,
			get: () => scrollTop,
			set: (v: number) => {
				scrollTop = v;
			}
		});

		const scrollToIndex = vi.fn((_index: number, _opts: unknown) => {
			// Immediate write like TanStack _scrollToOffset, then deferred reconcileScroll.
			scrollTop = 0;
			deferredReconcile = () => {
				scrollTop = 0;
			};
		});
		const fake = createFakeVirtualizerStore({ scrollToIndex });

		const list = createComposerVirtualizer({
			getCount: () => 10,
			getScrollElement: () => scroller as unknown as HTMLElement,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => list.dispose());

		await list.ensureStepVisible(0, { align: 'start', behavior: 'auto' });
		expect(scrollToIndex).toHaveBeenCalledOnce();
		// Caller restores mid-list pin…
		scrollTop = 200;
		// …then TanStack reconcileScroll rAF re-applies align-start target.
		deferredReconcile?.();
		expect(scrollTop).toBe(0);

		scrollToIndex.mockClear();
		deferredReconcile = null;
		scrollTop = 200;
		await list.ensureStepVisible(0, { mountOnly: true, align: 'start', behavior: 'auto' });
		expect(scrollToIndex).not.toHaveBeenCalled();
		deferredReconcile?.();
		expect(scrollTop).toBe(200);
	});

	it('keeps WeakMap keys stable across reorder and maps missing items to fallback', () => {
		const a = { id: 'a' };
		const b = { id: 'b' };
		const ka = stableItemKey(a, 0);
		const kb = stableItemKey(b, 1);
		expect(ka).not.toBe(kb);
		expect(stableItemKey(a, 99)).toBe(ka);
		expect(stableItemKey(undefined, 7)).toBe(7);
	});

	it('passes getItemKey through and remaps it only on syncAfterReorder', () => {
		const keys = ['a', 'b'];
		const fake = createFakeVirtualizerStore();
		const list = createComposerVirtualizer({
			getCount: () => keys.length,
			getScrollElement: () => null,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			getItemKey: (index) => keys[index] ?? index,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => list.dispose());

		const first = fake.setOptions.mock.calls.at(-1)?.[0] as {
			getItemKey?: (index: number) => string | number;
		};
		expect(first.getItemKey?.(0)).toBe('a');
		expect(first.getItemKey?.(1)).toBe('b');

		keys.reverse();
		list.syncAfterReorder();

		const after = fake.setOptions.mock.calls.at(-1)?.[0] as {
			getItemKey?: (index: number) => string | number;
		};
		expect(after.getItemKey).not.toBe(first.getItemKey);
		expect(after.getItemKey?.(0)).toBe('b');
		expect(after.getItemKey?.(1)).toBe('a');
	});

	it('restores scrollTop after syncAfterReorder rememo', () => {
		const scroller = { _top: 480 } as HTMLElement & { _top: number };
		Object.defineProperty(scroller, 'scrollTop', {
			configurable: true,
			get() {
				return scroller._top;
			},
			set(v: number) {
				scroller._top = v;
			}
		});

		const instance = {
			scrollToIndex: vi.fn(),
			setOptions: vi.fn(() => {
				scroller._top = 0;
			}),
			measureElement: vi.fn(),
			getVirtualItems: () => [],
			getTotalSize: () => 0,
			scrollOffset: 480 as number | null
		};
		const store = readable(instance as unknown as SvelteVirtualizer<HTMLElement, Element>);

		const list = createComposerVirtualizer({
			getCount: () => 2,
			getScrollElement: () => scroller,
			itemSelector: stepCardSelector,
			estimateSize: () => DEFAULT_STEP_ESTIMATE_SIZE,
			getItemKey: (index) => index,
			createVirtualizer: (() => store) as never
		});
		disposers.push(() => list.dispose());

		scroller._top = 480;
		instance.scrollOffset = 480;
		list.syncAfterReorder();

		expect(scroller._top).toBe(480);
		expect(instance.scrollOffset).toBe(480);
	});
});
