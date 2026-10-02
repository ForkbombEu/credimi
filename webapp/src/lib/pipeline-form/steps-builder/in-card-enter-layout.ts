// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * In-card enter layout protocol: owns form-host class transitions and lock
 * settled sizing around the anime primitive `playInCardEnter`.
 *
 * Caller's `onSettled` is for domain flags only (e.g. enterComplete) — layout
 * is applied imperatively here before `settleEnterFormHost` clears absolute fill.
 */

import {
	bodyMaxHeightWithinCard,
	playInCardEnter,
	type MotionHandle
} from './_partials/in-card-motion.js';

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

export type InCardEnterLayoutSettled = {
	/** Grown body max used for the lock (undefined when enter sized to form natural height). */
	bodyMaxPx: number | undefined;
};

export type InCardEnterLayoutOptions = {
	lock: HTMLElement;
	display: HTMLElement;
	form: HTMLElement;
	/** Card root — with `cardMaxHeightPx`, computes body max via `bodyMaxHeightWithinCard`. */
	card?: HTMLElement;
	/** Whole-card max height (px); chrome is subtracted when `card` is also provided. */
	cardMaxHeightPx?: number;
	/** Precomputed body max; wins over card + cardMaxHeightPx when finite. */
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
	const { bodyMaxPx, card, cardMaxHeightPx, display } = options;
	if (typeof bodyMaxPx === 'number' && Number.isFinite(bodyMaxPx)) {
		return Math.max(0, bodyMaxPx);
	}
	if (card && typeof cardMaxHeightPx === 'number' && Number.isFinite(cardMaxHeightPx)) {
		return bodyMaxHeightWithinCard(card, display, cardMaxHeightPx);
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
