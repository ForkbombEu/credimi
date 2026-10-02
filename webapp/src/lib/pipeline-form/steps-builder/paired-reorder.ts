// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { tick as svelteTick } from 'svelte';

import { restoreScrollTop as defaultRestoreScrollTop } from './composer-virtualizer.svelte.js';

export type PairedReorderLayout = {
	run(mutate: () => void | Promise<void>): Promise<void>;
};

export type PairedReorderArgs = {
	cardsScroller: HTMLElement | null | undefined;
	yamlScroller: HTMLElement | null | undefined;
	/** Rememo both cards + yaml virtualizers after domain mutate. */
	syncBoth: () => void;
	layout: PairedReorderLayout | null | undefined;
	/** Domain mutate only — e.g. `() => builder.shiftStep(i, d)`. */
	mutate: () => void;
	/** Inject for tests; defaults to Svelte `tick`. */
	tick?: () => Promise<void>;
	/** Inject for tests; defaults to composer-virtualizer `restoreScrollTop`. */
	restoreScrollTop?: (el: HTMLElement | null | undefined, top: number) => void;
};

/**
 * Snapshot both pane scrollTops, optionally FLIP via `layout.run`, then
 * mutate → syncBoth → tick → restore both tops. Same ritual as the former
 * `steps-builder.svelte` `shiftStep` helper.
 */
export async function runPairedReorder(args: PairedReorderArgs): Promise<void> {
	const {
		cardsScroller,
		yamlScroller,
		syncBoth,
		layout,
		mutate,
		tick = svelteTick,
		restoreScrollTop = defaultRestoreScrollTop
	} = args;

	const cardsTop = cardsScroller?.scrollTop ?? 0;
	const yamlTop = yamlScroller?.scrollTop ?? 0;

	const apply = () => {
		mutate();
		syncBoth();
	};

	const restoreScroll = () => {
		restoreScrollTop(cardsScroller, cardsTop);
		restoreScrollTop(yamlScroller, yamlTop);
	};

	if (!layout) {
		apply();
		await tick();
		restoreScroll();
		return;
	}

	await layout.run(async () => {
		apply();
		await tick();
		restoreScroll();
	});
}
