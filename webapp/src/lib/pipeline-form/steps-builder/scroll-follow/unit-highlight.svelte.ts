// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { sameUnit, type ActiveUnit } from './active-unit.js';

export type UnitHighlightInputs = {
	getIsManual: () => boolean;
	getEditingIndex: () => number | undefined;
	/** Section being edited; defaults to steps when omitted. */
	getEditingSection?: () => ActiveUnit['section'];
};

export function resolveSelectedUnit(
	isManual: boolean,
	editingIndex: number | undefined,
	pinnedUnit: ActiveUnit | null,
	editingSection: ActiveUnit['section'] = 'steps'
): ActiveUnit | null {
	if (isManual) return null;
	if (editingIndex !== undefined) return { section: editingSection, index: editingIndex };
	return pinnedUnit;
}

/**
 * After an adjacent steps swap (`fromIndex` ↔ `toIndex`), remap a unit so the
 * wash stays on the same step identity rather than the vacated slot index.
 */
export function remapUnitAfterAdjacentSwap(
	unit: ActiveUnit | null,
	fromIndex: number,
	toIndex: number
): ActiveUnit | null {
	if (!unit || unit.section !== 'steps') return unit;
	if (unit.index === fromIndex) return { section: 'steps', index: toIndex };
	if (unit.index === toIndex) return { section: 'steps', index: fromIndex };
	return unit;
}

/**
 * Hover / pin / edit-selected washes for Pipeline Composer cards and YAML.
 * Does not own scroll leadership (see PeerScrollFollow).
 */
export class UnitHighlight {
	hoveredUnit = $state.raw<ActiveUnit | null>(null);
	pinnedUnit = $state.raw<ActiveUnit | null>(null);

	selectedUnit = $derived.by(() =>
		resolveSelectedUnit(
			this.#getIsManual(),
			this.#getEditingIndex(),
			this.pinnedUnit,
			this.#getEditingSection()
		)
	);

	#getIsManual: () => boolean;
	#getEditingIndex: () => number | undefined;
	#getEditingSection: () => ActiveUnit['section'];
	#disposed = false;

	constructor(inputs: UnitHighlightInputs) {
		this.#getIsManual = inputs.getIsManual;
		this.#getEditingIndex = inputs.getEditingIndex;
		this.#getEditingSection = inputs.getEditingSection ?? (() => 'steps');
	}

	hoverCard(unit: ActiveUnit) {
		if (this.#disposed) return;
		this.#setHovered(unit);
	}

	/** Clear hover only when the leaving card still owns hover (avoids flicker). */
	clearHoverCard(unit: ActiveUnit) {
		if (this.#disposed) return;
		if (!sameUnit(this.hoveredUnit, unit)) return;
		this.#setHovered(null);
	}

	/** Unconditionally clear hover (YAML leave / pointer exit). */
	clearHover() {
		if (this.#disposed) return;
		this.#setHovered(null);
	}

	/** Pin selection from a YAML block / card unit (index-aligned preview). */
	pinUnit(unit: ActiveUnit): ActiveUnit | null {
		if (this.#disposed) return null;
		if (!sameUnit(this.pinnedUnit, unit)) {
			this.pinnedUnit = unit;
		}
		return unit;
	}

	/**
	 * Keep pin/hover on the moved steps after an adjacent reorder swap.
	 * Follow-ups are untouched (shift only moves steps).
	 */
	remapStepsAfterAdjacentSwap(fromIndex: number, toIndex: number) {
		if (this.#disposed) return;
		this.pinnedUnit = remapUnitAfterAdjacentSwap(this.pinnedUnit, fromIndex, toIndex);
		this.hoveredUnit = remapUnitAfterAdjacentSwap(this.hoveredUnit, fromIndex, toIndex);
	}

	isCardHovered(section: ActiveUnit['section'], index: number): boolean {
		return this.hoveredUnit?.section === section && this.hoveredUnit.index === index;
	}

	isCardSelected(section: ActiveUnit['section'], index: number): boolean {
		const selected = this.selectedUnit;
		return selected?.section === section && selected.index === index;
	}

	dispose() {
		if (this.#disposed) return;
		this.#disposed = true;
		this.hoveredUnit = null;
		this.pinnedUnit = null;
	}

	#setHovered(next: ActiveUnit | null) {
		if (sameUnit(this.hoveredUnit, next)) return;
		this.hoveredUnit = next;
	}
}
