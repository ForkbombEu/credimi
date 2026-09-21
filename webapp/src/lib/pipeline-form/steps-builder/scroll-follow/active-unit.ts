// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CardSection } from './yaml-ranges.js';

export type ActiveUnit = {
	section: CardSection;
	index: number;
};

const HYSTERESIS = 0.22;

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
 * the active unit does not flicker at boundaries.
 */
export function resolveViewportActiveCard(
	scrollContainer: HTMLElement,
	previous: ActiveUnit | null
): ActiveUnit | null {
	const cards = [
		...scrollContainer.querySelectorAll<HTMLElement>('[data-card-section][data-card-index]')
	];
	if (cards.length === 0) return null;

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
	behavior: ScrollBehavior
): void {
	const el = scrollContainer.querySelector<HTMLElement>(
		`[data-card-section="${unit.section}"][data-card-index="${unit.index}"]`
	);
	el?.scrollIntoView({ behavior, block: 'nearest' });
	el?.focus({ preventScroll: true });
}

export function scrollYamlLineIntoView(
	scroller: HTMLElement,
	line: number,
	behavior: ScrollBehavior
): void {
	const el = scroller.querySelector<HTMLElement>(`[data-line="${line}"]`);
	el?.scrollIntoView({ behavior, block: 'nearest' });
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
