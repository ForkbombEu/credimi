// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

const SCROLLABLE = new Set(['auto', 'scroll', 'overlay']);

function axisOverflow(el: HTMLElement, axis: 'x' | 'y'): string {
	const computed = getComputedStyle(el);
	const fromComputed = axis === 'y' ? computed.overflowY : computed.overflowX;
	if (fromComputed && fromComputed !== 'visible') return fromComputed;
	return axis === 'y' ? el.style.overflowY : el.style.overflowX;
}

function elementCanScrollDelta(el: Element, deltaX: number, deltaY: number): boolean {
	if (!(el instanceof HTMLElement)) return false;
	const vertical = Math.abs(deltaY) >= Math.abs(deltaX);
	if (vertical) {
		if (!SCROLLABLE.has(axisOverflow(el, 'y'))) return false;
		if (deltaY < 0) return el.scrollTop > 0;
		if (deltaY > 0) return el.scrollTop + el.clientHeight < el.scrollHeight - 1;
		return false;
	}
	if (!SCROLLABLE.has(axisOverflow(el, 'x'))) return false;
	if (deltaX < 0) return el.scrollLeft > 0;
	if (deltaX > 0) return el.scrollLeft + el.clientWidth < el.scrollWidth - 1;
	return false;
}

function pathToRoot(event: WheelEvent, root: EventTarget | null): EventTarget[] {
	if (typeof event.composedPath === 'function') {
		const path = event.composedPath();
		if (path.length > 0) return path;
	}
	const chain: EventTarget[] = [];
	let node: Node | null = event.target instanceof Node ? event.target : null;
	while (node) {
		chain.push(node);
		if (node === root) break;
		node = node.parentNode;
	}
	return chain;
}

/**
 * True when a descendant of `root` (not `root` itself) can consume this wheel.
 * Park wheel lock must not `preventDefault` in that case — In-card form body
 * scroll (trackpad) vs cards pane still-ness.
 */
export function nestedScrollerConsumesWheel(event: WheelEvent, root: EventTarget | null): boolean {
	for (const node of pathToRoot(event, root)) {
		if (node === root) break;
		if (elementCanScrollDelta(node, event.deltaX, event.deltaY)) return true;
	}
	return false;
}
