// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import type { AnimatableScroll } from './animatable-scroll.js';
import { PeerScrollFollow, type PeerScrollFollowClock } from './peer-scroll-follow.svelte.js';

type FakeClock = PeerScrollFollowClock & {
	flushRaf(): void;
	flushTimeouts(ms?: number): void;
};

/** Test double: records Animatable retargets and mirrors onto scroller.scrollTo for assertions. */
function fakeCreateAnimatableScroll(scroller: HTMLElement): AnimatableScroll {
	const scrollTo = vi.fn((top: number, _durationMs?: number) => {
		(scroller as { scrollTop: number }).scrollTop = top;
		const scrollToNative = (scroller as unknown as { scrollTo: ReturnType<typeof vi.fn> })
			.scrollTo;
		scrollToNative?.({ top, behavior: 'auto' });
	});
	return {
		scrollTo,
		getScrollTop: () => (scroller as { scrollTop: number }).scrollTop,
		dispose: vi.fn(),
		animatable: {} as AnimatableScroll['animatable']
	};
}

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
			const wantCard = selector.includes('[data-card-section]');
			const wantYaml = selector.includes('[data-yaml-section]');
			const wantLine = selector.includes('[data-line]');
			const out: ElementStub[] = [];
			const walk = (node: ElementStub) => {
				if (wantCard && node.getAttribute('data-card-section') != null) out.push(node);
				if (wantYaml && node.getAttribute('data-yaml-section') != null) out.push(node);
				if (wantLine && node.getAttribute('data-line') != null) out.push(node);
				for (const c of node._children) walk(c);
			};
			walk(el);
			return out;
		},
		querySelector(selector: string) {
			const cardMatch = selector.match(
				/\[data-card-section="([^"]+)"\]\[data-card-index="([^"]+)"\]/
			);
			if (cardMatch) {
				return (
					el
						.querySelectorAll('[data-card-section]')
						.find(
							(c: ElementStub) =>
								c.getAttribute('data-card-section') === cardMatch[1] &&
								c.getAttribute('data-card-index') === cardMatch[2]
						) ?? null
				);
			}
			const yamlMatch = selector.match(
				/\[data-yaml-section="([^"]+)"\]\[data-yaml-index="([^"]+)"\]/
			);
			if (yamlMatch) {
				return (
					el
						.querySelectorAll('[data-yaml-section]')
						.find(
							(c: ElementStub) =>
								c.getAttribute('data-yaml-section') === yamlMatch[1] &&
								c.getAttribute('data-yaml-index') === yamlMatch[2]
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

function createYamlScroller(blockCenters: number[]) {
	const scroller = createElementStub();
	stubScrollerGeometry(scroller, { top: 0, bottom: 400, height: 400 });

	for (const [index, center] of blockCenters.entries()) {
		const block = createElementStub({
			'data-yaml-section': 'steps',
			'data-yaml-index': String(index)
		});
		block.getBoundingClientRect = () =>
			makeRect({ top: center - 40, bottom: center + 40, height: 80 });
		scroller.appendChild(block);
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

function createFollow(
	clock: FakeClock,
	extra?: Omit<
		ConstructorParameters<typeof PeerScrollFollow>[0],
		'clock' | 'createAnimatableScroll'
	>
) {
	return new PeerScrollFollow({
		clock,
		createAnimatableScroll: fakeCreateAnimatableScroll,
		...extra
	});
}

describe('PeerScrollFollow', () => {
	it('does not steal leadership or change activeUnit without user intent', () => {
		const clock = createFakeClock();
		const follow = createFollow(clock);
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
		const follow = createFollow(clock);
		follow.setEnabled(true);

		const scroller = createCardsScroller([50, 200, 450]);
		scroller.scrollTop = 100;
		const yaml = createYamlScroller([100, 300]);
		follow.cardsAttach(scroller as unknown as HTMLElement);
		follow.yamlAttach(yaml as unknown as HTMLElement);

		scroller.dispatchEvent(new Event('pointerdown'));
		scroller.dispatchEvent(new Event('scroll'));

		expect(follow.activeUnit).toEqual({ section: 'steps', index: 1 });
		follow.dispose();
	});

	it('no-ops cards→yaml follow while yaml is scroll leader', async () => {
		const clock = createFakeClock();
		const follow = createFollow(clock);
		follow.setEnabled(true);

		const cards = createCardsScroller([100, 300]);
		const yaml = createYamlScroller([800, 900]);
		follow.cardsAttach(cards as unknown as HTMLElement);
		follow.yamlAttach(yaml as unknown as HTMLElement);

		follow.followUnit({ section: 'steps', index: 0 }, 'cards');
		await vi.waitFor(() => {
			expect(yaml.scrollTo).toHaveBeenCalled();
		});
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
		const follow = createFollow(clock);
		follow.setEnabled(true);
		follow.followUnit({ section: 'steps', index: 0 }, 'cards');
		expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });

		follow.setEnabled(false);

		expect(follow.enabled).toBe(false);
		expect(follow.activeUnit).toBeNull();
		follow.dispose();
	});

	it('queues onReveal until cardsAttach then flushes', async () => {
		const clock = createFakeClock();
		const follow = createFollow(clock);
		follow.setEnabled(true);

		follow.onReveal({ section: 'steps', index: 0 });
		expect(follow.activeUnit).toBeNull();

		const cards = createCardsScroller([100]);
		const yaml = createYamlScroller([100]);
		follow.yamlAttach(yaml as unknown as HTMLElement);
		follow.cardsAttach(cards as unknown as HTMLElement);

		await vi.waitFor(() => {
			expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });
		});
		follow.dispose();
	});

	it('calls ensureMounted when revealing an unmounted card', async () => {
		const clock = createFakeClock();
		const cards = createCardsScroller([]);
		const ensureMounted = vi.fn(async (unit: { section: string; index: number }) => {
			const card = createElementStub({
				'data-card-section': unit.section,
				'data-card-index': String(unit.index)
			});
			// Off-screen so center-align issues a scrollTo.
			card.getBoundingClientRect = () => makeRect({ top: 800, bottom: 880, height: 80 });
			cards.appendChild(card);
			return true;
		});
		const follow = createFollow(clock, { ensureMounted });
		follow.setEnabled(true);

		const yaml = createYamlScroller([100]);
		follow.cardsAttach(cards as unknown as HTMLElement);
		follow.yamlAttach(yaml as unknown as HTMLElement);

		follow.onReveal({ section: 'steps', index: 7 });

		await vi.waitFor(() => {
			expect(ensureMounted).toHaveBeenCalledWith({ section: 'steps', index: 7 });
			expect(follow.activeUnit).toEqual({ section: 'steps', index: 7 });
		});
		expect(cards.scrollTo).toHaveBeenCalled();
		follow.dispose();
	});

	it('yaml→cards follow calls ensureMounted for an unmounted peer card', async () => {
		const clock = createFakeClock();
		const cards = createCardsScroller([]);
		const ensureMounted = vi.fn(async (unit: { section: string; index: number }) => {
			const card = createElementStub({
				'data-card-section': unit.section,
				'data-card-index': String(unit.index)
			});
			card.getBoundingClientRect = () => makeRect({ top: 800, bottom: 880, height: 80 });
			cards.appendChild(card);
			return true;
		});
		const follow = createFollow(clock, { ensureMounted });
		follow.setEnabled(true);

		const yaml = createYamlScroller([100, 300]);
		follow.cardsAttach(cards as unknown as HTMLElement);
		follow.yamlAttach(yaml as unknown as HTMLElement);

		follow.followUnit({ section: 'steps', index: 1 }, 'yaml');

		await vi.waitFor(() => {
			expect(ensureMounted).toHaveBeenCalledWith({ section: 'steps', index: 1 });
		});
		expect(cards.scrollTo).toHaveBeenCalled();
		follow.dispose();
	});

	it('cards→yaml follow calls ensureMountedYaml for an unmounted yaml step', async () => {
		const clock = createFakeClock();
		const cards = createCardsScroller([100]);
		const yaml = createYamlScroller([]);
		const ensureMountedYaml = vi.fn(async (unit: { section: string; index: number }) => {
			const block = createElementStub({
				'data-yaml-section': unit.section,
				'data-yaml-index': String(unit.index)
			});
			block.getBoundingClientRect = () => makeRect({ top: 800, bottom: 880, height: 80 });
			yaml.appendChild(block);
			return true;
		});
		const follow = createFollow(clock, { ensureMountedYaml });
		follow.setEnabled(true);

		follow.cardsAttach(cards as unknown as HTMLElement);
		follow.yamlAttach(yaml as unknown as HTMLElement);

		follow.followUnit({ section: 'steps', index: 3 }, 'cards');

		await vi.waitFor(() => {
			expect(ensureMountedYaml).toHaveBeenCalledWith({ section: 'steps', index: 3 });
		});
		expect(yaml.scrollTo).toHaveBeenCalled();
		follow.dispose();
	});

	it('uses getCardLengths at list-end when resolving cards scroll', () => {
		const clock = createFakeClock();
		const follow = createFollow(clock, {
			getCardLengths: () => ({ steps: 1400, followUps: 2 })
		});
		follow.setEnabled(true);

		// Mounted window is mid-list indices only — without hints we'd get index 10.
		const scroller = createCardsScroller([100, 200, 300]);
		// Rewrite indices to look like a virtual mid-window
		const mounted = scroller.querySelectorAll('[data-card-section]');
		mounted[0]!.setAttribute('data-card-index', '10');
		mounted[1]!.setAttribute('data-card-index', '11');
		mounted[2]!.setAttribute('data-card-index', '12');
		scroller.scrollTop = 0;

		follow.cardsAttach(scroller as unknown as HTMLElement);
		scroller.dispatchEvent(new Event('pointerdown'));
		scroller.dispatchEvent(new Event('scroll'));

		expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });
		follow.dispose();
	});
});
