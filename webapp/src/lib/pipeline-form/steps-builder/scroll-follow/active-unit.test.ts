// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import { createFakeClock } from '../test-support/fake-clock.js';
import {
	CARD_PANE,
	YAML_PANE,
	computeAlignedScrollTop,
	computeNearestScrollTop,
	ensureMountedForStepsVirtualizer,
	resolveListEndUnit,
	resolveTopmostVisibleUnit,
	resolveViewportUnit,
	scrollUnitIntoView,
	watchDrivenScroll,
	type ActiveUnit
} from './active-unit.js';

describe('ensureMountedForStepsVirtualizer', () => {
	it('delegates steps indices to ensureStepVisible', async () => {
		const ensureStepVisible = vi.fn(async (index: number) => index === 7);
		const ensureMounted = ensureMountedForStepsVirtualizer(ensureStepVisible);

		await expect(ensureMounted({ section: 'steps', index: 7 })).resolves.toBe(true);
		expect(ensureStepVisible).toHaveBeenCalledWith(7);
	});

	it('treats follow-ups as already mounted without calling ensureStepVisible', async () => {
		const ensureStepVisible = vi.fn(async () => false);
		const ensureMounted = ensureMountedForStepsVirtualizer(ensureStepVisible);

		await expect(ensureMounted({ section: 'follow-ups', index: 2 })).resolves.toBe(true);
		expect(ensureStepVisible).not.toHaveBeenCalled();
	});
});

describe('computeNearestScrollTop', () => {
	const scroller = { top: 100, bottom: 500 }; // 400px tall

	it('returns null when fully visible', () => {
		expect(computeNearestScrollTop(50, 400, { top: 150, bottom: 200 }, scroller)).toBeNull();
	});

	it('scrolls up when above the viewport', () => {
		expect(computeNearestScrollTop(80, 400, { top: 60, bottom: 90 }, scroller)).toBe(40);
	});

	it('scrolls down when below the viewport', () => {
		expect(computeNearestScrollTop(0, 400, { top: 480, bottom: 560 }, scroller)).toBe(60);
	});

	it('mid-list double-down nearest reveal would eject the swap target above the fold', () => {
		// Documents why revealUnitNearest uses onlyIfOutside: after two center
		// downs with tall cards, nearest on the moved card scrolls enough that
		// the upward swap partner sits above the viewport top.
		const viewportH = 400;
		const scrollTop = 340;
		const port = { top: 100, bottom: 500 };
		const moved = { top: 100 + (880 - scrollTop), bottom: 100 + (1100 - scrollTop) };
		const next = computeNearestScrollTop(scrollTop, viewportH, moved, port);
		expect(next).not.toBeNull();
		const targetTopInPort = 100 + (660 - next!);
		expect(targetTopInPort).toBeLessThan(port.top);
	});
});

describe('computeAlignedScrollTop', () => {
	const scroller = { top: 100, bottom: 500 }; // 400px tall
	const scrollHeight = 2000;

	it('nearest returns null when fully visible', () => {
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 150, bottom: 200 },
				scroller,
				'nearest'
			)
		).toBeNull();
	});

	it('start aligns element top with padding even when already visible', () => {
		// el top at 200, scroller top 100 → delta 100; padding 16 → target scrollTop + 84
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 200, bottom: 220 },
				scroller,
				'start'
			)
		).toBe(134);
	});

	it('center aligns element mid to viewport mid', () => {
		// el mid 210, port mid 300 → delta -90 → target 50 - 90 = -40 → clamped 0
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 200, bottom: 220 },
				scroller,
				'center'
			)
		).toBe(0);
	});

	it('clamps to max scroll', () => {
		expect(
			computeAlignedScrollTop(0, 400, 500, { top: 800, bottom: 820 }, scroller, 'start')
		).toBe(100);
	});

	it('returns null when already within epsilon of start align', () => {
		// delta = 16, padding 16 → target = scrollTop
		expect(
			computeAlignedScrollTop(
				100,
				400,
				scrollHeight,
				{ top: 116, bottom: 136 },
				scroller,
				'start'
			)
		).toBeNull();
	});

	it('start-band skips when line already sits in the upper band', () => {
		// offset 80px into a 400px port → within 35% band (140px), after padding floor
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 180, bottom: 200 },
				scroller,
				'start-band'
			)
		).toBeNull();
	});

	it('start-band scrolls when line is below the band', () => {
		// offset 200px > 140px band
		expect(
			computeAlignedScrollTop(
				50,
				400,
				scrollHeight,
				{ top: 300, bottom: 320 },
				scroller,
				'start-band'
			)
		).toBe(234);
	});
});

it('start barely moves a near-top card while center re-homes it', () => {
	// Repro insight: with card N centered, card N-1 often already sits near the
	// scroller top — start/nearest barely move (or no-op), so the viewport still
	// "belongs" to N; the next off-screen unit then leaps.
	const scroller = { top: 100, bottom: 500 };
	const nearTop = { top: 120, bottom: 280 }; // fully visible, near top
	const startTop = computeAlignedScrollTop(200, 400, 2000, nearTop, scroller, 'start');
	const centerTop = computeAlignedScrollTop(200, 400, 2000, nearTop, scroller, 'center');
	expect(computeAlignedScrollTop(200, 400, 2000, nearTop, scroller, 'nearest')).toBeNull();
	expect(startTop).toBe(204); // only +4px
	expect(centerTop).toBe(100); // -100px — enough to change which card owns center
});

describe('resolveListEndUnit', () => {
	it('returns first step at start when steps exist', () => {
		expect(resolveListEndUnit({ steps: 1400, followUps: 2 }, 'start')).toEqual({
			section: 'steps',
			index: 0
		});
	});

	it('returns last follow-up at end when follow-ups exist', () => {
		expect(resolveListEndUnit({ steps: 1400, followUps: 2 }, 'end')).toEqual({
			section: 'follow-ups',
			index: 1
		});
	});

	it('falls back to last step when there are no follow-ups', () => {
		expect(resolveListEndUnit({ steps: 10, followUps: 0 }, 'end')).toEqual({
			section: 'steps',
			index: 9
		});
	});

	it('falls back to first follow-up at start when steps are empty', () => {
		expect(resolveListEndUnit({ steps: 0, followUps: 3 }, 'start')).toEqual({
			section: 'follow-ups',
			index: 0
		});
	});

	it('returns null when both lengths are zero', () => {
		expect(resolveListEndUnit({ steps: 0, followUps: 0 }, 'start')).toBeNull();
		expect(resolveListEndUnit({ steps: 0, followUps: 0 }, 'end')).toBeNull();
	});
});

describe('resolveViewportUnit (cards)', () => {
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

	type CardStub = {
		getAttribute(name: string): string | null;
		getBoundingClientRect(): DOMRect;
	};

	type ScrollerStub = {
		querySelectorAll(selector: string): CardStub[];
		getBoundingClientRect(): DOMRect;
		clientHeight: number;
		scrollHeight: number;
		scrollTop: number;
	};

	function createScroller(cards: { section: string; index: number; center: number }[]) {
		const cardEls: CardStub[] = cards.map((c) => ({
			getAttribute(name: string) {
				if (name === 'data-card-section') return c.section;
				if (name === 'data-card-index') return String(c.index);
				return null;
			},
			getBoundingClientRect: () =>
				makeRect({ top: c.center - 40, bottom: c.center + 40, height: 80 })
		}));

		const scroller: ScrollerStub = {
			querySelectorAll() {
				return cardEls;
			},
			getBoundingClientRect: () => makeRect({ top: 0, bottom: 400, height: 400 }),
			clientHeight: 400,
			scrollHeight: 2000,
			scrollTop: 0
		};
		return scroller;
	}

	it('uses first mounted card at top when lengths are absent', () => {
		// Virtual window: mounted indices 5–7 only; without hints we wrongly pick 5.
		const scroller = createScroller([
			{ section: 'steps', index: 5, center: 100 },
			{ section: 'steps', index: 6, center: 200 },
			{ section: 'steps', index: 7, center: 300 }
		]);
		scroller.scrollTop = 0;
		expect(resolveViewportUnit(scroller as unknown as HTMLElement, null, CARD_PANE)).toEqual({
			section: 'steps',
			index: 5
		});
	});

	it('uses length hints at top so true list start wins over first mounted', () => {
		const scroller = createScroller([
			{ section: 'steps', index: 5, center: 100 },
			{ section: 'steps', index: 6, center: 200 },
			{ section: 'steps', index: 7, center: 300 }
		]);
		scroller.scrollTop = 0;
		expect(
			resolveViewportUnit(scroller as unknown as HTMLElement, null, CARD_PANE, {
				steps: 1400,
				followUps: 2
			})
		).toEqual({ section: 'steps', index: 0 });
	});

	it('uses length hints at bottom so true list end wins over last mounted', () => {
		const scroller = createScroller([
			{ section: 'steps', index: 1390, center: 100 },
			{ section: 'steps', index: 1391, center: 200 },
			{ section: 'steps', index: 1392, center: 300 }
		]);
		scroller.scrollTop = 1600; // maxScroll = 1600
		expect(
			resolveViewportUnit(scroller as unknown as HTMLElement, null, CARD_PANE, {
				steps: 1400,
				followUps: 2
			})
		).toEqual({ section: 'follow-ups', index: 1 });
	});

	it('uses last mounted card at bottom when lengths are absent', () => {
		const scroller = createScroller([
			{ section: 'steps', index: 1390, center: 100 },
			{ section: 'steps', index: 1391, center: 200 },
			{ section: 'steps', index: 1392, center: 300 }
		]);
		scroller.scrollTop = 1600;
		expect(resolveViewportUnit(scroller as unknown as HTMLElement, null, CARD_PANE)).toEqual({
			section: 'steps',
			index: 1392
		});
	});
});

describe('resolveTopmostVisibleUnit (cards)', () => {
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

	type CardStub = {
		getAttribute(name: string): string | null;
		getBoundingClientRect(): DOMRect;
	};

	function createScroller(
		cards: { section: string; index: number; top: number; bottom: number }[]
	) {
		const cardEls: CardStub[] = cards.map((c) => ({
			getAttribute(name: string) {
				if (name === 'data-card-section') return c.section;
				if (name === 'data-card-index') return String(c.index);
				return null;
			},
			getBoundingClientRect: () =>
				makeRect({ top: c.top, bottom: c.bottom, height: c.bottom - c.top })
		}));

		return {
			querySelectorAll() {
				return cardEls;
			},
			getBoundingClientRect: () => makeRect({ top: 0, bottom: 400, height: 400 }),
			clientHeight: 400,
			scrollHeight: 2000,
			scrollTop: 100
		};
	}

	it('returns null when no cards are mounted', () => {
		const scroller = createScroller([]);
		expect(
			resolveTopmostVisibleUnit(scroller as unknown as HTMLElement, CARD_PANE)
		).toBeNull();
	});

	it('picks the topmost intersecting card over the vertical center', () => {
		// Short card at top; tall middle card owns the viewport center (200).
		const scroller = createScroller([
			{ section: 'steps', index: 0, top: 10, bottom: 90 },
			{ section: 'steps', index: 1, top: 90, bottom: 500 },
			{ section: 'steps', index: 2, top: 500, bottom: 580 }
		]);
		expect(resolveViewportUnit(scroller as unknown as HTMLElement, null, CARD_PANE)).toEqual({
			section: 'steps',
			index: 1
		});
		expect(
			resolveTopmostVisibleUnit(scroller as unknown as HTMLElement, CARD_PANE)
		).toEqual({ section: 'steps', index: 0 });
	});

	it('skips a top-clipped sliver so the next in-port card leads follow', () => {
		// Index 0 barely peeks from above; index 1 has its top inside the port.
		const scroller = createScroller([
			{ section: 'steps', index: 0, top: -60, bottom: 20 },
			{ section: 'steps', index: 1, top: 20, bottom: 400 },
			{ section: 'steps', index: 2, top: 400, bottom: 480 }
		]);
		expect(
			resolveTopmostVisibleUnit(scroller as unknown as HTMLElement, CARD_PANE)
		).toEqual({ section: 'steps', index: 1 });
	});

	it('falls back to a top-clipped card when it is the only intersection', () => {
		const scroller = createScroller([
			{ section: 'steps', index: 0, top: -40, bottom: 30 }
		]);
		expect(
			resolveTopmostVisibleUnit(scroller as unknown as HTMLElement, CARD_PANE)
		).toEqual({ section: 'steps', index: 0 });
	});

	it('ignores cards fully below the viewport', () => {
		const scroller = createScroller([
			{ section: 'steps', index: 0, top: 500, bottom: 580 },
			{ section: 'steps', index: 1, top: 600, bottom: 680 }
		]);
		expect(
			resolveTopmostVisibleUnit(scroller as unknown as HTMLElement, CARD_PANE)
		).toBeNull();
	});

	it('uses length hints at absolute top so true list start wins', () => {
		const scroller = createScroller([
			{ section: 'steps', index: 5, top: 20, bottom: 100 },
			{ section: 'steps', index: 6, top: 100, bottom: 180 }
		]);
		scroller.scrollTop = 0;
		expect(
			resolveTopmostVisibleUnit(scroller as unknown as HTMLElement, CARD_PANE, {
				steps: 1400,
				followUps: 0
			})
		).toEqual({ section: 'steps', index: 0 });
	});
});

describe('scrollUnitIntoView (cards)', () => {
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

	function createCardsScroller(indices: number[]) {
		const cards = new Map<
			string,
			{
				getBoundingClientRect: () => DOMRect;
				focus: ReturnType<typeof vi.fn>;
			}
		>();

		// Place cards below the viewport so center-align must scroll.
		const offscreen = () => makeRect({ top: 800, bottom: 880, height: 80 });

		for (const index of indices) {
			cards.set(`steps:${index}`, {
				getBoundingClientRect: offscreen,
				focus: vi.fn()
			});
		}

		const scroller = {
			querySelector(selector: string) {
				const match = selector.match(
					/\[data-card-section="([^"]+)"\]\[data-card-index="([^"]+)"\]/
				);
				if (!match) return null;
				return cards.get(`${match[1]}:${match[2]}`) ?? null;
			},
			getBoundingClientRect: () => makeRect({ top: 0, bottom: 400, height: 400 }),
			scrollTo: vi.fn(),
			clientHeight: 400,
			scrollHeight: 2000,
			scrollTop: 0,
			mount(unit: ActiveUnit) {
				cards.set(`${unit.section}:${unit.index}`, {
					getBoundingClientRect: offscreen,
					focus: vi.fn()
				});
			}
		};
		return scroller;
	}

	it('returns false when the card is missing and ensureMounted is absent', async () => {
		const scroller = createCardsScroller([0]);
		await expect(
			scrollUnitIntoView(
				scroller as unknown as HTMLElement,
				{ section: 'steps', index: 99 },
				'auto',
				CARD_PANE
			)
		).resolves.toBe(false);
		expect(scroller.scrollTo).not.toHaveBeenCalled();
	});

	it('calls ensureMounted for a missing card then scrolls once mounted', async () => {
		const scroller = createCardsScroller([0]);
		const ensureMounted = vi.fn(async (unit: ActiveUnit) => {
			scroller.mount(unit);
			return true;
		});

		await expect(
			scrollUnitIntoView(
				scroller as unknown as HTMLElement,
				{ section: 'steps', index: 42 },
				'auto',
				CARD_PANE,
				{ ensureMounted, focus: false }
			)
		).resolves.toBe(true);

		expect(ensureMounted).toHaveBeenCalledWith({ section: 'steps', index: 42 });
		expect(scroller.scrollTo).toHaveBeenCalled();
	});

	it('returns false when ensureMounted fails to mount', async () => {
		const scroller = createCardsScroller([0]);
		const ensureMounted = vi.fn(async () => false);

		await expect(
			scrollUnitIntoView(
				scroller as unknown as HTMLElement,
				{ section: 'steps', index: 42 },
				'auto',
				CARD_PANE,
				{ ensureMounted }
			)
		).resolves.toBe(false);

		expect(ensureMounted).toHaveBeenCalledTimes(1);
		expect(scroller.scrollTo).not.toHaveBeenCalled();
	});

	it('does not call ensureMounted when the card is already mounted', async () => {
		const scroller = createCardsScroller([3]);
		const ensureMounted = vi.fn(async () => true);

		await scrollUnitIntoView(
			scroller as unknown as HTMLElement,
			{ section: 'steps', index: 3 },
			'auto',
			CARD_PANE,
			{ ensureMounted, focus: false }
		);

		expect(ensureMounted).not.toHaveBeenCalled();
		expect(scroller.scrollTo).toHaveBeenCalled();
	});
});

describe('watchDrivenScroll', () => {
	function stubEl() {
		const listeners = new Map<string, Set<EventListener>>();
		return {
			addEventListener(type: string, listener: EventListener) {
				if (!listeners.has(type)) listeners.set(type, new Set());
				listeners.get(type)!.add(listener);
			},
			removeEventListener(type: string, listener: EventListener) {
				listeners.get(type)?.delete(listener);
			},
			dispatch(type: string) {
				for (const listener of listeners.get(type) ?? []) {
					listener(new Event(type));
				}
			},
			listenerCount(type: string) {
				return listeners.get(type)?.size ?? 0;
			}
		};
	}

	it('clears via injected clock after auto timeout (120ms)', () => {
		const clock = createFakeClock();
		const el = stubEl();
		const onClear = vi.fn();
		watchDrivenScroll(el as unknown as HTMLElement, 'auto', onClear, clock);
		expect(onClear).not.toHaveBeenCalled();
		clock.flushTimeouts(119);
		expect(onClear).not.toHaveBeenCalled();
		clock.flushTimeouts(1);
		expect(onClear).toHaveBeenCalledTimes(1);
	});

	it('clears on scrollend before timeout and cancels the timer', () => {
		const clock = createFakeClock();
		const el = stubEl();
		const onClear = vi.fn();
		watchDrivenScroll(el as unknown as HTMLElement, 'smooth', onClear, clock);
		el.dispatch('scrollend');
		expect(onClear).toHaveBeenCalledTimes(1);
		clock.flushTimeouts(650);
		expect(onClear).toHaveBeenCalledTimes(1);
	});

	it('uses idleTimeoutMs when provided (Animatable-driven)', () => {
		const clock = createFakeClock();
		const el = stubEl();
		const onClear = vi.fn();
		watchDrivenScroll(el as unknown as HTMLElement, 'auto', onClear, clock, {
			idleTimeoutMs: 360
		});
		clock.flushTimeouts(359);
		expect(onClear).not.toHaveBeenCalled();
		clock.flushTimeouts(1);
		expect(onClear).toHaveBeenCalledTimes(1);
	});
});

describe('resolveViewportUnit (yaml)', () => {
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

	function createYamlScroller(blocks: { section: string; index: number; center: number }[]) {
		const blockEls = blocks.map((b) => ({
			getAttribute(name: string) {
				if (name === 'data-yaml-section') return b.section;
				if (name === 'data-yaml-index') return String(b.index);
				return null;
			},
			getBoundingClientRect: () =>
				makeRect({ top: b.center - 40, bottom: b.center + 40, height: 80 })
		}));

		return {
			querySelectorAll() {
				return blockEls;
			},
			querySelector(selector: string) {
				const match = selector.match(
					/\[data-yaml-section="([^"]+)"\]\[data-yaml-index="([^"]+)"\]/
				);
				if (!match) return null;
				return (
					blockEls.find(
						(el) =>
							el.getAttribute('data-yaml-section') === match[1] &&
							el.getAttribute('data-yaml-index') === match[2]
					) ?? null
				);
			},
			getBoundingClientRect: () => makeRect({ top: 0, bottom: 400, height: 400 }),
			clientHeight: 400,
			scrollHeight: 2000,
			scrollTop: 100
		};
	}

	it('picks the yaml block nearest the viewport center', () => {
		const scroller = createYamlScroller([
			{ section: 'steps', index: 0, center: 80 },
			{ section: 'steps', index: 1, center: 220 },
			{ section: 'steps', index: 2, center: 360 }
		]);
		expect(resolveViewportUnit(scroller as unknown as HTMLElement, null, YAML_PANE)).toEqual({
			section: 'steps',
			index: 1
		});
	});

	it('returns null when no yaml block intersects the viewport (header-only)', () => {
		const scroller = createYamlScroller([
			{ section: 'steps', index: 0, center: 800 },
			{ section: 'steps', index: 1, center: 900 }
		]);
		scroller.scrollTop = 0;
		expect(resolveViewportUnit(scroller as unknown as HTMLElement, null, YAML_PANE)).toBeNull();
	});
});

describe('scrollUnitIntoView (yaml)', () => {
	it('calls ensureMounted when the yaml step block is missing', async () => {
		const children: Array<{
			getAttribute(name: string): string | null;
			getBoundingClientRect(): DOMRect;
			focus: ReturnType<typeof vi.fn>;
		}> = [];

		const scroller = {
			querySelector(selector: string) {
				const match = selector.match(
					/\[data-yaml-section="([^"]+)"\]\[data-yaml-index="([^"]+)"\]/
				);
				if (!match) return null;
				return (
					children.find(
						(c) =>
							c.getAttribute('data-yaml-section') === match[1] &&
							c.getAttribute('data-yaml-index') === match[2]
					) ?? null
				);
			},
			getBoundingClientRect: () =>
				({
					top: 0,
					bottom: 400,
					height: 400,
					left: 0,
					right: 100,
					width: 100,
					x: 0,
					y: 0,
					toJSON() {
						return this;
					}
				}) as DOMRect,
			clientHeight: 400,
			scrollHeight: 2000,
			scrollTop: 0,
			scrollTo: vi.fn()
		};

		const ensureMounted = vi.fn(async (unit: ActiveUnit) => {
			children.push({
				getAttribute(name: string) {
					if (name === 'data-yaml-section') return unit.section;
					if (name === 'data-yaml-index') return String(unit.index);
					return null;
				},
				getBoundingClientRect: () =>
					({
						top: 800,
						bottom: 880,
						height: 80,
						left: 0,
						right: 100,
						width: 100,
						x: 0,
						y: 800,
						toJSON() {
							return this;
						}
					}) as DOMRect,
				focus: vi.fn()
			});
			return true;
		});

		await expect(
			scrollUnitIntoView(
				scroller as unknown as HTMLElement,
				{ section: 'steps', index: 4 },
				'auto',
				YAML_PANE,
				{ ensureMounted, focus: false }
			)
		).resolves.toBe(true);
		expect(ensureMounted).toHaveBeenCalledWith({ section: 'steps', index: 4 });
		expect(scroller.scrollTo).toHaveBeenCalled();
	});

	it('prefers Animatable scrollTo over native scrollTo when provided', async () => {
		const children: Array<{
			getAttribute(name: string): string | null;
			getBoundingClientRect(): DOMRect;
			focus: ReturnType<typeof vi.fn>;
		}> = [
			{
				getAttribute(name: string) {
					if (name === 'data-yaml-section') return 'steps';
					if (name === 'data-yaml-index') return '0';
					return null;
				},
				getBoundingClientRect: () =>
					({
						top: 800,
						bottom: 880,
						height: 80,
						left: 0,
						right: 100,
						width: 100,
						x: 0,
						y: 800,
						toJSON() {
							return this;
						}
					}) as DOMRect,
				focus: vi.fn()
			}
		];

		const scroller = {
			querySelector() {
				return children[0] ?? null;
			},
			getBoundingClientRect: () =>
				({
					top: 0,
					bottom: 400,
					height: 400,
					left: 0,
					right: 100,
					width: 100,
					x: 0,
					y: 0,
					toJSON() {
						return this;
					}
				}) as DOMRect,
			clientHeight: 400,
			scrollHeight: 2000,
			scrollTop: 0,
			scrollTo: vi.fn()
		};

		const animatableScroll = {
			scrollTo: vi.fn(),
			getScrollTop: () => 0,
			dispose: vi.fn(),
			animatable: {} as never
		};

		await expect(
			scrollUnitIntoView(
				scroller as unknown as HTMLElement,
				{ section: 'steps', index: 0 },
				'auto',
				YAML_PANE,
				{ focus: false, animatableScroll, durationMs: 280 }
			)
		).resolves.toBe(true);

		expect(animatableScroll.scrollTo).toHaveBeenCalledWith(expect.any(Number), 280);
		expect(scroller.scrollTo).not.toHaveBeenCalled();
	});
});
