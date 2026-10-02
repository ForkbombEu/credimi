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

/**
 * Anime.js FLIP for a virtualized card list.
 *
 * `createLayout` rewrites `position`/`top` during the tween, which fights
 * TanStack's absolute `style:top`. This helper only animates `y` (translate).
 */
export function createCardListLayout(
	root: HTMLElement,
	options?: CreateCardListLayoutOptions
): CardListLayout {
	const duration = options?.durationMs ?? LAYOUT_SWAP_DURATION_MS;
	const ease = options?.ease ?? LAYOUT_SWAP_EASE;
	const children = options?.children ?? DEFAULT_CARD_LAYOUT_CHILDREN;
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
			for (const el of measure(root, children).keys()) clearFlipTransform(el);

			const first = measure(root, children);
			await mutate();
			if (disposed || gen !== generation) return;

			const last = measure(root, children);
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
			for (const el of measure(root, children).keys()) clearFlipTransform(el);
		}
	};
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
