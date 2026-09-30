// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { sameUnit, type ActiveUnit } from './active-unit.js';
import {
	findNearestUnitToLine,
	findRangeForUnit,
	findUnitAtLine,
	type YamlCardRange
} from './yaml-ranges.js';

/** Inclusive 0-based line span for CodeDisplay washes. */
export type LineRange = {
	start: number;
	end: number;
};

export type UnitHighlightInputs = {
	getIsManual: () => boolean;
	getEditingIndex: () => number | undefined;
	/** Section being edited; defaults to steps when omitted. */
	getEditingSection?: () => ActiveUnit['section'];
	getRanges: () => YamlCardRange[];
};

export function linesForUnit(unit: ActiveUnit | null, ranges: YamlCardRange[]): LineRange | null {
	if (!unit) return null;
	const range = findRangeForUnit(ranges, unit.section, unit.index);
	if (!range) return null;
	return { start: range.startLine, end: range.endLine };
}

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

	selectedLines = $derived.by(() => linesForUnit(this.selectedUnit, this.#getRanges()));

	hoverLines = $derived.by(() => linesForUnit(this.hoveredUnit, this.#getRanges()));

	#getIsManual: () => boolean;
	#getEditingIndex: () => number | undefined;
	#getEditingSection: () => ActiveUnit['section'];
	#getRanges: () => YamlCardRange[];
	#disposed = false;

	constructor(inputs: UnitHighlightInputs) {
		this.#getIsManual = inputs.getIsManual;
		this.#getEditingIndex = inputs.getEditingIndex;
		this.#getEditingSection = inputs.getEditingSection ?? (() => 'steps');
		this.#getRanges = inputs.getRanges;
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

	hoverYamlLine(line: number | null) {
		if (this.#disposed) return;
		if (line === null) {
			this.#setHovered(null);
			return;
		}
		const hit = findNearestUnitToLine(this.#getRanges(), line);
		if (!hit) return;
		this.#setHovered({ section: hit.section, index: hit.index });
	}

	/**
	 * Pin selection from an exact YAML line hit. Returns the pinned unit, or null
	 * when the line is outside any step range (caller should not peer-follow).
	 */
	pinYamlLine(line: number): ActiveUnit | null {
		if (this.#disposed) return null;
		const hit = findUnitAtLine(this.#getRanges(), line);
		if (!hit) return null;
		const next: ActiveUnit = { section: hit.section, index: hit.index };
		if (!sameUnit(this.pinnedUnit, next)) {
			this.pinnedUnit = next;
		}
		return next;
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
