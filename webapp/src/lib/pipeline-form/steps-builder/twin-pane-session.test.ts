// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import type { ComposerVirtualizer } from './composer-virtualizer.svelte.js';
import type { PairedShiftArgs } from './paired-shift.js';
import type { ActiveUnit } from './scroll-follow/active-unit.js';

import { createTwinPaneSession } from './twin-pane-session.svelte.js';

function fakeVirtualizer(
	label: string,
	order: string[],
	overrides: Partial<ComposerVirtualizer> = {}
): ComposerVirtualizer {
	return {
		virtualizer: {
			subscribe: () => () => {}
		} as ComposerVirtualizer['virtualizer'],
		getVirtualItems: () => [],
		getTotalSize: () => 0,
		measureElement: () => {},
		syncAfterReorder: () => {
			order.push(`sync:${label}`);
		},
		ensureStepVisible: vi.fn(async (index: number) => {
			order.push(`ensure:${label}:${index}`);
			return true;
		}),
		dispose: () => {
			order.push(`dispose:${label}`);
		},
		...overrides
	};
}

function baseOptions(order: string[], extras: Record<string, unknown> = {}) {
	const stepsVirt = fakeVirtualizer('cards', order);
	const yamlVirt = fakeVirtualizer('yaml', order);
	const notePairedReorder = vi.fn(() => {
		order.push('mute');
	});
	const peerDispose = vi.fn(() => {
		order.push('dispose:peer');
	});
	const highlightDispose = vi.fn(() => {
		order.push('dispose:highlight');
	});
	const layoutDispose = vi.fn(() => {
		order.push('dispose:layout');
	});
	let boundHandlers: {
		onRevealStep?: (index: number) => void;
		onEditFocus?: (stepIndex: number) => void;
	} = {};
	let virtCalls = 0;

	return {
		order,
		stepsVirt,
		yamlVirt,
		notePairedReorder,
		peerDispose,
		highlightDispose,
		layoutDispose,
		getBoundHandlers: () => boundHandlers,
		options: {
			getStepsCount: () => 4,
			getYamlStepsCount: () => 4,
			getItemKey: (index: number) => index,
			getIsManual: () => false,
			getEditingIndex: () => undefined as number | undefined,
			getEditingSection: () => 'steps' as const,
			getYamlPreview: () => 'steps:\n  - id: a\n',
			getFollowUpsLength: () => 0,
			getCreatedCard: () => null,
			getYamlScrollMargin: () => 0,
			canShiftStep: () => true,
			mutateShiftStep: () => {
				order.push('mutate');
			},
			bindComposerScroll: (handlers: typeof boundHandlers) => {
				boundHandlers = handlers;
				order.push(Object.keys(handlers).length === 0 ? 'unbind' : 'bind');
			},
			createComposerVirtualizer: () => {
				virtCalls += 1;
				order.push(`virt:${virtCalls === 1 ? 'cards' : 'yaml'}`);
				return virtCalls === 1 ? stepsVirt : yamlVirt;
			},
			createPeerScrollFollow: () =>
				({
					enabled: true,
					notePairedReorder,
					dispose: peerDispose,
					setEnabled: vi.fn(),
					onEditFocus: vi.fn(),
					onReveal: vi.fn(),
					followUnit: vi.fn(),
					onYamlTextChanged: vi.fn(() => () => {}),
					cardsAttach: () => {},
					yamlAttach: () => {}
				}) as never,
			createUnitHighlight: () =>
				({
					pinUnit: (unit: ActiveUnit) => unit,
					hoverCard: vi.fn(),
					clearHover: vi.fn(),
					clearHoverCard: vi.fn(),
					isCardSelected: () => false,
					isCardHovered: () => false,
					dispose: highlightDispose
				}) as never,
			createMultiListLayout: () =>
				({
					run: async (fn: () => void | Promise<void>) => {
						await fn();
					},
					dispose: layoutDispose
				}) as never,
			tick: async () => {
				order.push('tick');
			},
			restoreScrollTop: () => {
				order.push('restore');
			},
			...extras
		}
	};
}

describe('createTwinPaneSession', () => {
	it('shiftStep no-ops when canShiftStep is false', async () => {
		const order: string[] = [];
		const runPairedShift = vi.fn(async () => {
			order.push('paired');
		});
		const { options } = baseOptions(order, {
			canShiftStep: () => false,
			runPairedShift
		});
		const session = createTwinPaneSession(options);

		await session.shiftStep(1, 1);

		expect(runPairedShift).not.toHaveBeenCalled();
		expect(order).not.toContain('paired');
		session.dispose();
	});

	it('shiftStep owns mute → mount → mutate/sync via runPairedShift wiring', async () => {
		const order: string[] = [];
		const { options, stepsVirt, yamlVirt, notePairedReorder } = baseOptions(order, {
			runPairedShift: async (args: PairedShiftArgs) => {
				order.push(`paired:${args.fromIndex}->${args.toIndex}`);
				args.notePairedReorder();
				await args.ensureSwapMounted(args.fromIndex, args.toIndex);
				args.mutate();
				args.syncBoth();
				args.pinScrollports(11, 22);
			}
		});
		const session = createTwinPaneSession(options);
		session.cardsScroller = { scrollTop: 40 } as HTMLElement;
		session.yamlScroller = { scrollTop: 50 } as HTMLElement;

		await session.shiftStep(2, 1);

		expect(notePairedReorder).toHaveBeenCalledOnce();
		expect(stepsVirt.ensureStepVisible).toHaveBeenCalledWith(2, {
			behavior: 'auto',
			align: 'start',
			mountOnly: true
		});
		expect(stepsVirt.ensureStepVisible).toHaveBeenCalledWith(3, {
			behavior: 'auto',
			align: 'start',
			mountOnly: true
		});
		expect(yamlVirt.ensureStepVisible).toHaveBeenCalledWith(2, {
			behavior: 'auto',
			align: 'start',
			mountOnly: true
		});
		expect(yamlVirt.ensureStepVisible).toHaveBeenCalledWith(3, {
			behavior: 'auto',
			align: 'start',
			mountOnly: true
		});
		expect(order).toEqual([
			'virt:cards',
			'virt:yaml',
			'bind',
			'paired:2->3',
			'mute',
			'ensure:cards:2',
			'ensure:cards:3',
			'ensure:yaml:2',
			'ensure:yaml:3',
			'restore',
			'restore',
			'tick',
			'mutate',
			'sync:cards',
			'sync:yaml',
			'restore',
			'restore'
		]);
		session.dispose();
	});

	it('mountSwapIndices keeps mid-list scrollTop when pair units are already mounted', async () => {
		/**
		 * Guilty writer without mountOnly: ensureStepVisible → scrollToIndex(align:start)
		 * (+ TanStack reconcile) toward ≈0. With mountOnly, already-mounted units must not
		 * schedule that scroll; cards scrollTop stays at the pre-swap pin (e.g. 200).
		 */
		let cardsTop = 200;
		let yamlTop = 200;
		const order: string[] = [];
		const cardsScroller = {
			get scrollTop() {
				return cardsTop;
			},
			set scrollTop(v: number) {
				cardsTop = v;
			},
			querySelector(sel: string) {
				const m = /data-card-index="(\d+)"/.exec(sel);
				return m && (m[1] === '0' || m[1] === '1') ? ({} as Element) : null;
			}
		} as HTMLElement;
		const yamlScroller = {
			get scrollTop() {
				return yamlTop;
			},
			set scrollTop(v: number) {
				yamlTop = v;
			},
			querySelector(sel: string) {
				const m = /data-yaml-index="(\d+)"/.exec(sel);
				return m && (m[1] === '0' || m[1] === '1') ? ({} as Element) : null;
			}
		} as HTMLElement;

		const stepsVirt = fakeVirtualizer('cards', order, {
			ensureStepVisible: vi.fn(async (index: number, opts?: { mountOnly?: boolean }) => {
				order.push(`ensure:cards:${index}`);
				if (opts?.mountOnly && cardsScroller.querySelector(`[data-card-index="${index}"]`)) {
					return true;
				}
				// Legacy align:start yank toward list start (the smoke bug writer).
				cardsTop = 0;
				return true;
			})
		});
		const yamlVirt = fakeVirtualizer('yaml', order, {
			ensureStepVisible: vi.fn(async (index: number, opts?: { mountOnly?: boolean }) => {
				order.push(`ensure:yaml:${index}`);
				if (opts?.mountOnly && yamlScroller.querySelector(`[data-yaml-index="${index}"]`)) {
					return true;
				}
				yamlTop = 0;
				return true;
			})
		});

		const { options } = baseOptions(order, {
			createComposerVirtualizer: (() => {
				let n = 0;
				return () => {
					n += 1;
					order.push(`virt:${n === 1 ? 'cards' : 'yaml'}`);
					return n === 1 ? stepsVirt : yamlVirt;
				};
			})(),
			runPairedShift: async (args: PairedShiftArgs) => {
				args.notePairedReorder();
				await args.ensureSwapMounted(args.fromIndex, args.toIndex);
				expect(cardsScroller.scrollTop).toBe(200);
				expect(yamlScroller.scrollTop).toBe(200);
				args.mutate();
				args.syncBoth();
			},
			restoreScrollTop: (el, top) => {
				order.push(`restore:${top}`);
				if (el) el.scrollTop = top;
			}
		});
		const session = createTwinPaneSession(options);
		session.cardsScroller = cardsScroller;
		session.yamlScroller = yamlScroller;

		await session.shiftStep(0, 1);

		expect(cardsTop).toBe(200);
		expect(yamlTop).toBe(200);
		expect(stepsVirt.ensureStepVisible).toHaveBeenCalledWith(0, expect.objectContaining({ mountOnly: true }));
		expect(stepsVirt.ensureStepVisible).toHaveBeenCalledWith(1, expect.objectContaining({ mountOnly: true }));
		session.dispose();
	});

	it('dispose cleans peer, highlight, virtualizers, layout, and unbinds composer scroll', () => {
		const order: string[] = [];
		const { options } = baseOptions(order);
		const session = createTwinPaneSession(options);

		session.dispose();

		expect(order).toContain('unbind');
		expect(order).toContain('dispose:peer');
		expect(order).toContain('dispose:highlight');
		expect(order).toContain('dispose:cards');
		expect(order).toContain('dispose:yaml');
		// layout may be null until roots attach — dispose is still safe
		session.dispose(); // idempotent
	});

	it('onUnitClick pins then follows the peer pane', () => {
		const followUnit = vi.fn();
		const pinUnit = vi.fn((unit: ActiveUnit) => unit);
		const order: string[] = [];
		const { options } = baseOptions(order, {
			createPeerScrollFollow: () =>
				({
					enabled: true,
					notePairedReorder: vi.fn(),
					dispose: vi.fn(),
					setEnabled: vi.fn(),
					onEditFocus: vi.fn(),
					onReveal: vi.fn(),
					followUnit,
					onYamlTextChanged: vi.fn(() => () => {}),
					cardsAttach: () => {},
					yamlAttach: () => {}
				}) as never,
			createUnitHighlight: () =>
				({
					pinUnit,
					hoverCard: vi.fn(),
					clearHover: vi.fn(),
					clearHoverCard: vi.fn(),
					isCardSelected: () => false,
					isCardHovered: () => false,
					dispose: vi.fn()
				}) as never
		});
		const session = createTwinPaneSession(options);
		const unit: ActiveUnit = { section: 'steps', index: 1 };

		session.onUnitClick(unit, 'cards');

		expect(pinUnit).toHaveBeenCalledWith(unit);
		expect(followUnit).toHaveBeenCalledWith(unit, 'cards');
		session.dispose();
	});
});
