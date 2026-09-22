// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import type { YamlCardRange } from './yaml-ranges.js';

import {
	createPeerScrollFollow,
	type PeerScrollFollowClock
} from './peer-scroll-follow.js';

type FakeClock = PeerScrollFollowClock & {
	flushRaf(): void;
	flushTimeouts(ms?: number): void;
};

function createFakeClock(): FakeClock {
	let nextRaf = 1;
	const rafQueue = new Map<number, FrameRequestCallback>();
	let nextTimer = 1;
	const timers = new Map<number, { due: number; handler: () => void }>();
	let now = 0;

	return {
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
		flushTimeouts(ms = 0) {
			now += ms;
			const due = [...timers.entries()].filter(([, t]) => t.due <= now);
			for (const [id, t] of due) {
				timers.delete(id);
				t.handler();
			}
		}
	};
}

type Rect = { top: number; bottom: number; height: number };

function makeRect(rect: Rect): DOMRect {
	return {
		top: rect.top,
		bottom: rect.bottom,
		height: rect.height,
		left: 0,
		right: 100,
		width: 100,
		x: 0,
		y: rect.top,
		toJSON() {
			return this;
		}
	} as DOMRect;
}

/** Minimal Element stub for node vitest (no happy-dom/jsdom). */
function createElementStub(attrs: Record<string, string> = {}) {
	const listeners = new Map<string, Set<EventListener>>();
	const children: ReturnType<typeof createElementStub>[] = [];
	const store = { ...attrs };

	const el = {
		getAttribute(name: string) {
			return store[name] ?? null;
		},
		setAttribute(name: string, value: string) {
			store[name] = value;
		},
		appendChild(child: ReturnType<typeof createElementStub>) {
			children.push(child);
			return child;
		},
		querySelectorAll(selector: string) {
			const wantSection = selector.includes('[data-card-section]');
			const wantLine = selector.includes('[data-line]');
			const out: ReturnType<typeof createElementStub>[] = [];
			const walk = (node: ReturnType<typeof createElementStub>) => {
				if (wantSection && node.getAttribute('data-card-section') != null) out.push(node);
				if (wantLine && node.getAttribute('data-line') != null) out.push(node);
				for (const c of node._children) walk(c);
			};
			walk(el);
			return out;
		},
		querySelector(selector: string) {
			const match = selector.match(
				/\[data-card-section="([^"]+)"\]\[data-card-index="([^"]+)"\]/
			);
			if (match) {
				return (
					el
						.querySelectorAll('[data-card-section]')
						.find(
							(c) =>
								c.getAttribute('data-card-section') === match[1] &&
								c.getAttribute('data-card-index') === match[2]
						) ?? null
				);
			}
			const lineMatch = selector.match(/\[data-line="([^"]+)"\]/);
			if (lineMatch) {
				return (
					el
						.querySelectorAll('[data-line]')
						.find((c) => c.getAttribute('data-line') === lineMatch[1]) ?? null
				);
			}
			return null;
		},
		addEventListener(type: string, listener: EventListener) {
			if (!listeners.has(type)) listeners.set(type, new Set());
			listeners.get(type)!.add(listener);
		},
		removeEventListener(type: string, listener: EventListener) {
			listeners.get(type)?.delete(listener);
		},
		dispatchEvent(event: Event) {
			for (const listener of listeners.get(event.type) ?? []) {
				listener.call(el, event);
			}
			return true;
		},
		getBoundingClientRect: () => makeRect({ top: 0, bottom: 0, height: 0 }),
		focus: vi.fn(),
		scrollTo: vi.fn(),
		clientHeight: 400,
		scrollHeight: 2000,
		scrollTop: 0,
		get _children() {
			return children;
		}
	};

	return el;
}

function createCardsScroller(cardCenters: number[]) {
	const scroller = createElementStub();
	stubScrollerGeometry(scroller, { top: 0, bottom: 400, height: 400 });

	for (const [index, center] of cardCenters.entries()) {
		const card = createElementStub({
			'data-card-section': 'steps',
			'data-card-index': String(index)
		});
		card.getBoundingClientRect = () =>
			makeRect({ top: center - 40, bottom: center + 40, height: 80 });
		scroller.appendChild(card);
	}

	return scroller;
}

function createYamlScroller(lineTops: number[]) {
	const scroller = createElementStub();
	stubScrollerGeometry(scroller, { top: 0, bottom: 400, height: 400 });

	for (const [i, top] of lineTops.entries()) {
		const line = createElementStub({ 'data-line': String(i) });
		line.getBoundingClientRect = () => makeRect({ top, bottom: top + 20, height: 20 });
		scroller.appendChild(line);
	}

	return scroller;
}

function stubScrollerGeometry(
	el: ReturnType<typeof createElementStub>,
	rect: Rect
) {
	el.clientHeight = rect.height;
	el.scrollHeight = 2000;
	el.scrollTop = 0;
	el.getBoundingClientRect = () => makeRect(rect);
	el.scrollTo = vi.fn();
}

const ranges: YamlCardRange[] = [
	{ section: 'steps', index: 0, startLine: 0, endLine: 2 },
	{ section: 'steps', index: 1, startLine: 3, endLine: 5 }
];

describe('createPeerScrollFollow', () => {
	it('does not steal leadership or change activeUnit without user intent', () => {
		const clock = createFakeClock();
		const follow = createPeerScrollFollow({ clock });
		follow.setEnabled(true);

		const scroller = createCardsScroller([100, 300, 500]);
		const unbind = follow.bindCards(scroller as unknown as HTMLElement);

		expect(follow.activeUnit).toBeNull();
		scroller.dispatchEvent(new Event('scroll'));
		expect(follow.activeUnit).toBeNull();

		unbind();
		follow.dispose();
	});

	it('updates activeUnit from cards scroll after cards intent', () => {
		const clock = createFakeClock();
		const follow = createPeerScrollFollow({ clock });
		follow.setEnabled(true);

		// Viewport center 200 → card centers 50 / 200 / 450 → card 1 wins.
		// scrollTop must be > 2 so edge preference does not force the first card.
		const scroller = createCardsScroller([50, 200, 450]);
		scroller.scrollTop = 100;
		const yaml = createYamlScroller([0, 20, 40, 60, 80, 100]);
		follow.bindCards(scroller as unknown as HTMLElement);
		follow.bindYaml(yaml as unknown as HTMLElement, () => ranges);

		scroller.dispatchEvent(new Event('pointerdown'));
		scroller.dispatchEvent(new Event('scroll'));

		expect(follow.activeUnit).toEqual({ section: 'steps', index: 1 });
		follow.dispose();
	});

	it('no-ops cards→yaml follow while yaml is scroll leader', () => {
		const clock = createFakeClock();
		const follow = createPeerScrollFollow({ clock });
		follow.setEnabled(true);

		const cards = createCardsScroller([100, 300]);
		// Line 0 starts far below the viewport so start-align must scroll.
		const yaml = createYamlScroller([800, 820, 840, 860, 880, 900]);
		follow.bindCards(cards as unknown as HTMLElement);
		follow.bindYaml(yaml as unknown as HTMLElement, () => ranges);

		follow.followUnit({ section: 'steps', index: 0 }, 'cards');
		expect(yaml.scrollTo).toHaveBeenCalled();
		vi.mocked(yaml.scrollTo).mockClear();

		yaml.dispatchEvent(new Event('wheel'));
		const cancel = follow.onYamlTextChanged();
		clock.flushTimeouts(130);

		expect(yaml.scrollTo).not.toHaveBeenCalled();
		cancel();
		follow.dispose();
	});

	it('setEnabled(false) clears activeUnit and notifies subscribers', () => {
		const clock = createFakeClock();
		const follow = createPeerScrollFollow({ clock });
		follow.setEnabled(true);
		follow.followUnit({ section: 'steps', index: 0 }, 'cards');
		expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });

		const listener = vi.fn();
		follow.subscribe(listener);
		follow.setEnabled(false);

		expect(follow.enabled).toBe(false);
		expect(follow.activeUnit).toBeNull();
		expect(listener).toHaveBeenCalled();
		follow.dispose();
	});
});
