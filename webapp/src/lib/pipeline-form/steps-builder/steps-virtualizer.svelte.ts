// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	createVirtualizer,
	type ScrollToOptions,
	type SvelteVirtualizer,
	type VirtualItem
} from '@tanstack/svelte-virtual';
import { get, type Readable } from 'svelte/store';

/** Rough collapsed StepCard height used before measureElement runs. */
export const DEFAULT_STEP_ESTIMATE_SIZE = 140;

/** Keep ±1 neighbors mounted near the viewport edge for animate:flip. */
export const DEFAULT_OVERSCAN = 8;

export const DEFAULT_ENSURE_VISIBLE_TIMEOUT_MS = 2000;

export type StepsVirtualizerClock = {
	now: () => number;
	raf: (callback: FrameRequestCallback) => number;
	cancelRaf: (handle: number) => void;
	setTimeout: (handler: () => void, timeout?: number) => ReturnType<typeof setTimeout>;
	clearTimeout: (handle: ReturnType<typeof setTimeout>) => void;
};

export type EnsureStepVisibleOptions = ScrollToOptions & {
	timeoutMs?: number;
};

export type StepsVirtualizerOptions = {
	getCount: () => number;
	getScrollElement: () => HTMLElement | null;
	estimateSize?: (index: number) => number;
	overscan?: number;
	ensureVisibleTimeoutMs?: number;
	clock?: StepsVirtualizerClock;
	/**
	 * Injected for tests. Defaults to `@tanstack/svelte-virtual` createVirtualizer.
	 */
	createVirtualizer?: typeof createVirtualizer;
};

export type StepsVirtualizer = {
	/** TanStack Readable store — subscribe with `$virtualizer` in components. */
	virtualizer: Readable<SvelteVirtualizer<HTMLElement, Element>>;
	getVirtualItems: () => VirtualItem[];
	getTotalSize: () => number;
	measureElement: (node: Element | null) => void;
	/**
	 * Scroll index into view, then wait until the step card is mounted in the scroller
	 * (`[data-card-section=steps][data-card-index="…"]`) or until timeout.
	 */
	ensureStepVisible: (index: number, options?: EnsureStepVisibleOptions) => Promise<boolean>;
	dispose: () => void;
};

function defaultClock(): StepsVirtualizerClock {
	return {
		now: () => globalThis.performance?.now?.() ?? Date.now(),
		raf: (...args) => globalThis.requestAnimationFrame(...args),
		cancelRaf: (...args) => globalThis.cancelAnimationFrame(...args),
		setTimeout: (...args) => globalThis.setTimeout(...args),
		clearTimeout: (...args) => globalThis.clearTimeout(...args)
	};
}

export function stepCardSelector(index: number): string {
	return `[data-card-section="steps"][data-card-index="${index}"]`;
}

/**
 * Poll until `scrollElement` contains `selector`, or `timeoutMs` elapses.
 * Returns true when found.
 */
export function waitForSelectorInScroller(
	scrollElement: ParentNode,
	selector: string,
	timeoutMs: number,
	clock: StepsVirtualizerClock = defaultClock()
): Promise<boolean> {
	const deadline = clock.now() + timeoutMs;

	return new Promise((resolve) => {
		let rafHandle: number | null = null;
		let timeoutHandle: ReturnType<typeof setTimeout> | null = null;
		let settled = false;

		const finish = (found: boolean) => {
			if (settled) return;
			settled = true;
			if (rafHandle != null) clock.cancelRaf(rafHandle);
			if (timeoutHandle != null) clock.clearTimeout(timeoutHandle);
			resolve(found);
		};

		const poll = () => {
			if (scrollElement.querySelector(selector)) {
				finish(true);
				return;
			}
			if (clock.now() >= deadline) {
				finish(false);
				return;
			}
			rafHandle = clock.raf(poll);
		};

		timeoutHandle = clock.setTimeout(() => finish(false), timeoutMs);
		poll();
	});
}

/**
 * Svelte 5-safe wrapper around TanStack `createVirtualizer` for Pipeline Composer step cards.
 *
 * The store adapter does not auto-track option getters, so count (and scroll element identity)
 * are pushed via `setOptions` inside an `$effect.root` (TanStack Virtual #866 / Table guide).
 */
export function createStepsVirtualizer(options: StepsVirtualizerOptions): StepsVirtualizer {
	const create = options.createVirtualizer ?? createVirtualizer;
	const clock = options.clock ?? defaultClock();
	const estimateSize = options.estimateSize ?? (() => DEFAULT_STEP_ESTIMATE_SIZE);
	const overscan = options.overscan ?? DEFAULT_OVERSCAN;
	const ensureVisibleTimeoutMs =
		options.ensureVisibleTimeoutMs ?? DEFAULT_ENSURE_VISIBLE_TIMEOUT_MS;

	const virtualizer = create<HTMLElement, Element>({
		count: options.getCount(),
		getScrollElement: () => options.getScrollElement(),
		estimateSize,
		overscan
	});

	const syncOptions = () => {
		const count = options.getCount();
		// Synchronous read so rune-backed scrollport rebinds re-run the `$effect` (TanStack #866).
		options.getScrollElement();
		get(virtualizer).setOptions({
			count,
			getScrollElement: () => options.getScrollElement(),
			estimateSize,
			overscan
		});
	};

	// Eager sync so consumers (and unit tests) see the current count without waiting for
	// a browser `$effect` flush. The effect below re-runs when rune-backed getters change.
	syncOptions();

	const destroyEffect = $effect.root(() => {
		$effect(() => {
			// getCount / getScrollElement reads are tracked when called from syncOptions.
			syncOptions();
		});
	});

	let disposed = false;

	return {
		virtualizer,
		getVirtualItems() {
			return get(virtualizer).getVirtualItems();
		},
		getTotalSize() {
			return get(virtualizer).getTotalSize();
		},
		measureElement(node) {
			if (disposed) return;
			get(virtualizer).measureElement(node);
		},
		async ensureStepVisible(index, ensureOptions = {}) {
			if (disposed) return false;
			const { timeoutMs = ensureVisibleTimeoutMs, ...scrollOptions } = ensureOptions;
			// Default `auto`: estimated-size smooth scrollToIndex fights measureElement and
			// any caller smooth realign. Pass `behavior: 'smooth'` only when this is the
			// sole scroll (e.g. focus a newly created card).
			get(virtualizer).scrollToIndex(index, {
				align: scrollOptions.align ?? 'center',
				behavior: scrollOptions.behavior ?? 'auto'
			});
			const scrollElement = options.getScrollElement();
			if (!scrollElement) return false;
			return waitForSelectorInScroller(
				scrollElement,
				stepCardSelector(index),
				timeoutMs,
				clock
			);
		},
		dispose() {
			if (disposed) return;
			disposed = true;
			destroyEffect();
		}
	};
}
