// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * animejs v4 helpers for the in-card form shell (Pipeline Composer).
 *
 * Enter sequence (after scroll settles):
 * 1. lock card body height to the current summary height
 * 2. fade summary out (height unchanged)
 * 3. fade form in (still at locked height)
 * 4. grow the body to the column body max (or form natural height)
 * 5. caller applies settled flex classes, then `settleEnterFormHost` clears overlay
 *
 * - One running animation per element: starting a new one cancels the previous
 *   one in place (no revert), so retargeting mid-flight continues from the
 *   current height/opacity instead of jumping.
 * - `cancel()` and `cancelMotion()` are idempotent and never throw.
 * - Honors `prefers-reduced-motion` by applying the end state synchronously.
 */

import { animate, type JSAnimation } from 'animejs';

//

export const IN_CARD_MOTION_MS = 200;
export const IN_CARD_MOTION_EASE = 'outQuad';

export type MotionHandle = {
	/** Stops the animation, keeping the current visual state. Safe to call repeatedly. */
	cancel: () => void;
	/** Resolves when the animation completes or is cancelled. Never rejects. */
	finished: Promise<void>;
};

export type MotionOptions = {
	durationMs?: number;
	/** Runs only when the animation reaches its end (not when cancelled). */
	onComplete?: () => void | Promise<void>;
};

export type InCardEnterOptions = MotionOptions & {
	/** Height-locked wrapper around summary + form. */
	lock: HTMLElement;
	/** Summary / details layer that fades out first. */
	display: HTMLElement;
	/** Form host that fades in at the locked height, then drives the grow. */
	form: HTMLElement;
	/**
	 * Cap (px) for the grown *body* height (below the type header / color bar).
	 * Pass `bodyMaxHeightWithinCard(...)` so this matches the card's max-height
	 * minus chrome. When set, enter grows to fill this height (form scrolls inside);
	 * when omitted, enter grows to the form's natural height.
	 */
	maxHeightPx?: number;
};

/**
 * How tall the form body may grow when the whole card is capped at `cardMaxHeightPx`.
 * Chrome = color bar + type header (everything in the card above `display`/`lock`).
 */
export function bodyMaxHeightWithinCard(
	card: HTMLElement,
	display: HTMLElement,
	cardMaxHeightPx: number
): number {
	const cardHeight = card.getBoundingClientRect().height;
	const displayHeight = display.getBoundingClientRect().height;
	const chromePx = Math.max(0, cardHeight - displayHeight);
	return Math.max(0, cardMaxHeightPx - chromePx);
}

//

const running = new WeakMap<Element, MotionHandle>();

function prefersReducedMotion(): boolean {
	return (
		typeof window !== 'undefined' &&
		typeof window.matchMedia === 'function' &&
		window.matchMedia('(prefers-reduced-motion: reduce)').matches
	);
}

const NOOP_HANDLE: MotionHandle = { cancel: () => {}, finished: Promise.resolve() };

/** Cancels the animation currently running on `el`, if any. Leaves styles as they are. */
export function cancelMotion(el: Element | null | undefined): void {
	if (!el) return;
	running.get(el)?.cancel();
	running.delete(el);
}

type Props = Record<string, unknown>;

function run(el: HTMLElement, props: Props, options: MotionOptions, onDone?: () => void) {
	cancelMotion(el);

	let resolve: () => void = () => {};
	const finished = new Promise<void>((r) => {
		resolve = r;
	});

	const handle: MotionHandle = {
		finished,
		cancel: () => {
			if (running.get(el) === handle) running.delete(el);
			try {
				animation.cancel();
			} catch {
				// already disposed
			}
			resolve();
		}
	};
	running.set(el, handle);

	const animation: JSAnimation = animate(el, {
		...props,
		duration: options.durationMs ?? IN_CARD_MOTION_MS,
		ease: IN_CARD_MOTION_EASE,
		onComplete: () => {
			if (running.get(el) === handle) running.delete(el);
			onDone?.();
			options.onComplete?.();
			resolve();
		}
	});

	return handle;
}

/**
 * Natural height of the absolutely-overlaid form host.
 * Must clear `inset`/`bottom` — with top+bottom pinned, `height: auto` still equals the lock.
 */
function measureFormNaturalHeight(form: HTMLElement): number {
	const style = form.style;
	const previous = {
		position: style.position,
		inset: style.inset,
		top: style.top,
		right: style.right,
		bottom: style.bottom,
		left: style.left,
		width: style.width,
		height: style.height,
		overflow: style.overflow
	};

	style.position = 'absolute';
	style.inset = '';
	style.top = '0';
	style.left = '0';
	style.right = '0';
	style.bottom = 'auto';
	style.width = '100%';
	style.height = 'auto';
	style.overflow = 'visible';

	const natural = form.getBoundingClientRect().height;

	style.position = previous.position;
	style.inset = previous.inset;
	style.top = previous.top;
	style.right = previous.right;
	style.bottom = previous.bottom;
	style.left = previous.left;
	style.width = previous.width;
	style.height = previous.height;
	style.overflow = previous.overflow;

	return natural;
}

function clearInlineBox(el: HTMLElement) {
	el.style.height = '';
	el.style.overflow = '';
	el.style.opacity = '';
}

function prepareFormForEnter(form: HTMLElement) {
	form.style.position = 'absolute';
	form.style.inset = '0';
	form.style.width = '100%';
	form.style.height = '100%';
	form.style.overflow = 'hidden';
	form.style.opacity = '0';
	form.style.visibility = 'visible';
	form.style.pointerEvents = 'none';
}

function finishEnterLayout(
	lock: HTMLElement,
	display: HTMLElement,
	form: HTMLElement,
	/** Keep the lock at this height so the card fills the column after grow. */
	lockHeightPx?: number
) {
	display.style.opacity = '0';
	display.style.pointerEvents = 'none';
	display.style.visibility = 'hidden';

	// Keep the form absolutely filling the lock until `settleEnterFormHost` runs.
	// Clearing absolute before settled flex host classes paint would collapse the
	// overlay for a frame and jump the save footer.
	prepareFormForEnter(form);
	form.style.opacity = '1';
	form.style.pointerEvents = '';

	if (typeof lockHeightPx === 'number' && Number.isFinite(lockHeightPx)) {
		lock.style.overflow = 'hidden';
		lock.style.height = `${lockHeightPx}px`;
		lock.style.opacity = '';
	} else {
		clearInlineBox(lock);
	}
}

/**
 * Clears enter overlay styles after the caller has applied the settled flex
 * host classes (e.g. after `enterComplete` + `tick()`).
 */
export function settleEnterFormHost(form: HTMLElement) {
	form.style.position = '';
	form.style.inset = '';
	form.style.width = '';
	form.style.height = '';
	form.style.overflow = '';
	form.style.opacity = '';
	form.style.visibility = '';
	form.style.pointerEvents = '';
}

async function afterEnterComplete(
	onComplete: (() => void | Promise<void>) | undefined,
	form: HTMLElement,
	isCancelled: () => boolean
) {
	await Promise.resolve(onComplete?.());
	if (isCancelled()) return;
	settleEnterFormHost(form);
}

/** Fill the column when a body max is provided; otherwise size to form content. */
function resolveEnterHeight(form: HTMLElement, maxHeightPx?: number): number {
	if (typeof maxHeightPx === 'number' && Number.isFinite(maxHeightPx)) {
		return Math.max(0, maxHeightPx);
	}
	return measureFormNaturalHeight(form);
}

//

/**
 * Enter motion after the card has scrolled into place:
 * fade summary (height held) → fade form in → grow to fill the column body
 * (or the form's natural height when no max is provided).
 */
export function playInCardEnter(options: InCardEnterOptions): MotionHandle {
	const { lock, display, form, maxHeightPx, onComplete, durationMs } = options;

	cancelMotion(lock);
	cancelMotion(display);
	cancelMotion(form);

	const fromHeight = display.getBoundingClientRect().height;

	lock.style.overflow = 'hidden';
	lock.style.height = `${fromHeight}px`;
	prepareFormForEnter(form);

	if (prefersReducedMotion()) {
		let cancelled = false;
		const finished = (async () => {
			// Measure natural height with bottom:auto when no body max is set.
			form.style.position = 'absolute';
			form.style.inset = '';
			form.style.top = '0';
			form.style.left = '0';
			form.style.right = '0';
			form.style.bottom = 'auto';
			form.style.width = '100%';
			form.style.height = 'auto';
			form.style.opacity = '1';
			form.style.visibility = 'visible';
			form.style.pointerEvents = '';
			form.style.overflow = 'hidden';
			const toHeight = resolveEnterHeight(form, maxHeightPx);
			lock.style.height = `${toHeight}px`;
			finishEnterLayout(lock, display, form, toHeight);
			await afterEnterComplete(onComplete, form, () => cancelled);
		})();
		return {
			cancel: () => {
				cancelled = true;
			},
			finished
		};
	}

	let cancelled = false;
	let step: MotionHandle | undefined;

	let resolveFinished: () => void = () => {};
	const finished = new Promise<void>((r) => {
		resolveFinished = r;
	});

	const handle: MotionHandle = {
		finished,
		cancel: () => {
			cancelled = true;
			step?.cancel();
			resolveFinished();
		}
	};

	const completeEnter = (toHeight: number) => {
		finishEnterLayout(lock, display, form, toHeight);
		void afterEnterComplete(onComplete, form, () => cancelled).then(() => {
			if (!cancelled) resolveFinished();
		});
	};

	const stepOpts = { durationMs };

	const grow = () => {
		if (cancelled) return;
		const toHeight = resolveEnterHeight(form, maxHeightPx);
		// Keep the form clipped to the lock while the lock height animates.
		prepareFormForEnter(form);
		form.style.opacity = '1';
		form.style.pointerEvents = '';
		const from = lock.getBoundingClientRect().height;
		if (Math.abs(toHeight - from) < 1) {
			completeEnter(toHeight);
			return;
		}
		step = run(
			lock,
			{ height: [`${from}px`, `${toHeight}px`] },
			stepOpts,
			() => {
				if (cancelled) return;
				completeEnter(toHeight);
			}
		);
	};

	const fadeFormIn = () => {
		if (cancelled) return;
		form.style.pointerEvents = 'none';
		step = run(form, { opacity: 1 }, stepOpts, () => {
			if (cancelled) return;
			form.style.pointerEvents = '';
			grow();
		});
	};

	// 1) Summary fades out; height stays locked.
	step = run(display, { opacity: 0 }, stepOpts, () => {
		if (cancelled) return;
		display.style.pointerEvents = 'none';
		// 2) Form fades in at the locked height, then 3) grow.
		fadeFormIn();
	});

	return handle;
}

export type InCardExitOptions = MotionOptions & {
	lock: HTMLElement;
	display: HTMLElement;
	form: HTMLElement;
};

/**
 * Exit motion (reverse of enter):
 * shrink body to the summary height → fade form out → fade summary in.
 */
export function playInCardExit(options: InCardExitOptions): MotionHandle {
	const { lock, display, form, onComplete, durationMs } = options;

	cancelMotion(lock);
	cancelMotion(display);
	cancelMotion(form);

	const fromHeight = lock.getBoundingClientRect().height;

	lock.style.overflow = 'hidden';
	lock.style.height = `${fromHeight}px`;

	// Both layers overlay the lock so remounting the summary does not inflate height.
	prepareFormForEnter(form);
	form.style.opacity = '1';
	form.style.pointerEvents = 'none';

	display.style.position = 'absolute';
	display.style.inset = '';
	display.style.top = '0';
	display.style.left = '0';
	display.style.right = '0';
	display.style.bottom = 'auto';
	display.style.width = '100%';
	display.style.height = 'auto';
	display.style.overflow = 'visible';
	display.style.opacity = '0';
	display.style.visibility = 'visible';
	display.style.pointerEvents = 'none';

	const toHeight = measureFormNaturalHeight(display);

	const finish = () => {
		display.style.position = '';
		display.style.inset = '';
		display.style.top = '';
		display.style.left = '';
		display.style.right = '';
		display.style.bottom = '';
		display.style.width = '';
		display.style.height = '';
		display.style.overflow = '';
		display.style.opacity = '';
		display.style.visibility = '';
		display.style.pointerEvents = '';

		form.style.position = '';
		form.style.inset = '';
		form.style.width = '';
		form.style.height = '';
		form.style.overflow = '';
		form.style.opacity = '0';
		form.style.visibility = 'hidden';
		form.style.pointerEvents = 'none';

		clearInlineBox(lock);
	};

	if (prefersReducedMotion()) {
		lock.style.height = `${toHeight}px`;
		finish();
		onComplete?.();
		return NOOP_HANDLE;
	}

	let cancelled = false;
	let step: MotionHandle | undefined;

	let resolveFinished: () => void = () => {};
	const finished = new Promise<void>((r) => {
		resolveFinished = r;
	});

	const handle: MotionHandle = {
		finished,
		cancel: () => {
			cancelled = true;
			step?.cancel();
			resolveFinished();
		}
	};

	const stepOpts = { durationMs };

	const fadeDisplayIn = () => {
		if (cancelled) return;
		step = run(display, { opacity: 1 }, stepOpts, () => {
			if (cancelled) return;
			finish();
			onComplete?.();
			resolveFinished();
		});
	};

	const fadeFormOut = () => {
		if (cancelled) return;
		step = run(form, { opacity: 0 }, stepOpts, () => {
			if (cancelled) return;
			fadeDisplayIn();
		});
	};

	const shrink = () => {
		if (cancelled) return;
		const from = lock.getBoundingClientRect().height;
		if (Math.abs(toHeight - from) < 1) {
			fadeFormOut();
			return;
		}
		step = run(lock, { height: [`${from}px`, `${toHeight}px`] }, stepOpts, () => {
			if (cancelled) return;
			fadeFormOut();
		});
	};

	shrink();
	return handle;
}
