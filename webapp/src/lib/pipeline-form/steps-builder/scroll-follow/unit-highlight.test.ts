// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { YamlCardRange } from './yaml-ranges.js';

import { UnitHighlight, resolveSelectedUnit, linesForUnit } from './unit-highlight.svelte.js';

const ranges: YamlCardRange[] = [
	{ section: 'steps', index: 0, startLine: 0, endLine: 2 },
	{ section: 'steps', index: 1, startLine: 4, endLine: 6 }
];

describe('resolveSelectedUnit / linesForUnit', () => {
	it('prefers edit focus over pin; clears in manual', () => {
		const pinned = { section: 'steps' as const, index: 1 };
		expect(resolveSelectedUnit(false, undefined, pinned)).toEqual(pinned);
		expect(resolveSelectedUnit(false, 0, pinned)).toEqual({ section: 'steps', index: 0 });
		expect(resolveSelectedUnit(true, 0, pinned)).toBeNull();
		expect(linesForUnit(pinned, ranges)).toEqual({ start: 4, end: 6 });
	});
});

describe('UnitHighlight', () => {
	it('maps wash lines from pin; edit focus from inputs wins', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => undefined,
			getRanges: () => ranges
		});

		highlight.pinYamlLine(5);
		expect(highlight.selectedUnit).toEqual({ section: 'steps', index: 1 });
		expect(highlight.selectedLines).toEqual({ start: 4, end: 6 });

		const editing = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => 0,
			getRanges: () => ranges
		});
		editing.pinYamlLine(5);
		expect(editing.selectedUnit).toEqual({ section: 'steps', index: 0 });
		expect(editing.selectedLines).toEqual({ start: 0, end: 2 });
		expect(editing.isCardSelected('steps', 0)).toBe(true);
		expect(editing.isCardSelected('steps', 1)).toBe(false);

		highlight.dispose();
		editing.dispose();
	});

	it('clears selection wash when inputs say manual (pin retained)', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => true,
			getEditingIndex: () => undefined,
			getRanges: () => ranges
		});

		highlight.pinYamlLine(1);
		expect(highlight.pinnedUnit).toEqual({ section: 'steps', index: 0 });
		expect(highlight.selectedUnit).toBeNull();
		expect(highlight.selectedLines).toBeNull();
		highlight.dispose();
	});

	it('hovers cards and sticky yaml gaps; leave only clears owning card', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => undefined,
			getRanges: () => ranges
		});

		highlight.hoverCard({ section: 'steps', index: 0 });
		expect(highlight.isCardHovered('steps', 0)).toBe(true);
		expect(highlight.hoverLines).toEqual({ start: 0, end: 2 });

		highlight.clearHoverCard({ section: 'steps', index: 1 });
		expect(highlight.isCardHovered('steps', 0)).toBe(true);

		highlight.clearHoverCard({ section: 'steps', index: 0 });
		expect(highlight.hoveredUnit).toBeNull();

		highlight.hoverYamlLine(3);
		expect(highlight.hoveredUnit).toEqual({ section: 'steps', index: 0 });

		highlight.hoverYamlLine(null);
		expect(highlight.hoveredUnit).toBeNull();
		highlight.dispose();
	});

	it('pinYamlLine ignores non-hits', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => undefined,
			getRanges: () => ranges
		});

		expect(highlight.pinYamlLine(3)).toBeNull();

		const pinned = highlight.pinYamlLine(0);
		expect(pinned).toEqual({ section: 'steps', index: 0 });
		expect(highlight.pinnedUnit).toEqual({ section: 'steps', index: 0 });
		highlight.dispose();
	});
});
