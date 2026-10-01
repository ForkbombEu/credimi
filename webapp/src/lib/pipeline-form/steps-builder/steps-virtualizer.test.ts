// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { readable, type Readable } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { SvelteVirtualizer, VirtualItem } from '@tanstack/svelte-virtual';

import {
	createStepsVirtualizer,
	DEFAULT_OVERSCAN,
	DEFAULT_STEP_ESTIMATE_SIZE,
	stepCardSelector,
	waitForSelectorInScroller,
	type StepsVirtualizerClock
} from './steps-virtualizer.svelte.js';

type FakeClock = StepsVirtualizerClock & {
	flushRaf(): void;
	advance(ms: number): void;
};

function createFakeClock(): FakeClock {
	let nextRaf = 1;
	const rafQueue = new Map<number, FrameRequestCallback>();
	let nextTimer = 1;
	const timers = new Map<number, { due: number; handler: () => void }>();
	let now = 0;

	return {
		now: () => now,
		raf(callback) {
			const id = nextRaf++;
			rafQueue.set(id, callback);
			return id;
		},
		cancelRaf(handle) {
			rafQueue.delete(handle);
		},
		setTimeout(handler, timeout = 0) {
			const id = nextTimer++;
			timers.set(id, { due: now + timeout, handler });
			return id as unknown as ReturnType<typeof setTimeout>;
		},
		clearTimeout(handle) {
			timers.delete(handle as unknown as number);
		},
		flushRaf() {
			const queued = [...rafQueue.entries()];
			rafQueue.clear();
			for (const [, cb] of queued) cb(now);
		},
		advance(ms) {
			now += ms;
			const due = [...timers.entries()].filter(([, t]) => t.due <= now);
			for (const [id, t] of due) {
				timers.delete(id);
				t.handler();
			}
		}
	};
}

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
			const match = selector.match(
				/\[data-card-section="([^"]+)"\]\[data-card-index="([^"]+)"\]/
			);
			if (!match) return [];
			const [, section, index] = match;
			const out: ElementStub[] = [];
			const walk = (node: ElementStub) => {
				if (
					node.getAttribute('data-card-section') === section &&
					node.getAttribute('data-card-index') === index
				) {
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

describe('stepCardSelector', () => {
	it('targets steps section by card index', () => {
		expect(stepCardSelector(12)).toBe('[data-card-section="steps"][data-card-index="12"]');
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

describe('createStepsVirtualizer', () => {
	const disposers: Array<() => void> = [];

	afterEach(() => {
		for (const dispose of disposers.splice(0)) dispose();
	});

	it('uses StepCard-ish estimate and overscan defaults when creating the virtualizer', () => {
		const fake = createFakeVirtualizerStore();
		const createVirtualizer = vi.fn((_opts: unknown) => fake.store);

		const steps = createStepsVirtualizer({
			getCount: () => 10,
			getScrollElement: () => null,
			createVirtualizer: createVirtualizer as never
		});
		disposers.push(() => steps.dispose());

		expect(createVirtualizer).toHaveBeenCalledOnce();
		const opts = createVirtualizer.mock.calls[0]?.[0] as unknown as {
			count: number;
			estimateSize: (index: number) => number;
			overscan: number;
		};
		expect(opts.count).toBe(10);
		expect(opts.overscan).toBe(DEFAULT_OVERSCAN);
		expect(opts.estimateSize(0)).toBe(DEFAULT_STEP_ESTIMATE_SIZE);
	});

	it('pushes initial count through setOptions on mount', () => {
		const fake = createFakeVirtualizerStore();

		const steps = createStepsVirtualizer({
			getCount: () => 17,
			getScrollElement: () => null,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => steps.dispose());

		expect(fake.setOptions).toHaveBeenCalled();
		const last = fake.setOptions.mock.calls.at(-1)?.[0] as { count: number };
		expect(last.count).toBe(17);
	});

	it('delegates getVirtualItems, getTotalSize, and measureElement to the store instance', () => {
		const items: VirtualItem[] = [{ key: 0, index: 0, start: 0, end: 140, size: 140, lane: 0 }];
		const fake = createFakeVirtualizerStore({
			getVirtualItems: () => items,
			getTotalSize: () => 1400
		});

		const steps = createStepsVirtualizer({
			getCount: () => 10,
			getScrollElement: () => null,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => steps.dispose());

		expect(steps.getVirtualItems()).toBe(items);
		expect(steps.getTotalSize()).toBe(1400);

		const node = {} as Element;
		steps.measureElement(node);
		expect(fake.measureElement).toHaveBeenCalledWith(node);
	});

	it('scrollToIndex then waits until the step card exists in the scroller', async () => {
		const clock = createFakeClock();
		const scroller = createElementStub();
		const fake = createFakeVirtualizerStore();

		const steps = createStepsVirtualizer({
			getCount: () => 100,
			getScrollElement: () => scroller as unknown as HTMLElement,
			clock,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => steps.dispose());

		const pending = steps.ensureStepVisible(42, { align: 'start', behavior: 'auto' });

		expect(fake.scrollToIndex).toHaveBeenCalledWith(42, {
			align: 'start',
			behavior: 'auto'
		});

		clock.flushRaf();
		scroller.appendChild(
			createElementStub({ 'data-card-section': 'steps', 'data-card-index': '42' })
		);
		clock.flushRaf();

		await expect(pending).resolves.toBe(true);
	});

	it('returns false from ensureStepVisible when the card never mounts', async () => {
		const clock = createFakeClock();
		const scroller = createElementStub();
		const fake = createFakeVirtualizerStore();

		const steps = createStepsVirtualizer({
			getCount: () => 100,
			getScrollElement: () => scroller as unknown as HTMLElement,
			clock,
			ensureVisibleTimeoutMs: 40,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => steps.dispose());

		const pending = steps.ensureStepVisible(7);
		clock.flushRaf();
		clock.advance(40);

		await expect(pending).resolves.toBe(false);
		expect(fake.scrollToIndex).toHaveBeenCalledWith(7, {
			align: 'center',
			behavior: 'smooth'
		});
	});

	it('returns false from ensureStepVisible when there is no scroll element', async () => {
		const fake = createFakeVirtualizerStore();
		const steps = createStepsVirtualizer({
			getCount: () => 1,
			getScrollElement: () => null,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => steps.dispose());

		await expect(steps.ensureStepVisible(0)).resolves.toBe(false);
		expect(fake.scrollToIndex).toHaveBeenCalled();
	});
});
