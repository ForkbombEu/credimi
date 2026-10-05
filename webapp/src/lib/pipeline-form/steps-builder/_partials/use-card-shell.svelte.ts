// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { onDestroy } from 'svelte';

import type { StepsBuilder } from '../steps-builder.svelte.js';

import { useHeldFormMode } from './use-held-form-mode.svelte.js';

/**
 * Drop the held form and unlock Twin-pane still-ness in the same moment.
 * `noteExitComplete` is idempotent — exit-complete plus onDestroy may both invoke this.
 */
export function completeInCardExit(clearHeld: () => void, onExitUnlock: () => void): void {
	clearHeld();
	onExitUnlock();
}

/**
 * Destroy must unlock when this card still holds the exit form (motion cancelled
 * or remount skipped `onExitComplete`). Do not unlock while still editing — a
 * virtualizer remount of the open card must not idle the pane.
 */
export function shouldCompleteExitOnDestroy(holdingForm: boolean, editing: boolean): boolean {
	return holdingForm && !editing;
}

/**
 * Held form + paired unlock for Step / Follow-up cards.
 */
export function useCardShell(
	getBuilder: () => StepsBuilder,
	getEditing: () => boolean,
	onExitUnlock: () => void
) {
	const held = useHeldFormMode(getBuilder, getEditing);

	function completeExit() {
		completeInCardExit(() => held.clear(), onExitUnlock);
	}

	onDestroy(() => {
		if (!shouldCompleteExitOnDestroy(held.mode !== null, getEditing())) return;
		completeExit();
	});

	return {
		get mode() {
			return held.mode;
		},
		completeExit
	};
}
