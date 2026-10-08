// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * ADR 0012 still-ness phases. Cards unlock only via {@link InCardSession.noteExitComplete}.
 */
export type InCardPhase = 'idle' | 'aligning' | 'still' | 'exiting';

export function isInCardStill(phase: InCardPhase): boolean {
	return phase === 'still' || phase === 'exiting';
}

/** Enter expand waits for still — not aligning. */
export function isInCardExpandReady(editing: boolean, phase: InCardPhase): boolean {
	return editing && phase === 'still';
}

export type InCardSession = {
	get phase(): InCardPhase;
	get still(): boolean;
	/**
	 * Edit-focus start: enter start-align in flight. Resets from any phase
	 * (self-heal if a prior host failed to unlock).
	 */
	noteEnterStart(): void;
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
	dispose(): void;
};

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
