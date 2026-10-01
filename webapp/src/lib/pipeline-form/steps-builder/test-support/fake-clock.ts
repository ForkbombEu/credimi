// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ComposerClock } from '../composer-clock.js';

/**
 * Deterministic raf + setTimeout without happy-dom/jsdom timers.
 * Implements ComposerClock plus flush helpers for unit tests.
 */
export type FakeClock = ComposerClock & {
	flushRaf(): void;
	/** Advance virtual time and fire due timers (composer-virtualizer call site). */
	advance(ms: number): void;
	/** Advance virtual time and fire due timers (peer-scroll-follow call site; ms defaults to 0). */
	flushTimeouts(ms?: number): void;
};

export function createFakeClock(): FakeClock {
	let nextRaf = 1;
	const rafQueue = new Map<number, FrameRequestCallback>();
	let nextTimer = 1;
	const timers = new Map<number, { due: number; handler: () => void }>();
	let now = 0;

	const advance = (ms: number) => {
		now += ms;
		const due = [...timers.entries()].filter(([, t]) => t.due <= now);
		for (const [id, t] of due) {
			timers.delete(id);
			t.handler();
		}
	};

	return {
		now: () => now,
		raf(callback) {
			const id = nextRaf++;
			rafQueue.set(id, callback);
			return id;
		},
		cancelRaf(handle) {
			rafQueue.delete(handle);
		},
		setTimeout(handler, timeout = 0) {
			const id = nextTimer++;
			timers.set(id, { due: now + timeout, handler });
			return id as unknown as ReturnType<typeof setTimeout>;
		},
		clearTimeout(handle) {
			timers.delete(handle as unknown as number);
		},
		flushRaf() {
			const queued = [...rafQueue.entries()];
			rafQueue.clear();
			for (const [, cb] of queued) cb(now);
		},
		advance,
		flushTimeouts(ms = 0) {
			advance(ms);
		}
	};
}
