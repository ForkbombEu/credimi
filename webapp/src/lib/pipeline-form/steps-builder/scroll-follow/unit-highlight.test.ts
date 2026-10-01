// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { UnitHighlight, resolveSelectedUnit } from './unit-highlight.svelte.js';

describe('resolveSelectedUnit', () => {
	it('prefers edit focus over pin; clears in manual', () => {
		const pinned = { section: 'steps' as const, index: 1 };
		expect(resolveSelectedUnit(false, undefined, pinned)).toEqual(pinned);
		expect(resolveSelectedUnit(false, 0, pinned)).toEqual({ section: 'steps', index: 0 });
		expect(resolveSelectedUnit(true, 0, pinned)).toBeNull();
	});
});

describe('UnitHighlight', () => {
	it('pins units; edit focus from inputs wins', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => undefined
		});

		highlight.pinUnit({ section: 'steps', index: 1 });
		expect(highlight.selectedUnit).toEqual({ section: 'steps', index: 1 });

		const editing = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => 0
		});
		editing.pinUnit({ section: 'steps', index: 1 });
		expect(editing.selectedUnit).toEqual({ section: 'steps', index: 0 });
		expect(editing.isCardSelected('steps', 0)).toBe(true);
		expect(editing.isCardSelected('steps', 1)).toBe(false);

		highlight.dispose();
		editing.dispose();
	});

	it('clears selection wash when inputs say manual (pin retained)', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => true,
			getEditingIndex: () => undefined
		});

		highlight.pinUnit({ section: 'steps', index: 0 });
		expect(highlight.pinnedUnit).toEqual({ section: 'steps', index: 0 });
		expect(highlight.selectedUnit).toBeNull();
		highlight.dispose();
	});

	it('hovers cards; leave only clears owning card; clearHover always clears', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => undefined
		});

		highlight.hoverCard({ section: 'steps', index: 0 });
		expect(highlight.isCardHovered('steps', 0)).toBe(true);

		highlight.clearHoverCard({ section: 'steps', index: 1 });
		expect(highlight.isCardHovered('steps', 0)).toBe(true);

		highlight.clearHoverCard({ section: 'steps', index: 0 });
		expect(highlight.hoveredUnit).toBeNull();

		highlight.hoverCard({ section: 'steps', index: 1 });
		highlight.clearHover();
		expect(highlight.hoveredUnit).toBeNull();
		highlight.dispose();
	});
});
