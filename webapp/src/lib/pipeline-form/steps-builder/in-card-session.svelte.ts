// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * ADR 0012 still-ness phases for Pipeline Composer In-card edit.
 * Twin-pane session owns this module and drives enter/exit transitions;
 * cards unlock only via {@link InCardSession.noteExitComplete}.
 */
export type InCardPhase = 'idle' | 'aligning' | 'still' | 'exiting';

/** True while the cards pane must stay still (overflow lock, wheel block, peer mute). */
export function isInCardStill(phase: InCardPhase): boolean {
	return phase === 'still' || phase === 'exiting';
}

/**
 * Enter-settled phases — expand/crossfade may run (same set as still-ness today;
 * kept separate so a later deepen can diverge if needed).
 */
export function isInCardEnterSettled(phase: InCardPhase): boolean {
	return phase === 'still' || phase === 'exiting';
}

export type InCardSession = {
	get phase(): InCardPhase;
	/** Derived still-ness — `still | exiting`. */
	get still(): boolean;
	/**
	 * Edit-focus start: enter start-align in flight. Resets from any phase
	 * (self-heal if a prior host failed to unlock).
	 */
	noteEnterStart(): void;
	/**
	 * Enter start-align settled. → `still` while still In-card edit; else → `idle`.
	 */
	noteEnterSettled(isInCardEdit: boolean): void;
	/**
	 * Builder left form mode while `still` — held form may still be mounted.
	 * Stays still until {@link noteExitComplete}.
	 */
	noteEditingEnded(): void;
	/**
	 * Sole unlock. Cards call from exit-complete AND onDestroy/abort.
	 * Idempotent. Always → `idle`.
	 */
	noteExitComplete(): void;
	/** Twin-pane dispose — always → `idle`. */
	dispose(): void;
};

/**
 * Nested In-card phase machine. Created inside {@link createTwinPaneSession};
 * public Twin-pane surface re-exports `phase` + `noteExitComplete` only.
 */
export function createInCardSession(): InCardSession {
	let phase = $state<InCardPhase>('idle');
	let disposed = false;

	function noteEnterStart() {
		if (disposed) return;
		phase = 'aligning';
	}

	function noteEnterSettled(isInCardEdit: boolean) {
		if (disposed) {
			phase = 'idle';
			return;
		}
		phase = isInCardEdit ? 'still' : 'idle';
	}

	function noteEditingEnded() {
		if (disposed) return;
		if (phase !== 'still') return;
		phase = 'exiting';
	}

	function noteExitComplete() {
		phase = 'idle';
	}

	function dispose() {
		disposed = true;
		phase = 'idle';
	}

	return {
		get phase() {
			return phase;
		},
		get still() {
			return isInCardStill(phase);
		},
		noteEnterStart,
		noteEnterSettled,
		noteEditingEnded,
		noteExitComplete,
		dispose
	};
}
