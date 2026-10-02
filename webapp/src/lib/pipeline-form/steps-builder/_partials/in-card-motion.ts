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
 * 4. grow the body to the form's natural height
 *
 * - One running animation per element: starting a new one cancels the previous
 *   one in place (no revert), so retargeting (expand -> collapse mid-flight)
 *   continues from the current height/opacity instead of jumping.
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
	onComplete?: () => void;
};

export type ExpandOptions = MotionOptions & {
	/** Reset the region to height 0 / opacity 0 before animating (use right after mount). */
	startCollapsed?: boolean;
};

export type InCardEnterOptions = MotionOptions & {
	/** Height-locked wrapper around summary + form. */
	lock: HTMLElement;
	/** Summary / details layer that fades out first. */
	display: HTMLElement;
	/** Form host that fades in at the locked height, then drives the grow. */
	form: HTMLElement;
	/** Optional cap (px) for the grown height (card max-height). */
	maxHeightPx?: number;
};

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

function settled(onComplete?: () => void): MotionHandle {
	onComplete?.();
	return NOOP_HANDLE;
}

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

/** Natural height of `el` with `height: auto`, respecting any ancestor max-height. */
function measureAutoHeight(el: HTMLElement): number {
	const previous = el.style.height;
	el.style.height = 'auto';
	const natural = el.getBoundingClientRect().height;
	el.style.height = previous;
	return natural;
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

function finishEnterLayout(lock: HTMLElement, display: HTMLElement, form: HTMLElement) {
	display.style.opacity = '0';
	display.style.pointerEvents = 'none';
	display.style.visibility = 'hidden';

	form.style.position = '';
	form.style.inset = '';
	form.style.width = '';
	form.style.height = '';
	form.style.overflow = '';
	// Keep opacity:1 until the caller flips enterComplete (drops opacity-0 class), then clear.
	form.style.opacity = '1';
	form.style.visibility = '';
	form.style.pointerEvents = '';

	clearInlineBox(lock);
}

function clearFormEnterOpacity(form: HTMLElement) {
	form.style.opacity = '';
}

//

/** Animates a form region from its current size to its natural height and fades it in. */
export function expandRegion(el: HTMLElement, options: ExpandOptions = {}): MotionHandle {
	cancelMotion(el);

	if (options.startCollapsed) {
		el.style.height = '0px';
		el.style.opacity = '0';
	}

	const from = el.getBoundingClientRect().height;
	const to = measureAutoHeight(el);

	const finish = () => {
		clearInlineBox(el);
	};

	if (prefersReducedMotion()) {
		finish();
		return settled(options.onComplete);
	}

	el.style.overflow = 'hidden';
	el.style.height = `${from}px`;

	return run(el, { height: [`${from}px`, `${to}px`], opacity: 1 }, options, finish);
}

/** Animates a form region to height 0 and fades it out. */
export function collapseRegion(el: HTMLElement, options: MotionOptions = {}): MotionHandle {
	cancelMotion(el);

	if (prefersReducedMotion()) {
		el.style.overflow = 'hidden';
		el.style.height = '0px';
		el.style.opacity = '0';
		return settled(options.onComplete);
	}

	const from = el.getBoundingClientRect().height;
	el.style.overflow = 'hidden';
	el.style.height = `${from}px`;

	return run(el, { height: [`${from}px`, '0px'], opacity: 0 }, options);
}

/** Fades one or more elements to `opacity`. Cancelling the handle cancels all of them. */
export function fadeTo(
	targets: HTMLElement | readonly HTMLElement[],
	opacity: number,
	options: MotionOptions = {}
): MotionHandle {
	const elements = Array.isArray(targets) ? [...targets] : [targets as HTMLElement];
	if (elements.length === 0) return settled(options.onComplete);

	if (prefersReducedMotion()) {
		for (const el of elements) {
			cancelMotion(el);
			el.style.opacity = String(opacity);
		}
		return settled(options.onComplete);
	}

	let remaining = elements.length;
	const handles = elements.map((el) =>
		run(el, { opacity }, { durationMs: options.durationMs }, () => {
			remaining -= 1;
			if (remaining === 0) options.onComplete?.();
		})
	);

	return {
		cancel: () => {
			for (const handle of handles) handle.cancel();
		},
		finished: Promise.all(handles.map((handle) => handle.finished)).then(() => {})
	};
}

/**
 * Enter motion after the card has scrolled into place:
 * fade summary (height held) → fade form in → grow to form height.
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

	const snapOpen = () => {
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
		let toHeight = measureFormNaturalHeight(form);
		if (typeof maxHeightPx === 'number' && Number.isFinite(maxHeightPx)) {
			toHeight = Math.min(toHeight, maxHeightPx);
		}
		lock.style.height = `${toHeight}px`;
		finishEnterLayout(lock, display, form);
		onComplete?.();
		clearFormEnterOpacity(form);
	};

	if (prefersReducedMotion()) {
		snapOpen();
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

	const grow = () => {
		if (cancelled) return;
		let toHeight = measureFormNaturalHeight(form);
		if (typeof maxHeightPx === 'number' && Number.isFinite(maxHeightPx)) {
			toHeight = Math.min(toHeight, maxHeightPx);
		}
		// Keep the form clipped to the lock while the lock height animates.
		prepareFormForEnter(form);
		form.style.opacity = '1';
		form.style.pointerEvents = '';
		const from = lock.getBoundingClientRect().height;
		if (Math.abs(toHeight - from) < 1) {
			finishEnterLayout(lock, display, form);
			onComplete?.();
			clearFormEnterOpacity(form);
			resolveFinished();
			return;
		}
		step = run(
			lock,
			{ height: [`${from}px`, `${toHeight}px`] },
			stepOpts,
			() => {
				if (cancelled) return;
				finishEnterLayout(lock, display, form);
				onComplete?.();
				clearFormEnterOpacity(form);
				resolveFinished();
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
