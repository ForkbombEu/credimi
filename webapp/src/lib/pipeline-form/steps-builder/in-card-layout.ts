// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * In-card enter/exit layout protocol: form-host classes, lock settled sizing,
 * chrome→body max, and abort when enter never settled. Anime stays in
 * `playInCardEnter` / `playInCardExit`.
 *
 * Caller's `onSettled` / `onComplete` are domain flags only.
 */

import { playInCardEnter, playInCardExit, type MotionHandle } from './_partials/in-card-motion.js';

/**
 * Form-host classes for In-card enter vs settled/exit.
 * Enter keeps absolute fill over the growing lock, but must still be a flex column
 * so the shell's grow pushes Save to the column bottom during the grow (not only
 * after settle).
 */
export function inCardFormHostClass(settled: boolean): string {
	if (settled) return 'flex min-h-0 grow flex-col overflow-hidden';
	return 'pointer-events-none invisible absolute inset-0 flex min-h-0 flex-col overflow-hidden opacity-0';
}

/**
 * How tall the form body may grow when the whole card is capped at `cardFillMaxPx`.
 * Chrome = color bar + type header (everything in the card above `display`/`lock`).
 */
export function bodyMaxHeightWithinCard(
	card: HTMLElement,
	display: HTMLElement,
	cardFillMaxPx: number
): number {
	const cardHeight = card.getBoundingClientRect().height;
	const displayHeight = display.getBoundingClientRect().height;
	const chromePx = Math.max(0, cardHeight - displayHeight);
	return Math.max(0, cardFillMaxPx - chromePx);
}

export type InCardEnterLayoutSettled = {
	/** Grown body max used for the lock (undefined when enter sized to form natural height). */
	bodyMaxPx: number | undefined;
};

export type InCardEnterLayoutOptions = {
	lock: HTMLElement;
	display: HTMLElement;
	form: HTMLElement;
	/** Card root — with `cardFillMaxPx`, computes body max via `bodyMaxHeightWithinCard`. */
	card?: HTMLElement;
	/** Whole-card fill max (px) from Twin-pane `cardFillMaxPx`. */
	cardFillMaxPx?: number;
	/** Precomputed body max; wins over card + cardFillMaxPx when finite (tests). */
	bodyMaxPx?: number;
	/** Domain flags only — runs after settled host/lock layout, before absolute fill clears. */
	onSettled?: (info: InCardEnterLayoutSettled) => void | Promise<void>;
	durationMs?: number;
};

/** Ensure lock is a flex grow column; keep `relative` / existing `min-h-0` tokens. */
function applyLockSettledLayout(lock: HTMLElement, bodyMaxPx: number | undefined): void {
	const tokens = new Set(lock.className.split(/\s+/).filter(Boolean));
	tokens.add('flex');
	tokens.add('min-h-0');
	tokens.add('grow');
	tokens.add('flex-col');
	lock.className = [...tokens].join(' ');

	if (typeof bodyMaxPx === 'number' && Number.isFinite(bodyMaxPx)) {
		lock.style.height = `${bodyMaxPx}px`;
		lock.style.maxHeight = `${bodyMaxPx}px`;
	}
}

function resolveBodyMaxPx(options: InCardEnterLayoutOptions): number | undefined {
	const { bodyMaxPx, card, cardFillMaxPx, display } = options;
	if (typeof bodyMaxPx === 'number' && Number.isFinite(bodyMaxPx)) {
		return Math.max(0, bodyMaxPx);
	}
	if (card && typeof cardFillMaxPx === 'number' && Number.isFinite(cardFillMaxPx)) {
		return bodyMaxHeightWithinCard(card, display, cardFillMaxPx);
	}
	return undefined;
}

/**
 * Enter protocol: absolute-fill host → anime grow → settled flex host + lock height,
 * then domain `onSettled`, then motion clears absolute fill.
 */
export function playInCardEnterLayout(options: InCardEnterLayoutOptions): MotionHandle {
	const { lock, display, form, onSettled, durationMs } = options;
	const bodyMaxPx = resolveBodyMaxPx(options);

	form.className = inCardFormHostClass(false);

	return playInCardEnter({
		lock,
		display,
		form,
		maxHeightPx: bodyMaxPx,
		durationMs,
		onComplete: async () => {
			form.className = inCardFormHostClass(true);
			applyLockSettledLayout(lock, bodyMaxPx);
			await onSettled?.({ bodyMaxPx });
		}
	});
}

export type InCardExitLayoutOptions = {
	lock?: HTMLElement;
	display?: HTMLElement;
	form?: HTMLElement;
	/** False when enter never finished — teardown without shrink. */
	enterSettled: boolean;
	onComplete?: () => void | Promise<void>;
	durationMs?: number;
};

/**
 * Exit protocol: if enter never settled, complete immediately; else shrink via anime.
 */
export function playInCardExitLayout(options: InCardExitLayoutOptions): MotionHandle {
	const { lock, display, form, enterSettled, onComplete, durationMs } = options;
	if (!enterSettled || !lock || !display || !form) {
		const finished = Promise.resolve().then(async () => {
			await onComplete?.();
		});
		return { cancel: () => {}, finished };
	}
	return playInCardExit({
		lock,
		display,
		form,
		onComplete,
		durationMs
	});
}
