// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	UnitHighlight,
	remapUnitAfterAdjacentSwap,
	resolveSelectedUnit
} from './unit-highlight.svelte.js';

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

	it('remapStepsAfterAdjacentSwap keeps pin and hover on the moved step identity', () => {
		const highlight = new UnitHighlight({
			getIsManual: () => false,
			getEditingIndex: () => undefined
		});

		highlight.pinUnit({ section: 'steps', index: 1 });
		highlight.hoverCard({ section: 'steps', index: 2 });

		// Swap 1 ↔ 2: selected step moves to index 2; hovered partner to 1.
		highlight.remapStepsAfterAdjacentSwap(1, 2);
		expect(highlight.pinnedUnit).toEqual({ section: 'steps', index: 2 });
		expect(highlight.hoveredUnit).toEqual({ section: 'steps', index: 1 });
		expect(highlight.isCardSelected('steps', 2)).toBe(true);
		expect(highlight.isCardSelected('steps', 1)).toBe(false);

		// Unrelated swap leaves pin alone.
		highlight.remapStepsAfterAdjacentSwap(3, 4);
		expect(highlight.pinnedUnit).toEqual({ section: 'steps', index: 2 });

		// Follow-ups pins are not remapped by a steps swap.
		highlight.pinUnit({ section: 'follow-ups', index: 0 });
		highlight.remapStepsAfterAdjacentSwap(0, 1);
		expect(highlight.pinnedUnit).toEqual({ section: 'follow-ups', index: 0 });

		highlight.dispose();
	});
});

describe('remapUnitAfterAdjacentSwap', () => {
	it('swaps the two indices; leaves others and other sections alone', () => {
		expect(remapUnitAfterAdjacentSwap(null, 1, 2)).toBeNull();
		expect(remapUnitAfterAdjacentSwap({ section: 'steps', index: 1 }, 1, 2)).toEqual({
			section: 'steps',
			index: 2
		});
		expect(remapUnitAfterAdjacentSwap({ section: 'steps', index: 2 }, 1, 2)).toEqual({
			section: 'steps',
			index: 1
		});
		expect(remapUnitAfterAdjacentSwap({ section: 'steps', index: 0 }, 1, 2)).toEqual({
			section: 'steps',
			index: 0
		});
		expect(remapUnitAfterAdjacentSwap({ section: 'follow-ups', index: 1 }, 1, 2)).toEqual({
			section: 'follow-ups',
			index: 1
		});
	});
});
