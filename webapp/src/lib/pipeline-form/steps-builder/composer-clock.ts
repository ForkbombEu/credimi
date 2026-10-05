// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Shared clock surface for Pipeline Composer virtualizer + peer scroll-follow.
 * Production uses `browserClock()`; tests inject `FakeClock`.
 */
export type ComposerClock = {
	now: () => number;
	raf: (callback: FrameRequestCallback) => number;
	cancelRaf: (handle: number) => void;
	setTimeout: (handler: () => void, timeout?: number) => ReturnType<typeof setTimeout>;
	clearTimeout: (handle: ReturnType<typeof setTimeout>) => void;
};

export function browserClock(): ComposerClock {
	return {
		now: () => globalThis.performance?.now?.() ?? Date.now(),
		raf: (...args) => globalThis.requestAnimationFrame(...args),
		cancelRaf: (...args) => globalThis.cancelAnimationFrame(...args),
		setTimeout: (...args) => globalThis.setTimeout(...args),
		clearTimeout: (...args) => globalThis.clearTimeout(...args)
	};
}
