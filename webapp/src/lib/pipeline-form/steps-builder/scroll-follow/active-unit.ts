// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CardSection } from './yaml-ranges.js';

export type ActiveUnit = {
	section: CardSection;
	index: number;
};

export type ScrollAlign = 'nearest' | 'start' | 'center' | 'start-band';

const HYSTERESIS = 0.22;
const ALIGN_EPSILON_PX = 1;
const START_PADDING_PX = 16;
/** Fraction of viewport height — start-band only scrolls if the line is outside this zone. */
const START_BAND_RATIO = 0.35;

function parseCard(el: Element): ActiveUnit | null {
	const section = el.getAttribute('data-card-section');
	const indexRaw = el.getAttribute('data-card-index');
	if (section !== 'steps' && section !== 'follow-ups') return null;
	if (indexRaw == null) return null;
	const index = Number(indexRaw);
	if (!Number.isInteger(index) || index < 0) return null;
	return { section, index };
}

export function sameUnit(a: ActiveUnit | null, b: ActiveUnit | null): boolean {
	if (a === null || b === null) return a === b;
	return a.section === b.section && a.index === b.index;
}

/**
 * Pick the card nearest the scrollport vertical center, with sticky hysteresis so
 * the active unit does not flicker at boundaries. At scroll edges, prefer the
 * first/last card so the ends of the list remain reachable.
 */
export function resolveViewportActiveCard(
	scrollContainer: HTMLElement,
	previous: ActiveUnit | null
): ActiveUnit | null {
	const cards = [
		...scrollContainer.querySelectorAll<HTMLElement>('[data-card-section][data-card-index]')
	];
	if (cards.length === 0) return null;

	const maxScroll = scrollContainer.scrollHeight - scrollContainer.clientHeight;
	if (maxScroll > 0) {
		if (scrollContainer.scrollTop <= 2) {
			return parseCard(cards[0]!) ?? null;
		}
		if (scrollContainer.scrollTop >= maxScroll - 2) {
			return parseCard(cards[cards.length - 1]!) ?? null;
		}
	}

	const port = scrollContainer.getBoundingClientRect();
	const centerY = port.top + port.height / 2;
	const thresholdPx = port.height * HYSTERESIS;

	let best: { unit: ActiveUnit; distance: number; el: HTMLElement } | null = null;
	for (const el of cards) {
		const unit = parseCard(el);
		if (!unit) continue;
		const rect = el.getBoundingClientRect();
		const cardCenter = rect.top + rect.height / 2;
		const distance = Math.abs(cardCenter - centerY);
		if (!best || distance < best.distance) {
			best = { unit, distance, el };
		}
	}
	if (!best) return null;

	if (previous && sameUnit(previous, best.unit)) return previous;

	if (previous) {
		const prevEl = cards.find((el) => {
			const u = parseCard(el);
			return u && sameUnit(u, previous);
		});
		if (prevEl) {
			const prevRect = prevEl.getBoundingClientRect();
			const prevCenter = prevRect.top + prevRect.height / 2;
			const prevDistance = Math.abs(prevCenter - centerY);
			if (prevDistance - best.distance < thresholdPx) {
				return previous;
			}
		}
	}

	return best.unit;
}

export function scrollCardIntoView(
	scrollContainer: HTMLElement,
	unit: ActiveUnit,
	behavior: ScrollBehavior,
	options?: { focus?: boolean; align?: ScrollAlign }
): boolean {
	const el = scrollContainer.querySelector<HTMLElement>(
		`[data-card-section="${unit.section}"][data-card-index="${unit.index}"]`
	);
	if (!el) return false;
	const align = options?.align ?? 'center';
	const scrolled = scrollChildIntoScroller(scrollContainer, el, behavior, align);
	if (options?.focus !== false) {
		el.focus({ preventScroll: true });
	}
	return scrolled;
}

/** Scroll `el` into `scroller` without using Element.scrollIntoView (avoids wrong ancestors). */
export function scrollChildIntoScroller(
	scroller: HTMLElement,
	el: HTMLElement,
	behavior: ScrollBehavior,
	align: ScrollAlign = 'nearest'
): boolean {
	const nextTop = computeAlignedScrollTop(
		scroller.scrollTop,
		scroller.clientHeight,
		scroller.scrollHeight,
		el.getBoundingClientRect(),
		scroller.getBoundingClientRect(),
		align
	);
	if (nextTop === null) return false;
	scroller.scrollTo({ top: nextTop, behavior });
	return true;
}

/**
 * Returns the scroller's next scrollTop for `align`, or null if already within epsilon.
 * - nearest: only move when not fully visible (minimal delta)
 * - start: place element top near scroller top (with padding)
 * - start-band: like start, but skip when the line already sits in the upper band
 * - center: place element center on scroller center
 */
export function computeAlignedScrollTop(
	scrollTop: number,
	clientHeight: number,
	scrollHeight: number,
	elRect: { top: number; bottom: number },
	scrollerRect: { top: number; bottom: number },
	align: ScrollAlign
): number | null {
	const maxScroll = Math.max(0, scrollHeight - clientHeight);
	let target: number;

	if (align === 'nearest') {
		const nearest = computeNearestScrollTop(scrollTop, clientHeight, elRect, scrollerRect);
		if (nearest === null) return null;
		target = nearest;
	} else if (align === 'start' || align === 'start-band') {
		if (align === 'start-band') {
			const offset = elRect.top - scrollerRect.top;
			const bandBottom = clientHeight * START_BAND_RATIO;
			if (offset >= START_PADDING_PX && offset <= bandBottom) {
				return null;
			}
		}
		target = scrollTop + (elRect.top - scrollerRect.top) - START_PADDING_PX;
	} else {
		const elCenter = (elRect.top + elRect.bottom) / 2;
		const portCenter = (scrollerRect.top + scrollerRect.bottom) / 2;
		target = scrollTop + (elCenter - portCenter);
	}

	target = Math.min(maxScroll, Math.max(0, target));
	if (Math.abs(target - scrollTop) < ALIGN_EPSILON_PX) return null;
	return target;
}

/**
 * Returns the scroller's next scrollTop so `elRect` is fully visible inside `scrollerRect`,
 * or null if already fully visible (nearest semantics).
 */
export function computeNearestScrollTop(
	scrollTop: number,
	clientHeight: number,
	elRect: { top: number; bottom: number },
	scrollerRect: { top: number; bottom: number }
): number | null {
	const visibleTop = scrollerRect.top;
	const visibleBottom = scrollerRect.bottom;

	if (elRect.top >= visibleTop && elRect.bottom <= visibleBottom) {
		return null;
	}

	if (elRect.top < visibleTop) {
		return scrollTop + (elRect.top - visibleTop);
	}

	// el extends past the bottom
	return scrollTop + (elRect.bottom - visibleBottom);
}

export function scrollYamlLineIntoView(
	scroller: HTMLElement,
	line: number,
	behavior: ScrollBehavior,
	align: ScrollAlign = 'start'
): boolean {
	const el = scroller.querySelector<HTMLElement>(`[data-line="${line}"]`);
	if (!el) return false;
	return scrollChildIntoScroller(scroller, el, behavior, align);
}

/** Line nearest the vertical center of a YAML scroller; null if above first step. */
export function resolveViewportYamlLine(
	scroller: HTMLElement,
	firstStepLine: number | null
): number | null {
	const lines = [...scroller.querySelectorAll<HTMLElement>('[data-line]')];
	if (lines.length === 0) return null;

	const port = scroller.getBoundingClientRect();
	const centerY = port.top + port.height / 2;

	let best: { line: number; distance: number } | null = null;
	for (const el of lines) {
		const line = Number(el.getAttribute('data-line'));
		if (!Number.isInteger(line)) continue;
		const rect = el.getBoundingClientRect();
		const mid = rect.top + rect.height / 2;
		const distance = Math.abs(mid - centerY);
		if (!best || distance < best.distance) {
			best = { line, distance };
		}
	}
	if (!best) return null;
	if (firstStepLine != null && best.line < firstStepLine) return null;
	return best.line;
}

/**
 * Mark a scroller as programmatically driven until scrollend (or timeout fallback).
 * Peer scroll handlers should ignore events while their side is driven.
 */
export type DrivenScrollClock = {
	setTimeout: (handler: () => void, timeout?: number) => ReturnType<typeof setTimeout>;
	clearTimeout: (handle: ReturnType<typeof setTimeout>) => void;
};

const defaultDrivenClock: DrivenScrollClock = {
	setTimeout: (...args) => globalThis.setTimeout(...args),
	clearTimeout: (...args) => globalThis.clearTimeout(...args)
};

export function watchDrivenScroll(
	el: HTMLElement,
	behavior: ScrollBehavior,
	onClear: () => void,
	clock: DrivenScrollClock = defaultDrivenClock
): () => void {
	let cleared = false;
	const clear = () => {
		if (cleared) return;
		cleared = true;
		el.removeEventListener('scrollend', clear);
		clock.clearTimeout(timer);
		onClear();
	};
	el.addEventListener('scrollend', clear, { once: true });
	const timer = clock.setTimeout(clear, behavior === 'smooth' ? 650 : 120);
	return clear;
}
