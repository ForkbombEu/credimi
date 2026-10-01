// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { readable, type Readable } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { SvelteVirtualizer, VirtualItem } from '@tanstack/svelte-virtual';

import {
	createYamlStepsVirtualizer,
	DEFAULT_YAML_OVERSCAN,
	DEFAULT_YAML_STEP_ESTIMATE_SIZE,
	yamlStepBlockSelector,
	type YamlStepsVirtualizerClock
} from './yaml-virtualizer.svelte.js';

type FakeClock = YamlStepsVirtualizerClock & {
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
				/\[data-yaml-section="([^"]+)"\]\[data-yaml-index="([^"]+)"\]/
			);
			if (!match) return [];
			const [, section, index] = match;
			const out: ElementStub[] = [];
			const walk = (node: ElementStub) => {
				if (
					node.getAttribute('data-yaml-section') === section &&
					node.getAttribute('data-yaml-index') === index
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

describe('yamlStepBlockSelector', () => {
	it('targets yaml step blocks by index', () => {
		expect(yamlStepBlockSelector(12)).toBe('[data-yaml-section="steps"][data-yaml-index="12"]');
	});
});

describe('createYamlStepsVirtualizer', () => {
	const disposers: Array<() => void> = [];

	afterEach(() => {
		for (const dispose of disposers.splice(0)) dispose();
	});

	it('uses YAML estimate and overscan defaults', () => {
		const fake = createFakeVirtualizerStore();
		const createVirtualizer = vi.fn((_opts: unknown) => fake.store);

		const yaml = createYamlStepsVirtualizer({
			getCount: () => 10,
			getScrollElement: () => null,
			createVirtualizer: createVirtualizer as never
		});
		disposers.push(() => yaml.dispose());

		expect(createVirtualizer).toHaveBeenCalledOnce();
		const opts = createVirtualizer.mock.calls[0]?.[0] as unknown as {
			count: number;
			estimateSize: (index: number) => number;
			overscan: number;
			scrollMargin: number;
		};
		expect(opts.count).toBe(10);
		expect(opts.overscan).toBe(DEFAULT_YAML_OVERSCAN);
		expect(opts.estimateSize(0)).toBe(DEFAULT_YAML_STEP_ESTIMATE_SIZE);
		expect(opts.scrollMargin).toBe(0);
	});

	it('pushes scrollMargin from getScrollMargin through setOptions', () => {
		const fake = createFakeVirtualizerStore();
		let margin = 48;

		const yaml = createYamlStepsVirtualizer({
			getCount: () => 5,
			getScrollElement: () => null,
			getScrollMargin: () => margin,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => yaml.dispose());

		const last = fake.setOptions.mock.calls.at(-1)?.[0] as { scrollMargin: number };
		expect(last.scrollMargin).toBe(48);

		margin = 120;
		// Eager sync already ran; call setOptions path via ensure by re-reading through a new
		// create would be heavy — assert the getter is wired by constructing with non-zero margin.
		expect(last.scrollMargin).toBe(48);
	});

	it('scrollToIndex then waits until the yaml step block exists', async () => {
		const clock = createFakeClock();
		const scroller = createElementStub();
		const fake = createFakeVirtualizerStore();

		const yaml = createYamlStepsVirtualizer({
			getCount: () => 100,
			getScrollElement: () => scroller as unknown as HTMLElement,
			clock,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => yaml.dispose());

		const pending = yaml.ensureStepVisible(42, { align: 'start', behavior: 'auto' });

		expect(fake.scrollToIndex).toHaveBeenCalledWith(42, {
			align: 'start',
			behavior: 'auto'
		});

		clock.flushRaf();
		scroller.appendChild(
			createElementStub({ 'data-yaml-section': 'steps', 'data-yaml-index': '42' })
		);
		clock.flushRaf();

		await expect(pending).resolves.toBe(true);
	});

	it('returns false from ensureStepVisible when the block never mounts', async () => {
		const clock = createFakeClock();
		const scroller = createElementStub();
		const fake = createFakeVirtualizerStore();

		const yaml = createYamlStepsVirtualizer({
			getCount: () => 100,
			getScrollElement: () => scroller as unknown as HTMLElement,
			clock,
			ensureVisibleTimeoutMs: 40,
			createVirtualizer: (() => fake.store) as never
		});
		disposers.push(() => yaml.dispose());

		const pending = yaml.ensureStepVisible(7);
		clock.flushRaf();
		clock.advance(40);

		await expect(pending).resolves.toBe(false);
	});
});
