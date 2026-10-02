// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { animate, type JSAnimation } from 'animejs';

/**
 * Duration / ease for step-card reorder (shift up/down).
 * Matches former `animate:flip={{ duration: 300 }}`; `out(3)` matches peer-scroll.
 */
export const LAYOUT_SWAP_DURATION_MS = 300;
export const LAYOUT_SWAP_EASE = 'out(3)';

export const DEFAULT_CARD_LAYOUT_CHILDREN = '[data-card-section="steps"]';
export const DEFAULT_YAML_LAYOUT_CHILDREN = '[data-yaml-section="steps"]';

export type FlipInvert = {
	el: HTMLElement;
	/** translateY needed so the node stays at `first` after `last` placement. */
	y: number;
};

export type CardListLayout = {
	/**
	 * FLIP around a DOM mutation. Record first tops, run `mutate` (must
	 * `await tick()` so TanStack `top` updates), then invert + tween `y` only.
	 * Never writes `top` / `position` — the virtualizer owns those.
	 */
	run(mutate: () => void | Promise<void>): Promise<void>;
	dispose(): void;
};

export type MeasureCardTops = (root: HTMLElement, selector: string) => Map<HTMLElement, number>;

export type AnimateFlip = (
	targets: HTMLElement[],
	params: { y: number[]; duration: number; ease: string }
) => { revert?: () => void; pause?: () => void; cancel?: () => void };

export type CreateCardListLayoutOptions = {
	children?: string;
	durationMs?: number;
	ease?: string;
	measureTops?: MeasureCardTops;
	animate?: AnimateFlip;
};

export type MultiListLayoutEntry = {
	root: HTMLElement;
	/** Defaults to `options.children` then `DEFAULT_CARD_LAYOUT_CHILDREN`. */
	children?: string;
};

const FLIP_EPSILON_PX = 0.5;

/**
 * Invert deltas for nodes present in both snapshots (same element identity).
 * Svelte keyed `{#each}` reuses the step's node; `data-card-index` is not stable.
 */
export function flipInverts(
	first: ReadonlyMap<HTMLElement, number>,
	last: ReadonlyMap<HTMLElement, number>
): FlipInvert[] {
	const out: FlipInvert[] = [];
	for (const [el, firstTop] of first) {
		const lastTop = last.get(el);
		if (lastTop == null) continue;
		const y = firstTop - lastTop;
		if (Math.abs(y) < FLIP_EPSILON_PX) continue;
		out.push({ el, y });
	}
	return out;
}

export function measureOffsetTops(root: HTMLElement, selector: string): Map<HTMLElement, number> {
	const tops = new Map<HTMLElement, number>();
	for (const node of root.querySelectorAll(selector)) {
		if (!(node instanceof HTMLElement)) continue;
		tops.set(node, node.offsetTop);
	}
	return tops;
}

function clearFlipTransform(el: HTMLElement) {
	el.style?.removeProperty('translate');
	el.style?.removeProperty('transform');
}

function mergeMeasuredTops(
	entries: MultiListLayoutEntry[],
	defaultChildren: string,
	measure: MeasureCardTops
): Map<HTMLElement, number> {
	const merged = new Map<HTMLElement, number>();
	for (const entry of entries) {
		const selector = entry.children ?? defaultChildren;
		for (const [el, top] of measure(entry.root, selector)) {
			merged.set(el, top);
		}
	}
	return merged;
}

function clearMeasuredTransforms(
	entries: MultiListLayoutEntry[],
	defaultChildren: string,
	measure: MeasureCardTops
) {
	for (const entry of entries) {
		const selector = entry.children ?? defaultChildren;
		for (const el of measure(entry.root, selector).keys()) clearFlipTransform(el);
	}
}

/**
 * Anime.js FLIP across one or more virtualized list roots (cards + YAML).
 *
 * Measures all roots before mutate, mutates once, collects inverts from every
 * root, then runs a single animate call. Never writes `top` / `position`.
 */
export function createMultiListLayout(
	entries: MultiListLayoutEntry[],
	options?: CreateCardListLayoutOptions
): CardListLayout {
	const duration = options?.durationMs ?? LAYOUT_SWAP_DURATION_MS;
	const ease = options?.ease ?? LAYOUT_SWAP_EASE;
	const defaultChildren = options?.children ?? DEFAULT_CARD_LAYOUT_CHILDREN;
	const measure = options?.measureTops ?? measureOffsetTops;
	const runAnimate = options?.animate ?? defaultAnimate;

	let disposed = false;
	let generation = 0;
	let playing: ReturnType<AnimateFlip> | null = null;

	const resetPlaying = () => {
		if (!playing) return;
		playing.cancel?.();
		playing.pause?.();
		playing = null;
	};

	return {
		async run(mutate) {
			if (disposed) {
				await mutate();
				return;
			}
			const gen = ++generation;
			resetPlaying();
			clearMeasuredTransforms(entries, defaultChildren, measure);

			const first = mergeMeasuredTops(entries, defaultChildren, measure);
			await mutate();
			if (disposed || gen !== generation) return;

			const last = mergeMeasuredTops(entries, defaultChildren, measure);
			const inverts = flipInverts(first, last);
			if (inverts.length === 0) return;

			playing = runAnimate(
				inverts.map((item) => item.el),
				{
					y: inverts.map((item) => item.y),
					duration,
					ease
				}
			);
		},
		dispose() {
			if (disposed) return;
			disposed = true;
			generation += 1;
			resetPlaying();
			clearMeasuredTransforms(entries, defaultChildren, measure);
		}
	};
}

/**
 * Anime.js FLIP for a single virtualized card list.
 * Thin wrapper over {@link createMultiListLayout}.
 */
export function createCardListLayout(
	root: HTMLElement,
	options?: CreateCardListLayoutOptions
): CardListLayout {
	return createMultiListLayout([{ root, children: options?.children }], options);
}

function defaultAnimate(
	targets: HTMLElement[],
	params: { y: number[]; duration: number; ease: string }
): JSAnimation {
	return animate(targets, {
		y: (_el, i) => [params.y[i] ?? 0, 0],
		duration: params.duration,
		ease: params.ease,
		composition: 'replace'
	});
}
