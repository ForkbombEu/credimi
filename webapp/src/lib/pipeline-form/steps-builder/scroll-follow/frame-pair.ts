// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

const ALIGN_EPSILON_PX = 1;
/**
 * Tolerate intentional row chrome that rides on the measured unit (cards `pb-3`
 * gap padding). Must stay below the YAML clipped-lower smoke delta (20px) so
 * real content past the fold still frames.
 */
const VISIBLE_EDGE_SLOP_PX = 12;

export type FramePairRect = { top: number; bottom: number };

function intersects(el: FramePairRect, port: FramePairRect): boolean {
	return el.bottom > port.top && el.top < port.bottom;
}

function fullyVisible(el: FramePairRect, port: FramePairRect): boolean {
	return (
		el.top >= port.top - VISIBLE_EDGE_SLOP_PX && el.bottom <= port.bottom + VISIBLE_EDGE_SLOP_PX
	);
}

/**
 * Next scrollTop that frames a swapped pair, or null if no move is needed.
 *
 * - Both already fully visible (within row-gap edge slop) → null.
 * - Pair taller than the viewport → null if both already intersect; else center the gap
 *   (top/bottom may clip).
 * - Pair fits → minimal Δscroll so both are fully visible (partial intersect is not enough).
 *
 * `upper` / `lower` are document order (upper above lower), in viewport coordinates
 * matching `getBoundingClientRect()` relative to `scrollerRect`.
 */
export function framePairScrollTop(
	scrollTop: number,
	clientHeight: number,
	scrollHeight: number,
	upper: FramePairRect,
	lower: FramePairRect,
	scrollerRect: FramePairRect
): number | null {
	if (fullyVisible(upper, scrollerRect) && fullyVisible(lower, scrollerRect)) {
		return null;
	}

	const maxScroll = Math.max(0, scrollHeight - clientHeight);
	const upperTop = scrollTop + (upper.top - scrollerRect.top);
	const upperBottom = scrollTop + (upper.bottom - scrollerRect.top);
	const lowerTop = scrollTop + (lower.top - scrollerRect.top);
	const lowerBottom = scrollTop + (lower.bottom - scrollerRect.top);
	const pairHeight = lowerBottom - upperTop;

	let target: number;
	if (pairHeight >= clientHeight) {
		if (intersects(upper, scrollerRect) && intersects(lower, scrollerRect)) {
			return null;
		}
		const gapCenter = (upperBottom + lowerTop) / 2;
		target = gapCenter - clientHeight / 2;
	} else {
		const minScroll = lowerBottom - clientHeight;
		const maxScrollForPair = upperTop;
		if (scrollTop < minScroll) target = minScroll;
		else if (scrollTop > maxScrollForPair) target = maxScrollForPair;
		else return null;
	}

	target = Math.min(maxScroll, Math.max(0, target));
	if (Math.abs(target - scrollTop) < ALIGN_EPSILON_PX) return null;
	return target;
}
