// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * In-card enter/exit layout protocol: form-host classes, lock settled sizing,
 * chrome→body max, overlay prepare/finish, natural-height measure, and abort
 * when enter never settled. Tweens stay in `playInCardEnter` / `playInCardExit`.
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

export type InCardExitLayoutOptions = {
	lock?: HTMLElement;
	display?: HTMLElement;
	form?: HTMLElement;
	/** False when enter never finished — teardown without shrink. */
	enterSettled: boolean;
	onComplete?: () => void | Promise<void>;
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
 * Natural height of an absolutely-overlaid layer.
 * Must clear `inset`/`bottom` — with top+bottom pinned, `height: auto` still equals the lock.
 */
function measureNaturalHeight(el: HTMLElement): number {
	const style = el.style;
	const previous = {
		position: style.position,
		inset: style.inset,
		top: style.top,
		right: style.right,
		bottom: style.bottom,
		left: style.left,
		width: style.width,
		height: style.height,
		overflow: style.overflow
	};

	style.position = 'absolute';
	style.inset = '';
	style.top = '0';
	style.left = '0';
	style.right = '0';
	style.bottom = 'auto';
	style.width = '100%';
	style.height = 'auto';
	style.overflow = 'visible';

	const natural = el.getBoundingClientRect().height;

	style.position = previous.position;
	style.inset = previous.inset;
	style.top = previous.top;
	style.right = previous.right;
	style.bottom = previous.bottom;
	style.left = previous.left;
	style.width = previous.width;
	style.height = previous.height;
	style.overflow = previous.overflow;

	return natural;
}

function applyEnterOverlay(form: HTMLElement) {
	form.style.position = 'absolute';
	form.style.inset = '0';
	form.style.width = '100%';
	form.style.height = '100%';
	form.style.overflow = 'hidden';
	form.style.opacity = '0';
	form.style.visibility = 'visible';
	form.style.pointerEvents = 'none';
}

function hideEnterDisplay(display: HTMLElement) {
	display.style.opacity = '0';
	display.style.pointerEvents = 'none';
	display.style.visibility = 'hidden';
}

function settleEnterFormHost(form: HTMLElement) {
	form.style.position = '';
	form.style.inset = '';
	form.style.width = '';
	form.style.height = '';
	form.style.overflow = '';
	form.style.opacity = '';
	form.style.visibility = '';
	form.style.pointerEvents = '';
}

function finishEnterLayout(
	lock: HTMLElement,
	display: HTMLElement,
	form: HTMLElement,
	lockHeightPx: number
) {
	hideEnterDisplay(display);

	// Keep the form absolutely filling the lock until `settleEnterFormHost` runs.
	// Clearing absolute before settled flex host classes paint would collapse the
	// overlay for a frame and jump the save footer.
	applyEnterOverlay(form);
	form.style.opacity = '1';
	form.style.pointerEvents = '';

	lock.style.overflow = 'hidden';
	lock.style.height = `${lockHeightPx}px`;
	lock.style.opacity = '';
}

function lockToSummaryHeight(lock: HTMLElement, display: HTMLElement) {
	lock.style.overflow = 'hidden';
	lock.style.height = `${display.getBoundingClientRect().height}px`;
}

function applyExitOverlay(display: HTMLElement, form: HTMLElement) {
	applyEnterOverlay(form);
	form.style.opacity = '1';
	form.style.pointerEvents = 'none';

	display.style.position = 'absolute';
	display.style.inset = '';
	display.style.top = '0';
	display.style.left = '0';
	display.style.right = '0';
	display.style.bottom = 'auto';
	display.style.width = '100%';
	display.style.height = 'auto';
	display.style.overflow = 'visible';
	display.style.opacity = '0';
	display.style.visibility = 'visible';
	display.style.pointerEvents = 'none';
}

function teardownExitOverlay(lock: HTMLElement, display: HTMLElement, form: HTMLElement) {
	display.style.position = '';
	display.style.inset = '';
	display.style.top = '';
	display.style.left = '';
	display.style.right = '';
	display.style.bottom = '';
	display.style.width = '';
	display.style.height = '';
	display.style.overflow = '';
	display.style.opacity = '';
	display.style.visibility = '';
	display.style.pointerEvents = '';

	form.style.position = '';
	form.style.inset = '';
	form.style.width = '';
	form.style.height = '';
	form.style.overflow = '';
	form.style.opacity = '0';
	form.style.visibility = 'hidden';
	form.style.pointerEvents = 'none';

	lock.style.height = '';
	lock.style.overflow = '';
	lock.style.opacity = '';
}

/**
 * Enter protocol: overlay form + lock to summary → tweens → settled flex host +
 * lock height, then domain `onSettled`, then clear absolute fill.
 */
export function playInCardEnterLayout(options: InCardEnterLayoutOptions): MotionHandle {
	const { lock, display, form, onSettled, durationMs } = options;
	const bodyMaxPx = resolveBodyMaxPx(options);

	form.className = inCardFormHostClass(false);
	lockToSummaryHeight(lock, display);
	applyEnterOverlay(form);

	const toHeight =
		typeof bodyMaxPx === 'number' && Number.isFinite(bodyMaxPx)
			? Math.max(0, bodyMaxPx)
			: measureNaturalHeight(form);

	return playInCardEnter({
		lock,
		display,
		form,
		toHeight,
		durationMs,
		onComplete: async () => {
			finishEnterLayout(lock, display, form, toHeight);
			form.className = inCardFormHostClass(true);
			applyLockSettledLayout(lock, bodyMaxPx);
			await onSettled?.({ bodyMaxPx });
			settleEnterFormHost(form);
		}
	});
}

/**
 * Exit protocol: if enter never settled, complete immediately; else overlay
 * both layers, tween shrink/fade, then teardown overlay.
 */
export function playInCardExitLayout(options: InCardExitLayoutOptions): MotionHandle {
	const { lock, display, form, enterSettled, onComplete, durationMs } = options;
	if (!enterSettled || !lock || !display || !form) {
		const finished = Promise.resolve().then(async () => {
			await onComplete?.();
		});
		return { cancel: () => {}, finished };
	}

	const fromHeight = lock.getBoundingClientRect().height;
	lock.style.overflow = 'hidden';
	lock.style.height = `${fromHeight}px`;
	applyExitOverlay(display, form);
	const toHeight = measureNaturalHeight(display);

	return playInCardExit({
		lock,
		display,
		form,
		toHeight,
		durationMs,
		onComplete: async () => {
			teardownExitOverlay(lock, display, form);
			await onComplete?.();
		}
	});
}
