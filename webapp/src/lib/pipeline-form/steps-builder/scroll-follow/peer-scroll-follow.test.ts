// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import type { AnimatableScroll } from './animatable-scroll.js';

import { createFakeClock, type FakeClock } from '../test-support/fake-clock.js';
import { PeerScrollFollow } from './peer-scroll-follow.svelte.js';

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
			const out: ElementStub[] = [];
			const walk = (node: ElementStub) => {
				if (wantCard && node.getAttribute('data-card-section') != null) out.push(node);
				if (wantYaml && node.getAttribute('data-yaml-section') != null) out.push(node);
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

function createCardsScroller(cardCenters: number[], followUpCenters: number[] = []) {
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

	for (const [index, center] of followUpCenters.entries()) {
		const card = createElementStub({
			'data-card-section': 'follow-ups',
			'data-card-index': String(index)
		});
		card.getBoundingClientRect = () =>
			makeRect({ top: center - 40, bottom: center + 40, height: 80 });
		scroller.appendChild(card);
	}

	return scroller;
}

function createYamlScroller(blockCenters: number[], followUpCenters: number[] = []) {
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

	for (const [index, center] of followUpCenters.entries()) {
		const block = createElementStub({
			'data-yaml-section': 'follow-ups',
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

		// Topmost intersecting (index 0 at center 50) — not vertical center (index 1).
		expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });
		follow.dispose();
	});

	it('cards continuous follow tracks topmost intersecting card under long geometry', async () => {
		const clock = createFakeClock();
		const follow = createFollow(clock);
		follow.setEnabled(true);

		const cards = createElementStub();
		stubScrollerGeometry(cards, { top: 0, bottom: 400, height: 400 });
		cards.scrollTop = 100;
		// Short top card + tall middle (owns center) + third below fold.
		const geometries = [
			{ index: 0, top: 10, bottom: 90 },
			{ index: 1, top: 90, bottom: 500 },
			{ index: 2, top: 500, bottom: 580 }
		];
		for (const g of geometries) {
			const card = createElementStub({
				'data-card-section': 'steps',
				'data-card-index': String(g.index)
			});
			card.getBoundingClientRect = () =>
				makeRect({ top: g.top, bottom: g.bottom, height: g.bottom - g.top });
			cards.appendChild(card);
		}

		const yaml = createYamlScroller([800, 200, 360]);
		follow.cardsAttach(cards as unknown as HTMLElement);
		follow.yamlAttach(yaml as unknown as HTMLElement);

		cards.dispatchEvent(new Event('pointerdown'));
		cards.dispatchEvent(new Event('scroll'));

		// Topmost card (0), not the tall center-owning card (1).
		expect(follow.activeUnit).toEqual({ section: 'steps', index: 0 });
		clock.flushRaf();
		await vi.waitFor(() => {
			expect(yaml.scrollTo).toHaveBeenCalled();
		});
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

	it('skips yaml regen discrete follow after notePairedReorder; follows without note', async () => {
		const clock = createFakeClock();
		const follow = createFollow(clock);
		follow.setEnabled(true);

		const cards = createCardsScroller([100, 300]);
		const yaml = createYamlScroller([100, 500]);
		follow.cardsAttach(cards as unknown as HTMLElement);
		follow.yamlAttach(yaml as unknown as HTMLElement);

		follow.followUnit({ section: 'steps', index: 1 }, 'cards');
		await vi.waitFor(() => {
			expect(yaml.scrollTo).toHaveBeenCalled();
		});
		vi.mocked(yaml.scrollTo).mockClear();

		follow.notePairedReorder();
		const cancelSuppressed = follow.onYamlTextChanged();
		clock.flushTimeouts(130);
		expect(yaml.scrollTo).not.toHaveBeenCalled();
		cancelSuppressed();

		clock.flushTimeouts(100);
		const cancelFollow = follow.onYamlTextChanged();
		clock.flushTimeouts(130);
		await vi.waitFor(() => {
			expect(yaml.scrollTo).toHaveBeenCalled();
		});
		cancelFollow();
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

	describe('onEditFocus (In-card edit hard start)', () => {
		function setup(enabled: boolean) {
			const clock = createFakeClock();
			const follow = createFollow(clock);
			follow.setEnabled(enabled);
			const cards = createCardsScroller([100, 300], [500]);
			const yaml = createYamlScroller([100, 300], [500]);
			follow.cardsAttach(cards as unknown as HTMLElement);
			follow.yamlAttach(yaml as unknown as HTMLElement);
			return { clock, follow, cards, yaml };
		}

		it('start-aligns the card and its yaml peer, keeping scroll follow on', () => {
			const { follow, cards, yaml } = setup(true);

			void follow.onEditFocus({ section: 'steps', index: 1 });

			expect(follow.activeUnit).toEqual({ section: 'steps', index: 1 });
			expect(follow.enabled).toBe(true);
			// card/block top 260 − scroller top 0 − 16px padding
			expect(cards.scrollTo).toHaveBeenCalledWith({ top: 244, behavior: 'auto' });
			expect(yaml.scrollTo).toHaveBeenCalledWith({ top: 244, behavior: 'auto' });
			expect(cards.focus).not.toHaveBeenCalled();
			follow.dispose();
		});

		it('supports follow-up units', () => {
			const { follow, cards, yaml } = setup(true);

			void follow.onEditFocus({ section: 'follow-ups', index: 0 });

			expect(follow.activeUnit).toEqual({ section: 'follow-ups', index: 0 });
			expect(cards.scrollTo).toHaveBeenCalledWith({ top: 444, behavior: 'auto' });
			expect(yaml.scrollTo).toHaveBeenCalledWith({ top: 444, behavior: 'auto' });
			follow.dispose();
		});

		it('hard-starts again even when the unit is already active', () => {
			const { follow, cards, yaml } = setup(true);
			follow.followUnit({ section: 'steps', index: 1 }, 'cards');
			vi.mocked(cards.scrollTo).mockClear();
			vi.mocked(yaml.scrollTo).mockClear();

			void follow.onEditFocus({ section: 'steps', index: 1 });

			expect(cards.scrollTo).toHaveBeenCalledTimes(1);
			expect(yaml.scrollTo).toHaveBeenCalledTimes(1);
			follow.dispose();
		});

		it('still start-aligns yaml when scroll follow is off, without enabling it', () => {
			const { follow, cards, yaml } = setup(false);

			void follow.onEditFocus({ section: 'steps', index: 1 });

			expect(follow.enabled).toBe(false);
			expect(cards.scrollTo).toHaveBeenCalledWith({ top: 244, behavior: 'auto' });
			expect(yaml.scrollTo).toHaveBeenCalledWith({ top: 244, behavior: 'auto' });
			follow.dispose();
		});

		it('resolves after the cards scroll settles so expand can wait', async () => {
			const { clock, follow, cards } = setup(true);

			let settled = false;
			const done = follow.onEditFocus({ section: 'steps', index: 1 }).then(() => {
				settled = true;
			});

			expect(settled).toBe(false);
			expect(cards.scrollTo).toHaveBeenCalled();
			clock.flushTimeouts(700);
			await done;
			expect(settled).toBe(true);
			follow.dispose();
		});

		it('resolves immediately when the card is already start-aligned', async () => {
			const { follow, cards } = setup(true);
			// Place card 1 already at the start-align target (top ≈ 16 with 16px padding).
			cards.scrollTop = 244;
			const card = cards.querySelector('[data-card-section="steps"][data-card-index="1"]');
			expect(card).not.toBeNull();
			card!.getBoundingClientRect = () => makeRect({ top: 16, bottom: 96, height: 80 });

			vi.mocked(cards.scrollTo).mockClear();
			await follow.onEditFocus({ section: 'steps', index: 1 });
			expect(cards.scrollTo).not.toHaveBeenCalled();
			follow.dispose();
		});

		it('does not let YAML start-align peer-center cards before expand/park', async () => {
			/**
			 * Repro: beginDriven(cards) flips scroll leader to yaml; YAML hard-start
			 * scroll then peer-centers cards (clears cards driven early). Expand parks
			 * with the card mid-viewport → grown card top is cropped.
			 */
			const { clock, follow, cards, yaml } = setup(true);

			const focusDone = follow.onEditFocus({ section: 'steps', index: 1 });
			expect(cards.scrollTo).toHaveBeenCalledWith({ top: 244, behavior: 'auto' });
			vi.mocked(cards.scrollTo).mockClear();

			// Mid hard-start: YAML viewport center claims a different unit (index 0).
			const yaml0 = yaml.querySelector('[data-yaml-section="steps"][data-yaml-index="0"]');
			const yaml1 = yaml.querySelector('[data-yaml-section="steps"][data-yaml-index="1"]');
			expect(yaml0).not.toBeNull();
			expect(yaml1).not.toBeNull();
			yaml0!.getBoundingClientRect = () => makeRect({ top: 120, bottom: 200, height: 80 });
			yaml1!.getBoundingClientRect = () => makeRect({ top: 320, bottom: 400, height: 80 });
			yaml.dispatchEvent(new Event('scroll'));
			clock.flushRaf();
			await Promise.resolve();

			expect(cards.scrollTo).not.toHaveBeenCalled();

			clock.flushTimeouts(700);
			await focusDone;
			follow.dispose();
		});

		it('snaps cards to start-align before resolve when settle left the card clipped', async () => {
			const { clock, follow, cards } = setup(true);
			const card = cards.querySelector('[data-card-section="steps"][data-card-index="1"]');
			expect(card).not.toBeNull();

			const focusDone = follow.onEditFocus({ section: 'steps', index: 1 });
			expect(cards.scrollTo).toHaveBeenCalledWith({ top: 244, behavior: 'auto' });

			// Simulate Animatable/settle finishing while the card is still clipped above.
			cards.scrollTop = 100;
			card!.getBoundingClientRect = () => makeRect({ top: -40, bottom: 40, height: 80 });
			vi.mocked(cards.scrollTo).mockClear();

			clock.flushTimeouts(700);
			await focusDone;

			// Instant snap: scrollTop + (elTop - portTop) - START_PADDING = 100 + (-40) - 16 = 44
			expect(cards.scrollTo).toHaveBeenCalledWith({ top: 44, behavior: 'auto' });
			follow.dispose();
		});

		it('does not suppress later peer sync after the driven scrolls settle', () => {
			const { clock, follow, cards } = setup(true);
			void follow.onEditFocus({ section: 'steps', index: 0 });
			clock.flushTimeouts(700);

			cards.scrollTop = 100;
			cards.dispatchEvent(new Event('pointerdown'));
			cards.dispatchEvent(new Event('scroll'));

			expect(follow.enabled).toBe(true);
			expect(follow.activeUnit).not.toBeNull();
			follow.dispose();
		});
	});

	describe('isCardsParked (YAML→cards mute)', () => {
		it('no-ops YAML→cards follow while parked; preference stays on; resumes when unparked', async () => {
			const clock = createFakeClock();
			let parked = true;
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
			const follow = createFollow(clock, {
				ensureMounted,
				isCardsParked: () => parked
			});
			follow.setEnabled(true);

			const yaml = createYamlScroller([100, 300]);
			follow.cardsAttach(cards as unknown as HTMLElement);
			follow.yamlAttach(yaml as unknown as HTMLElement);

			follow.followUnit({ section: 'steps', index: 1 }, 'yaml');
			expect(ensureMounted).not.toHaveBeenCalled();
			expect(cards.scrollTo).not.toHaveBeenCalled();
			expect(follow.enabled).toBe(true);

			parked = false;
			follow.followUnit({ section: 'steps', index: 1 }, 'yaml');
			await vi.waitFor(() => {
				expect(ensureMounted).toHaveBeenCalledWith({ section: 'steps', index: 1 });
			});
			expect(cards.scrollTo).toHaveBeenCalled();
			follow.dispose();
		});

		it('yaml scroll does not move cards while parked', () => {
			const clock = createFakeClock();
			const follow = createFollow(clock, { isCardsParked: () => true });
			follow.setEnabled(true);

			const cards = createCardsScroller([100, 300]);
			const yaml = createYamlScroller([50, 200, 450]);
			follow.cardsAttach(cards as unknown as HTMLElement);
			follow.yamlAttach(yaml as unknown as HTMLElement);

			yaml.dispatchEvent(new Event('pointerdown'));
			yaml.dispatchEvent(new Event('scroll'));
			clock.flushRaf();

			expect(cards.scrollTo).not.toHaveBeenCalled();
			expect(follow.enabled).toBe(true);
			follow.dispose();
		});

		it('no-ops cards→YAML follow while parked', async () => {
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
			const follow = createFollow(clock, {
				ensureMountedYaml,
				isCardsParked: () => true
			});
			follow.setEnabled(true);
			follow.cardsAttach(cards as unknown as HTMLElement);
			follow.yamlAttach(yaml as unknown as HTMLElement);

			follow.followUnit({ section: 'steps', index: 0 }, 'cards');
			await Promise.resolve();
			expect(ensureMountedYaml).not.toHaveBeenCalled();
			expect(yaml.scrollTo).not.toHaveBeenCalled();
			expect(follow.enabled).toBe(true);
			follow.dispose();
		});
	});
});
