// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { tick as svelteTick } from 'svelte';

import { stepCardSelector, yamlStepBlockSelector } from './composer-virtualizer.svelte.js';
import { framePairScrollTop, type FramePairRect } from './scroll-follow/frame-pair.js';

/** Default lag between cards and YAML pair framing (helps peer sync). */
export const DEFAULT_YAML_FRAME_DELAY_MS = 50;

export type PairedShiftLayout = {
	run(mutate: () => void | Promise<void>): Promise<void>;
};

export type MeasuredPair = {
	scrollTop: number;
	clientHeight: number;
	scrollHeight: number;
	upper: FramePairRect;
	lower: FramePairRect;
	scrollerRect: FramePairRect;
};

export type PairedShiftArgs = {
	fromIndex: number;
	toIndex: number;
	/** Mute continuous Scroll follow around the ritual. */
	notePairedReorder: () => void;
	/**
	 * Opaque swap-mount seam — twin-pane wires `mountSwapIndices`.
	 * Must not know mountOnly / align / virtualizer scrollOffset options.
	 */
	ensureSwapMounted: (fromIndex: number, toIndex: number) => Promise<void>;
	cardsScroller: HTMLElement | null | undefined;
	yamlScroller: HTMLElement | null | undefined;
	/** Rememo both cards + yaml virtualizers after domain mutate. */
	rememoAfterMutate: () => void;
	layout: PairedShiftLayout | null | undefined;
	/** Domain mutate only — e.g. `() => builder.shiftStep(i, d)`. */
	mutate: () => void;
	/**
	 * Pin both panes after mutate/FLIP — twin-pane wires DOM+TanStack via
	 * `pinBothScrollports`. Paired-shift must not call virt-less restore.
	 */
	pinScrollports: (cardsTop: number, yamlTop: number) => void;
	/** Inject for tests; defaults to Svelte `tick`. */
	tick?: () => Promise<void>;
	/**
	 * Measure swapped-pair geometry in a pane. Return null to skip framing that pane.
	 * Defaults to querySelector + getBoundingClientRect via card/yaml selectors.
	 */
	measurePair?: (
		scroller: HTMLElement,
		upperIndex: number,
		lowerIndex: number
	) => MeasuredPair | null;
	/** Apply framed scrollTop; defaults to `scrollTo({ top, behavior: 'auto' })`. */
	scrollTo?: (scroller: HTMLElement, top: number) => void;
	/** Delay before YAML framing; default {@link DEFAULT_YAML_FRAME_DELAY_MS}. */
	yamlFrameDelayMs?: number;
	/** Inject for tests; defaults to `setTimeout`. */
	delay?: (ms: number) => Promise<void>;
};

function defaultDelay(ms: number): Promise<void> {
	if (ms <= 0) return Promise.resolve();
	return new Promise((resolve) => {
		setTimeout(resolve, ms);
	});
}

function defaultScrollTo(scroller: HTMLElement, top: number): void {
	scroller.scrollTo({ top, behavior: 'auto' });
}

function rectFromEl(el: Element): FramePairRect {
	const r = el.getBoundingClientRect();
	return { top: r.top, bottom: r.bottom };
}

function queryPairElements(
	scroller: HTMLElement,
	upperIndex: number,
	lowerIndex: number
): { upperEl: Element; lowerEl: Element } | null {
	for (const selector of [stepCardSelector, yamlStepBlockSelector]) {
		const upperEl = scroller.querySelector(selector(upperIndex));
		const lowerEl = scroller.querySelector(selector(lowerIndex));
		if (upperEl && lowerEl) return { upperEl, lowerEl };
	}
	return null;
}

function defaultMeasurePair(
	scroller: HTMLElement,
	upperIndex: number,
	lowerIndex: number
): MeasuredPair | null {
	const pair = queryPairElements(scroller, upperIndex, lowerIndex);
	if (!pair) return null;
	const port = scroller.getBoundingClientRect();
	return {
		scrollTop: scroller.scrollTop,
		clientHeight: scroller.clientHeight,
		scrollHeight: scroller.scrollHeight,
		upper: rectFromEl(pair.upperEl),
		lower: rectFromEl(pair.lowerEl),
		scrollerRect: { top: port.top, bottom: port.bottom }
	};
}

function framePanePair(
	scroller: HTMLElement | null | undefined,
	upperIndex: number,
	lowerIndex: number,
	measurePair: NonNullable<PairedShiftArgs['measurePair']>,
	scrollTo: NonNullable<PairedShiftArgs['scrollTo']>
): void {
	if (!scroller) return;
	const measured = measurePair(scroller, upperIndex, lowerIndex);
	if (!measured) return;
	const next = framePairScrollTop(
		measured.scrollTop,
		measured.clientHeight,
		measured.scrollHeight,
		measured.upper,
		measured.lower,
		measured.scrollerRect
	);
	if (next == null) return;
	scrollTo(scroller, next);
}

/**
 * Paired step-shift ritual: mute follow → mount swap indices → mutate+FLIP
 * restore → frame swapped pair in cards → delay → frame YAML independently.
 */
export async function runPairedShift(args: PairedShiftArgs): Promise<void> {
	const {
		fromIndex,
		toIndex,
		notePairedReorder,
		ensureSwapMounted,
		cardsScroller,
		yamlScroller,
		rememoAfterMutate,
		layout,
		mutate,
		pinScrollports,
		tick = svelteTick,
		measurePair = defaultMeasurePair,
		scrollTo = defaultScrollTo,
		yamlFrameDelayMs = DEFAULT_YAML_FRAME_DELAY_MS,
		delay = defaultDelay
	} = args;

	const upperIndex = Math.min(fromIndex, toIndex);
	const lowerIndex = Math.max(fromIndex, toIndex);

	notePairedReorder();
	await ensureSwapMounted(fromIndex, toIndex);

	const cardsTop = cardsScroller?.scrollTop ?? 0;
	const yamlTop = yamlScroller?.scrollTop ?? 0;

	const apply = () => {
		mutate();
		rememoAfterMutate();
	};

	const restoreScroll = () => {
		pinScrollports(cardsTop, yamlTop);
	};

	if (!layout) {
		apply();
		await tick();
		restoreScroll();
	} else {
		await layout.run(async () => {
			apply();
			await tick();
			restoreScroll();
		});
	}

	framePanePair(cardsScroller, upperIndex, lowerIndex, measurePair, scrollTo);
	await delay(yamlFrameDelayMs);
	framePanePair(yamlScroller, upperIndex, lowerIndex, measurePair, scrollTo);
}
