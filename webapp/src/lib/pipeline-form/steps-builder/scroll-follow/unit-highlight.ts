// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { sameUnit, type ActiveUnit } from './active-unit.js';
import { findNearestUnitToLine, findRangeForUnit, findUnitAtLine, type YamlCardRange } from './yaml-ranges.js';

/** Inclusive 0-based line span for CodeDisplay washes. */
export type LineRange = {
	start: number;
	end: number;
};

export type UnitHighlightSnapshot = {
	hoveredUnit: ActiveUnit | null;
	selectedUnit: ActiveUnit | null;
	selectedLines: LineRange | null;
	hoverLines: LineRange | null;
};

export type UnitHighlight = {
	readonly snapshot: UnitHighlightSnapshot;
	subscribe(listener: () => void): () => void;
	/** Manual mode clears selection wash; hover may still clear via leave handlers. */
	setManual(isManual: boolean): void;
	setEditingIndex(stepIndex: number | undefined): void;
	setRanges(ranges: YamlCardRange[]): void;
	hoverCard(unit: ActiveUnit): void;
	/** Clear hover only when the leaving card still owns hover (avoids flicker). */
	clearHoverCard(unit: ActiveUnit): void;
	hoverYamlLine(line: number | null): void;
	/**
	 * Pin selection from an exact YAML line hit. Returns the pinned unit, or null
	 * when the line is outside any step range (caller should not peer-follow).
	 */
	pinYamlLine(line: number): ActiveUnit | null;
	isCardHovered(section: ActiveUnit['section'], index: number): boolean;
	isCardSelected(section: ActiveUnit['section'], index: number): boolean;
	dispose(): void;
};

function linesForUnit(unit: ActiveUnit | null, ranges: YamlCardRange[]): LineRange | null {
	if (!unit) return null;
	const range = findRangeForUnit(ranges, unit.section, unit.index);
	if (!range) return null;
	return { start: range.startLine, end: range.endLine };
}

function resolveSelectedUnit(
	isManual: boolean,
	editingIndex: number | undefined,
	pinnedUnit: ActiveUnit | null
): ActiveUnit | null {
	if (isManual) return null;
	if (editingIndex !== undefined) return { section: 'steps', index: editingIndex };
	return pinnedUnit;
}

export function createUnitHighlight(): UnitHighlight {
	let isManual = false;
	let editingIndex: number | undefined;
	let ranges: YamlCardRange[] = [];
	let hoveredUnit: ActiveUnit | null = null;
	let pinnedUnit: ActiveUnit | null = null;
	let disposed = false;
	const listeners = new Set<() => void>();

	function notify() {
		for (const listener of listeners) listener();
	}

	function selectedUnit(): ActiveUnit | null {
		return resolveSelectedUnit(isManual, editingIndex, pinnedUnit);
	}

	function buildSnapshot(): UnitHighlightSnapshot {
		const selected = selectedUnit();
		return {
			hoveredUnit,
			selectedUnit: selected,
			selectedLines: linesForUnit(selected, ranges),
			hoverLines: linesForUnit(hoveredUnit, ranges)
		};
	}

	function setHovered(next: ActiveUnit | null) {
		if (sameUnit(hoveredUnit, next)) return;
		hoveredUnit = next;
		notify();
	}

	function subscribe(listener: () => void): () => void {
		listeners.add(listener);
		return () => {
			listeners.delete(listener);
		};
	}

	function setManual(next: boolean) {
		if (disposed) return;
		if (isManual === next) return;
		isManual = next;
		notify();
	}

	function setEditingIndex(stepIndex: number | undefined) {
		if (disposed) return;
		if (editingIndex === stepIndex) return;
		editingIndex = stepIndex;
		notify();
	}

	function setRanges(next: YamlCardRange[]) {
		if (disposed) return;
		if (ranges === next) return;
		ranges = next;
		notify();
	}

	function hoverCard(unit: ActiveUnit) {
		if (disposed) return;
		setHovered(unit);
	}

	function clearHoverCard(unit: ActiveUnit) {
		if (disposed) return;
		if (!sameUnit(hoveredUnit, unit)) return;
		setHovered(null);
	}

	function hoverYamlLine(line: number | null) {
		if (disposed) return;
		if (line === null) {
			setHovered(null);
			return;
		}
		const hit = findNearestUnitToLine(ranges, line);
		if (!hit) return;
		setHovered({ section: hit.section, index: hit.index });
	}

	function pinYamlLine(line: number): ActiveUnit | null {
		if (disposed) return null;
		const hit = findUnitAtLine(ranges, line);
		if (!hit) return null;
		const next: ActiveUnit = { section: hit.section, index: hit.index };
		if (!sameUnit(pinnedUnit, next)) {
			pinnedUnit = next;
			notify();
		}
		return next;
	}

	function isCardHovered(section: ActiveUnit['section'], index: number): boolean {
		return hoveredUnit?.section === section && hoveredUnit.index === index;
	}

	function isCardSelected(section: ActiveUnit['section'], index: number): boolean {
		const selected = selectedUnit();
		return selected?.section === section && selected.index === index;
	}

	function dispose() {
		if (disposed) return;
		disposed = true;
		listeners.clear();
		hoveredUnit = null;
		pinnedUnit = null;
		ranges = [];
	}

	return {
		get snapshot() {
			return buildSnapshot();
		},
		subscribe,
		setManual,
		setEditingIndex,
		setRanges,
		hoverCard,
		clearHoverCard,
		hoverYamlLine,
		pinYamlLine,
		isCardHovered,
		isCardSelected,
		dispose
	};
}
