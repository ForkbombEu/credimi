// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * animejs v4 tweens for the in-card form shell (Pipeline Composer).
 *
 * One running animation per element: starting a new one cancels the previous
 * one in place (no revert), so retargeting mid-flight continues from the
 * current height/opacity instead of jumping.
 * `cancel()` and `cancelMotion()` are idempotent and never throw.
 * Honors `prefers-reduced-motion` by applying tween end values synchronously.
 */

import { animate, type JSAnimation } from 'animejs';

export const IN_CARD_MOTION_MS = 150;
export const IN_CARD_MOTION_EASE = 'outQuad';

export type MotionHandle = {
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
	lock: HTMLElement;
	display: HTMLElement;
	form: HTMLElement;
	toHeight: number;
};

export type InCardExitOptions = MotionOptions & {
	lock: HTMLElement;
	display: HTMLElement;
	form: HTMLElement;
	toHeight: number;
};

const running = new WeakMap<Element, MotionHandle>();

function prefersReducedMotion(): boolean {
	return (
		typeof window !== 'undefined' &&
		typeof window.matchMedia === 'function' &&
		window.matchMedia('(prefers-reduced-motion: reduce)').matches
	);
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

function applyEnterTweenEnd(
	lock: HTMLElement,
	display: HTMLElement,
	form: HTMLElement,
	toHeight: number
) {
	display.style.opacity = '0';
	form.style.opacity = '1';
	lock.style.height = `${toHeight}px`;
}

function applyExitTweenEnd(
	lock: HTMLElement,
	display: HTMLElement,
	form: HTMLElement,
	toHeight: number
) {
	lock.style.height = `${toHeight}px`;
	form.style.opacity = '0';
	display.style.opacity = '1';
}

function reducedMotionHandle(onComplete?: () => void | Promise<void>): MotionHandle {
	let cancelled = false;
	const finished = (async () => {
		if (cancelled) return;
		await Promise.resolve(onComplete?.());
	})();
	return {
		cancel: () => {
			cancelled = true;
		},
		finished
	};
}

export function playInCardEnter(options: InCardEnterOptions): MotionHandle {
	const { lock, display, form, toHeight, onComplete, durationMs } = options;

	cancelMotion(lock);
	cancelMotion(display);
	cancelMotion(form);

	if (prefersReducedMotion()) {
		applyEnterTweenEnd(lock, display, form, toHeight);
		return reducedMotionHandle(onComplete);
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

	const completeEnter = () => {
		applyEnterTweenEnd(lock, display, form, toHeight);
		void Promise.resolve(onComplete?.()).then(() => {
			if (!cancelled) resolveFinished();
		});
	};

	const stepOpts = { durationMs };

	const grow = () => {
		if (cancelled) return;
		form.style.pointerEvents = '';
		const from = lock.getBoundingClientRect().height;
		if (Math.abs(toHeight - from) < 1) {
			completeEnter();
			return;
		}
		step = run(lock, { height: [`${from}px`, `${toHeight}px`] }, stepOpts, () => {
			if (cancelled) return;
			completeEnter();
		});
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

	step = run(display, { opacity: 0 }, stepOpts, () => {
		if (cancelled) return;
		display.style.pointerEvents = 'none';
		fadeFormIn();
	});

	return handle;
}

export function playInCardExit(options: InCardExitOptions): MotionHandle {
	const { lock, display, form, toHeight, onComplete, durationMs } = options;

	cancelMotion(lock);
	cancelMotion(display);
	cancelMotion(form);

	if (prefersReducedMotion()) {
		applyExitTweenEnd(lock, display, form, toHeight);
		return reducedMotionHandle(onComplete);
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
			applyExitTweenEnd(lock, display, form, toHeight);
			void Promise.resolve(onComplete?.()).then(() => {
				if (!cancelled) resolveFinished();
			});
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
