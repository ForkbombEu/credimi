// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { AnimatableScroll } from './animatable-scroll.js';
import type { CardSection } from './yaml-ranges.js';

export type ActiveUnit = {
	section: CardSection;
	index: number;
};

export type ScrollAlign = 'nearest' | 'start' | 'center' | 'start-band';

/**
 * True list lengths for cards sequence (steps then follow-ups).
 * Under virtualization, first/last *mounted* cards may not be the true ends.
 */
export type CardListLengths = {
	steps: number;
	followUps: number;
};

/**
 * Mount (or scroll-to-index) a card that is not yet in the DOM; resolves true when ready.
 * Always async so PeerScrollFollow / scrollUnitIntoView can await uniformly.
 */
export type EnsureMounted = (unit: ActiveUnit) => Promise<boolean>;

/**
 * Peer-scroll bridge for a steps-only virtualizer: follow-ups stay fully mounted,
 * steps call `ensureStepVisible(index)` (e.g. TanStack scrollToIndex + wait).
 */
export function ensureMountedForStepsVirtualizer(
	ensureStepVisible: (index: number) => boolean | Promise<boolean>
): EnsureMounted {
	return async (unit) => {
		if (unit.section !== 'steps') return true;
		return await ensureStepVisible(unit.index);
	};
}

export type ScrollUnitIntoViewOptions = {
	focus?: boolean;
	align?: ScrollAlign;
	/** When the unit is missing, call this before retrying the DOM query. */
	ensureMounted?: EnsureMounted;
	/**
	 * When set, eases via Anime.js Animatable (retargetable) instead of native
	 * `scrollTo({ behavior })`. Mount helpers should still use `behavior: 'auto'`.
	 */
	animatableScroll?: AnimatableScroll;
	/** Per-call Animatable duration override (ms). */
	durationMs?: number;
};

/**
 * How a pane resolves the active unit at the scrollport top (and for header-only YAML).
 * - `claim`: take the list-start unit (cards).
 * - `intersect`: only claim when the unit's element intersects the scroller (YAML header).
 */
export type TopEdgePolicy = 'claim' | 'intersect';

/** DOM attribute / query adapter for a cards or YAML twin pane. */
export type PaneAdapter = {
	sectionAttr: string;
	indexAttr: string;
	listQuery: string;
	topEdgePolicy: TopEdgePolicy;
};

export const CARD_PANE: PaneAdapter = {
	sectionAttr: 'data-card-section',
	indexAttr: 'data-card-index',
	listQuery: '[data-card-section][data-card-index]',
	topEdgePolicy: 'claim'
};

export const YAML_PANE: PaneAdapter = {
	sectionAttr: 'data-yaml-section',
	indexAttr: 'data-yaml-index',
	listQuery: '[data-yaml-section][data-yaml-index]',
	topEdgePolicy: 'intersect'
};

const HYSTERESIS = 0.22;
const ALIGN_EPSILON_PX = 1;
const START_PADDING_PX = 16;
/** Fraction of viewport height — start-band only scrolls if the line is outside this zone. */
const START_BAND_RATIO = 0.35;

function parseUnit(el: Element, pane: PaneAdapter): ActiveUnit | null {
	const section = el.getAttribute(pane.sectionAttr);
	const indexRaw = el.getAttribute(pane.indexAttr);
	if (section !== 'steps' && section !== 'follow-ups') return null;
	if (indexRaw == null) return null;
	const index = Number(indexRaw);
	if (!Number.isInteger(index) || index < 0) return null;
	return { section, index };
}

function findUnit(
	scrollContainer: HTMLElement,
	unit: ActiveUnit,
	pane: PaneAdapter
): HTMLElement | null {
	return scrollContainer.querySelector<HTMLElement>(
		`[${pane.sectionAttr}="${unit.section}"][${pane.indexAttr}="${unit.index}"]`
	);
}

/**
 * Logical first/last unit from injected lengths (steps then follow-ups).
 * Returns null when both counts are zero or lengths are invalid.
 */
export function resolveListEndUnit(
	lengths: CardListLengths,
	edge: 'start' | 'end'
): ActiveUnit | null {
	const steps = Math.max(0, Math.floor(lengths.steps));
	const followUps = Math.max(0, Math.floor(lengths.followUps));
	if (steps === 0 && followUps === 0) return null;

	if (edge === 'start') {
		if (steps > 0) return { section: 'steps', index: 0 };
		return { section: 'follow-ups', index: 0 };
	}
	if (followUps > 0) return { section: 'follow-ups', index: followUps - 1 };
	return { section: 'steps', index: steps - 1 };
}

export function sameUnit(a: ActiveUnit | null, b: ActiveUnit | null): boolean {
	if (a === null || b === null) return a === b;
	return a.section === b.section && a.index === b.index;
}

/**
 * Pick the unit nearest the scrollport vertical center, with sticky hysteresis so
 * the active unit does not flicker at boundaries. At scroll edges, prefer the
 * first/last unit so the ends of the list remain reachable.
 *
 * When `lengths` is provided, list-end edges use the true sequence ends instead of
 * the first/last *mounted* unit (needed under virtualization).
 *
 * YAML (`topEdgePolicy: 'intersect'`) refuses to claim when only the header is
 * visible (no block intersects the viewport).
 */
export function resolveViewportUnit(
	scrollContainer: HTMLElement,
	previous: ActiveUnit | null,
	pane: PaneAdapter,
	lengths?: CardListLengths
): ActiveUnit | null {
	const items = [...scrollContainer.querySelectorAll<HTMLElement>(pane.listQuery)];
	if (items.length === 0) return null;

	const maxScroll = scrollContainer.scrollHeight - scrollContainer.clientHeight;
	if (maxScroll > 0) {
		if (scrollContainer.scrollTop <= 2) {
			if (pane.topEdgePolicy === 'intersect') {
				// At top: header may still own the viewport — only claim start when it intersects.
				const start = lengths
					? resolveListEndUnit(lengths, 'start')
					: parseUnit(items[0]!, pane);
				if (!start) return null;
				const startEl = findUnit(scrollContainer, start, pane);
				if (!startEl || !elementIntersectsScroller(startEl, scrollContainer)) return null;
				return start;
			}
			return (
				(lengths ? resolveListEndUnit(lengths, 'start') : null) ??
				parseUnit(items[0]!, pane) ??
				null
			);
		}
		if (scrollContainer.scrollTop >= maxScroll - 2) {
			return (
				(lengths ? resolveListEndUnit(lengths, 'end') : null) ??
				parseUnit(items[items.length - 1]!, pane) ??
				null
			);
		}
	}

	const port = scrollContainer.getBoundingClientRect();
	const centerY = port.top + port.height / 2;
	const thresholdPx = port.height * HYSTERESIS;

	let best: { unit: ActiveUnit; distance: number; el: HTMLElement } | null = null;
	for (const el of items) {
		const unit = parseUnit(el, pane);
		if (!unit) continue;
		const rect = el.getBoundingClientRect();
		const itemCenter = rect.top + rect.height / 2;
		const distance = Math.abs(itemCenter - centerY);
		if (!best || distance < best.distance) {
			best = { unit, distance, el };
		}
	}
	if (!best) return null;

	if (
		pane.topEdgePolicy === 'intersect' &&
		!elementIntersectsScroller(best.el, scrollContainer)
	) {
		return null;
	}

	if (previous && sameUnit(previous, best.unit)) return previous;

	if (previous) {
		const prevEl = items.find((el) => {
			const u = parseUnit(el, pane);
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

/**
 * Scroll a pane unit into view. When the unit is not mounted and `ensureMounted` is
 * provided, awaits mount then retries. Without `ensureMounted`, missing units
 * still return false (today's non-virtual path).
 */
export async function scrollUnitIntoView(
	scrollContainer: HTMLElement,
	unit: ActiveUnit,
	behavior: ScrollBehavior,
	pane: PaneAdapter,
	options?: ScrollUnitIntoViewOptions
): Promise<boolean> {
	let el = findUnit(scrollContainer, unit, pane);
	if (!el && options?.ensureMounted) {
		const mounted = await options.ensureMounted(unit);
		if (!mounted) return false;
		el = findUnit(scrollContainer, unit, pane);
	}
	if (!el) return false;
	const align = options?.align ?? 'center';
	const scrolled = scrollChildIntoScroller(scrollContainer, el, behavior, align, {
		animatableScroll: options?.animatableScroll,
		durationMs: options?.durationMs
	});
	if (options?.focus !== false) {
		el.focus({ preventScroll: true });
	}
	return scrolled;
}

export type ScrollChildIntoScrollerOptions = {
	animatableScroll?: AnimatableScroll;
	durationMs?: number;
};

/** Scroll `el` into `scroller` without using Element.scrollIntoView (avoids wrong ancestors). */
export function scrollChildIntoScroller(
	scroller: HTMLElement,
	el: HTMLElement,
	behavior: ScrollBehavior,
	align: ScrollAlign = 'nearest',
	options?: ScrollChildIntoScrollerOptions
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
	const animatable = options?.animatableScroll;
	if (animatable) {
		animatable.scrollTo(nextTop, options?.durationMs);
	} else {
		scroller.scrollTo({ top: nextTop, behavior });
	}
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

function elementIntersectsScroller(el: HTMLElement, scroller: HTMLElement): boolean {
	const elRect = el.getBoundingClientRect();
	const port = scroller.getBoundingClientRect();
	return elRect.bottom > port.top && elRect.top < port.bottom;
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

export type WatchDrivenScrollOptions = {
	/**
	 * Explicit idle timeout (ms). Prefer this for Animatable-driven scrolls
	 * (`durationMs + pad`) so peer handlers stay muted until the tween settles.
	 * When omitted, falls back to behavior heuristics (`smooth` → 650, else 120).
	 */
	idleTimeoutMs?: number;
};

export function watchDrivenScroll(
	el: HTMLElement,
	behavior: ScrollBehavior,
	onClear: () => void,
	clock: DrivenScrollClock = defaultDrivenClock,
	options?: WatchDrivenScrollOptions
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
	const idleMs = options?.idleTimeoutMs ?? (behavior === 'smooth' ? 650 : 120);
	const timer = clock.setTimeout(clear, idleMs);
	return clear;
}
