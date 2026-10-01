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

/** Rough collapsed YAML step block height before measureElement runs. */
export const DEFAULT_YAML_STEP_ESTIMATE_SIZE = 96;

/** Keep ±1 neighbors mounted near the viewport edge for animate:flip / preview. */
export const DEFAULT_OVERSCAN = 8;

export const DEFAULT_ENSURE_VISIBLE_TIMEOUT_MS = 2000;

export type ComposerVirtualizerClock = {
	now: () => number;
	raf: (callback: FrameRequestCallback) => number;
	cancelRaf: (handle: number) => void;
	setTimeout: (handler: () => void, timeout?: number) => ReturnType<typeof setTimeout>;
	clearTimeout: (handle: ReturnType<typeof setTimeout>) => void;
};

export type EnsureStepVisibleOptions = ScrollToOptions & {
	timeoutMs?: number;
};

export type ComposerVirtualizerOptions = {
	getCount: () => number;
	getScrollElement: () => HTMLElement | null;
	/** Selector for the mounted item used by `ensureStepVisible`. */
	itemSelector: (index: number) => string;
	estimateSize: (index: number) => number;
	overscan?: number;
	/** Offset of the virtual list within a shared scroller (e.g. YAML header height). */
	getScrollMargin?: () => number;
	ensureVisibleTimeoutMs?: number;
	clock?: ComposerVirtualizerClock;
	/**
	 * Injected for tests. Defaults to `@tanstack/svelte-virtual` createVirtualizer.
	 */
	createVirtualizer?: typeof createVirtualizer;
};

export type ComposerVirtualizer = {
	/** TanStack Readable store — subscribe with `$virtualizer` in components. */
	virtualizer: Readable<SvelteVirtualizer<HTMLElement, Element>>;
	getVirtualItems: () => VirtualItem[];
	getTotalSize: () => number;
	measureElement: (node: Element | null) => void;
	/**
	 * Scroll index into view, then wait until `itemSelector(index)` is mounted in the
	 * scroller or until timeout.
	 */
	ensureStepVisible: (index: number, options?: EnsureStepVisibleOptions) => Promise<boolean>;
	dispose: () => void;
};

function defaultClock(): ComposerVirtualizerClock {
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

export function yamlStepBlockSelector(index: number): string {
	return `[data-yaml-section="steps"][data-yaml-index="${index}"]`;
}

/**
 * Poll until `scrollElement` contains `selector`, or `timeoutMs` elapses.
 * Returns true when found.
 */
export function waitForSelectorInScroller(
	scrollElement: ParentNode,
	selector: string,
	timeoutMs: number,
	clock: ComposerVirtualizerClock = defaultClock()
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
 * Svelte 5-safe wrapper around TanStack `createVirtualizer` for Pipeline Composer lists
 * (step cards and YAML step blocks).
 *
 * The store adapter does not auto-track option getters, so count (and scroll element identity)
 * are pushed via `setOptions` inside an `$effect.root` (TanStack Virtual #866 / Table guide).
 */
export function createComposerVirtualizer(
	options: ComposerVirtualizerOptions
): ComposerVirtualizer {
	const create = options.createVirtualizer ?? createVirtualizer;
	const clock = options.clock ?? defaultClock();
	const estimateSize = options.estimateSize;
	const overscan = options.overscan ?? DEFAULT_OVERSCAN;
	const ensureVisibleTimeoutMs =
		options.ensureVisibleTimeoutMs ?? DEFAULT_ENSURE_VISIBLE_TIMEOUT_MS;
	const getScrollMargin = options.getScrollMargin;
	const itemSelector = options.itemSelector;

	const initialOptions = {
		count: options.getCount(),
		getScrollElement: () => options.getScrollElement(),
		estimateSize,
		overscan,
		...(getScrollMargin ? { scrollMargin: getScrollMargin() } : {})
	};

	const virtualizer = create<HTMLElement, Element>(initialOptions);

	const syncOptions = () => {
		const count = options.getCount();
		// Synchronous read so rune-backed scrollport rebinds re-run the `$effect` (TanStack #866).
		options.getScrollElement();
		const scrollMargin = getScrollMargin?.();
		get(virtualizer).setOptions({
			count,
			getScrollElement: () => options.getScrollElement(),
			estimateSize,
			overscan,
			...(scrollMargin !== undefined ? { scrollMargin } : {})
		});
	};

	// Eager sync so consumers (and unit tests) see the current count without waiting for
	// a browser `$effect` flush. The effect below re-runs when rune-backed getters change.
	syncOptions();

	const destroyEffect = $effect.root(() => {
		$effect(() => {
			// getCount / getScrollElement / getScrollMargin reads are tracked via syncOptions.
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
			return waitForSelectorInScroller(scrollElement, itemSelector(index), timeoutMs, clock);
		},
		dispose() {
			if (disposed) return;
			disposed = true;
			destroyEffect();
		}
	};
}
