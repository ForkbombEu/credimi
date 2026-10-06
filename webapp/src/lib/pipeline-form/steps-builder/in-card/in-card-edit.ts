// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ActiveUnit } from '../scroll-follow/active-unit.js';

/**
 * Structural view of the Steps builder mode — only the fields In-card edit reads.
 * Kept structural so this module stays pure (no import of the builder / its Svelte component).
 */
export type InCardEditMode =
	| { id: 'form'; intent: string; stepIndex?: number; section?: ActiveUnit['section'] }
	| { id: 'idle' }
	| { id: 'manual' };

export function isInCardEdit(mode: InCardEditMode): boolean {
	return mode.id === 'form' && mode.intent === 'edit';
}

export function editingUnit(mode: InCardEditMode): ActiveUnit | null {
	if (mode.id !== 'form' || mode.intent !== 'edit') return null;
	if (mode.stepIndex === undefined) return null;
	return { section: mode.section ?? 'steps', index: mode.stepIndex };
}
