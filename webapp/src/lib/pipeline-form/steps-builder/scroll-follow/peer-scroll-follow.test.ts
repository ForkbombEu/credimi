// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import type { YamlCardRange } from './yaml-ranges.js';

import { PeerScrollFollow, type PeerScrollFollowClock } from './peer-scroll-follow.svelte.js';

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

type ElementStub = {
	getAttribute(name: string): string | null;
	setAttribute(name: string, value: string): void;
	appendChild(child: ElementStub): ElementStub;
	querySelectorAll(selector: string): ElementStub[];
	querySelector(selector: string): ElementStub | null;
	addEventListener(type: string, listener: EventListener): void;
	removeEventListener(type: string, listener: EventListener): void;
	dispatchEvent(event: Event): boolean;
	getBoundingClientRect(): DOMRect;
	focus: ReturnType<typeof vi.fn>;
	scrollTo: ReturnType<typeof vi.fn>;
	clientHeight: number;
	scrollHeight: number;
	scrollTop: number;
	readonly _children: ElementStub[];
};

/** Minimal Element stub for node vitest (no happy-dom/jsdom). */
function createElementStub(attrs: Record<string, string> = {}): ElementStub {
	const listeners = new Map<string, Set<EventListener>>();
	const children: ElementStub[] = [];
	const store = { ...attrs };

	const el: ElementStub = {
		getAttribute(name: string) {
			return store[name] ?? null;
		},
		setAttribute(name: string, value: string) {
			store[name] = value;
		},
		appendChild(child: ElementStub) {
			children.push(child);
			return child;
		},
		querySelectorAll(selector: string) {
			const wantSection = selector.includes('[data-card-section]');
			const wantLine = selector.includes('[data-line]');
			const out: ElementStub[] = [];
			const walk = (node: ElementStub) => {
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
							(c: ElementStub) =>
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
						.find((c: ElementStub) => c.getAttribute('data-line') === lineMatch[1]) ??
					null
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

function stubScrollerGeometry(el: ElementStub, rect: Rect) {
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

describe('PeerScrollFollow', () => {
	it('does not steal leadership or change activeUnit without user intent', () => {
		const clock = createFakeClock();
		const follow = new PeerScrollFollow({ clock });
		follow.setEnabled(true);

		const scroller = createCardsScroller([100, 300, 500]);
		const unbind = follow.cardsAttach(scroller as unknown as HTMLElement);

		expect(follow.activeUnit).toBeNull();
		scroller.dispatchEvent(new Event('scroll'));
		expect(follow.activeUnit).toBeNull();

		unbind?.();
		follow.dispose();
	});

	it('updates activeUnit from cards scroll after cards intent', () => {
		const clock = createFakeClock();
		const follow = new PeerScrollFollow({ clock });
		follow.setEnabled(true);

		const scroller = createCardsScroller([50, 200, 450]);
		scroller.scrollTop = 100;
		const yaml = createYamlScroller([0, 20, 40, 60, 80, 100]);
		follow.cardsAttach(scroller as unknown as HTMLElement);
		follow.yamlAttach(() => ranges)(yaml as unknown as HTMLElement);

		scroller.dispatchEvent(new Event('pointerdown'));
		scroller.dispatchEvent(new Event('scroll'));

		expect(follow.activeUnit).toEqual({ section: 'steps', index: 1 });
		follow.dispose();
	});

	it('no-ops cards→yaml follow while yaml is scroll leader', () => {
		const clock = createFakeClock();
		const follow = new PeerScrollFollow({ clock });
		follow.setEnabled(true);

		const cards = createCardsScroller([100, 300]);
		const yaml = createYamlScroller([800, 820, 840, 860, 880, 900]);
		follow.cardsAttach(cards as unknown as HTMLElement);
		follow.yamlAttach(() => ranges)(yaml as unknown as HTMLElement);

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

	it('setEnabled(false) clears activeUnit', () => {
		const clock = createFakeClock();
		const follow = new PeerScrollFollow({ clock });
		follow.setEnabled(true);
		follow.followUnit({ section: 'steps', index: 0 }, 'cards');
		expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });

		follow.setEnabled(false);

		expect(follow.enabled).toBe(false);
		expect(follow.activeUnit).toBeNull();
		follow.dispose();
	});

	it('queues onReveal until cardsAttach then flushes', () => {
		const clock = createFakeClock();
		const follow = new PeerScrollFollow({ clock });
		follow.setEnabled(true);

		follow.onReveal({ section: 'steps', index: 0 });
		expect(follow.activeUnit).toBeNull();

		const cards = createCardsScroller([100]);
		const yaml = createYamlScroller([0, 20, 40, 60, 80, 100]);
		follow.yamlAttach(() => ranges)(yaml as unknown as HTMLElement);
		follow.cardsAttach(cards as unknown as HTMLElement);

		expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });
		follow.dispose();
	});
});
