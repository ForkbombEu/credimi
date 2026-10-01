// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createAnimatable, type AnimatableObject } from 'animejs';

/**
 * Duration / ease defaults for composer peer-scroll.
 *
 * Continuous follow (~280ms): short so rapid active-unit retargets replace the
 * in-flight tween instead of stacking native `behavior: 'smooth'` animations.
 * Discrete reveal / edit / click (~350ms): slightly longer settle for intentional jumps.
 * Ease `out(3)`: snappy deceleration without overshoot.
 */
export const FOLLOW_SCROLL_DURATION_MS = 280;
export const DISCRETE_SCROLL_DURATION_MS = 350;
export const SCROLL_EASE = 'out(3)';

/** Extra ms past the tween duration before driven-scroll idle fallback clears. */
export const ANIMATABLE_DRIVEN_IDLE_PAD_MS = 80;

export type AnimatableScroll = {
	/** Retarget scrollTop; optional per-call duration override (ms). */
	scrollTo(top: number, durationMs?: number): void;
	/** Current scrollTop from the Animatable getter. */
	getScrollTop(): number;
	/** Tear down the persistent Animatable (attach cleanup / dispose). */
	dispose(): void;
	readonly animatable: AnimatableObject;
};

export type CreateAnimatableScrollOptions = {
	/** Default duration for `scrollTo` when no override is passed. */
	durationMs?: number;
	ease?: string;
	/** Fires when the Animatable callbacks animation completes (may lag retargets). */
	onComplete?: () => void;
};

/**
 * One persistent Anime.js Animatable per scrollport.
 * Call `scrollTo` to ease (and retarget) `scrollTop` — do not use native smooth scrollTo.
 */
export function createAnimatableScroll(
	scroller: HTMLElement,
	options?: CreateAnimatableScrollOptions
): AnimatableScroll {
	const durationMs = options?.durationMs ?? FOLLOW_SCROLL_DURATION_MS;
	const ease = options?.ease ?? SCROLL_EASE;
	const animatable = createAnimatable(scroller, {
		scrollTop: durationMs,
		ease,
		onComplete: options?.onComplete
	});

	return {
		animatable,
		scrollTo(top, overrideDuration) {
			if (overrideDuration != null) {
				animatable.scrollTop(top, overrideDuration);
			} else {
				animatable.scrollTop(top);
			}
		},
		getScrollTop() {
			const value = animatable.scrollTop();
			return typeof value === 'number' ? value : Number(value);
		},
		dispose() {
			animatable.revert();
		}
	};
}

/** Idle timeout for watchDrivenScroll when Animatable drives the scroller. */
export function animatableDrivenIdleMs(durationMs: number): number {
	return Math.max(0, durationMs) + ANIMATABLE_DRIVEN_IDLE_PAD_MS;
}
