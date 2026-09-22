// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import type { YamlCardRange } from './yaml-ranges.js';

import { createUnitHighlight } from './unit-highlight.js';

const ranges: YamlCardRange[] = [
	{ section: 'steps', index: 0, startLine: 0, endLine: 2 },
	{ section: 'steps', index: 1, startLine: 4, endLine: 6 }
];

describe('createUnitHighlight', () => {
	it('resolves selected from edit focus over pin, and maps wash lines', () => {
		const highlight = createUnitHighlight();
		highlight.setRanges(ranges);
		highlight.pinYamlLine(5);
		expect(highlight.snapshot.selectedUnit).toEqual({ section: 'steps', index: 1 });
		expect(highlight.snapshot.selectedLines).toEqual({ start: 4, end: 6 });

		highlight.setEditingIndex(0);
		expect(highlight.snapshot.selectedUnit).toEqual({ section: 'steps', index: 0 });
		expect(highlight.snapshot.selectedLines).toEqual({ start: 0, end: 2 });
		expect(highlight.isCardSelected('steps', 0)).toBe(true);
		expect(highlight.isCardSelected('steps', 1)).toBe(false);

		highlight.dispose();
	});

	it('clears selection in manual mode without clearing the pin', () => {
		const highlight = createUnitHighlight();
		highlight.setRanges(ranges);
		highlight.pinYamlLine(1);
		highlight.setManual(true);
		expect(highlight.snapshot.selectedUnit).toBeNull();
		expect(highlight.snapshot.selectedLines).toBeNull();

		highlight.setManual(false);
		expect(highlight.snapshot.selectedUnit).toEqual({ section: 'steps', index: 0 });
		highlight.dispose();
	});

	it('hovers cards and sticky yaml gaps; leave only clears owning card', () => {
		const highlight = createUnitHighlight();
		highlight.setRanges(ranges);

		highlight.hoverCard({ section: 'steps', index: 0 });
		expect(highlight.isCardHovered('steps', 0)).toBe(true);
		expect(highlight.snapshot.hoverLines).toEqual({ start: 0, end: 2 });

		highlight.clearHoverCard({ section: 'steps', index: 1 });
		expect(highlight.isCardHovered('steps', 0)).toBe(true);

		highlight.clearHoverCard({ section: 'steps', index: 0 });
		expect(highlight.snapshot.hoveredUnit).toBeNull();

		// Gap line 3 is equidistant; sticky prefers the earlier range.
		highlight.hoverYamlLine(3);
		expect(highlight.snapshot.hoveredUnit).toEqual({ section: 'steps', index: 0 });

		highlight.hoverYamlLine(null);
		expect(highlight.snapshot.hoveredUnit).toBeNull();
		highlight.dispose();
	});

	it('pinYamlLine ignores non-hits and notifies on pin', () => {
		const highlight = createUnitHighlight();
		highlight.setRanges(ranges);
		const listener = vi.fn();
		highlight.subscribe(listener);

		expect(highlight.pinYamlLine(3)).toBeNull();
		expect(listener).not.toHaveBeenCalled();

		const pinned = highlight.pinYamlLine(0);
		expect(pinned).toEqual({ section: 'steps', index: 0 });
		expect(listener).toHaveBeenCalled();
		highlight.dispose();
	});
});
