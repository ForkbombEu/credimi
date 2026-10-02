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

import { browserClock, type ComposerClock } from './composer-clock.js';

/** Rough collapsed StepCard height used before measureElement runs. */
export const DEFAULT_STEP_ESTIMATE_SIZE = 140;

/** Rough collapsed YAML step block height before measureElement runs. */
export const DEFAULT_YAML_STEP_ESTIMATE_SIZE = 96;

/** Keep ±1 neighbors mounted near the viewport edge for reorder animation / preview. */
export const DEFAULT_OVERSCAN = 8;

export const DEFAULT_ENSURE_VISIBLE_TIMEOUT_MS = 2000;

export type EnsureStepVisibleOptions = ScrollToOptions & {
	timeoutMs?: number;
};

export type ComposerVirtualizerOptions = {
	getCount: () => number;
	getScrollElement: () => HTMLElement | null;
	/** Selector for the mounted item used by `ensureStepVisible`. */
	itemSelector: (index: number) => string;
	estimateSize: (index: number) => number;
	/**
	 * Stable identity for TanStack `itemSizeCache` (and `{#each}` keys).
	 * Must follow the step across reorder — default index keys leave the old
	 * height on the old slot, so swapped cards/YAML blocks overlap.
	 */
	getItemKey?: (index: number) => string | number;
	overscan?: number;
	/** Offset of the virtual list within a shared scroller (e.g. YAML header height). */
	getScrollMargin?: () => number;
	ensureVisibleTimeoutMs?: number;
	clock?: ComposerClock;
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
	 * After a reorder: bump `getItemKey` identity so TanStack rememos row→key
	 * mapping, then restore the scroller's `scrollTop` (rememo from index 0
	 * otherwise jumps mid-list viewports to the start).
	 */
	syncAfterReorder: () => void;
	/**
	 * Scroll index into view, then wait until `itemSelector(index)` is mounted in the
	 * scroller or until timeout.
	 */
	ensureStepVisible: (index: number, options?: EnsureStepVisibleOptions) => Promise<boolean>;
	dispose: () => void;
};

export function stepCardSelector(index: number): string {
	return `[data-card-section="steps"][data-card-index="${index}"]`;
}

export function yamlStepBlockSelector(index: number): string {
	return `[data-yaml-section="steps"][data-yaml-index="${index}"]`;
}

const itemKeyIds = new WeakMap<object, number>();
let nextItemKeyId = 1;

/** Stable primitive key for a step tuple (survives splice reorder). */
export function stableItemKey(item: object | null | undefined, fallback: number): number {
	if (item == null) return fallback;
	let id = itemKeyIds.get(item);
	if (id == null) {
		id = nextItemKeyId++;
		itemKeyIds.set(item, id);
	}
	return id;
}

/** Pin scrollTop on a scroller (and optional TanStack instance) after a layout rememo. */
export function restoreScrollTop(
	scroller: HTMLElement | null | undefined,
	top: number,
	virtualizer?: { scrollOffset: number | null }
): void {
	if (!scroller) return;
	if (scroller.scrollTop !== top) scroller.scrollTop = top;
	if (virtualizer && virtualizer.scrollOffset !== top) {
		virtualizer.scrollOffset = top;
	}
}

/**
 * Poll until `scrollElement` contains `selector`, or `timeoutMs` elapses.
 * Returns true when found.
 */
export function waitForSelectorInScroller(
	scrollElement: ParentNode,
	selector: string,
	timeoutMs: number,
	clock: ComposerClock = browserClock()
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
	const clock = options.clock ?? browserClock();
	const estimateSize = options.estimateSize;
	const overscan = options.overscan ?? DEFAULT_OVERSCAN;
	const ensureVisibleTimeoutMs =
		options.ensureVisibleTimeoutMs ?? DEFAULT_ENSURE_VISIBLE_TIMEOUT_MS;
	const getScrollMargin = options.getScrollMargin;
	const itemSelector = options.itemSelector;
	const getItemKey = options.getItemKey;

	/** Stable wrapper — only replaced in `syncAfterReorder` to force TanStack rememo. */
	let itemKeyFn = getItemKey ? (index: number) => getItemKey(index) : undefined;

	const readItemKeys = (count: number) => {
		if (!getItemKey) return;
		for (let i = 0; i < count; i++) getItemKey(i);
	};

	const virtualizerOptions = (count: number, scrollMargin?: number) => ({
		count,
		getScrollElement: () => options.getScrollElement(),
		estimateSize,
		overscan,
		...(itemKeyFn ? { getItemKey: itemKeyFn } : {}),
		...(scrollMargin !== undefined ? { scrollMargin } : {})
	});

	const initialCount = options.getCount();
	const virtualizer = create<HTMLElement, Element>(
		virtualizerOptions(initialCount, getScrollMargin?.())
	);

	const syncOptions = () => {
		const count = options.getCount();
		// Synchronous read so rune-backed scrollport rebinds re-run the `$effect` (TanStack #866).
		options.getScrollElement();
		readItemKeys(count);
		const scrollMargin = getScrollMargin?.();
		get(virtualizer).setOptions(virtualizerOptions(count, scrollMargin));
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
		syncAfterReorder() {
			if (disposed) return;
			const scroller = options.getScrollElement();
			const top = scroller?.scrollTop ?? 0;
			if (getItemKey) {
				itemKeyFn = (index: number) => getItemKey(index);
			}
			syncOptions();
			restoreScrollTop(scroller, top, get(virtualizer));
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
