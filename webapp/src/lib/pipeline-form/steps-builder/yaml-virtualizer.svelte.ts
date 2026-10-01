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

import {
	waitForSelectorInScroller,
	type StepsVirtualizerClock
} from './steps-virtualizer.svelte.js';

/** Rough collapsed YAML step block height before measureElement runs. */
export const DEFAULT_YAML_STEP_ESTIMATE_SIZE = 96;

/** Keep ±1 neighbors mounted near the viewport edge. */
export const DEFAULT_YAML_OVERSCAN = 8;

export const DEFAULT_YAML_ENSURE_VISIBLE_TIMEOUT_MS = 2000;

export type YamlStepsVirtualizerClock = StepsVirtualizerClock;

export type EnsureYamlStepVisibleOptions = ScrollToOptions & {
	timeoutMs?: number;
};

export type YamlStepsVirtualizerOptions = {
	getCount: () => number;
	getScrollElement: () => HTMLElement | null;
	/** Offset of the virtual step list within the shared YAML scroller (header height). */
	getScrollMargin?: () => number;
	estimateSize?: (index: number) => number;
	overscan?: number;
	ensureVisibleTimeoutMs?: number;
	clock?: YamlStepsVirtualizerClock;
	createVirtualizer?: typeof createVirtualizer;
};

export type YamlStepsVirtualizer = {
	virtualizer: Readable<SvelteVirtualizer<HTMLElement, Element>>;
	getVirtualItems: () => VirtualItem[];
	getTotalSize: () => number;
	measureElement: (node: Element | null) => void;
	ensureStepVisible: (index: number, options?: EnsureYamlStepVisibleOptions) => Promise<boolean>;
	dispose: () => void;
};

function defaultClock(): YamlStepsVirtualizerClock {
	return {
		now: () => globalThis.performance?.now?.() ?? Date.now(),
		raf: (...args) => globalThis.requestAnimationFrame(...args),
		cancelRaf: (...args) => globalThis.cancelAnimationFrame(...args),
		setTimeout: (...args) => globalThis.setTimeout(...args),
		clearTimeout: (...args) => globalThis.clearTimeout(...args)
	};
}

export function yamlStepBlockSelector(index: number): string {
	return `[data-yaml-section="steps"][data-yaml-index="${index}"]`;
}

/**
 * TanStack virtualizer for per-step Shiki blocks in the YAML preview pane.
 * Header and follow-ups stay outside this list; `scrollMargin` accounts for the header.
 */
export function createYamlStepsVirtualizer(
	options: YamlStepsVirtualizerOptions
): YamlStepsVirtualizer {
	const create = options.createVirtualizer ?? createVirtualizer;
	const clock = options.clock ?? defaultClock();
	const estimateSize = options.estimateSize ?? (() => DEFAULT_YAML_STEP_ESTIMATE_SIZE);
	const overscan = options.overscan ?? DEFAULT_YAML_OVERSCAN;
	const ensureVisibleTimeoutMs =
		options.ensureVisibleTimeoutMs ?? DEFAULT_YAML_ENSURE_VISIBLE_TIMEOUT_MS;
	const getScrollMargin = options.getScrollMargin ?? (() => 0);

	const virtualizer = create<HTMLElement, Element>({
		count: options.getCount(),
		getScrollElement: () => options.getScrollElement(),
		estimateSize,
		overscan,
		scrollMargin: getScrollMargin()
	});

	const syncOptions = () => {
		const count = options.getCount();
		options.getScrollElement();
		const scrollMargin = getScrollMargin();
		get(virtualizer).setOptions({
			count,
			getScrollElement: () => options.getScrollElement(),
			estimateSize,
			overscan,
			scrollMargin
		});
	};

	syncOptions();

	const destroyEffect = $effect.root(() => {
		$effect(() => {
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
			// Default `auto` — see steps-virtualizer ensureStepVisible.
			get(virtualizer).scrollToIndex(index, {
				align: scrollOptions.align ?? 'center',
				behavior: scrollOptions.behavior ?? 'auto'
			});
			const scrollElement = options.getScrollElement();
			if (!scrollElement) return false;
			return waitForSelectorInScroller(
				scrollElement,
				yamlStepBlockSelector(index),
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
