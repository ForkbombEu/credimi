// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { StepsBuilder } from '../steps-builder.svelte.js';

//

type FormMode = Extract<StepsBuilder['mode'], { id: 'form' }>;

/**
 * Exposes the builder's form mode while a card is editing, and keeps the last one around
 * after the card stops editing so the in-card shell can render the form during its collapse
 * animation (the builder drops the mode immediately on save / dismiss).
 * Call `clear()` from the shell's `onCollapsed` so the held form does not stick forever.
 */
export function useHeldFormMode(getBuilder: () => StepsBuilder, getEditing: () => boolean) {
	let last = $state.raw<FormMode | null>(null);

	const live = $derived.by(() => {
		if (!getEditing()) return null;
		const mode = getBuilder().mode;
		return mode.id === 'form' ? mode : null;
	});

	$effect.pre(() => {
		if (live) last = live;
	});

	return {
		get mode(): FormMode | null {
			return live ?? last;
		},
		clear() {
			last = null;
		}
	};
}
