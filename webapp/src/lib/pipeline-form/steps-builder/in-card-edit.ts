// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ActiveUnit } from './scroll-follow/active-unit.js';
import type { CardSection } from './scroll-follow/yaml-ranges.js';

/**
 * Structural view of the Steps builder mode — only the fields In-card edit reads.
 * Kept structural so this module stays pure (no import of the builder / its Svelte component).
 */
export type InCardEditMode =
	| { id: 'form'; intent: string; stepIndex?: number; section?: CardSection }
	| { id: 'idle' }
	| { id: 'manual' };

/** True while a Step or Follow-up card hosts its edit form (form mode, edit intent). */
export function isInCardEdit(mode: InCardEditMode): boolean {
	return mode.id === 'form' && mode.intent === 'edit';
}

/**
 * Form-host classes for In-card enter vs settled/exit.
 * Enter keeps absolute fill over the growing lock, but must still be a flex column
 * so the shell's grow pushes Save to the column bottom during the grow (not only
 * after `enterComplete`).
 */
export function inCardFormHostClass(settled: boolean): string {
	if (settled) return 'flex min-h-0 grow flex-col overflow-hidden';
	return 'pointer-events-none invisible absolute inset-0 flex min-h-0 flex-col overflow-hidden opacity-0';
}

/** The unit being edited in place, or null when not in In-card edit (or no index yet). */
export function editingUnit(mode: InCardEditMode): ActiveUnit | null {
	if (mode.id !== 'form' || mode.intent !== 'edit') return null;
	if (mode.stepIndex === undefined) return null;
	return { section: mode.section ?? 'steps', index: mode.stepIndex };
}
